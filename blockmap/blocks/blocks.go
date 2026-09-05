package dioblocks

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
)

type Block interface {
	Climbable() bool
	DFblock() world.Block
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

type defaultPorp struct {
	world.Block
}

func (defaultPorp) Climbable() bool{
	return false
}

func (dp defaultPorp) DFblock() world.Block{
	return dp.Block
}

type Ladder struct{
	defaultPorp
}

func (Ladder) Climbable() bool{
	return true
}

type Vines struct{
	defaultPorp
}

func (Vines) Climbable() bool{
	return true
}

type InvisibleBedrock struct{
	defaultPorp
	block.InvisibleBedrock
}

func (InvisibleBedrock) DFblock() world.Block{
	return block.InvisibleBedrock{}
}

type Air struct{
	defaultPorp
	block.Air
}
func (Air) DFblock() world.Block{
	return block.Air{}
}