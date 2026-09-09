package dioblocks

import (
	"github.com/df-mc/dragonfly/server/world"
)

const(
	PixelHeight float64 = 0.125
)

type Fiuld interface {
	Height()  float64
	DFfiuld() world.Liquid
}

func DFfiuldToFiuld(fu world.Liquid) Fiuld{
	return DefaultFiuld{Liquid: fu}
}

type DefaultFiuld struct{world.Liquid}
func (d DefaultFiuld) Height() float64{return 1 - PixelHeight * float64(d.LiquidDepth())}
func (d DefaultFiuld) DFfiuld() world.Liquid{return d.Liquid}
