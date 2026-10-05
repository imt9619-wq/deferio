package utils

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	ProbeOffset = 0.003
)

func SubChunkVecFromSubIndex(index int32) SubChunkVec{
	index -= 1
	s := SubChunkVec{}
	s[0] = index/9-1
	index = index%9
	s[1] = index/3-1
	index = index%3
	s[2] = index-1
	return s
}

func SubChunkVecFromVec3(vec3 mgl64.Vec3) SubChunkVec{
	return SubChunkVec{int32(math.Floor(vec3[0])) >> 4, int32(math.Floor(vec3[1])) >> 4, int32(math.Floor(vec3[2])) >> 4}
}

func Mgl64FromSubChunkVec(s SubChunkVec) mgl64.Vec3{
	return mgl64.Vec3{float64(s[0] << 4), float64(s[1] << 4), float64(s[2] << 4)}
}

func SubChunkVecFromCubePos(pos cube.Pos) SubChunkVec{
	return SubChunkVec{int32(pos[0]) >> 4, int32(pos[1]) >> 4, int32(pos[2]) >> 4}
}

func Mgl64Vec2FromMgl32(vec mgl32.Vec2) mgl64.Vec2{
	return mgl64.Vec2{float64(vec[0]), float64(vec[0])}
}

func Mgl32FromMgl64(vec mgl64.Vec3) mgl32.Vec3{
	return mgl32.Vec3{float32(vec[0]), float32(vec[1]), float32(vec[2])}
}

func Mgl64FromMgl32(vec mgl32.Vec3) mgl64.Vec3{
	return mgl64.Vec3{float64(vec[0]), float64(vec[1]), float64(vec[2])}
}

func Mgl32FromCubePos(pos cube.Pos) mgl32.Vec3{
	return mgl32.Vec3{float32(pos[0]), float32(pos[1]), float32(pos[2])}
}

func Box(min, max mgl64.Vec3) cube.BBox{
	return cube.Box(min[0], min[1], min[2], max[0], max[1], max[2])
}

func CubePosFromVec3(vec mgl32.Vec3) cube.Pos{
	return cube.Pos{int(math32.Floor(vec[0])), int(math32.Floor(vec[0])), int(math32.Floor(vec[0]))}
}

func PosFromVec3(vec mgl32.Vec3) protocol.BlockPos{
	return protocol.BlockPos{int32(math32.Floor(vec[0])), int32(math32.Floor(vec[0])), int32(math32.Floor(vec[0]))}
}

func ChunkPosFromPos(pos protocol.BlockPos) protocol.ChunkPos{
	return protocol.ChunkPos{pos[0] >> 4, pos[2] >> 4}
} 

func SubchunkPosAddOffset(pos protocol.SubChunkPos, offset protocol.SubChunkOffset) protocol.SubChunkPos{
	return protocol.SubChunkPos{pos[0]+int32(offset[0]), pos[1]+int32(offset[1]), pos[2]+int32(offset[2])}
}