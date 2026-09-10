package diohandler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/deferio/diohandler/forwarder"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type DioHandler struct{
	player.Handler
	s  *session.Session
	p  *playerCache
	fw *forwarder.PlayerConn
}

const(
	ReachDist = 3.0
)

func NewDioHandler(p *player.Player) *DioHandler{
	dih := &DioHandler{
		s: p.Data().Session,
	}
	conn, ok := SessionDioConn(p.Data().Session)
	xuid, err := strconv.ParseUint(p.XUID(), 10, 64)
	if err != nil{
		panic(fmt.Sprintf("NewDioHandler: Cannot convert XUID from string to uint64 for %s (xuid: %s)", p.Name(), p.XUID()))
	}
	dih.fw = &forwarder.PlayerConn{
		Conn: conn.fw,
		XUID: xuid,
	}
	dih.fw.SetShieldIDWithGameData(conn.Conn.(*minecraft.Conn).GameData())
	dih.fw.ForwardPacket(&forwarder.IncomingPlayerPacket{}, forwarder.SourceDioHandlerPacket)
	dih.p = newPlayerCache(p)
	dih.registerSessionHandlers()
	if ok{
		conn.h = dih
	}
	return dih
}

func SetPlayerHandler(p *player.Player, h player.Handler) error{
	defer p.Handle(h)
	ph := p.Handler()
	dih, ok := ph.(*DioHandler)
	if !ok{
		dih = NewDioHandler(p)
	}
	dih.Handler = h
	h = dih
	return nil
}

func (d *DioHandler) HandleClientPacket(pk packet.Packet){
	h, ok := IDToDioClientPacketHandler[pk.ID()]
	if ok{
		h.HandlePacket(pk, d)
	}
}

func (d *DioHandler) HandleServerPacket(pk packet.Packet){
	h, ok := IDToDioServerPacketHandler[pk.ID()]
	if ok{
		h.HandlePacket(pk, d)
	}
}

func (d *DioHandler) HandleBlockPlace(ctx *player.Context, pos cube.Pos, b world.Block){
	bbs := b.Model().BBox(pos, ctx.Tx)
	if len(bbs) != 0{
		for ent := range ctx.Tx.EntitiesWithin(cube.Box(0, 0, 0, 1, 1, 1).Translate(pos.Vec3())){
			for _, bb := range bbs{
				if ent.H().Type().BBox(ent).Translate(ent.Position()).IntersectsWith(bb.Translate(pos.Vec3())){
					ctx.Cancel()
					return
				}
			}
		}
	}
	d.Handler.HandleBlockPlace(ctx, pos, b)
}

func (d *DioHandler) HandleHurt(ctx *player.Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource){
	if ent, ok := src.(entity.AttackDamageSource); ok{
		if nearby, ok := ent.Attacker.(*player.Player); ok{
			reachDist, ok := utils.RayTraceFromOrigin(
				utils.PlayerBBox(ctx.Player()).Grow(0.05), 
				nearby.Position().Add(mgl64.Vec3{0, nearby.EyeHeight()}), 
				utils.DirNorm(nearby.Rotation()))
			if !ok || reachDist[0] > ReachDist{
				ctx.Cancel()
				return
			}
		}
	}
	d.Handler.HandleHurt(ctx, damage, immune, attackImmunity, src)
}