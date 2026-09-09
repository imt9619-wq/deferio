package utils

import (
	"github.com/chewxy/math32"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const(
	ProbeOffset = 0.003
)

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