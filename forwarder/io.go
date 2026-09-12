package forwarder

import (
	"encoding/binary"
	"io"
	"time"
)

type Header struct {
	id         uint16
    timeOffset time.Duration
    packetID   uint32
    dioPacket  bool
    source     uint8
}

func (h *Header) Write(w io.Writer) error{
	// id: 15 | timeOffset(mcs): 37 | packetID: 15 | dioPacket: 1 | source: 4, total 9 byte
	var off, pid uint64 = 0, uint64(h.packetID) & 0x7FFF
	if us := h.timeOffset.Microseconds(); us > 0 {
		off = uint64(us) & 0x1FFFFFFFFF
	}
	var buf [HeaderByteSize]byte
	binary.BigEndian.PutUint64(buf[0:8], (uint64(h.id)&0x7FFF)<<49 | off<<12 | pid>>3)

	var dio uint64
	if h.dioPacket {
		dio = 1
	}
	buf[8] = byte((pid&7)<<5 | dio<<4 | (uint64(h.source)&0xF))
	
	_, err := w.Write(buf[:])
	return err
}

func (h *Header) Read(r io.Reader) error{
	var buf [HeaderByteSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}

	u := binary.BigEndian.Uint64(buf[0:8])
	h.id = uint16((u>>49)&0x7FFF)
	h.timeOffset = time.Duration((u>>12)&0x1FFFFFFFFF) * time.Microsecond

	last := uint64(buf[8])
	h.packetID = uint32((u&0xFFF)<<3 | (last>>5)&0x7)
	h.dioPacket = (last>>4)&1 == 1
	h.source = uint8(last & 0xF)
	
	return nil
}

func TimeTodayMidnight(now time.Time) time.Time{
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}