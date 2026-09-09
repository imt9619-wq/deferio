package utils

import (
	"iter"
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func BBoxOnBBoxFaceWithThreshold(self cube.BBox, face cube.Face, threshold float64) cube.BBox {
	min, max := self.Min(), self.Max()
	switch face {
	case cube.FaceUp:
		min[1] = max[1]
		max[1] += threshold
	case cube.FaceDown:
		max[1] = min[1]
		min[1] -= threshold
	case cube.FaceNorth:
		max[2] = min[2]
		min[2] -= threshold
	case cube.FaceEast:
		min[0] = max[0]
		max[0] += threshold
	case cube.FaceSouth:
		min[2] = max[2]
		max[2] += threshold
	default:
		max[0] = min[0]
		min[0] -= threshold
	}
	return cube.Box(min[0], min[1], min[2], max[0], max[1], max[2])
}

func BBoxIntersectsSolid(tx *world.Tx, pBBox cube.BBox) bool{
	for nearby := range BBoxesInBBox(tx, pBBox){
		if pBBox.IntersectsWith(nearby){
			return true
		}
	}
	return false
}

func BBoxesInBBox(tx *world.Tx, bb cube.BBox) iter.Seq[cube.BBox]{
	return func(yield func(cube.BBox) bool) {
		for pos := range CubePosWithInBBox(bb){
			for _, nearby := range BBoxFromWorld(pos, tx){
				if !yield(nearby.Translate(pos.Vec3())){
					return 
				}
			}
		}
	}
}

func CubePosWithInBBox(bb cube.BBox) iter.Seq[cube.Pos]{
	return func(yield func(cube.Pos) bool) {
		for x := int(math.Floor(bb.Min()[0])); x <= int(math.Floor(bb.Max()[0])); x++{
			for y := int(math.Floor(bb.Min()[0])); y <= int(math.Floor(bb.Max()[0])); y++{
				for z := int(math.Floor(bb.Min()[0])); z <= int(math.Floor(bb.Max()[0])); z++{
					if !yield(cube.Pos{x, y, z}){
						return 
					}
				}
			}
		}
	}
}

func BBoxFromWorld(pos cube.Pos, tx *world.Tx) []cube.BBox{
	return tx.Block(pos).Model().BBox(pos, tx)
}

func SetVec3AxisTo(vec mgl64.Vec3, axis int, to float64) mgl64.Vec3{
	vec[axis] = to
	return vec
}

func DirNorm(ro cube.Rotation) mgl64.Vec3{
	yawRad := ro.Yaw() * math.Pi / 180.0
	pitchRad := ro.Pitch() * math.Pi / 180.0

	x := -math.Sin(yawRad) * math.Cos(pitchRad)
	y := -math.Sin(pitchRad)
	z := math.Cos(yawRad) * math.Cos(pitchRad)

	return mgl64.Vec3{x, y, z}.Normalize()
}

func FaceOnDeltaAxis(delta mgl64.Vec3, axis int) cube.Face{
	switch axis{
	case 0:
		if delta[axis] > 0{
			return cube.FaceEast
		}else{
			return cube.FaceWest
		}
	case 1:
		if delta[axis] > 0{
			return cube.FaceUp
		}else{
			return cube.FaceDown
		}
	default:
		if delta[axis] > 0{
			return cube.FaceSouth
		}else{
			return cube.FaceNorth
		}
	}
}

func RayTraceFromOrigin(aabb cube.BBox, origin mgl64.Vec3, dir mgl64.Vec3) (mgl64.Vec2, bool){
	tmin := math.Inf(-1)
	tmax := math.Inf(1)
	for axis := range 3{
		if dir[axis] != 0.0 {
			tx1 := (aabb.Min()[axis] - origin[axis]) / dir[axis]
			tx2 := (aabb.Max()[axis] - origin[axis]) / dir[axis]
			tmin = math.Max(tmin, math.Min(tx1, tx2))
			tmax = math.Min(tmax, math.Max(tx1, tx2))
		} else if origin[axis] < aabb.Min()[axis] || origin[axis] > aabb.Max()[axis]{
			return mgl64.Vec2{}, false
		}
	}
	if tmax >= 0 && tmin <= tmax {
		if tmin < 0 {
			return mgl64.Vec2{}, true
		}
		return mgl64.Vec2{tmin, tmax}, true
	}
	return mgl64.Vec2{}, false
}

func PlayerBBox(p *player.Player) cube.BBox{
	return p.H().Type().BBox(p).Translate(p.Position())
}