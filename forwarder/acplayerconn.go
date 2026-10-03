package forwarder

import (
	"errors"
	"fmt"
	"iter"
	"sync"
)

type ACplayerConn struct{
	id   uint16
	xuid uint64
	data *PlayerGameData
	pkStream chan PacketInRaw
	taker *Taker
	close chan struct{}
	closeOnce *sync.Once
}

func newACPlayerConn(p *PacketWrapper, t *Taker) *ACplayerConn{
	pk := p.pk.(*IncomingPlayerPacket)
	a := &ACplayerConn{
		id: pk.id,
		xuid: pk.XUID,
		data: pk.data,
		pkStream: make(chan PacketInRaw, 32),
		close: make(chan struct{}),
		taker: t,
		closeOnce: &sync.Once{},
	}
	return a
}

func (p *ACplayerConn) sendPacketToPlayer(raw PacketInRaw){
	select{
	case <-p.close:
		return
	case p.pkStream <-raw:
	default:
		p.taker.conf.Log.Debug(fmt.Sprintf("Raw packet is dropped on full pkStream: id: %d", raw.id))
	}
}

func (p *ACplayerConn) ReadPacketTilDisconnect() iter.Seq[PacketResult]{
	return func(yield func(PacketResult) bool) {
		defer p.Close()
		for{
			select{
			case <-p.close:
				return
			case raw, ok := <-p.pkStream:
				if !ok{
					return 
				}
				pw, err := p.taker.readPacketFromRaw(raw)
				if err != nil{
					if _, ok := errors.AsType[decodeError](err); ok{
						continue
					}
					return
				}
				pk := PacketResult{
					Source: pw.data.source,
					T: p.taker.timeFromOffset(pw.data.timeOffset),
					Packet: pw.pk,
				}
				if !yield(pk){
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