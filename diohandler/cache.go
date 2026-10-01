package diohandler

import (
	diosim "github.com/deferio/diohandler/sim"
	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type playerCache struct{
	lastTickDelta      mgl64.Vec3
    lastTick           uint64
    lastTickBlockUnder world.Block
	jumpCooldown       int

	lastTickEffects map[int]int
    movements       *diosim.MovementInput
    tickTODOs       map[uint64][]func()

    serverInpause *utils.OrderedMap[uint64, mgl32.Vec3]
    serverReset   *utils.OrderedMap[uint64, movePlayerData]
    serverEffects *utils.OrderedMap[uint64, effectData]
}

func newPlayerCache(p *player.Player) *playerCache{
	c := &playerCache{
		lastTickEffects: make(map[int]int, 28),
		movements: &diosim.MovementInput{},
		jumpCooldown: diosim.PlayerJumpCooldown,
		serverInpause: utils.NewOrderedCache[uint64, mgl32.Vec3](100),
		serverReset: utils.NewOrderedCache[uint64, movePlayerData](100),
		serverEffects: utils.NewOrderedCache[uint64, effectData](100),
	}
	diosim.InitializeMovementInput(p, c.movements)
	return c
}

func cacheFromPlayer(p *player.Player) *playerCache{
	dih, ok := p.Handler().(*DioHandler)
	if ok{
		return dih.p
	}
	return nil
}

func (c *playerCache) tickTo(to uint64){
	if to <= c.lastTick{
		return
	}
	for i := c.lastTick+1; i < to; i++{
		c.lastTick = i
		if dos, ok := c.tickTODOs[i]; ok{
			for j, do := range dos{
				do()
				dos[j] = nil
			}
			delete(c.tickTODOs, i)
		}
		c.handleEffectData()
	}
	c.lastTick = to
	c.handleEffectData()
}

func (c *playerCache) handleEffectData(){
	if eData, ok := c.serverEffects.Get(c.lastTick); ok{
		et := int(eData.effectType)
		switch eData.operation{
		case packet.MobEffectAdd:
			c.lastTickEffects[et] = int(eData.level)
			if d := eData.duration; d > 0{
				c.addTODO(c.lastTick+uint64(d), func(){
					delete(c.lastTickEffects, et)
				})
			}
		case packet.MobEffectModify:
			if _, ok := c.lastTickEffects[et]; ok{
				c.lastTickEffects[int(eData.effectType)] = int(eData.level)
			}
		case packet.MobEffectRemove:
			delete(c.lastTickEffects, int(eData.effectType))
		}
	}
}

func (c *playerCache) addTODO(tick uint64, do func()){
	if _, ok := c.tickTODOs[tick]; !ok{
		c.tickTODOs[tick] = make([]func(), 0, 5)
	}
	c.tickTODOs[tick] = append(c.tickTODOs[tick], do)
}

// Effect return effect level if effect exist
func (c *playerCache) Effect(e int) (int, bool){
	lvl, ok := c.lastTickEffects[e]
	return lvl, ok
}