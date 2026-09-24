package dioblocks

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

const(
	BlockDefaultSlipperiness = 0.6
)

type Block interface {
	Climbable() bool
	DFblock()   world.Block
	Slipperiness() float64
}

func DFblockToBlock(b world.Block) Block{
	switch b.(type) {
	case block.Ladder:
		return Ladder{defaultPorp: defaultPorp{Block: b}}
	case block.Vines:
		return Vines{defaultPorp: defaultPorp{Block: b}}
	default:
		return defaultPorp{Block: b}
	}
}

type defaultPorp struct{world.Block}
func (defaultPorp) Climbable() bool{return false}
func (dp defaultPorp) DFblock() world.Block{return dp.Block}
func (dp defaultPorp) Slipperiness() float64{
	if bl, ok := dp.Block.(interface{Friction() float64}); ok{
		return bl.Friction()
	}
	return BlockDefaultSlipperiness
}

type Ladder struct{defaultPorp}
func (Ladder) Climbable() bool{return true}

type Vines struct{defaultPorp}
func (Vines) Climbable() bool{return true}