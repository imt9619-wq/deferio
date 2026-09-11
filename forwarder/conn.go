package forwarder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/deferio/diohandler/internel"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	HeaderByteSize = 9
	PacketLenghtByteSize = 4
	ServerID = (1 << 15) - 1 
)

type ForwarderConfig struct{
	FlushRate       time.Duration
    ConnF           func(addess string) (net.Conn, error)
    Address         string
    BytePerWrite    int
    MaxBufferedByte int
}

type Conn struct{
	net.Conn
    conf      *ForwarderConfig
    closeOnce *sync.Once
    close     chan struct{}

	sendBufMu             *sync.Mutex
    sendBuf, sendBufSpare [][]byte
    sendBufLen            int
	flushNow              chan struct{}
	
	done      chan struct{}
    shieldIDsetOnce *sync.Once
    shieldID        int32

    idMu        *sync.Mutex
    emptyIdSlot []int
    idToXuid    []uint64
}

func (f ForwarderConfig) Dial() (*Conn, error){
	if f.FlushRate == 0{
		f.FlushRate = time.Millisecond * 50
	}
	if f.Address == ""{
		f.Address = "127.0.0.1:19135"
	}
	if f.ConnF == nil{
		f.ConnF = func(addess string) (net.Conn, error){
			return net.Dial("tcp", addess)
		}
	}
	if f.BytePerWrite == 0{
		f.BytePerWrite = 16 * 1024
	}
	if f.MaxBufferedByte == 0{
		f.MaxBufferedByte = 4 * f.BytePerWrite
	}
	conn, err := f.ConnF(f.Address)
	if err != nil{
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	c := &Conn{
		Conn: conn,
		conf: &f,
		closeOnce: &sync.Once{},
		close: make(chan struct{}),
		sendBufMu: &sync.Mutex{},
		sendBuf: make([][]byte, 0, 4096),
		sendBufSpare: make([][]byte, 0, 4096),
		flushNow: make(chan struct{}, 1),
		shieldIDsetOnce: &sync.Once{},
		idMu: &sync.Mutex{},
		emptyIdSlot: make([]int, 0, 128),
		idToXuid: make([]uint64, 0, 128),
		done: make(chan struct{}),
	}
	go c.flushLoop()
	c.forwardPacket(&NewDialPacket{serverTime: time.Now().Unix()}, ServerID, SourceDioHandlerPacket)
	return c, nil
}

func (c *Conn) Close() error {
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

	pk.pk.Marshal(protocol.NewWriter(buf, c.shieldID))
	raw := make([]byte, PacketLenghtByteSize, PacketLenghtByteSize+len(buf.Bytes()))
	binary.BigEndian.PutUint32(raw[0:4], uint32(len(buf.Bytes())))
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

func (c *Conn) forwardPacket(pk ForwardPacket, id uint16, source uint8) error{
	hdr := Header{
		id:         id,
		timeOffset: TimeSinceToday(),
		packetID:   pk.ID(),
		source:     source,
	}
	_, ok := pk.(DioPacket)
	hdr.dioPacket = ok
	return c.writePacket(&PacketWrapper{
		pk:  pk,
		hdr: &hdr,
	})
}

func TimeSinceToday() time.Duration {
	now := time.Now().UTC()
	return now.Sub(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
}