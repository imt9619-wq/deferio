package diosim

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (in *MovementInput) isNoMove() bool{
	return in.isStop() && !in.isJump()
}

func (in *MovementInput) isStop() bool{
	return in.RawMoveVector.LenSqr() < MomentumThresholdSq
}

func (in *MovementInput) isSprint() bool{
	return in.Flags.Load(packet.InputFlagSprinting) && !in.isSneak()
}

func (in *MovementInput) isSneak() bool{
	return in.Flags.Load(packet.InputFlagSneaking)
}

func (in *MovementInput) isJump() bool{
	return in.Flags.Load(packet.InputFlagJumpDown)
}
