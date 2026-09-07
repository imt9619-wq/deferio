package diosim

import (
	dioblocks "github.com/deferio/diohandler/blocks"
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
    onClimb                      bool
    yaw, pitch                   float64
    slipperiness                 float64
    baseSpeed                    float64
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

func SimMovement(in MovementInput) MovementResult{
	i := &in
	i.setJumpCooldown()
	i.setOnClimb()
	return MovementResult{
		Position: i.Position,
		Velocity: i.Velocity,
		OnGround: i.OnGround,
		JumpCooldown: i.JumpCooldown,
	}
}

func (in *MovementInput) setJumpCooldown(){
	if !in.Jump{
		in.JumpCooldown = 0
		return
	}else{
		in.JumpCooldown = max(0, in.JumpCooldown-1)
	}
}

func (in *MovementInput) setOnClimb(){
	in.onClimb = dioblocks.DFblockToBlock(in.Tx.Block(cube.PosFromVec3(in.Position))).Climbable()
}