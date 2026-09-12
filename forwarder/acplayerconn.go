package forwarder

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type ACplayerConn struct{
	id   uint16
	XUID uint64
	Data *PlayerGameData
	pkStream chan PacketResult
	hSetted *atomic.Bool
	h    ACplayerConnHandler
	close chan struct{}
	closeOnce *sync.Once
}

type PacketResult struct{
	Source uint8
    T      time.Time
    Packet packet.Packet
}

type ACplayerConnHandler interface {
	HandlePacket(pk PacketResult)
}

func newAcPlayerConn(p *PacketWrapper) *ACplayerConn{
	pk := p.pk.(*IncomingPlayerPacket)
	a := &ACplayerConn{
		id: p.hdr.id,
		XUID: pk.XUID,
		Data: pk.data,
		pkStream: make(chan PacketResult, 64),
		hSetted: &atomic.Bool{},
		close: make(chan struct{}),
		closeOnce: &sync.Once{},
	}
	return a
}

func (p *ACplayerConn) acPlayerHandlerPacketLoop(){
	for{
		select{
		case <-p.close:
			return
		case pk := <-p.pkStream:
			p.h.HandlePacket(pk)
		}
	}
}

func (p *ACplayerConn) SetHandlerOnce(h ACplayerConnHandler){
	if p.hSetted.Load(){
		panic("Trying to ACplayerConn Handler twice")
	}
	p.h = h
	p.hSetted.Store(true)
	go p.acPlayerHandlerPacketLoop()
}

func (p *ACplayerConn) sendPacketToPlayer(pk PacketResult){
	select{
	case p.pkStream <- pk:
	default:
		panic("ACplayerConn Handler isn't set")
	}
}