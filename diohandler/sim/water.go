package diosim

import (
	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

var horiFaces = []cube.Pos{{-1}, {1}, {0, 0, -1}, {0, 0, 1}}

func fiuldFlowOnPlayer[T world.Liquid](in *MovementInput) (flow mgl64.Vec3, exist bool){
	aabb := in.bbox()
	sum := mgl64.Vec3{}
	for pos := range utils.CubePosWithInBBox(aabb){
		l, ok := liquidOf[T](in, pos)
		if !ok{
			continue
		}
		surface := float64(pos[1]) + PixelHeight*float64(l.LiquidDepth())
		if surface <= aabb.Min()[1]{
			continue
		}
		exist = true
		sum = sum.Add(liquidFlowVec[T](in, pos, l))
	}
	if sum.LenSqr() > 0{
		flow = sum.Normalize()
	}
	return
}

func liquidOf[T world.Liquid](in *MovementInput, pos cube.Pos) (T, bool){
	var zero T
	fiuld, ok := in.Tx().Liquid(pos)
	if !ok {
		return zero, false
	}
	l, ok := fiuld.(T)
	return l, ok
}

func liquidDecay(l world.Liquid) int{
	if l.LiquidFalling(){
		return 0
	}
	return 8 - l.LiquidDepth()
}

func canFlowInto(in *MovementInput, pos cube.Pos) bool{
	if _, ok := in.Tx().Liquid(pos); ok{
		return true
	}
	return len(in.Tx().Block(pos).Model().BBox(pos, in.Tx())) == 0
}

func liquidFlowVec[T world.Liquid](in *MovementInput, pos cube.Pos, l T) mgl64.Vec3{
	decay := liquidDecay(l)
	var x, y, z float64
	for _, d := range horiFaces{
		side := pos.Add(d)
		if n, ok := liquidOf[T](in, side); ok{
			rd := float64(liquidDecay(n) - decay)
			x += float64(d[0]) * rd
			z += float64(d[2]) * rd
			continue
		}
		if canFlowInto(in, side){
			if n, ok := liquidOf[T](in, side.Add(cube.Pos{0, -1, 0})); ok{
				rd := float64(liquidDecay(n) - (decay - 8))
				x += float64(d[0]) * rd
				z += float64(d[2]) * rd
			}
		}
	}
	vec := mgl64.Vec3{x, y, z}
	if l.LiquidFalling(){
		for _, d := range horiFaces{
			side := pos.Add(d)
			if !canFlowInto(in, side) || !canFlowInto(in, side.Add(cube.Pos{0, 1, 0})){
				if vec.LenSqr() > 0{
					vec = vec.Normalize()
				}
				vec = vec.Add(mgl64.Vec3{0, -6, 0})
				break
			}
		}
	}
	if vec.LenSqr() == 0{
		return mgl64.Vec3{}
	}
	return vec.Normalize()
}

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
		speed += (in.Speed() - speed) * efficenty
	}
	in.applyFriction(waterMul)
	in.Velocity[1] *= WaterDrag

	// gravity...
	in.applyFiuldGravity()

	// move...
	in.fiuldSinkNJump()
	if !in.OnGround{
		efficenty /= DepthStriderAirborneMul
	}
	in.fiuldFlowPush(WaterFlowMul*(1-efficenty))
	// TODO add magma and soul sand bubble sink and push when added to df
	if in.Swimming(){
		lookY := utils.DirNorm(in.Rotation)[1]
		scale := SwimLookScale
		if lookY < SwimLookDownThreshold{
			scale = SwimLookDownScale
		}
		head := cube.PosFromVec3(in.position.Add(mgl64.Vec3{0, SwimEyeOffset}))
		_, fluidOnHead := in.Tx().Liquid(head)
		if lookY <= 0 || in.Space || fluidOnHead{
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
	if in.Space{
		in.Velocity[1] += LiquidJumpSpeed
	}
}

func (in *MovementInput) applyFiuldGravity(){
	if !in.appliedLevitation(){
		in.Velocity[1] -= FiuldGravity
	}
}