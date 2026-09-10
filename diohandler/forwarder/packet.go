package forwarder

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const(
	ClientPacket = iota + 1
	ServerPacket 
	DioHandlerPacket
	DioAntiCheatPacket
)

type ForwardPacket interface{
	packet.Packet
	IsGtPacket() bool
	Source() uint32
}

var _ interface{Marshal(IO)} = &PacketWrapper{}

type PacketWrapper struct{
	pk ForwardPacket
	xuid string
	t time.Time
}

func (p *PacketWrapper) Marshal(io IO){
	p.pk.Marshal(io)
	io.String(&p.xuid)
	io.Time(&p.t)
}

const(
	IDIncomingPlayerPacket = iota + 1
)

type IncomingPlayerPacket struct{}
func (*IncomingPlayerPacket) ID() uint32{return IDIncomingPlayerPacket}
func (*IncomingPlayerPacket) Source() uint32{return DioHandlerPacket}
func (*IncomingPlayerPacket) IsGtPacket() bool{return false}
func (*IncomingPlayerPacket) Marshal(io protocol.IO){}