package forwarder

import (
	"sync"
	"time"
)

type Forwarder struct {
	*Conn
    idMu            *sync.Mutex
    emptyIdSlot     []int
    idToXuid        []uint64
}

func ForwarderDial(d DialConfig) (*Forwarder, error){
	conn, err := d.dial()
	if err != nil{
		return nil, err
	}
	fw := &Forwarder{
		Conn: conn,
		idMu: &sync.Mutex{},
		emptyIdSlot: make([]int, 0, 128),
		idToXuid: make([]uint64, 0, 128),
	}
	fw.forwardPacket(&NewDialPacket{serverTime: time.Now().Unix()}, ServerID, SourceDeferioPacket)
	return fw, nil
}

func (fw *Forwarder) forwardPacket(pk ForwardPacket, id uint16, source uint8) error{
	now := time.Now().UTC()
	hdr := Header{
		timeOffset: now.Sub(TimeTodayMidnight(now)),
	}
	return fw.Conn.forwardPacket(&PacketWrapper{pk: pk, hdr: &hdr}, id, source)
}