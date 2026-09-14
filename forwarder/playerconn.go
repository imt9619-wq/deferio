package forwarder

import (
	"github.com/sandertv/gophertunnel/minecraft"
)

type PlayerConn struct {
	*Forwarder
    XUID        uint64
    id          uint16
    data        *minecraft.GameData
}

func (c *PlayerConn) NewIncomingPlayer(data minecraft.GameData){
	dataCopy := data
	c.data = &dataCopy
	c.idMu.Lock()
	if lenght := len(c.emptyIdSlot); lenght > 0{
		id := c.emptyIdSlot[lenght-1]
		c.emptyIdSlot = c.emptyIdSlot[:lenght-1]
		c.idToPconn[id] = c
		c.id = uint16(id)
	} else{
		c.idToPconn = append(c.idToPconn, c)
		c.id = uint16(len(c.idToPconn) - 1)
	}
	c.idMu.Unlock()
	c.sendIncPlayer()
}

func (c *PlayerConn) sendIncPlayer(){
	data := c.data
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
	c.idToPconn[c.id] = nil
	c.idMu.Unlock()
}

func (c *PlayerConn) ForwardPacket(pk ForwardPacket, source uint8) error{
	return c.forwardPacket(pk, c.id, source)
}
