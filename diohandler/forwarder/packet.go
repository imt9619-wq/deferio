package forwarder

import (
	"time"
)

const(
	ClientPacket = iota + 1
	ServerPacket 
	DioHandlerPacket
	DioAntiCheatPacket
)

type ForwardPacket interface{
	ID()     uint16
	Encode() []byte
	Source() uint16
}

type PacketWrapper struct{
	ForwardPacket
	t time.Time
}

func (p *PacketWrapper) Encode() []byte{
	return []byte{}
}

const(
	IDIncomingPlayerPacket = iota + 1
)

type IncomingPlayerPacket struct{
	XUID string
}

func (i IncomingPlayerPacket) ID() uint16{
	return IDIncomingPlayerPacket
}

func (i IncomingPlayerPacket) Encode() []byte{
	return []byte{}
}

func (i IncomingPlayerPacket) Source() uint16{
	return DioHandlerPacket
}