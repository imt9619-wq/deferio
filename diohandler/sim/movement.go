package diosim

import (
	"math"

	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

const (
	PlayerJumpCooldown      = 10
	AirborneSprintAccel     = 0.026
	AirborneDefaultAccel    = 0.02
	SlipperinessToFriction  = 0.91
	SprintMovementMul       = 1.3
	SprintJumpBoost         = 0.2
	JumpSpeed               = 0.42
	MomentumThreshold       = 0.003
	MaxStepHeight           = 0.6
	ClimbSpeed              = 0.1176
	SneakProbeBBoxShrinks   = 0.025
	SneakMovementMul        = 0.3
	CobwebVerticalSpeed     = 0.05
	CobwebHorizontalSpeed   = 0.2
	WaterDefaultSpeed       = 0.08
	DepthStriderAirborneMul = 0.5
	SlowFallingGravity      = 0.01
	LiquidSinkSpeed         = 0.04
	LiquidJumpSpeed         = 0.04
	SwimLookDownThreshold   = -0.2
	SwimLookDownScale       = 0.085
	SwimLookScale           = 0.06
	SwimEyeOffset           = 0.9
	Gravity                 = 0.08
	WaterDrag               = 0.8
	AirVerticalDrag         = 0.98
	WaterFlowMul            = 0.014
	PixelHeight             = 0.125
	LevitationMul           = 0.05
	LevitationDrag          = 0.2
	SoulSandStick           = 0.4
	LavaDrag                = 0.5
	LavaFlowPushForce       = 0.014 
	LavaSpeed               = 0.02
	FiuldGravity            = 0.02
)

type MovementInput struct{
	*player.Player
	cube.Rotation
    Up, Down, Left, Right bool
    Shift, Space, Ctrl    bool
    OnGround              bool
    Velocity              mgl64.Vec3           
    JumpCooldown          uint
	LastSlipperiness      float64
	blockUnder            world.Block
    position              mgl64.Vec3
	flow                  mgl64.Vec3
}

type MovementResult struct{
	Position mgl64.Vec3
	Velocity mgl64.Vec3
	OnGround bool
	JumpCooldown uint
	Slipperiness float64
}

// lots of the movement logic is referenced on LivingEntity.travel() from 
// https://mcsrc.dev/2/26.2/net/minecraft/world/entity/LivingEntity#L2429
func SimMovement(in *MovementInput) MovementResult{
	for axis := range 3{
		if math.Abs(in.Velocity[axis]) < MomentumThreshold{
			in.Velocity[axis] = 0
		}
	}
	in.position = in.Position()
	in.blockUnder = in.Tx().Block(cube.PosFromVec3(in.position.Sub(mgl64.Vec3{0, 0.5, 0})))

	if flow, exist := fiuldFlowOnPlayer[block.Water](in); exist{
		in.flow = flow
		in.travelWater()
	}else if flow, exist := fiuldFlowOnPlayer[block.Lava](in); exist{
		in.flow = flow
		in.travelLava()
	}else{
		in.travelAir()
	}
	in.slowOnCobweb()
	in.stopOnEdge()
	
	maxDt := in.collide()
	in.position = in.position.Add(maxDt)
	if maxDt[0] != in.Velocity[0]{in.Velocity[0] = 0}
	yCollision, isFalling := maxDt[1] != in.Velocity[1], in.Velocity[1] < 0
	if yCollision{in.Velocity[1] = 0}
	if yCollision && isFalling{
		in.OnGround = true
		if _, ok := in.Tx().Block(cube.PosFromVec3(in.position.Sub(mgl64.Vec3{0, 0.5, 0}))).(block.Slime); 
		ok && !in.isSneak() && !in.Space{
			in.Velocity[1] = -in.Velocity[1]
		}
	}else{
		in.setOnGround()
	}
	if maxDt[2] != in.Velocity[2]{in.Velocity[2] = 0}

	return MovementResult{
		Position: in.position,
		Velocity: in.Velocity,
		OnGround: in.OnGround,
		JumpCooldown: in.JumpCooldown,
		Slipperiness: dioblocks.DFblockToBlock(in.blockUnder).Slipperiness(),
	}
}

func (in *MovementInput) bbox() cube.BBox{
	return in.H().Type().BBox(in.Player).Translate(in.position)
}

func (in *MovementInput) stopOnEdge() {
	if !(in.isSneak() && in.OnGround && in.Velocity[1] <= 0) {
		return
	}
	in.Velocity[1] = 0
	probeOnEdge := func(axis int) {
		planeSign := math.Abs(in.Velocity[axis]) / in.Velocity[axis]
		planeFinal := in.Velocity[axis]
		planeFinal -= planeSign * 0.05
		if planeFinal != 0 {
			if math.Abs(planeFinal)/planeFinal != planeSign {
				planeFinal = 0
			}
		}
		in.Velocity[axis] = planeFinal
	}
	probeBBox := utils.BBoxOnBBoxFaceWithThreshold(in.bbox().Grow(-SneakProbeBBoxShrinks),
		cube.FaceDown,
		MaxStepHeight+utils.ProbeOffset+SneakProbeBBoxShrinks)
	for in.Velocity[0] != 0 {
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(utils.SetVec3AxisTo(in.Velocity, 2, 0))) {
			break
		}
		probeOnEdge(0)
	}
	for in.Velocity[2] != 0 {
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(utils.SetVec3AxisTo(in.Velocity, 0, 0))) {
			break
		}
		probeOnEdge(2)
	}
	for in.Velocity[0] != 0 && in.Velocity[2] != 0 {
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(in.Velocity)) {
			break
		}
		probeOnEdge(0)
		probeOnEdge(2)
	}
}

