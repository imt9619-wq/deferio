package dioblockmap

import (
	"bytes"
	"fmt"
	_ "unsafe"

	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Chunk struct {
	*chunk.Chunk
	// hold player xuid for each subchunk where the subchunk is within 16 block distance of the player, have 
	// the same lenght as sub in Chunk
	subEntities []string 
}

func (c *Chunk) DFchunk() *chunk.Chunk{
	return c.Chunk
}

func newChunk(c *chunk.Chunk) *Chunk{
	return &Chunk{Chunk: c, subEntities: make([]string, 0)}
}

// we assume the clientCache is disenbled, if not, error will occur
func (b *BlockMap) InsertSubChunk(pk *packet.SubChunk) {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	dim, _ := world.DimensionByID(int(pk.Dimension))
	r := dim.Range()
	buf := bytes.NewBuffer(nil)
	for _, entry := range pk.SubChunkEntries {
		entryPos := utils.SubchunkPosAddOffset(pk.Position, entry.Offset)
		if !(entry.Result == protocol.SubChunkResultSuccess || entry.Result == protocol.SubChunkResultSuccessAllAir) {
			continue
		}
		c, ok := b.chunkMap[protocol.ChunkPos{entryPos[0], entryPos[2]}]
		if !ok {
			continue
		}
		ind := uint8(entryPos[1]-int32(r[0]>>4))
		var sub *chunk.SubChunk
		if entry.Result == protocol.SubChunkResultSuccessAllAir{
			sub = chunk.NewSubChunk(b.reg.AirRuntimeID())
		}else{
			rawPayLoad, ok := entry.RawPayload.Value()
			if !ok{
				continue
			}
			buf.Write(rawPayLoad)
			s, err := decodeSubChunk(buf, c.DFchunk(), &ind, chunk.NetworkEncoding)
			buf.Reset()
			if err != nil {
				fmt.Printf("Error when networkdecode subChunk: %s\n", err)
				continue
			}
			sub = s
		}
		c.Sub()[ind] = sub
	}
}

func (b *BlockMap) InsertLevelChunk(pk *packet.LevelChunk) {
	b.rmu.Lock()
	defer b.rmu.Unlock()
	b.currentDim = pk.Dimension
	dim, ok := world.DimensionByID(int(pk.Dimension))
	if !ok{
		pk.Dimension = 0
	}
	r := dim.Range()
	_, ok = pk.SubChunkLimit.Value()
	if ok{		
		b.insertChunk(pk.Position, newChunk(chunk.New(b.reg, r)))
		return
	}
	if pk.CacheEnabled == true{
		
		return
	}
	c, err := chunk.NetworkDecode(b.reg, pk.RawPayload, int(pk.SubChunkCount), r)
	if err != nil {
		fmt.Printf("Error when networkdecode chunk: %s\n", err)
		return
	}
	b.insertChunk(pk.Position, newChunk(c))
}

// noinspection ALL
//
//go:linkname decodeSubChunk github.com/df-mc/dragonfly/server/world/chunk.decodeSubChunk
func decodeSubChunk(buf *bytes.Buffer, c *chunk.Chunk, index *byte, e chunk.Encoding) (*chunk.SubChunk, error)
