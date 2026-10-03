package forwarder

import (
	"encoding/binary"
	"io"
	"time"
)

type PacketData struct{
	timeOffset time.Duration
    packetID   uint32
    dioPacket  bool
    source     uint8
}

func (d *PacketData) Write(w io.Writer) error{
	// timeOffset µs: 37 | packetID: 15 | dioPacket: 1 | source: 3
	var off uint64
	if us := d.timeOffset.Microseconds(); us > 0{
		off = uint64(us) & 0x1FFFFFFFFF
	}
	pid := uint64(d.packetID) & 0x7FFF
	var dio uint64
	if d.dioPacket {
		dio = 1
	}
	src := uint64(d.source) & 0x7

	var buf [PacketDataByteSize]byte
	binary.BigEndian.PutUint32(buf[0:4], uint32(off>>5))
	rest := (off&0x1F)<<19 | pid<<4 | dio<<3 | src
	binary.BigEndian.PutUint16(buf[4:6], uint16(rest>>8))
	buf[6] = byte(rest)
	_, err := w.Write(buf[:])
	return err
}

func (d *PacketData) Read(r io.Reader) error{
	var buf [PacketDataByteSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil{
		return err
	}
	off := uint64(binary.BigEndian.Uint32(buf[0:4]))<<5 | uint64(buf[4]>>3)
	rest := uint64(buf[4])<<16 | uint64(buf[5])<<8 | uint64(buf[6])
	d.timeOffset = time.Duration(off) * time.Microsecond
	d.packetID = uint32((rest >> 4) & 0x7FFF)
	d.dioPacket = (rest>>3)&1 == 1
	d.source = uint8(rest & 0x7)
	return nil
}

type PacketInRaw struct{
	id  uint16
    raw []byte
}

type PacketWrapper struct{
	pk   ForwardPacket
    data *PacketData
    id   uint16
}

type PacketResult struct {
	Source uint8
	T      time.Time
	Packet ForwardPacket
}