package diohandler

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

type dioIncomingPacketHandler interface {
	HandlePacket(packet.Packet, *DioHandler)
}

var IDToDioClientPacketHandler map[uint32]dioIncomingPacketHandler = map[uint32]dioIncomingPacketHandler{}
var IDToDioServerPacketHandler map[uint32]dioIncomingPacketHandler = map[uint32]dioIncomingPacketHandler{}