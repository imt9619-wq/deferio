package forwarder

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/player"
)

type ForwarderConfig struct{}

func (f ForwarderConfig) Dial(address string) (*Conn, error){
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("Forwarder: Failed to dial: %v", err)
	}
	c := &Conn{
		Conn: conn,
		inc: make(chan PacketWrapper, 4096),
		closeOnce: &sync.Once{},
		close: make(chan struct{}),
		}
	go c.startForwardLoop()
	return c, nil
}

type Conn struct{
	net.Conn
	inc chan PacketWrapper
	closeOnce *sync.Once
	close chan struct{}
}

func (c *Conn) Close() error{
	c.closeOnce.Do(func() {
		close(c.close)
	})
	return c.Conn.Close()
}

func (c *Conn) startForwardLoop(){
	for{
		select{
		case <-c.close:
			return
		case inc := <-c.inc:
			_, err := c.Write(inc.Encode())
			if err != nil{
				c.Close()
			}
		}
	}
}

func (c *Conn) IncomingPlayer(p *player.Player){
	c.incoming(IncomingPlayerPacket{
		XUID: p.XUID(),
	})
}

func (c *Conn) incoming(pk ForwardPacket){
	c.inc <- PacketWrapper{
		ForwardPacket: pk,
		t: time.Now(),
	}
}
