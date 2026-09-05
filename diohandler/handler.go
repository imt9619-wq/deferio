package diohandler

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type DioHandler struct{
	player.Handler
	s *session.Session
	p *dioPlayer
}

func NewDioHandler(p *player.Player, h player.Handler) (*DioHandler, error){
	dih := &DioHandler{
		s: p.Data().Session,
		Handler: h,
	}
	conn, ok := SessionDioConn(p.Data().Session)
	if !ok{
		return nil, fmt.Errorf("session.conn is not DioSessionCon for: %s", p.Name())
	}
	conn.h = dih
	return dih, nil
}

func (d *DioHandler) HandleClientPacket(pk packet.Packet){
	switch pk := pk.(type){
	case *packet.PlayerAuthInput:
		_ = pk
	}
}

func (d *DioHandler) HandleServerPacket(pk packet.Packet){
	switch pk := pk.(type){
	case *packet.AddPlayer:
		_ = pk
	}
}

func (d *DioHandler) HandleMove(ctx *player.Context, pos mgl64.Vec3, c cube.Rotation){
	ctx.Player().Position()
}