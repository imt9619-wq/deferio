package diosim

import "math"

const (
	PlayerJumpCooldown       = 10
	AirborneSlipperiness     = 1
	AirborneAccelration      = 0.026
	SlipperinessToFriction   = 0.91
	SprintMovementMult       = 1.3
	SprintJumpBoost          = 0.2
	JumpSpeed                = 0.42
	MomentumThreshold        = 0.003
	MaxStepHeight            = 0.6
	ClimbSpeed               = 0.1176
	DefaultBaseSpeed         = 0.1
	SneakProbeBBoxShrinks    = 0.025
	SprintMovementMultiplier = 1.3
	WalkMovementMultiplier   = 1
	SneakMovementMultiplier  = 0.3
)

func (in *MovementInput) movementMultiplier() float64 {
	dirMul := func() float64 {
		if in.keyOffset() == 45 || in.keyOffset() == -45 {
			if in.Shift {
				return math.Sqrt(2) * 0.98
			}
			return 1
		}
		if in.isStop() {
			return 0
		}
		return 0.98
	}()
	moveMul := func() float64 {
		if in.isStop() {
			return 0
		}
		if in.isSneak() {
			return SneakMovementMultiplier
		}
		if in.isSprint() {
			return SprintMovementMultiplier
		}
		return WalkMovementMultiplier
	}()
	return moveMul * dirMul
}

func (in *MovementInput) isStop() bool{
	return !(in.Up || in.Left || in.Down || in.Right)
}

func (in *MovementInput) isSprint() bool{
	return in.Ctrl && in.Up
}

func (in *MovementInput) isSneak() bool{
	return in.Shift
}

var directionToOffsets = [9]float64{-135.0, 180.0, 135.0, -90.0, 0.0, 90.0, -45.0, 0.0, 45.0}

func (in *MovementInput) keyOffset() float64 {
	var frontBack, rightLeft int8
	if !(in.Up == in.Down) {
		if in.Up {
			frontBack = 1
		} else {
			frontBack = -1
		}
	}
	if !(in.Right == in.Left) {
		if in.Left {
			rightLeft = 1
		} else {
			rightLeft = -1
		}
	}
	return directionToOffsets[(frontBack*3)+rightLeft+4]
}