package forwarder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	internal "github.com/deferio/diohandler/internel"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	HeaderByteSize = 20
	PacketLenghtByteSize = 4
)

type ForwarderConfig struct{
	FlushRate time.Duration
    ConnF     func(addess string) (net.Conn, error)
    Address   string
}

type Conn struct{
	net.Conn
    conf      *ForwarderConfig
    closeOnce *sync.Once
    close     chan struct{}
    sendBuf   *sendBuffer

    shieldIDsetOnce *sync.Once
    shieldID        int32
}

func (f ForwarderConfig) Dial() (*Conn, error){
	if f.FlushRate == 0{
		f.FlushRate = time.Millisecond * 50
	}
	if f.Address == ""{
		f.Address = "127.0.0.1:19135"
	}
	if f.ConnF == nil{
		f.ConnF = func(addess string) (net.Conn, error) {
			return net.Dial("tcp", addess)
		}
	}
	conn, err := f.ConnF(f.Address)
	if err != nil {
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	c := &Conn{
		Conn: conn,
		conf: &f,
		closeOnce: &sync.Once{},
		close: make(chan struct{}),
		sendBuf: newSendBuf(),
		shieldIDsetOnce: &sync.Once{},
	}
	go c.flushLoop()
	return c, nil
}

func (c *Conn) Close() error{
	c.closeOnce.Do(func() {
		close(c.close)
	})
	return c.Conn.Close()
}

func (c *Conn) flushLoop(){
	ticker := time.NewTicker(c.conf.FlushRate)
	defer ticker.Stop()
	for{
		select{
		case <-c.close:
			c.Flush()
			return
		case <-ticker.C:
			err := c.Flush()
			if err != nil{
				c.Close()
			}
		}
	}
}

func (c *Conn) writePacket(pk *PacketWrapper) error{
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
	c.sendBuf.append(append(raw, buf.Bytes()...))
	return nil
} 

func (c *Conn) Flush() error{
	for pk := range c.sendBuf.swapAndSend(){
		_, err := c.Write(pk)
		if err != nil{
			return err
		}
	}
	return nil
}

func (c *Conn) SetShieldIDWithGameData(data minecraft.GameData){
	c.shieldIDsetOnce.Do(func() {
		for _, it := range data.Items {
		if it.Name == "minecraft:shield" {
			c.shieldID = int32(it.RuntimeID)
		}
	}
	})
}

type PlayerConn struct{
	*Conn
	XUID uint64
}

func (c *PlayerConn) ForwardPacket(pk ForwardPacket, source uint8) error{
	hdr := Header{
		xuid: c.XUID,
		time: time.Now(),
		packetID: pk.ID(),
		source: source,
	}
	_, ok := pk.(DioPacket)
	hdr.dioPacket = ok
	return c.writePacket(&PacketWrapper{
		pk: pk,
		hdr: &hdr,
	})
}