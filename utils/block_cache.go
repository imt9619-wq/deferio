package utils

import (
	"iter"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl64"
)

type SubChunkVec [3]int32

func (s SubChunkVec) Add(pos SubChunkVec) SubChunkVec{
	return SubChunkVec{s[0]+pos[0], s[1]+pos[1], s[2]+pos[2]}
}

func (s SubChunkVec) Sub(pos SubChunkVec) SubChunkVec{
	return SubChunkVec{s[0]-pos[0], s[1]-pos[1], s[2]-pos[2]}
}

func (s SubChunkVec) PosWithin(min, max SubChunkVec) bool{
	return min[0] <= s[0] && max[0] > s[0] && min[1] <= s[1] && max[1] > s[1] && min[2] <= s[2] && max[2] > s[2]
}

func (s SubChunkVec) subIndex() int32{
	return 9*s[0]+3*s[1]+s[2]+1
}

type BlockCache struct {
	startFrom    SubChunkVec
    worldRange   [2]int32
    reg          world.BlockRegistry
    air          uint32
    subs         [][]*chunk.PalettedStorage
}

// A tiny Cache for subchunk near the player, only storing block and liquid, BlockCache is not safe to be accessed concurrently
func NewBlockCache(reg world.BlockRegistry) *BlockCache{
	bc := &BlockCache{
		air: reg.AirRuntimeID(),
		reg: reg,
		subs: make([][]*chunk.PalettedStorage, 27),
	}
	for index := range bc.subs{
		bc.subs[index] = make([]*chunk.PalettedStorage, 0)
	}
	return bc
}

func (bc *BlockCache) offsetBBox(offset SubChunkVec) cube.BBox{
	min := Mgl64FromSubChunkVec(offset.Add(bc.startFrom))
	return Box(min, min.Add(mgl64.Vec3{48, 48, 48}))
}

// CacheNearWithBBox will clone tx's subchunks's parette storages within bbox, changing bc.startFrom and 
// index of bc.subs if bbox position is too far. bbox must have its width, height and lenght all below 16
func (bc *BlockCache) CacheNearWithinBBox(tx *world.Tx, bbox cube.BBox){
	to := SubChunkVecFromVec3(bbox.Min()) // TODO: maybe dont get bbox.min directly but instead some sort of predict position
	shiftBy := to.Sub(bc.startFrom)
	canSwap := shiftBy[0] > 2 || shiftBy[0] < -2 || shiftBy[1] > 2 || shiftBy[1] < -2 || shiftBy[2] > 2 || shiftBy[2] < -2
	if !canSwap && to != bc.startFrom{
		bc.startFrom = to
		for offset, swap := range subOffsetsNSwap(shiftBy){
			if swap{
				bc.subs[offset.Sub(shiftBy).subIndex()] = bc.subs[offset.subIndex()]
			}
			index := offset.subIndex()
			bc.clearSub(index)
			bc.subs[index] = make([]*chunk.PalettedStorage, 0)
		}
		for offset, swap := range subOffsetsNSwap(shiftBy){
			if !swap && bc.offsetBBox(offset).IntersectsWith(bbox){
				bc.cloneToSub(tx, offset)
			}
		}
		return
	}
	for offset := range bc.cacheableBBoxCorners(bbox){
		bc.cloneToSub(tx, offset)
	}
}

func (bc *BlockCache) cloneToSub(tx *world.Tx, offset SubChunkVec){
	storage := chunkAt(tx, world.ChunkPos{offset[0], offset[2]}).SubChunk(int16((bc.worldRange[1]+offset[1]) << 4)).Layers()
	layers := bc.subs[offset.subIndex()]
	for layer := range layers{
		layers[layer] = nil
	}
	if len(layers) != len(storage){
		layers = make([]*chunk.PalettedStorage, len(storage))
	}
	for layer := range layers{
		layers[layer] = storage[layer].Clone()
	}
}

