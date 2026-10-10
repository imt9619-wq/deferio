package forwarder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deferio/internal"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	PacketDataByteSize = 7
	IDByteSize = 2
	PacketLenghtByteSize = 4
	ServerID = (1 << 15) - 1
	maxFrameBytes = 16 << 20
)

type ConnConfig struct{
	FlushRate       time.Duration
    Address         string
    BytePerWrite    int
    MaxBufferedByte int
	Log             slog.Logger
	// will be called when Conn is closing, can return a fallback function to run after closing
	onClose         func()(onAfterClose func())
}

func (f ConnConfig) defaultConnConfig() ConnConfig{
	if f.FlushRate == 0{
		f.FlushRate = time.Millisecond * 50
	}
	if f.Address == ""{
		f.Address = filepath.Join(os.TempDir(), "deferio_anticheat_server.sock")
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
	conn      net.Conn
    conf      ConnConfig
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

func (f ConnConfig) getEmptyConn() *Conn{
	return &Conn{
		conf: f,
		sendBufMu: &sync.Mutex{},
		sendBuf: make([][]byte, 0, 4096),
		sendBufSpare: make([][]byte, 0, 4096),
		flushNow: make(chan struct{}, 1),
		shieldID: &atomic.Int32{},
		shieldIDSet: &atomic.Bool{},
	}
}

func (c *Conn) newNetConn(conn net.Conn){
	c.reset()
	c.conn = conn
	go c.flushLoop()
}

func (c *Conn) reset(){
	c.conn = nil
	c.closeOnce = &sync.Once{}
	c.close = make(chan struct{})
	c.sendBufMu.Lock()
	c.sendBuf = c.sendBuf[:0]
	c.sendBufLen = 0
	c.sendBufMu.Unlock()
}

func (c *Conn) Close(){
	c.closeOnce.Do(func(){
		var onAfterClose func()
		if c.conf.onClose != nil{
			onAfterClose = c.conf.onClose()
		}
		close(c.close)
		if c.done != nil{
			<-c.done
		}
		if c.conn != nil{
			_ = c.conn.Close()
			c.conn = nil
		}
		if onAfterClose != nil{
			go onAfterClose()
		}
	})
}

func (c *Conn) flushLoop(){
	ticker := time.NewTicker(c.conf.FlushRate)
	lastWrite := time.Now()
	c.done = make(chan struct{})
	defer func(){
		ticker.Stop()
		close(c.done)
		c.Close()
	}()
	for{
		select{
		case <-c.close:
			_ = c.flush()
			return
		case t := <-ticker.C:
			if t.Before(lastWrite.Add(c.conf.FlushRate)){
				continue
			}
			if err := c.flush(); err != nil{
				return
			}
			lastWrite = time.Now()
		// we flush right away before tick flush if we are buffering a large amount of bytes
		case <-c.flushNow:
			if err := c.flush(); err != nil{
				return
			}
			lastWrite = time.Now()
		}
	}
}

func (c *Conn) Flush(){
	select{
	case c.flushNow <- struct{}{}:
	default:
	}
}

func isIOError(err error) bool{
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[encodeError](err); ok {
		return false
	}
	if _, ok := errors.AsType[decodeError](err); ok {
		return false
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return true
	}
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, net.ErrClosed)
}

func (c *Conn) writePacket(pk *PacketWrapper) (err error){
	select{
	case <-c.close:
		return fmt.Errorf("Forwarder Conn: trying to write packet on closed Conn")
	default:
	}

	buf := internal.BufferPool.Get().(*bytes.Buffer)
	defer func(){
		buf.Reset()
		internal.BufferPool.Put(buf)
		if err != nil && isIOError(err){
			c.Close()
		}
	}()
	buf.Reset()
	buf.Write([]byte{byte(pk.id >> 8), byte(pk.id)})
	if err = pk.data.Write(buf); err != nil{
		return err
	}
	if err = c.encodePacket(pk.pk, buf); err != nil{
		return err
	}

	raw := make([]byte, PacketLenghtByteSize, PacketLenghtByteSize+len(buf.Bytes()))
	binary.BigEndian.PutUint32(raw[:PacketLenghtByteSize], uint32(len(buf.Bytes())))
	frame := append(raw, buf.Bytes()...)
	c.sendBufMu.Lock()

	select{
	case <-c.close:
		c.sendBufMu.Unlock()
		return fmt.Errorf("Forwarder Conn: trying to write packet on closed Conn")
	default:
	}

	c.sendBuf = append(c.sendBuf, frame)
	c.sendBufLen += len(frame)
	flushNow := c.sendBufLen >= c.conf.MaxBufferedByte
	c.sendBufMu.Unlock()
	
	if flushNow{
		select{
		case c.flushNow <- struct{}{}:
		default:
		}
	}
	return nil
}

type encodeError struct{error}
func (c *Conn) encodePacket(pk ForwardPacket, buf *bytes.Buffer) (err error){
	defer func(){
		if r := recover(); r != nil{
			if e, ok := r.(error); ok{
				err = encodeError{fmt.Errorf("encode packet %T: %w", pk, e)}
			}else{
				err = encodeError{fmt.Errorf("encode packet %T: %v", pk, r)}
			}
		}
	}()
	pk.Marshal(protocol.NewWriter(buf, c.shieldID.Load()))
	return nil
}

// most copied from gophertunnel/minecraft.(*Conn).Flush()
func (c *Conn) flush() error{
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
			if _, err := c.conn.Write(buf.Bytes()); err != nil{
				return err
			}
			buf.Reset()
		}
		if len(pk) > c.conf.BytePerWrite{
			if _, err := c.conn.Write(pk); err != nil{
				return err
			}
			continue
		}
		buf.Write(pk)
	}
	if buf.Len() > 0{
		_, err := c.conn.Write(buf.Bytes())
		return err
	}
	return nil
}

