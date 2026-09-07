package diohandler

import (
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type packetHandler interface {
	Handle(p packet.Packet, s *session.Session, tx *world.Tx, c session.Controllable) error
}

type handleCanceller interface{
	cancelPacketHandle(pk packet.Packet, p *player.Player) bool
}

type packetHandlerWrapper struct{
	dfhandler packetHandler
	diohandler handleCanceller
}

func (*DioHandler) sessionHandleCanceller() map[uint32]handleCanceller{
	return map[uint32]handleCanceller{
		packet.IDPlayerAuthInput: &DioClientPlayerAuthInputHandler{},
	}
}

func (w *packetHandlerWrapper) Handle(p packet.Packet, s *session.Session, tx *world.Tx, c session.Controllable) error{
	if w.diohandler.cancelPacketHandle(p, c.(*player.Player)){
		return nil
	}
	if w.dfhandler != nil{
		return w.dfhandler.Handle(p, s, tx, c)
	}
	return nil
}

func (dih *DioHandler) registerSessionHandlers(){
	handlers, _ := utils.PrivateFieldByName[map[uint32]packetHandler](dih.s, "handlers")
	for id, dioPkHandler := range dih.sessionHandleCanceller(){
		oldHandler := handlers[id]
		handlers[id] = &packetHandlerWrapper{
			dfhandler: oldHandler,
			diohandler: dioPkHandler,
		}
	}
}

type DioClientPlayerAuthInputHandler struct{}
func (*DioClientPlayerAuthInputHandler) cancelPacketHandle(pk packet.Packet, p *player.Player) bool{
	pa := pk.(*packet.PlayerAuthInput)
	cache := cacheFromPlayer(p)
	if cache == nil{
		return false
	}
	_ = pa
	return false
}



