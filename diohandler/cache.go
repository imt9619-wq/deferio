package diohandler

import (
	diosim "github.com/deferio/diohandler/sim"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type playerCache struct{
	lastTickDelta mgl64.Vec3
    lastTick      uint64
    jumpCooldown  int
    serverInpause *utils.TickCache[mgl64.Vec3]
	serverReset   *utils.TickCache[*packet.MovePlayer]
}

func newPlayerCache(p *player.Player) *playerCache{
	return &playerCache{
		jumpCooldown: diosim.PlayerJumpCooldown,
		serverInpause: utils.NewTickCache[mgl64.Vec3](100),
		serverReset: utils.NewTickCache[*packet.MovePlayer](100),
	}
}

func cacheFromPlayer(p *player.Player) *playerCache{
	dih, ok := p.Handler().(*DioHandler)
	if ok{
		return dih.p
	}
	return nil
}