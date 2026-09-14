package forwarder

import (
	"errors"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
	"time"
)

type Taker struct {
	*Conn
    idToPlayerRmu *sync.RWMutex
    idToPlayer    map[uint16]*ACplayerConn
    timePoint     time.Time
	expects       chan Header
	inc           chan *ACplayerConn
	serverAddr    *atomic.Value
}

func (t *Taker) HandlePlayers(){
	defer func(){
		t.idToPlayerRmu.Lock()
		for _, p := range t.idToPlayer {
			p.Close()
		}
		clear(t.idToPlayer)
		t.idToPlayerRmu.Unlock()
	}()
	for{
		select{
		case <-t.close:
			return
		default:
		}
		p, err := t.readPacket()
		if err != nil{
			t.conf.Log.Error(fmt.Sprintf("Error when readingPackets: %s", err), fmt.Sprintf("Taker(%s)", t.ServerAddress()), "handlePlayers")
			var de decodeError
			if errors.As(err, &de){
				continue
			}
			t.Close()
			return
		}
		if !t.asExpected(*p.hdr){
			t.conf.Log.Error(fmt.Sprintf("Unexpected Packet: %T", p.pk), fmt.Sprintf("Taker(%s)", t.ServerAddress()), "handlePlayers")
			t.Close()
		}
		switch pk := p.pk.(type){
		case *NewDialPacket:
			t.timePoint = time.UnixMicro(pk.serverTime)
			t.serverAddr.Store(pk.serverAddr)
			continue
		case *IncomingPlayerPacket:
			t.setShieldID(pk)
			t.idToPlayerRmu.Lock()
			a := newAcPlayerConn(p)
			t.idToPlayer[p.hdr.id] = a
			t.idToPlayerRmu.Unlock()
			select{
			case <-t.close:
				a.Close()
				return
			case t.inc <-a:
			}
			continue
		case *DisconnectedPlayerPacket:
			t.idToPlayerRmu.Lock()
			pconn, ok := t.idToPlayer[p.hdr.id]
			delete(t.idToPlayer, p.hdr.id)
			t.idToPlayerRmu.Unlock()
			if ok{
				pconn.Close()
			}
			continue
		}
		t.idToPlayerRmu.RLock()
		player, ok := t.idToPlayer[p.hdr.id]
		t.idToPlayerRmu.RUnlock()
		if ok{
			player.sendPacketToPlayer(PacketResult{
				Source: p.hdr.source,
				Packet: p.pk,
				T: t.timeFromOffset(p.hdr.timeOffset),
			})
		}
	}
}

func (t *Taker) ServerAddress() string {
	v := t.serverAddr.Load()
	if v == nil{
		return ""
	}
	s, _ := v.(string)
	return s
}

func (t *Taker) expect(hdr Header){
	t.expects <- hdr
}

func (t *Taker) asExpected(hdr Header) bool{
	select{
	case ex := <- t.expects:
		return hdr.id == ex.id && hdr.dioPacket == ex.dioPacket && hdr.packetID == ex.packetID
	default:
		return true
	}
}

func (t *Taker) timeFromOffset(offset time.Duration) time.Time{
	since := t.timePoint.Add(offset)
	days := time.Since(t.timePoint)/Day
	if offset > (time.Since(t.timePoint)%Day){
		days -= 1
	}
	return since.Add(days*Day)
}

func (t *Taker) AcceptPlayerConn() iter.Seq[*ACplayerConn]{
	return func(yield func(*ACplayerConn) bool) {
		for{
			select{
			case <-t.close:
				return
			case inc := <-t.inc:
				if !yield(inc){
					inc.Close()
					return
				}
			}
		}
	}
}