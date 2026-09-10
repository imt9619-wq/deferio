package forwarder

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
)

type ForwarderConfig struct{
	FlushRate time.Duration
}

func (f ForwarderConfig) Dial(address string) (*Conn, error){
	if f.FlushRate == 0{
		f.FlushRate = time.Millisecond * 50
	}
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	c := &Conn{
		Conn: conn,
		conf: f,
		inc: make(chan *PacketWrapper, 4096),
		closeOnce: &sync.Once{},
		close: make(chan struct{}),
		sendBuf: newSendBuf(),
		}
	go c.startForwardLoop()
	return c, nil
}

type Conn struct{
	net.Conn
    conf      ForwarderConfig
    inc       chan *PacketWrapper
    closeOnce *sync.Once
    close     chan struct{}
    sendBuf   *sendBuffer
}

func (c *Conn) Close() error{
	c.closeOnce.Do(func() {
		close(c.close)
	})
	return c.Conn.Close()
}

func (c *Conn) flushLoop(){
	ticker := time.NewTicker(c.conf.FlushRate)
	for{
		select{
		case <-c.close:
			return
		case <-ticker.C:
			err := c.Flush()
			if err != nil{
				c.Close()
			}
		}
	}
}

func (c *Conn) startForwardLoop(){
	for{
		select{
		case <-c.close:
			return
		case inc := <-c.inc:
			err := c.WritePacket(inc)
			if err != nil{
				c.Close()
			}
		}
	}
}

func (c *Conn) WritePacket(pk *PacketWrapper) error{} 

func (c *Conn) Flush() error{}

type PlayerConn struct{
	*Conn
	XUID string
}

func (c *PlayerConn) IncomingPlayer(p *player.Player){
	c.incoming(&IncomingPlayerPacket{})
}

func (c *PlayerConn) incoming(pk ForwardPacket){
	c.inc <- &PacketWrapper{
		pk: pk,
		xuid: c.XUID,
		t: time.Now(),
	}
}

