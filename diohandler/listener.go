package diohandler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deferio/forwarder"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type dioListener struct{
	*minecraft.Listener
	fw *forwarder.Conn
}

type DioHandlerConfig struct{
	ForwarderConf forwarder.ForwarderConfig
}

func (d DioHandlerConfig) InterceptPacket(conf server.Config, address string) server.Config{
	conf.Listeners = []func(conf server.Config) (server.Listener, error){
		func(conf server.Config) (server.Listener, error) {
			cfg := minecraft.ListenConfig{
				MaximumPlayers:         conf.MaxPlayers,
				StatusProvider:         conf.StatusProvider,
				AuthenticationDisabled: conf.AuthDisabled,
				ResourcePacks:          conf.Resources,
				TexturePacksRequired:   conf.ResourcesRequired,
				Compression:            conf.Compression,
			}
			if conf.Log.Enabled(context.Background(), slog.LevelDebug) {
				cfg.ErrorLog = conf.Log.With("net origin", "gophertunnel")
			}
			l, err := cfg.Listen("raknet", address)
			if err != nil {
				return nil, fmt.Errorf("create minecraft listener: %w", err)
			}
			conf.Log.Info("Listener running.", "addr", l.Addr())
			fw, err := d.ForwarderConf.Dial()
			if err != nil {
				l.Close()
				return nil, fmt.Errorf("dio: dial with forwarder: %w", err)
			}
			return &dioListener{Listener: l, fw: fw}, nil
		},
	}
	return conf
}

type dioSessionConn struct{
	session.Conn
	h  *DioHandler
	fw *forwarder.Conn
}

func (d *dioListener) Accept() (session.Conn, error){
	conn, err := d.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &dioSessionConn{
		Conn: conn.(session.Conn), 
		fw: d.fw,
	}, nil
}

func (d *dioListener) Disconnect(conn session.Conn, reason string) error{
	if wrapped, ok := conn.(*dioSessionConn); ok {
		conn = wrapped.Conn
	}
	return d.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}

func (c *dioSessionConn) ReadPacket() (packet.Packet, error){
	pk, err := c.Conn.ReadPacket()
	if err != nil {
		return nil, err
	}
	if c.h != nil{
		c.h.HandleClientPacket(pk)
	}
	return pk, nil
}

func (c *dioSessionConn) WritePacket(pk packet.Packet) error{
	if c.h != nil{
		c.h.HandleServerPacket(pk)
	}
	return c.Conn.WritePacket(pk)
}

func SessionDioConn(s *session.Session) (*dioSessionConn, bool){
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