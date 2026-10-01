package diosim

import (
	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/go-gl/mathgl/mgl64"
)

func (in *MovementInput) travelWater(){
	// friction and drag...
	var efficenty float64 = 0
	if en, ok := in.Armour().Boots().Enchantment(enchantment.DepthStrider); ok{
		efficenty = min(float64(en.Level())/3, 1)
		if !in.OnGround{
			efficenty *= DepthStriderAirborneMul
		}
	}
	speed, waterMul := WaterDefaultSpeed, WaterDrag
	if efficenty > 0{
		waterMul += (dioblocks.BlockDefaultSlipperiness * SlipperinessToFriction - waterMul) * efficenty
		speed += (in.speed() - speed) * efficenty
	}
	in.applyFriction(waterMul)
	in.Velocity[1] *= WaterDrag

	// gravity...
	if !in.appliedLevitation(){
		in.Velocity[1] -= WaterGravity
	}

	// move...
	in.fiuldSinkNJump()
	if !in.OnGround{
		efficenty /= DepthStriderAirborneMul
	}
	in.fiuldFlowPush(WaterFlowMul*(1-efficenty))
	// TODO: add magma and soul sand bubble sink and push when added to df
	if in.pose == Swimming{
		lookY := utils.DirNorm(in.Rotation)[1]
		scale := SwimLookScale
		if lookY < SwimLookDownThreshold{
			scale = SwimLookDownScale
		}
		head := cube.PosFromVec3(in.Position.Add(mgl64.Vec3{0, SwimEyeOffset}))
		_, fluidOnHead := in.Tx().Liquid(head)
		if lookY <= 0 || in.isJump() || fluidOnHead{
			in.Velocity[1] += (lookY - in.Velocity[1]) * scale
		}
	}
	if !in.isStop(){
		in.moveRelative(speed)
	}
}

func (in *MovementInput) fiuldFlowPush(force float64){
	in.Velocity = in.Velocity.Add(in.flow.Mul(force))
}

func (in *MovementInput) fiuldSinkNJump(){
	if in.isSneak(){
		in.Velocity[1] -= LiquidSinkSpeed
	}
	if in.isJump(){
		in.Velocity[1] += LiquidJumpSpeed
	}
}