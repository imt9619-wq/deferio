package diosim

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

func (in *MovementInput) travelFly(){
	friction := FlyFriction
	speed := FlySpeed
	if in.isSneak() && in.OnGround{
		friction = in.lastSlipperiness * SlipperinessToFriction
		speed = in.airOnGroundMoveMul()
	}else{
		in.Velocity[1] *= FlyVerticalDrag
		if in.isSprint(){
			speed *= 2
		}
		if in.Flags.Load(packet.InputFlagWantUp) || in.Flags.Load(packet.InputFlagAscend) || in.isJump(){
			in.Velocity[1] += speed * 3
		}
		if in.Flags.Load(packet.InputFlagWantDown) || in.Flags.Load(packet.InputFlagDescend) || in.Flags.Load(packet.InputFlagSneakDown){
			in.Velocity[1] -= speed * 3
		}
	}
	in.applyFriction(friction)
	if !in.isStop(){
		in.moveRelative(speed)
	}
}