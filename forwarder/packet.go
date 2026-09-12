package forwarder

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const(
	SourceClientPacket = iota + 1
	SourceServerPacket 
	SourceDeferioPacket
)

type DioPacket interface{
	DioPacket()
}

type ForwardPacket interface{
	// if is gt Packet, then id will be gt packet id, else dio packet id
	ID() uint32 
    Marshal(io protocol.IO)
}

type PacketWrapper struct{
	pk ForwardPacket
	hdr *Header
}

const(
	IDNewDialPacket = iota + 1
	IDIncomingPlayerPacket 
	IDDisconnectedPlayerPacket
)

var (
	gtClientPool = packet.NewClientPool()
	gtServerPool = packet.NewServerPool()
	dioPool      = map[uint32]func() ForwardPacket{
		IDNewDialPacket:            func() ForwardPacket { return &NewDialPacket{} },
		IDIncomingPlayerPacket:     func() ForwardPacket { return &IncomingPlayerPacket{data: &PlayerGameData{}} },
		IDDisconnectedPlayerPacket: func() ForwardPacket { return &DisconnectedPlayerPacket{} },
	}
)

func packetByHeader(h *Header) (ForwardPacket, error) {
	if h.dioPacket {
		f, ok := dioPool[h.packetID]
		if !ok {
			return nil, fmt.Errorf("forwarder: unknown dio packet id %d", h.packetID)
		}
		return f(), nil
	}
	pool := gtServerPool
	if h.source == SourceClientPacket {
		pool = gtClientPool
	}
	f, ok := pool[h.packetID]
	if !ok {
		return nil, fmt.Errorf("forwarder: unknown gt packet id %d", h.packetID)
	}
	return f(), nil
}

type dioPacket struct{}
func (dioPacket) DioPacket()

type NewDialPacket struct{
	dioPacket
	serverTime int64
}
func (*NewDialPacket) ID() uint32{return IDNewDialPacket}
func (n *NewDialPacket) Marshal(io protocol.IO){
	io.Int64(&n.serverTime)
}

type PlayerGameData struct{
    EntityUniqueID               int64
    EntityRuntimeID              uint64
    PlayerGameMode               int32
    PlayerPosition               mgl32.Vec3
    Pitch                        float32
    Yaw                          float32
    Dimension                    int32
    WorldSpawn                   protocol.BlockPos
    WorldGameMode                int32
    Hardcore                     bool
    GameRules                    []protocol.GameRule 
    Time                         int64
    CustomBlocks                 []protocol.BlockEntry 
    Items                        []protocol.ItemEntry 
    PlayerMovementSettings       protocol.PlayerMovementSettings
    ServerAuthoritativeInventory bool
    PlayerPermissions            byte
    ChunkRadius                  int32
    ChatRestrictionLevel         uint8
    DisablePlayerInteractions    bool
    UseBlockNetworkIDHashes      bool
    PropertyData                 map[string]any 
    Dimensions                   []protocol.DimensionDefinition 
}
func (i *PlayerGameData) Marshal(io protocol.IO){
    io.Int64(&i.EntityUniqueID)
    io.Uint64(&i.EntityRuntimeID)
    io.Int32(&i.PlayerGameMode)
    io.Vec3(&i.PlayerPosition)
    io.Float32(&i.Pitch)
    io.Float32(&i.Yaw)
    io.Int32(&i.Dimension)
    io.BlockPos(&i.WorldSpawn)
    io.Int32(&i.WorldGameMode)
    io.Bool(&i.Hardcore)
    protocol.FuncSlice(io, &i.GameRules, io.GameRule)
    io.Int64(&i.Time)
    protocol.Slice(io, &i.CustomBlocks)
    protocol.Slice(io, &i.Items)
    protocol.PlayerMoveSettings(io, &i.PlayerMovementSettings)
    io.Bool(&i.ServerAuthoritativeInventory)
    io.Uint8(&i.PlayerPermissions)
    io.Int32(&i.ChunkRadius)
    io.Uint8(&i.ChatRestrictionLevel)
    io.Bool(&i.DisablePlayerInteractions)
    io.Bool(&i.UseBlockNetworkIDHashes)
    io.NBT(&i.PropertyData, nbt.NetworkLittleEndian)
    protocol.Slice(io, &i.Dimensions)
}

type IncomingPlayerPacket struct{
	dioPacket
    XUID uint64
    data *PlayerGameData
}
func (*IncomingPlayerPacket) ID() uint32{return IDIncomingPlayerPacket}
func (i *IncomingPlayerPacket) Marshal(io protocol.IO){
	io.Uint64(&i.XUID)
	i.data.Marshal(io)
}

type DisconnectedPlayerPacket struct{dioPacket}
func (*DisconnectedPlayerPacket) ID() uint32{return IDDisconnectedPlayerPacket}
func (*DisconnectedPlayerPacket) Marshal(io protocol.IO){}