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
	expects       chan PacketData
	inc           chan *ACplayerConn
	serverAddr    *atomic.Value
}

func (t *Taker) HandlePlayers(){
	defer func(){
		t.Close()
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
		raw, err := t.readPacketRaw()
		if err != nil{
			t.conf.Log.Error(fmt.Sprintf("Error when readingPacketsInRaw: %s", err), fmt.Sprintf("Taker(%s)", t.ServerAddress()), "handlePlayers")
			return
		}
		if raw.id == ServerID{
			p, err := t.readPacketFromRaw(raw)
			if err != nil{
				t.conf.Log.Error(fmt.Sprintf("Error when ServerID packet: %s", err), fmt.Sprintf("Taker(%s)", t.ServerAddress()), "handlePlayers")
				if _, ok := errors.AsType[decodeError](err); ok{
					continue
				}
				return
			}
			if !t.asExpected(*p.data){
				t.conf.Log.Error(fmt.Sprintf("Unexpected Packet: %T", p.pk), fmt.Sprintf("Taker(%s)", t.ServerAddress()), "handlePlayers")
				return
			}
			switch pk := p.pk.(type){
			case *NewDialPacket:
				t.timePoint = time.UnixMicro(pk.serverTime)
				t.serverAddr.Store(pk.serverAddr)
			case *IncomingPlayerPacket:
				t.setShieldID(pk)
				t.idToPlayerRmu.Lock()
				a := newACPlayerConn(p, t)
				t.idToPlayer[pk.id] = a
				t.idToPlayerRmu.Unlock()
				select{
				case <-t.close:
					a.Close()
					return
				case t.inc <-a:
				}
			case *DisconnectedPlayerPacket:
				t.idToPlayerRmu.Lock()
				pconn, ok := t.idToPlayer[pk.id]
				delete(t.idToPlayer, pk.id)
				t.idToPlayerRmu.Unlock()
				if ok{
					pconn.Close()
				}
			}
			continue
		}
		
		t.idToPlayerRmu.RLock()
		player, ok := t.idToPlayer[raw.id]
		t.idToPlayerRmu.RUnlock()
		if ok{
			player.sendPacketToPlayer(raw)
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

func (t *Taker) expect(data PacketData){
	t.expects <- data
}

func (t *Taker) asExpected(data PacketData) bool{
	select{
	case ex := <- t.expects:
		return data.dioPacket == ex.dioPacket && data.packetID == ex.packetID
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