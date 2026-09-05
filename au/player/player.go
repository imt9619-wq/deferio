package dioplayer

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
)

type Player struct {
	pos mgl32.Vec3
	velocity mgl32.Vec3
}

func NewPlayer(data *minecraft.GameData) *Player{
	p := &Player{}
	p.pos = data.PlayerPosition
	return p
}