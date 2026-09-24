package main

import (
	"log/slog"
	"sync"

	"github.com/deferio/dioanticheat"
	"github.com/deferio/diohandler"
	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/player/chat"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	ready := make(chan struct{})
	go func(){
		deferAntiCheatExample(ready)
		wg.Done()
	}()
	<-ready
	go func(){
		deferioHandlerExample()
		wg.Done()
	}()
	wg.Wait()
}

func deferioHandlerExample(){
	slog.SetLogLoggerLevel(slog.LevelDebug)

	chat.Global.Subscribe(chat.StdoutSubscriber{})
	conf, err := server.DefaultConfig().Config(slog.Default())
	conf.Listeners = []func(conf server.Config)(server.Listener, error){
		diohandler.DioHandlerConfig{}.ListenerFWithConfig(":19133"),
	}
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
	a, err := dioanticheat.DioAntiCheatConfig{}.StartAntiCheatServer()
	if err != nil{
		panic(err)
	}
	close(startListen)
	a.WaitTilProgramEnd()
}
