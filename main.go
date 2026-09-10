package main

import (
	"log/slog"
	"sync"

	"github.com/deferio/diohandler"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func ()  {
		deferioHandlerExample()
		wg.Done()
	}()
	go func ()  {
		deferAntiCheatExample()
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

func deferAntiCheatExample(){

}
