package diohandler

import (
	"github.com/deferio/diohandler/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const(
	selfEntityID = 1
)

type dioPacketHandler interface {
	HandlePacket(packet.Packet, *DioHandler)
}

var IDToDioClientPacketHandler map[uint32]dioPacketHandler = map[uint32]dioPacketHandler{}

var IDToDioServerPacketHandler map[uint32]dioPacketHandler = map[uint32]dioPacketHandler{
	packet.IDMovePlayer: dioHandleMovePlayer{},
	packet.IDSetActorMotion: dioHandleSetActorMotion{},
}

type dioHandleSetActorMotion struct{}
func (dioHandleSetActorMotion) HandlePacket(p packet.Packet, dih *DioHandler){
	pk := p.(*packet.SetActorMotion)
	if pk.EntityRuntimeID == selfEntityID{
		dih.p.serverInpause.set(pk.Tick, utils.Mgl64FromMgl32(pk.Velocity))
	}
}

type dioHandleMovePlayer struct{}
func (dioHandleMovePlayer) HandlePacket(p packet.Packet, dih *DioHandler){
	pk := p.(*packet.MovePlayer)
	if pk.EntityRuntimeID == selfEntityID{
		dih.p.serverReset.set(pk.Tick, pk)
	}
}