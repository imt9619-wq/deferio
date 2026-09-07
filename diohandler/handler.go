package diohandler

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type DioHandler struct{
	player.Handler
	s *session.Session
	p *playerCache
}

func NewDioHandler(p *player.Player) *DioHandler{
	dih := &DioHandler{
		s: p.Data().Session,
	}
	conn, ok := SessionDioConn(p.Data().Session)
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