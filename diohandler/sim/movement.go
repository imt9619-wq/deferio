package diosim

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const(
	PlayerWidth = 0.6
	PlayerHeight = 1.8
	PlayerSwimHeight = 0.6
	PlayerSneakHeight = 1.2
)

type MovementInput struct{
	Up, Down, Left, Right        bool
    Sneak, Sprint, Jump, Swim    bool
	Position, Velocity, Inplause mgl64.Vec3
    OnGround                     bool
    JumpCooldown                 uint
    Tx                           *world.Tx
}

type MovementResult struct{
	Position mgl64.Vec3
	Velocity mgl64.Vec3
	OnGround bool
	JumpCooldown uint
}

func (in MovementInput) PlayerBBox(pos mgl64.Vec3) cube.BBox{
	if in.Sneak{
		return cube.Box(
			pos[0] - PlayerWidth/2, 
			pos[1], 
			pos[2] - PlayerWidth/2, 
			pos[0] + PlayerWidth/2, 
			pos[1] + PlayerSneakHeight, 
			pos[2] + PlayerWidth/2, 
		)
	}else if in.Swim{
		return cube.Box(
			pos[0] - PlayerWidth/2, 
			pos[1], 
			pos[2] - PlayerWidth/2, 
			pos[0] + PlayerWidth/2, 
			pos[1] + PlayerSwimHeight, 
			pos[2] + PlayerWidth/2, 
		)
	}else{
		return cube.Box(
			pos[0] - PlayerWidth/2, 
			pos[1], 
			pos[2] - PlayerWidth/2, 
			pos[0] + PlayerWidth/2, 
			pos[1] + PlayerHeight, 
			pos[2] + PlayerWidth/2, 
		)
	}
}

func SimMovement(in MovementInput) {
	
}