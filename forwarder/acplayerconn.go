package forwarder

import (
	"iter"
	"sync"
	"time"
)

type ACplayerConn struct{
	id   uint16
	xuid uint64
	data *PlayerGameData
	pkStream chan PacketResult
	close chan struct{}
	closeOnce *sync.Once
}

type PacketResult struct{
	Source uint8
    T      time.Time
    Packet ForwardPacket
}

func newAcPlayerConn(p *PacketWrapper) *ACplayerConn{
	pk := p.pk.(*IncomingPlayerPacket)
	a := &ACplayerConn{
		id: p.hdr.id,
		xuid: pk.XUID,
		data: pk.data,
		pkStream: make(chan PacketResult, 32),
		close: make(chan struct{}),
		closeOnce: &sync.Once{},
	}
	return a
}

func (p *ACplayerConn) sendPacketToPlayer(pk PacketResult){
	select{
	case <-p.close:
		return
	case p.pkStream <- pk:
	default:
	}
}

func (p *ACplayerConn) ReadPacketTilDisconnect() iter.Seq[PacketResult]{
	return func(yield func(PacketResult) bool) {
		for{
			select{
			case <-p.close:
				return
			case pk, ok := <-p.pkStream:
				if !ok || !yield(pk){
					p.Close()
					return
				}
			}
		}
	}
}

func (p *ACplayerConn) Close(){
	p.closeOnce.Do(func() {
		close(p.close)
	})
}

func (p *ACplayerConn) XUID() uint64{
	return p.xuid
}

func (p *ACplayerConn) GameData() *PlayerGameData{
	return p.data
}