package forwarder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deferio/diohandler/internel"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	HeaderByteSize = 9
	PacketLenghtByteSize = 4
	ServerID = (1 << 15) - 1
	maxFrameBytes = 16 << 20
)

type ForwarderConfig struct{
	FlushRate       time.Duration
    Address         string
    BytePerWrite    int
    MaxBufferedByte int
}

type DialConfig struct{
	ForwarderConfig
	DialF func(address string) (net.Conn, error)
}

func (f ForwarderConfig) defaultForwarderConfig() ForwarderConfig{
	if f.FlushRate == 0{
		f.FlushRate = time.Millisecond * 50
	}
	if f.Address == ""{
		f.Address = "127.0.0.1:19135"
	}
	if f.BytePerWrite == 0{
		f.BytePerWrite = 16 * 1024
	}
	if f.MaxBufferedByte == 0{
		f.MaxBufferedByte = 4 * f.BytePerWrite
	}
	return f
}

type Conn struct{
	net.Conn
    conf      *ForwarderConfig
    closeOnce *sync.Once
    close     chan struct{}
	done      chan struct{}

	sendBufMu             *sync.Mutex
    sendBuf, sendBufSpare [][]byte
	sendBufLen            int
	flushNow              chan struct{}

	readBuf     []byte
    shieldID    *atomic.Int32
    shieldIDSet *atomic.Bool
}

func (d DialConfig) dial() (*Conn, error){
	if d.Address == ""{
		d.Address = "127.0.0.1:19135"
	}
	if d.DialF == nil{
		d.DialF = func(address string) (net.Conn, error){
			return net.Dial("unix", address)
		}
	}
	d.ForwarderConfig = d.defaultForwarderConfig()
	conn, err := d.DialF(d.Address)
	if err != nil{
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	return d.newConn(conn), nil
}

func (f ForwarderConfig) newConn(conn net.Conn) *Conn{
	c := &Conn{
		Conn: conn,
		conf: &f,
		closeOnce: &sync.Once{},
		close: make(chan struct{}),
		sendBufMu: &sync.Mutex{},
		sendBuf: make([][]byte, 0, 4096),
		sendBufSpare: make([][]byte, 0, 4096),
		flushNow: make(chan struct{}, 1),
		done: make(chan struct{}),
		shieldID: &atomic.Int32{},
		shieldIDSet: &atomic.Bool{},
	}
	go c.flushLoop()
	return c
}

func (c *Conn) Close() error{
	c.closeOnce.Do(func(){
		close(c.close)
	})
	<-c.done
	return nil
}

func (c *Conn) flushLoop(){
	ticker := time.NewTicker(c.conf.FlushRate)
	lastWrite := time.Now()
	defer ticker.Stop()
	defer close(c.done)
	for{
		select{
		case <-c.close:
			_ = c.Flush()
			_ = c.Conn.Close()
			return
		case t := <-ticker.C:
			if t.Before(lastWrite.Add(c.conf.FlushRate)){
				continue
			}
			if err := c.Flush(); err != nil{
				c.closeOnce.Do(func(){close(c.close)})
				_ = c.Conn.Close()
				return
			}
			lastWrite = time.Now()
		// we flush right away before tick flush if we are buffering a large amount of bytes
		case <-c.flushNow:
			if err := c.Flush(); err != nil{
				c.closeOnce.Do(func(){close(c.close)})
				_ = c.Conn.Close()
				return
			}
			lastWrite = time.Now()
		}
	}
}

func (c *Conn) writePacket(pk *PacketWrapper) error{
	select {
	case <-c.close:
		return fmt.Errorf("Forwarder Conn: trying to write packet on closed Conn")
	default:
	}

	buf := internal.BufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		internal.BufferPool.Put(buf)
	}()
	buf.Reset()
	err := pk.hdr.Write(buf)
	if err != nil{
		return err
	}

	pk.pk.Marshal(protocol.NewWriter(buf, c.shieldID.Load()))
	raw := make([]byte, PacketLenghtByteSize, PacketLenghtByteSize+len(buf.Bytes()))
	binary.BigEndian.PutUint32(raw[0:PacketLenghtByteSize], uint32(len(buf.Bytes())))
	frame := append(raw, buf.Bytes()...)

	c.sendBufMu.Lock()
	c.sendBuf = append(c.sendBuf, frame)
	c.sendBufLen += len(frame)
	flushNow := c.sendBufLen >= c.conf.MaxBufferedByte
	c.sendBufMu.Unlock()

	if flushNow{
		select {
		case c.flushNow <- struct{}{}:
		default:
		}
	}
	return nil
} 

// most copied from gophertunnel/minecraft.(*Conn).Flush()
func (c *Conn) Flush() error{
	c.sendBufMu.Lock()
	if len(c.sendBuf) == 0{
		c.sendBufMu.Unlock()
		return nil
	}
	send := c.sendBuf
	c.sendBuf = c.sendBufSpare[:0]
	c.sendBufSpare = nil
	c.sendBufLen = 0
	c.sendBufMu.Unlock()
	buf := internal.BufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		internal.BufferPool.Put(buf)
		for i := range send {
			send[i] = nil
		}
		c.sendBufMu.Lock()
		c.sendBufSpare = send[:0]
		c.sendBufMu.Unlock()
	}()
	buf.Reset()
	for _, pk := range send{
		if buf.Len() > 0 && buf.Len()+len(pk) > c.conf.BytePerWrite{
			if _, err := c.Write(buf.Bytes()); err != nil{
				return err
			}
			buf.Reset()
		}
		if len(pk) > c.conf.BytePerWrite{
			if _, err := c.Write(pk); err != nil{
				return err
			}
			continue
		}
		buf.Write(pk)
	}
	if buf.Len() > 0{
		_, err := c.Write(buf.Bytes())
		return err
	}
	return nil
}

func (c *Conn) forwardPacket(pk *PacketWrapper, id uint16, source uint8) error{
	pk.hdr.id = id
	pk.hdr.packetID = pk.pk.ID()
	pk.hdr.source = source
	_, ok := pk.pk.(DioPacket)
	pk.hdr.dioPacket = ok
	return c.writePacket(pk)
}

func (c *Conn) readPacket() (*PacketWrapper, error){
	var lenBuf [PacketLenghtByteSize]byte
	if _, err := io.ReadFull(c.Conn, lenBuf[:]); err != nil{
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:]))
	if n < HeaderByteSize || n > maxFrameBytes{
		return nil, fmt.Errorf("forwarder: bad frame length %d", n)
	}
	if cap(c.readBuf) < n{
		c.readBuf = make([]byte, n)
	}else{
		c.readBuf = c.readBuf[:n]
	}
	if _, err := io.ReadFull(c.Conn, c.readBuf); err != nil{
		return nil, err
	}

	body := bytes.NewBuffer(c.readBuf)
	hdr := &Header{}
	if err := hdr.Read(body); err != nil{
		return nil, err
	}
	pk, err := packetByHeader(hdr)
	if err != nil{
		return nil, err
	}
	pk.Marshal(protocol.NewReader(body, c.shieldID.Load(), false))
	return &PacketWrapper{pk: pk, hdr: hdr}, nil
}

func (c *Conn) setShieldID(pk *IncomingPlayerPacket){
	if c.shieldIDSet.Load(){
		return
	}
	if pk.data != nil{
		for _, it := range pk.data.Items{
			if it.Name == "minecraft:shield"{
				c.shieldID.Store(int32(it.RuntimeID))
				c.shieldIDSet.Store(true)
				return
			}
		}
	}
}