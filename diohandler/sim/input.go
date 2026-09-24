package diosim

import "math"

func (in *MovementInput) inputLen() float64 {
	if in.keyOffset() == 45 || in.keyOffset() == -45 {
		if in.Shift {
			return math.Sqrt2 * 0.98
		}
		return 1
	} else if in.isStop() {
		return 0
	} else {
		return 0.98
	}
}

func (in *MovementInput) movementMul() float64 {
	var moveMul float64 = 1
	if in.isStop() {
		moveMul = 0
	} else if in.isSneak() {
		moveMul = SneakMovementMul
	} else if in.isSprint() {
		moveMul = SprintMovementMul
	}
	return moveMul
}

func (in *MovementInput) isNoMove() bool {
	return in.isStop() && !in.Space
}

func (in *MovementInput) isStop() bool {
	return !(in.Up || in.Left || in.Down || in.Right)
}

func (in *MovementInput) isSprint() bool {
	return in.Ctrl && in.Up
}

func (in *MovementInput) isSneak() bool {
	return in.Shift
}

var directionToOffsets = [9]float64{-135.0, 180.0, 135.0, -90.0, 0.0, 90.0, -45.0, 0.0, 45.0}

func (in *MovementInput) keyOffset() float64 {
	var frontBack, rightLeft int
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