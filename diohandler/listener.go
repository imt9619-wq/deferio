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
	server.Listener
	fw *forwarder.Forwarder
}

type DioHandlerConfig struct{
	forwarder.DialConfig
	ProxyListenerF func(conf server.Config) (server.Listener, error)
}

type MCListenerWrap struct{*minecraft.Listener}
func (w MCListenerWrap) Accept() (session.Conn, error){
	conn, err := w.Listener.Accept()
	return conn.(session.Conn), err
}
func (w MCListenerWrap) Disconnect(conn session.Conn, reason string) error{
	return w.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}

func (d DioHandlerConfig) ListenerFWithConfig(address string) func(conf server.Config) (server.Listener, error){
	return func(conf server.Config) (server.Listener, error){
		var l server.Listener
		if d.ProxyListenerF != nil{
			proxyl, err := d.ProxyListenerF(conf)
			if err != nil {
				return nil, fmt.Errorf("create session listener through ProxyListenF: %w", err)
			}
			l = proxyl
		}else{
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
			mcl, err := cfg.Listen("raknet", address)
			if err != nil {
				return nil, fmt.Errorf("create minecraft listener: %w", err)
			}
			conf.Log.Info("Listener running.", "addr", mcl.Addr())
			l = MCListenerWrap{Listener: mcl}
		}
		fw, err := forwarder.ForwarderConfig{DialConfig: d.DialConfig, ServerAddr: address}.Dial()
		if err != nil{
			return nil, fmt.Errorf("create forwarder: %w", err)
		}
		return &dioListener{Listener: l, fw: fw}, nil
	}
}

type dioSessionConn struct{
	session.Conn
	h  *DioHandler
	fw *forwarder.Forwarder
}

func (d *dioListener) Accept() (session.Conn, error){
	conn, err := d.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &dioSessionConn{
		Conn: conn, 
		fw: d.fw,
	}, nil
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