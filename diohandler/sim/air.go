package diosim

import (
	"math"

	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item/enchantment"
)

func (in *MovementInput) travelAir(){
	// friction...
	in.applyFriction(in.LastSlipperiness * SlipperinessToFriction)

	// drag and gravity...
	if !in.appliedLevitation() && !in.OnGround{
		gravity := Gravity
		if _, ok := in.Effect(effect.SlowFalling); in.isFalling() && ok{
			gravity = SlowFallingGravity
		}
		in.Velocity[1] = (in.Velocity[1] - gravity) * AirVerticalDrag
	}
	
	// move...
	_, isSand := in.blockUnder.(block.SoulSand)
	soulSpeed, hasSoulSpeed := in.Armour().Boots().Enchantment(enchantment.SoulSpeed)
	if !in.isStop(){
		var speedMul float64 = 1
		if e, ok := in.Effect(effect.Speed); ok{
			speedMul = 1 + 0.2*float64(e.Level())
		}
		if e, ok := in.Effect(effect.Slowness); ok{
			speedMul *= 1 - 0.15*float64(e.Level())
		}
		
		speed := AirborneDefaultAccel
		if in.isSprint(){
			speed = AirborneSprintAccel
		}
		if in.OnGround{
			speed = in.Speed() * in.movementMul() * max(speedMul, 0) * 
			math.Pow(0.6/dioblocks.DFblockToBlock(in.blockUnder).Slipperiness(), 3)
			if _, isSoil := in.blockUnder.(block.SoulSoil); hasSoulSpeed && (isSand || isSoil){
				speed *= 1.3 + 0.105 * float64(soulSpeed.Level())
			}
		}
		in.moveRelative(speed)
	}
	in.JumpCooldown = max(0, in.JumpCooldown-1)
	if dioblocks.DFblockToBlock(in.Tx().Block(cube.PosFromVec3(in.position))).Climbable(){
		if !in.isNoMove(){
			in.Velocity[1] = ClimbSpeed
		}else if in.OnGround || in.isSneak(){
			in.Velocity[1] = 0
		}else{
			in.Velocity[1] = -ClimbSpeed
		}
	}else if in.Space && in.OnGround && in.JumpCooldown == 0{
		leapLvl := 0
		if l, ok := in.Effect(effect.JumpBoost); ok{
			leapLvl = l.Level()
		}
		in.Velocity[1] = max(in.Velocity[1], JumpSpeed + 0.1*float64(leapLvl))
		in.JumpCooldown = PlayerJumpCooldown
		if in.isSprint(){
			yawRad := in.Yaw() * (math.Pi / 180)
			in.Velocity[0] += SprintJumpBoost * -math.Sin(yawRad)
			in.Velocity[2] += SprintJumpBoost * math.Cos(yawRad)
		}
	}
	if !in.Space{
		in.JumpCooldown = 0
	}
	// TODO add honey friction and silde when added to df
	if !hasSoulSpeed && isSand && in.OnGround{
		in.Velocity[0] *= SoulSandStick
		in.Velocity[2] *= SoulSandStick
	}
}

