package diohandler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/deferio/forwarder"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
)

type MCListenerWrap struct{*minecraft.Listener}
func (w MCListenerWrap) Accept() (session.Conn, error){
	conn, err := w.Listener.Accept()
	return conn.(session.Conn), err
}
func (w MCListenerWrap) Disconnect(conn session.Conn, reason string) error{
	return w.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}

type dioListener struct{
	server.Listener
	fw forwarder.ForwarderConn
}

type DioHandlerConfig struct{
	forwarder.ForwarderConfig
	ProxyListenerF func(conf server.Config) (server.Listener, error)
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
		d.ForwarderConfig.ServerAddr = address
		fw, err := d.ForwarderConfig.Dial()
		if err != nil{
			return nil, fmt.Errorf("create forwarder: %w", err)
		}
		return &dioListener{Listener: l, fw: fw}, nil
	}
}