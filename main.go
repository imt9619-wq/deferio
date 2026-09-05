package main

import (
	"log/slog"

	"github.com/deferio/diohandler"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
)

const address = "127.0.0.1:19133"

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	chat.Global.Subscribe(chat.StdoutSubscriber{})

	c := server.DefaultConfig()
	c.Network.Address = address
	conf, err := c.Config(slog.Default())
	diohandler.InjectDioListener(conf, address)
	
	if err != nil {
		panic(err)
	}
	srv := conf.New()
	srv.CloseOnProgramEnd()

	srv.Listen()
	for p := range srv.Accept() {
		h, err := diohandler.NewDioHandler(p, Handler{})
		if err != nil{
			panic(err)
		}
		p.Handle(h)
	}
}

type Handler struct{
	player.NopHandler
}