package forwarder

import (
	"fmt"
	"net"
	"sync"
)

type ListenerConfig struct{
	ForwarderConfig
	ListenerF       func(address string) (net.Listener, error)
	newPh NewPlayerHandler
}

type Listener struct{
	net.Listener
    closeOnce *sync.Once
    conf      ForwarderConfig
    connMu    *sync.Mutex
    conns     []*Conn
	newPh     NewPlayerHandler
}

func (conf ListenerConfig) Listen() (*Listener, error){
	if conf.ListenerF == nil{
		conf.ListenerF = func(address string) (net.Listener, error){
			return net.Listen("unix", address)
		}
	}
	conf.ForwarderConfig = conf.defaultForwarderConfig()
	l, err := conf.ListenerF(conf.Address)
	if err != nil{
		return nil, fmt.Errorf("FwListener: Failed to listen: %v", err)
	}
	fwL := &Listener{
		Listener: l,
		closeOnce: &sync.Once{},
		connMu: &sync.Mutex{},
		conns: make([]*Conn, 0, 4),
		newPh: conf.newPh,
	}
	return fwL, nil
}

func (l *Listener) Accept() (*Taker, error){
	c, err := l.Accept()
	if err != nil{
		return nil, err
	}
	conn := l.conf.newConn(c)
	l.connMu.Lock()
	l.conns = append(l.conns, conn)
	l.connMu.Unlock()
	t := &Taker{
		Conn: conn,
		idToPlayerRmu: &sync.RWMutex{},
		idToPlayer: make(map[uint16]*ACplayerConn, 128),
		ph: l.newPh,
	}
	return t, err
}

func (l *Listener) Close(){
	l.closeOnce.Do(func(){
		l.connMu.Lock()
		for _, conn := range l.conns{
			conn.Close()
		}
		l.connMu.Unlock()
		l.Listener.Close()
	})
}