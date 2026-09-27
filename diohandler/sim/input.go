package diosim

import (
	"math"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (in *MovementInput) inputLen() float64{
	// we assume that the flags are correct, i.e. if upleft is on, then so does up, left. Packet validation is the job of AC
	if in.Flags.Load(packet.InputFlagUpRight) || in.Flags.Load(packet.InputFlagUpLeft){
		if in.Flags.Load(packet.InputFlagSneaking){
			return math.Sqrt2 * 0.98
		}
		return 1
	}else if in.isStop(){
		return 0
	}else{
		return 0.98
	}
}

func (in *MovementInput) isNoMove() bool {
	return in.isStop() && !in.isJump()
}

func (in *MovementInput) isStop() bool {
	return !(in.Flags.Load(packet.InputFlagDown) || in.Flags.Load(packet.InputFlagUp) || 
	in.Flags.Load(packet.InputFlagRight) || in.Flags.Load(packet.InputFlagLeft))
}

func (in *MovementInput) isSprint() bool {
	// again, we assume that the client is sprinting if they say so, validations are left for AC
	return in.Flags.Load(packet.InputFlagUp) && in.Flags.Load(packet.InputFlagSprintDown)
}

func (in *MovementInput) isSneak() bool {
	return in.Flags.Load(packet.InputFlagSneaking)
}

func (in *MovementInput) isJump() bool{
	return in.Flags.Load(packet.InputFlagJumpDown)
}

var directionToOffsets = [9]float64{-135.0, 180.0, 135.0, -90.0, 0.0, 90.0, -45.0, 0.0, 45.0}

func (in *MovementInput) keyOffset() float64{
	var frontBack, rightLeft int
	if in.Flags.Load(packet.InputFlagUp) != in.Flags.Load(packet.InputFlagDown){
		if in.Flags.Load(packet.InputFlagUp){
			frontBack = 1
		}else{
			frontBack = -1
		}
	}
	if in.Flags.Load(packet.InputFlagLeft) != in.Flags.Load(packet.InputFlagRight){
		if in.Flags.Load(packet.InputFlagLeft){
			rightLeft = 1
		} else {
			rightLeft = -1
		}
	}
	return directionToOffsets[(frontBack*3)+rightLeft+4]
}