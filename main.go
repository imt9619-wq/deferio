package main

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/deferio/diohandler"
	"github.com/deferio/forwarder"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	ready := make(chan struct{})
	go func ()  {
		deferAntiCheatExample(ready)
		wg.Done()
	}()
	<-ready
	go func ()  {
		deferioHandlerExample()
		wg.Done()
	}()
}

func deferioHandlerExample(){
	const address = "127.0.0.1:19133"

	slog.SetLogLoggerLevel(slog.LevelDebug)
	chat.Global.Subscribe(chat.StdoutSubscriber{})

	c := server.DefaultConfig()
	c.Network.Address = address
	conf, err := c.Config(slog.Default())
	conf = diohandler.DioHandlerConfig{}.InterceptPacket(conf, address)
	
	if err != nil {
		panic(err)
	}
	srv := conf.New()
	srv.CloseOnProgramEnd()

	srv.Listen()
	for p := range srv.Accept() {
		diohandler.SetPlayerHandler(p, player.NopHandler{})
	}
}

func deferAntiCheatExample(startListen chan struct{}){
	l, err := forwarder.ListenerConfig{}.Listen()
	if err != nil{
		panic(err)
	}
	defer l.Close()
	close(startListen)
	for{
		conn, err := l.Accept()
		if err != nil{
			fmt.Printf("Error on accepted Taker: %s\n", err)
			continue
		}
		go conn.HandleClients()
	}
}