func (in *MovementInput) collide() mgl64.Vec3{
	if in.Velocity == (mgl64.Vec3{}){
		return in.Velocity
	}
	maxDt := in.maxDelta(in.bbox(), in.Velocity)
	xCollision := in.Velocity[0] != maxDt[0]
	yCollision := in.Velocity[1] != maxDt[1]
	zCollision := in.Velocity[2] != maxDt[2]
	onGroundAfterCollision := yCollision && maxDt[1] < 0.0
	// need to be on ground to do step assist
	// video on step assist: https://www.youtube.com/watch?v=Awa9mZQwVi8
	if !((onGroundAfterCollision || in.OnGround) && (xCollision || zCollision)) {
		return maxDt
	}
	horizenalVelocity := utils.SetVec3AxisTo(in.Velocity, 1, 0)
	stepDt := in.maxDelta(in.bbox().Translate(mgl64.Vec3{0, MaxStepHeight}), horizenalVelocity)
	stepDt = in.maxDelta(in.bbox().Translate(utils.SetVec3AxisTo(stepDt, 1, -MaxStepHeight)), mgl64.Vec3{0, -MaxStepHeight})
	if stepDt.LenSqr() > maxDt.LenSqr(){
		return stepDt
	}
	// there might be a ceil
	stepAABB := in.bbox().Extend(horizenalVelocity)
	stepDt = in.maxDelta(stepAABB, mgl64.Vec3{0, MaxStepHeight})
	stepAABB = in.bbox().Translate(mgl64.Vec3{0, stepDt[1]})
	stepDt = in.maxDelta(stepAABB, horizenalVelocity)
	stepDt = in.maxDelta(stepAABB.Translate(stepDt), mgl64.Vec3{0, -MaxStepHeight})
	if stepDt.LenSqr() > maxDt.LenSqr(){
		return stepDt
	}
	return maxDt
}

func (in *MovementInput) isFalling() bool{
	return in.Velocity[1] < 0
}

func (in *MovementInput) maxDelta(aabb cube.BBox, dt mgl64.Vec3) mgl64.Vec3 {
	if dt[1] != 0 {
		blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 1), math.Abs(dt[1]))
		for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn) {
			dt[1] = aabb.YOffset(bbox, dt[1])
		}
		aabb = aabb.Translate(mgl64.Vec3{0, dt[1]})
	}
	minX := func() {
		if dt[0] != 0 {
			blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 0), math.Abs(dt[0]))
			for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn) {
				dt[0] = aabb.XOffset(bbox, dt[0])
			}
			aabb = aabb.Translate(mgl64.Vec3{dt[0]})
		}
	}
	minZ := func() {
		if dt[2] != 0 {
			blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 2), math.Abs(dt[2]))
			for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn) {
				dt[2] = aabb.ZOffset(bbox, dt[2])
			}
			aabb = aabb.Translate(mgl64.Vec3{0, 0, dt[2]})
		}
	}
	if math.Abs(dt[2]) > math.Abs(dt[0]) {
		minX()
		minZ()
	} else {
		minZ()
		minX()
	}
	return dt
}

func (in *MovementInput) moveRelative(speed float64){
	off := in.keyOffset()
	dirRad := (off + in.Yaw()) * (math.Pi / 180)
	accel := speed * in.inputLen()
	in.Velocity[0] += accel * -math.Sin(dirRad)
	in.Velocity[2] += accel * math.Cos(dirRad)
}

func (in *MovementInput) setOnGround(){
	in.OnGround = false
	tinyBBox := utils.BBoxOnBBoxFaceWithThreshold(in.bbox(), cube.FaceDown, utils.ProbeOffset)
	if in.Velocity[1] == 0 && utils.BBoxIntersectsSolid(in.Tx(), tinyBBox) {
		in.OnGround = true
	}
}

func (in *MovementInput) slowOnCobweb(){
	aabb := in.bbox()
	for pos := range utils.CubePosWithInBBox(aabb) {
		if _, ok := in.Tx().Block(pos).(block.Cobweb); ok &&
			aabb.IntersectsWith(utils.PosBound(pos)){
			in.Velocity[1] *= CobwebVerticalSpeed
			in.Velocity[0] *= CobwebHorizontalSpeed
			in.Velocity[2] *= CobwebHorizontalSpeed
			break
		}
	}
}

func (in *MovementInput) applyFriction(friction float64){
	in.Velocity[0] = in.Velocity[0] * friction
	in.Velocity[2] = in.Velocity[2] * friction
}

func (in *MovementInput) appliedLevitation() bool{
	if l, ok := in.Effect(effect.Levitation); ok{
		in.Velocity[1] += (LevitationMul * float64(l.Level()) - in.Velocity[1]) * LevitationDrag
		return true
	}
	return false
}