package forwarder

import (
	"sync"
	"time"
)

type NewPlayerHandler interface{
	NewPlayer(p *ACplayerConn)
}

type Taker struct {
	*Conn
    idToPlayerRmu *sync.RWMutex
    idToPlayer    map[uint16]*ACplayerConn
    timeDesync    time.Duration
    ph            NewPlayerHandler
}

func (t *Taker) HandleClients(){
	for{
		p, err := t.readPacket()
		if err != nil{
			// Add logging
		}
		if !p.hdr.dioPacket{
			t.idToPlayerRmu.RLock()
			if player, ok := t.idToPlayer[p.hdr.id]; ok{
				player.sendPacketToPlayer(PacketResult{
					Source: p.hdr.source,
					Packet: p.pk,
					T: // TODO fix time sync,
				})
			}
			t.idToPlayerRmu.RUnlock()
		}
		switch pk := p.pk.(type){
		case *NewDialPacket:
			t.timeDesync = // TODO fix time sync
		case *IncomingPlayerPacket:
			t.setShieldID(pk)
			t.idToPlayerRmu.Lock()
			a := newAcPlayerConn(p)
			t.idToPlayer[p.hdr.id] = a
			t.idToPlayerRmu.Unlock()
			if t.ph != nil{
				t.ph.NewPlayer(a)
			}
		}
	}
}