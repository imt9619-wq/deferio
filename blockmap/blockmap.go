package dioblockmap

import (
	"sync"

	_ "github.com/df-mc/dragonfly/server/block"
	dioblocks "github.com/deferio/blockmap/blocks"
	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

type dioBlockRegistry struct{
	world.BlockRegistry
	runtimeToBlock map[uint32]dioblocks.Block
}

func newDioBlockRegistry(br world.BlockRegistry) dioBlockRegistry{
	dioB := dioBlockRegistry{
		BlockRegistry: br,
		runtimeToBlock: make(map[uint32]dioblocks.Block, len(br.Blocks())),
	}
	for rid, bl := range br.Blocks() {
		dioB.runtimeToBlock[uint32(rid)] = dioblocks.DFblockToBlock(bl)
	}
	return dioB
}

func (d *dioBlockRegistry) BlockByRuntimeID(rid uint32) (dioblocks.Block, bool){
	bl, ok := d.runtimeToBlock[rid]
	if !ok{
		bl = dioblocks.Air{}
	}
	return bl, ok
}

type BlockMap struct{
	reg         dioBlockRegistry
    rmu         *sync.RWMutex
    chunkMap    map[protocol.ChunkPos]*Chunk
    chunkRadius int32
    chunkCentre protocol.ChunkPos
    currentDim  int32
}

func NewBlockMap(data *minecraft.GameData, reg world.BlockRegistry) *BlockMap{
	bm := &BlockMap{}
	bm.reg = newDioBlockRegistry(reg)
	bm.rmu = &sync.RWMutex{}
	bm.chunkRadius = data.ChunkRadius
	bm.chunkCentre = utils.ChunkPosFromMgl32(data.PlayerPosition)
	bm.chunkMap = make(map[protocol.ChunkPos]*Chunk, (bm.chunkRadius+1)*(bm.chunkRadius+1))
	return bm
}

func (b *BlockMap) Dimension() world.Dimension{
	b.rmu.RLock()
	defer b.rmu.RUnlock()
	dim, _ := world.DimensionByID(int(b.currentDim))
	return dim
}

func (b *BlockMap) UpdateChunkCentre(pos mgl32.Vec3) {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	chunkCentre := utils.ChunkPosFromMgl32(pos)
	if b.chunkCentre == chunkCentre {
		return
	}
	b.chunkCentre = chunkCentre
}

func (b *BlockMap) RefreshMapWithRenderDistance() {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	for chunk := range b.chunkMap{
		if !b.isRenderedChunk(chunk) {
			delete(b.chunkMap, chunk)
		}
	}
}

func (b *BlockMap) UpdateChunkRadius(r int32) {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	b.chunkRadius = r
}

func (b *BlockMap) insertChunk(pos protocol.ChunkPos, chunk *Chunk) {
	if !b.isRenderedChunk(pos) {
		return
	}
	b.chunkMap[pos] = chunk
}

func (b *BlockMap) SetBlock(pos protocol.BlockPos, layer uint8, block uint32) {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	chunkPos := utils.ChunkPosFromPos(pos)
	chunk, ok := b.chunkMap[chunkPos]
	if !ok {
		return
	}
	x := uint8(pos[0] & 0xF)
	y := int16(pos[1])
	z := uint8(pos[2] & 0xF)
	chunk.SetBlock(x, y, z, layer, block)
}

func (b *BlockMap) block(pos cube.Pos, layer uint8) (bl dioblocks.Block){
	b.rmu.RLock()
	defer b.rmu.RUnlock()
	bl = dioblocks.Air{}
	if layer > 1 {
		return 
	}

	chunkPos := utils.ChunkPosFromCubePos(pos)
	c, ok := b.chunkMap[chunkPos]
	if !ok {
		bl = dioblocks.InvisibleBedrock{}
		return
	}

	localX := uint8(pos[0]) & 0xF
	localZ := uint8(pos[2]) & 0xF
	worldY := int16(pos[1])

	if !(c.Range()[0] <= int(worldY) && int(worldY) <= c.Range()[1]){
		return 
	}
	rid := c.Block(localX, worldY, localZ, layer)

	bl, _ = b.reg.BlockByRuntimeID(rid)
	return
}

func (b *BlockMap) BBoxes(pos cube.Pos) []cube.BBox32{
	bbs := b.block(pos, 0).DFblock().Model().BBox(pos, b)
	bb32s := make([]cube.BBox32, 0, len(bbs))
	for _, bb := range bbs{
		bb32s = append(bb32s, utils.BBox32FromBBox(bb).Translate(utils.Mgl32FromCubePos(pos)))
	}
	return bb32s
}

// Block implements world.BlockSource.
func (b *BlockMap) Block(pos cube.Pos) world.Block{
	return b.block(pos, 0).DFblock()
}

func (b *BlockMap) isRenderedChunk(chunk [2]int32) bool {
	return b.chunkCentre[0]-b.chunkRadius <= chunk[0] && chunk[0] <= b.chunkCentre[0]+b.chunkRadius &&
	b.chunkCentre[1]-b.chunkRadius <= chunk[1] && chunk[1] <= b.chunkCentre[1]+b.chunkRadius
}