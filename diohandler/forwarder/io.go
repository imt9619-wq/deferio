package forwarder

import (
	"encoding/binary"
	"io"
	"time"
)

type Header struct {
	xuid     uint64
	time     time.Time
	packetID uint32
	dioPacket bool
	source   uint8
}

func (h *Header) Write(w io.ByteWriter) error{
	var buf [HeaderByteSize]byte
	binary.BigEndian.PutUint64(buf[0:8], h.xuid)
	binary.BigEndian.PutUint64(buf[8:16], uint64(h.time.UnixNano()))
	pID := h.packetID & 0x7FFFFF
	var packed uint32
	if h.dioPacket {
		packed |= (1 << 31)
	}
	packed |= uint32(h.source) << 23
	packed |= pID

	binary.BigEndian.PutUint32(buf[16:20], packed)
	if writer, ok := w.(io.Writer); ok {
		_, err := writer.Write(buf[:])
		return err
	}
	for _, b := range buf {
		if err := w.WriteByte(b); err != nil {
			return err
		}
	}
	return nil
}