func subOffsetsNSwap(shiftBy SubChunkVec) iter.Seq2[SubChunkVec, bool]{
	return func(yield func(SubChunkVec, bool) bool) {
		var xFrom, xTo, xAdd int32 = 0, 2, 1
		var yFrom, yTo, yAdd int32 = 0, 2, 1
		var zFrom, zTo, zAdd int32 = 0, 2, 1
		if shiftBy[0] < 0{xFrom, xTo = xTo, xFrom; xAdd = -1}
		if shiftBy[1] < 0{yFrom, yTo = yTo, yFrom; yAdd = -1}
		if shiftBy[2] < 0{zFrom, zTo = zTo, zFrom; zAdd = -1}
		for i := xFrom; i*xAdd <= xTo; i += xAdd {
			xSwap := (i+shiftBy[0] >= 0 && i+shiftBy[0] <= 2)
			for j := yFrom; j*yAdd <= yTo; j += yAdd {
				ySwap := (j+shiftBy[1] >= 0 && j+shiftBy[1] <= 2)
				for k := zFrom; k*zAdd <= zTo; k += zAdd {
					zSwap := (k+shiftBy[2] >= 0 && k+shiftBy[2] <= 2)
					if !yield(SubChunkVec{i, j, k}, xSwap && ySwap && zSwap){
						return 
					}
				} 
			} 
		} 
	}
}

func (bc *BlockCache) clearSub(index int32){
	layers := bc.subs[index]
	for layer := range layers{
		layers[layer] = nil
	}
}

func (bc *BlockCache) ChangeWorldTo(tx *world.Tx){
	bc.reg = tx.World().BlockRegistry()
	bc.air = bc.reg.AirRuntimeID()
	bc.worldRange[0] = int32(tx.World().Range().Min())
	bc.worldRange[1] = int32(tx.World().Range().Max())
	for index := range int32(len(bc.subs)){
		bc.clearSub(index)
		bc.subs[index] = make([]*chunk.PalettedStorage, 0)
	}
}

func (bc *BlockCache) Block(pos cube.Pos) world.Block{
	rid, ok := bc.block(pos)
	if !ok || rid == bc.air{
		return bc.reg.Air()
	}
	return bc.reg.BlockByRuntimeIDOrAir(rid)
}

func (bc *BlockCache) cacheableBBoxCorners(bbox cube.BBox) iter.Seq[SubChunkVec]{
	return func(yield func(SubChunkVec) bool) {
		corners := make(map[SubChunkVec]struct{}, 4)
		for _, pos := range bbox.Corners(){
			corners[SubChunkVecFromVec3(pos)] = struct{}{}
		}
		for corner := range corners{
			if corner[1] <= bc.worldRange[0] || corner[1] > bc.worldRange[1]{
				continue
			}
			if !yield(corner){
				return 
			}
		}
	}
}

// The whole point of BlockCache is to avoid using tx for movement simulation, therefore, a tx should only be used for BlockCache 
// when AllSubsInCache is false or when the world of BlockCache is changed
func (bc *BlockCache) AllSubsInCache(bbox cube.BBox) bool{
	if !bc.bboxWithinCachingRange(bbox){
		return false
	}
	for subPos := range bc.cacheableBBoxCorners(bbox){
		if sub, _ := bc.subAt(subPos, 0); sub == nil{
			return false
		}
	}
	return true
}

func (bc *BlockCache) subAt(pos SubChunkVec, layer uint8) (*chunk.PalettedStorage, bool){
	layers := bc.subs[pos.Sub(bc.startFrom).subIndex()]
	if uint8(len(layers)) <= layer{
		return nil, false
	}
	return layers[layer], true
}

func (bc *BlockCache) block(pos cube.Pos) (uint32, bool){
	subPos := SubChunkVecFromCubePos(pos)
	if !subPos.PosWithin(bc.startFrom, bc.startFrom.Add(SubChunkVec{3, 3, 3})){
		return bc.air, false
	}
	sub, inCache := bc.subAt(subPos, 0)
	if !inCache || sub == nil{
		return bc.air, false
	}
	if p := sub.Palette(); p.Len() == 1 && p.Value(0) == bc.air{
		return bc.air, true
	}
	return sub.At(uint8(pos[0]), uint8(pos[1]), uint8(pos[2])), true
}

func (bc *BlockCache) bboxWithinCachingRange(bbox cube.BBox) bool{
	posMin, posMax := SubChunkVecFromVec3(bbox.Min()), SubChunkVecFromVec3(bbox.Max())
	return bc.startFrom[0] <= posMin[0] && bc.startFrom[0]+3 > posMax[0] &&
	bc.startFrom[2] <= posMin[2] && bc.startFrom[2]+3 > posMax[2] && 
	(bc.startFrom[1] <= posMin[1] || bc.startFrom[1] == bc.worldRange[0]) && 
	(bc.startFrom[1]+3 > posMax[1] || bc.startFrom[1]+3 == bc.worldRange[1])
}

// noinspection ALL
//
//go:linkname chunkAt github.com/df-mc/dragonfly/server/world.(*Tx).chunk
func chunkAt(tx *world.Tx, pos world.ChunkPos) *world.Column