package diosim

import (
	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var horiFaces = []cube.Pos{{-1}, {1}, {0, 0, -1}, {0, 0, 1}}

func fiuldFlowOnPlayer[T world.Liquid](in *MovementInput) (flow mgl64.Vec3, height float64) {
	aabb := in.bbox()
	sum := mgl64.Vec3{}
	for pos := range utils.CubePosWithInBBox(aabb) {
		l, ok := liquidOf[T](in, pos)
		if !ok {
			continue
		}
		heightOnPlayer := float64(pos[1]) + PixelHeight*float64(l.LiquidDepth()) - aabb.Min()[1]
		if heightOnPlayer <= 0 {
			continue
		}
		height = max(height, heightOnPlayer)
		sum = sum.Add(liquidFlowVec(in, pos, l))
	}
	if sum.LenSqr() > 0 {
		flow = sum.Normalize()
	}
	return
}

func liquidOf[T world.Liquid](in *MovementInput, pos cube.Pos) (T, bool) {
	var zero T
	fiuld, ok := in.Tx().Liquid(pos)
	if !ok {
		return zero, false
	}
	l, ok := fiuld.(T)
	return l, ok
}

func liquidDecay(l world.Liquid) int {
	if l.LiquidFalling() {
		return 0
	}
	return 8 - l.LiquidDepth()
}

func canFlowInto(in *MovementInput, pos cube.Pos) bool {
	if _, ok := in.Tx().Liquid(pos); ok {
		return true
	}
	return len(in.Tx().Block(pos).Model().BBox(pos, in.Tx())) == 0
}

func liquidFlowVec[T world.Liquid](in *MovementInput, pos cube.Pos, l T) mgl64.Vec3 {
	decay := liquidDecay(l)
	var x, y, z float64
	for _, d := range horiFaces {
		side := pos.Add(d)
		if n, ok := liquidOf[T](in, side); ok {
			rd := float64(liquidDecay(n) - decay)
			x += float64(d[0]) * rd
			z += float64(d[2]) * rd
			continue
		}
		if canFlowInto(in, side) {
			if n, ok := liquidOf[T](in, side.Add(cube.Pos{0, -1, 0})); ok {
				rd := float64(liquidDecay(n) - (decay - 8))
				x += float64(d[0]) * rd
				z += float64(d[2]) * rd
			}
		}
	}
	vec := mgl64.Vec3{x, y, z}
	if l.LiquidFalling(){
		for _, d := range horiFaces {
			side := pos.Add(d)
			if !canFlowInto(in, side) || !canFlowInto(in, side.Add(cube.Pos{0, 1, 0})) {
				if vec.LenSqr() > 0 {
					vec = vec.Normalize()
				}
				vec = vec.Add(mgl64.Vec3{0, -6, 0})
				break
			}
		}
	}
	if vec.LenSqr() == 0 {
		return mgl64.Vec3{}
	}
	return vec.Normalize()
}