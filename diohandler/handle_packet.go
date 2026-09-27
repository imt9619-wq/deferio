package diohandler

import (
	"github.com/deferio/diohandler/utils"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type dioPacketHandler interface {
	handlePacket(packet.Packet, *DioHandler)
}

var IDToDioClientPacketHandler map[uint32]dioPacketHandler = map[uint32]dioPacketHandler{}

var IDToDioServerPacketHandler map[uint32]dioPacketHandler = map[uint32]dioPacketHandler{
	packet.IDMovePlayer: dioHandleMovePlayer{},
	packet.IDSetActorMotion: dioHandleSetActorMotion{},
	packet.IDMobEffect: dioHandleMobEffect{},
}

type dioHandleSetActorMotion struct{}
func (dioHandleSetActorMotion) handlePacket(p packet.Packet, dih *DioHandler){
	pk := p.(*packet.SetActorMotion)
	if pk.EntityRuntimeID == utils.SelfEntityID{
		dih.p.serverInpause.Set(pk.Tick, pk.Velocity)
	}
}

type movePlayerData struct{
	position mgl32.Vec3
    pitch    float32
    yaw      float32
    headYaw  float32
    mode     byte
    onGround bool
}
type dioHandleMovePlayer struct{}
func (dioHandleMovePlayer) handlePacket(p packet.Packet, dih *DioHandler){
	pk := p.(*packet.MovePlayer)
	if pk.EntityRuntimeID == utils.SelfEntityID{
		dih.p.serverReset.Set(pk.Tick, movePlayerData{
			position: pk.Position,
			pitch: pk.Pitch,
			yaw: pk.Yaw,
			headYaw: pk.HeadYaw,
			mode: pk.Mode,
			onGround: pk.OnGround,
		})
	}
}

type effectData struct{
	operation  byte
    effectType int32
    duration   int32
    level      int32
}
type dioHandleMobEffect struct{}
func (dioHandleMobEffect) handlePacket(p packet.Packet, dih *DioHandler){
	pk := p.(*packet.MobEffect)
	if pk.EntityRuntimeID == utils.SelfEntityID{
		dih.p.serverEffects.Set(pk.Tick, effectData{
			operation: pk.Operation,
			effectType: pk.EffectType,
			duration: pk.Duration,
			level: pk.Amplifier+1,
		})
	}
}