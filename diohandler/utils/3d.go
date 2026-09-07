package utils

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
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
	for x := int(math.Floor(pBBox.Min()[0])); x <= int(math.Floor(pBBox.Max()[0])); x++{
		for y := int(math.Floor(pBBox.Min()[0])); y <= int(math.Floor(pBBox.Max()[0])); y++{
			for z := int(math.Floor(pBBox.Min()[0])); z <= int(math.Floor(pBBox.Max()[0])); z++{
				for _, nearby := range BBoxFromWorld(cube.Pos{x, y, z}, tx){
					if pBBox.IntersectsWith(nearby.Translate(cube.Pos{x, y, z}.Vec3())){
						return true
					}
				}
			}
		}
	}
	return false
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