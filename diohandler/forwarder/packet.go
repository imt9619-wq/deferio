package forwarder

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	SourceClientPacket = iota + 1
	SourceServerPacket 
	SourceDioHandlerPacket
	SourceDioAntiCheatPacket
)

type DioPacket interface{
	DioPacket()
}

type ForwardPacket interface{
	// if is gt Packet, then id will be gt packet id, else dio packet id
	ID() uint32 
    Marshal(io protocol.IO)
}

type PacketWrapper struct{
	pk ForwardPacket
	hdr *Header
}

const(
	IDIncomingPlayerPacket = iota + 1
)

type dioPacket struct{}
func (dioPacket) DioPacket()

type IncomingPlayerPacket struct{
	dioPacket
}
func (*IncomingPlayerPacket) ID() uint32{return IDIncomingPlayerPacket}
func (i *IncomingPlayerPacket) Marshal(io protocol.IO){}