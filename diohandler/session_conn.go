package diohandler

import (
	"sync/atomic"

	"github.com/deferio/utils"
	"github.com/deferio/forwarder"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type dioSessionConn struct {
	session.Conn
	h                  *DioHandler
	fw                 *forwarder.Forwarder
	handlerRegsistered *atomic.Bool
}

func (d *dioListener) Accept() (session.Conn, error) {
	conn, err := d.Listener.Accept()
	if err != nil {
		return nil, err
	}
	handlerRegsistered := &atomic.Bool{}
	handlerRegsistered.Store(true)
	return &dioSessionConn{
		Conn:               conn,
		fw:                 d.fw,
		handlerRegsistered: handlerRegsistered,
	}, nil
}

func (c *dioSessionConn) ReadPacket() (packet.Packet, error) {
	// since readpacket and session handler calling is synchronous, we should register session handlers at here
	// instead of at NewDioHandler
	if c.handlerRegsistered.CompareAndSwap(false, true) {
		c.h.registerSessionHandlers()
	}
	pk, err := c.Conn.ReadPacket()
	if err != nil {
		return nil, err
	}
	if c.h != nil {
		c.h.handleClientPacket(pk)
	}
	return pk, nil
}

func (c *dioSessionConn) WritePacket(pk packet.Packet) error {
	if c.h != nil {
		c.h.handleServerPacket(pk)
	}
	return c.Conn.WritePacket(pk)
}

func sessionDioConn(s *session.Session) (*dioSessionConn, bool) {
	if s == nil {
		return nil, false
	}
	conn, ok := utils.PrivateFieldByName[session.Conn](s, "conn")
	if !ok {
		return nil, false
	}
	wrapped, ok := conn.(*dioSessionConn)
	if !ok {
		return nil, false
	}
	return wrapped, true
}