package forwarder

import (
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

type ListenerConfig struct{
	ForwarderConnConfig
	ListenerF func(address string) (net.Listener, error)
}

type Listener struct{
	net.Listener
    closeOnce *sync.Once
    conf      ForwarderConnConfig
    connMu    *sync.Mutex
    conns     []*Conn
	inc       chan func()(*Taker, error)
}

func (conf ListenerConfig) Listen() (*Listener, error){
	if conf.ListenerF == nil{
		conf.ListenerF = func(address string) (net.Listener, error){
			os.Remove(address)
			return net.Listen("unix", address)
		}
	}
	conf.ForwarderConnConfig = conf.defaultForwarderConnConfig()
	l, err := conf.ListenerF(conf.Address)
	if err != nil{
		return nil, fmt.Errorf("FwListener: Failed to listen: %v", err)
	}
	fwL := &Listener{
		Listener: l,
		closeOnce: &sync.Once{},
		connMu: &sync.Mutex{},
		conf: conf.ForwarderConnConfig,
		conns: make([]*Conn, 0, 4),
		inc: make(chan func() (*Taker, error), 4),
	}
	return fwL, nil
}

func (l *Listener) accept() (*Taker, error){
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
		inc: make(chan *ACplayerConn),
		expects: make(chan Header, 10),
		serverAddr: &atomic.Value{},
	}
	t.expect(Header{packetID: IDNewDialPacket, id: ServerID, dioPacket: true})
	return t, nil
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

func (l *Listener) IncomingClients() chan func()(*Taker, error){
	return l.inc
}

func (l *Listener) StartHandleClients(){
	for{
		conn, err := l.accept()
		l.inc <- func() (*Taker, error){return conn, err}
		if err != nil{
			l.conf.Log.Error(fmt.Sprintf("Return on error: %s", err), "Listener", "StartHandleClients")
			return
		}
	}
}