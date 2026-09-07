package diohandler

import (
	diosim "github.com/deferio/diohandler/sim"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type tickCache[K any] struct{
	cache map[uint64]K
    size  int
}

func newTickCache[K any](size int) tickCache[K]{
	return tickCache[K]{
		cache: make(map[uint64]K, size),
		size: size,
	}
}

func (t tickCache[K]) set(tick uint64, data K){
	if len(t.cache) == t.size{
		for oldest := range t.cache{
			delete(t.cache, oldest)
			break
		}
	}
	t.cache[tick] = data
}

func (t tickCache[K]) get(tick uint64) (K, bool){
	data, ok := t.cache[tick]
	return data, ok
}

func (t tickCache[K]) tickTo(tick uint64){
	for passTick := range t.cache{
		if passTick < tick{
			delete(t.cache, passTick)
		}else{
			break
		}
	}
}

func (t tickCache[K]) popTick(tick uint64) (K, bool){
	data, ok := t.get(tick)
	t.tickTo(tick+1)
	return data, ok
}

type playerCache struct{
	lastTickDelta mgl64.Vec3
    lastTick      uint64
    jumpCooldown  int
    serverInpause tickCache[mgl64.Vec3]
	serverReset   tickCache[*packet.MovePlayer]
}

func newPlayerCache(p *player.Player) *playerCache{
	return &playerCache{
		jumpCooldown: diosim.PlayerJumpCooldown,
		serverInpause: newTickCache[mgl64.Vec3](100),
		serverReset: newTickCache[*packet.MovePlayer](100),
	}
}

func cacheFromPlayer(p *player.Player) *playerCache{
	dih, ok := p.Handler().(*DioHandler)
	if ok{
		return dih.p
	}
	return nil
}