package forwarder

import "github.com/sandertv/gophertunnel/minecraft"

type ForwarderPlayerConn interface{
	DisconnectedPlayer()
	ForwardPacket(pk ForwardPacket, source uint8) error
}

type NopForwarderPlayerConn struct{}
func (NopForwarderPlayerConn) DisconnectedPlayer(){}
func (NopForwarderPlayerConn) ForwardPacket(ForwardPacket, uint8) error { return nil }

type ForwarderConn interface {
	NewIncomingPlayer(minecraft.GameData, uint64) ForwarderPlayerConn
}

type NopForwarderConn struct{}
func (NopForwarderConn) NewIncomingPlayer(minecraft.GameData, uint64) ForwarderPlayerConn{
	return NopForwarderPlayerConn{}
}