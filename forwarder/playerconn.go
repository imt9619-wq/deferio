package forwarder

import (
	"github.com/sandertv/gophertunnel/minecraft"
)

type PlayerConn struct {
	*Forwarder
	XUID uint64
	id   uint16
}

func (c *PlayerConn) NewIncomingPlayer(data minecraft.GameData){
	c.idMu.Lock()
	if lenght := len(c.emptyIdSlot); lenght > 0 {
		id := c.emptyIdSlot[lenght-1]
		c.emptyIdSlot = c.emptyIdSlot[:lenght-1]
		c.idToXuid[id] = c.XUID
		c.id = uint16(id)
	} else {
		c.idToXuid = append(c.idToXuid, c.XUID)
		c.id = uint16(len(c.idToXuid) - 1)
	}
	c.idMu.Unlock()
	pk := &IncomingPlayerPacket{
		XUID: c.XUID,
		data: &PlayerGameData{
			EntityUniqueID: data.EntityUniqueID,
			EntityRuntimeID: data.EntityRuntimeID,
			PlayerGameMode: data.PlayerGameMode,
			PlayerPosition: data.PlayerPosition,
			Pitch: data.Pitch,
			Yaw: data.Yaw,
			Dimension: data.Dimension,
			WorldSpawn: data.WorldSpawn,
			WorldGameMode: data.WorldGameMode,
			Hardcore: data.Hardcore,
			GameRules: data.GameRules,
			Time: data.Time,
			CustomBlocks: data.CustomBlocks,
			Items: data.Items,
			PlayerMovementSettings: data.PlayerMovementSettings,
			ServerAuthoritativeInventory: data.ServerAuthoritativeInventory,
			PlayerPermissions: data.PlayerPermissions,
			ChunkRadius: data.ChunkRadius,
			ChatRestrictionLevel: data.ChatRestrictionLevel,
			DisablePlayerInteractions: data.DisablePlayerInteractions,
			UseBlockNetworkIDHashes: data.UseBlockNetworkIDHashes,
			PropertyData: data.PropertyData,
			Dimensions: data.Dimensions,
		},
	}
	c.setShieldID(pk)
	c.ForwardPacket(pk, SourceDeferioPacket)
}

func (c *PlayerConn) DisconnectedPlayer(){
	c.ForwardPacket(&DisconnectedPlayerPacket{}, SourceDeferioPacket)
	c.idMu.Lock()
	c.emptyIdSlot = append(c.emptyIdSlot, int(c.id))
	c.idMu.Unlock()
}

func (c *PlayerConn) ForwardPacket(pk ForwardPacket, source uint8) error {
	return c.forwardPacket(pk, c.id, source)
}