func (c *Conn) forwardPacket(pk *PacketWrapper, id uint16, source uint8) error{
	pk.id = id
	pk.data.packetID = pk.pk.ID()
	pk.data.source = source
	_, ok := pk.pk.(dioPacket)
	pk.data.dioPacket = ok
	return c.writePacket(pk)
}

func (c *Conn) readPacketRaw() (PacketInRaw, error){
	var lenBuf [PacketLenghtByteSize]byte
	if _, err := io.ReadFull(c.conn, lenBuf[:]); err != nil{
		return PacketInRaw{}, err
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:]))
	if n < PacketDataByteSize + IDByteSize || n > maxFrameBytes{
		return PacketInRaw{}, fmt.Errorf("forwarder: bad frame length %d", n)
	}
	if cap(c.readBuf) < n{
		c.readBuf = make([]byte, n)
	}else{
		c.readBuf = c.readBuf[:n]
	}
	if _, err := io.ReadFull(c.conn, c.readBuf); err != nil{
		return PacketInRaw{}, err
	}
	raw := make([]byte, (n - IDByteSize))
	copy(raw, c.readBuf[IDByteSize:])
	return PacketInRaw{
		id: binary.BigEndian.Uint16(c.readBuf[:IDByteSize]),
		raw: raw,
	}, nil
}

func (c *Conn) readPacketFromRaw(raw PacketInRaw) (*PacketWrapper, error){
	body := bytes.NewBuffer(raw.raw)
	data := &PacketData{}
	if err := data.Read(body); err != nil{
		return nil, err
	}
	pk, err := packetByHeader(data)
	if err != nil{
		return nil, err
	}
	if err := c.decodePacket(pk, body); err != nil{
		return nil, err
	}
	return &PacketWrapper{pk: pk, data: data, id: raw.id}, nil
}

func (c *Conn) readPacket() (*PacketWrapper, error){
	new, err := c.readPacketRaw()
	if err != nil{
		return nil, err
	}
	return c.readPacketFromRaw(new)
}

type decodeError struct{error}
func (c *Conn) decodePacket(pk ForwardPacket, body *bytes.Buffer) (err error){
	defer func(){
		if r := recover(); r != nil {
			if e, ok := r.(error); ok{
				err = decodeError{fmt.Errorf("Decode packet %T: %w", pk, e)}
			}else{
				err = decodeError{fmt.Errorf("Decode packet %T: %v", pk, r)}
			}
		}
	}()
	pk.Marshal(protocol.NewReader(body, c.shieldID.Load(), false))
	return nil
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