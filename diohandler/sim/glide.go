package diosim

import (
	"math"
)

func (in *MovementInput) travelGlide() {
	pitchRad := in.Pitch() * math.Pi / 180
	cosP := math.Cos(pitchRad)
	cosP *= cosP

	// gravity...
	in.applyAirGravity(cosP * 0.75 + DefaultGravityMul)
	
	// move...
	if cosP > Negiaible{
		var speed float64
		velH := math.Hypot(in.Velocity[0], in.Velocity[2])
		if in.isFalling(){
			d := in.Velocity[1] * -0.1 * cosP
			speed += d
			in.Velocity[1] += d
		}
		if pitchRad < 0{
			d := velH * -math.Sin(pitchRad) * 0.04
			speed -= d
			in.Velocity[1] += d * 3.2
		}
		sin, cos := in.yawSinCos()
		in.Velocity[0] += speed * -sin
		in.Velocity[2] += speed * cos
		in.Velocity[0] += (-sin*velH - in.Velocity[0]) * 0.1
		in.Velocity[2] += (cos*velH - in.Velocity[2]) * 0.1
	}

	// friction...
	in.applyFriction(0.99)

	// drag...
	in.Velocity[1] *= AirVerticalDrag
}