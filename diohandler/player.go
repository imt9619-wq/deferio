package diohandler

import "github.com/go-gl/mathgl/mgl64"

type dioPlayer struct{
	pos mgl64.Vec3
	lastTickSpeed mgl64.Vec3
	jumpCooldown int
	onClimb bool
	yaw float64
	serverInplause map[uint]mgl64.Vec3
}

func (d *dioPlayer) Position() mgl64.Vec3{
	return d.pos
}

func (d *dioPlayer) Velocity() mgl64.Vec3{
	return d.pos
}



