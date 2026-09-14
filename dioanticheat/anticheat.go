package dioanticheat

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/deferio/dioanticheat/detection"
	"github.com/deferio/forwarder"
)

type DioAntiCheatConfig struct{
	forwarder.ListenerConfig
	Handler detection.DetectionHandler
}

type DioACserver struct{
	conf      *DioAntiCheatConfig
    l         *forwarder.Listener
    close     chan struct{}
    closeOnce *sync.Once
}

func (conf DioAntiCheatConfig) StartAntiCheatServer() (*DioACserver, error){
	if conf.Handler == nil{
		conf.Handler = detection.NopDetectionHandler{}
	}
	l, err := conf.Listen()
	if err != nil{
		return nil, err
	}
	a := &DioACserver{
		conf: &conf,
		l: l,
		close: make(chan struct{}),
		closeOnce: &sync.Once{},
	}
	go a.StartDetecting()
	return a, nil
}

func (a *DioACserver) Close(){
	a.closeOnce.Do(func(){
		a.l.Close()
		close(a.close)
	})
}

func (a *DioACserver) WaitTilProgramEnd() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	a.Close()
}

func (a *DioACserver) StartDetecting(){
	go a.l.StartHandleClients()
	defer a.l.Close()
	for{
		select{
		case <-a.close:
			return
		case f := <-a.l.IncomingClients():
			t, err := f()
			if err != nil{
				a.conf.Log.Error(fmt.Sprintf("Error on Incoming Takers: %s", err), "DioACserver", "StartDetecting")
				continue
			}
			go a.HandleTaker(t)
		}
	}
}

func (a *DioACserver) HandleTaker(t *forwarder.Taker){
	go t.HandlePlayers()
	defer t.Close()
	for p := range t.AcceptPlayerConn(){
		go a.HandlePlayers(p)
	}
}

func (a *DioACserver) HandlePlayers(p *forwarder.ACplayerConn){
	for pk := range p.ReadPacketTilDisconnect(){
		_ = pk
		// handle packet...
	}
}
