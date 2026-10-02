package diosim

import (
	"math"

	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/utils"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const(
	Standing = iota
	Sneaking
	Swimming
	Crawling
	Gliding
)

const(
	PlayerJumpCooldown      = 10
	AirborneSprintAccel     = 0.026
	AirborneDefaultAccel    = 0.02
	SlipperinessToFriction  = 0.91
	DefaultPlayerSpeed      = 0.1
	SprintMovementMul       = 1.3
	SprintJumpBoost         = 0.2
	JumpSpeed               = 0.42
	MomentumThreshold       = 0.003
	MomentumThresholdSq     = MomentumThreshold * MomentumThreshold
	MaxStepHeight           = 0.6
	ClimbSpeed              = 0.1176
	ShrinkedProbeBBoxWidth  = 0.025
	SneakMovementMul        = 0.3
	CobwebVerticalSpeed     = 0.05
	CobwebHorizontalSpeed   = 0.2
	WaterDefaultSpeed       = 0.02
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
	FiuldSpeed              = 0.02
	LavaGravity             = 0.02
	WaterGravity            = 0.08
	ProbeOffset             = 0.003
	AirSlipperness          = 1.0
	FlyFriction             = 0.6
	FlySpeed                = 0.05
	FlyVerticalDrag         = 0.6
	Negiaible               = 1e-5
	DefaultGravityMul       = -1
)

type MovementInput struct{
	*player.Player

    cube.Rotation
    Flags         protocol.InputFlags
    RawMoveVector mgl64.Vec2

    OnGround bool
    Velocity mgl64.Vec3
    Position mgl64.Vec3

	flying           bool
    jumpCooldown     uint
    lastSlipperiness float64
    pose             int
    blockUnder       world.Block
    flow             mgl64.Vec3
    fiuldHeight      float64
}

type MovementResult struct{
	Position     mgl64.Vec3
    Velocity     mgl64.Vec3
    OnGround     bool
    Pose         int
}

func InitializeMovementInput(p *player.Player, in *MovementInput){
	in.Player = p
	in.setBlockUnder()
	in.setOnGround()
	in.lastSlipperiness = in.currSlippernessWithBlockUnder()
}

func (in *MovementInput) SeedFromPlayerAuthInputPacket(pk *packet.PlayerAuthInput){
	// TODO: seed position as well
	in.Flags = pk.InputData
	in.Rotation = cube.Rotation{float64(pk.Yaw), float64(pk.Pitch)}
	in.RawMoveVector = utils.Mgl64Vec2FromMgl32(pk.RawMoveVector)
}

// lots of the movement logic is referenced on LivingEntity.travel() from 
// https://mcsrc.dev/2/26.2/net/minecraft/world/entity/LivingEntity#L2429
func (in *MovementInput) SimMovement() MovementResult{
	if in.RawMoveVector.LenSqr() > 1{
		in.RawMoveVector = in.RawMoveVector.Normalize()
	}
	for axis := range 3{
		if math.Abs(in.Velocity[axis]) < MomentumThreshold{
			in.Velocity[axis] = 0
		}
	}
	in.setBlockUnder()

	if in.Flags.Load(packet.InputFlagStartFlying) && in.GameMode().AllowsFlying(){
		in.flying = true
	}else if in.Flags.Load(packet.InputFlagStopFlying) || (in.GameMode() != world.GameModeCreative && in.OnGround){
		in.flying = false
		in.pose = Standing
	}

	flow, waterHeight := fiuldFlowOnPlayer[block.Water](in)
	in.fiuldHeight = waterHeight
	c := in.Armour().Chestplate()
	if in.pose == Gliding && (c.Durability() == 0 || in.Flags.Load(packet.InputFlagStopGliding) || in.OnGround || waterHeight > 0){
		in.pose = Standing
	}
	if (waterHeight > 1 && in.isSprint()) || (in.pose == Swimming && waterHeight > 0 && in.isSprint()) && !in.flying{
		in.pose = Swimming
	}else if  _, ok := c.Item().(item.Elytra); (in.Flags.Load(packet.InputFlagStartGliding) || in.pose == Gliding) && 
	!in.OnGround && ok && c.Durability() >= 2 && !in.flying{
		in.pose = Gliding
	}else if standInBlock := utils.BBoxIntersectsSolid(in.Tx(), in.bboxWithPose(Standing));
	standInBlock && !utils.BBoxIntersectsSolid(in.Tx(), in.bboxWithPose(Sneaking)){
		in.pose = Sneaking
	}else if standInBlock && !utils.BBoxIntersectsSolid(in.Tx(), in.bboxWithPose(Crawling)){
		in.pose = Crawling
	}else if in.isSneak() && in.pose == Standing && !in.flying{
		in.pose = Sneaking
	}else{
		in.pose = Standing
	}

	if in.flying{
		in.travelFly()
	}else if waterHeight > 0{
		in.flow = flow
		in.travelWater()
	}else if flow, lavaHeight := fiuldFlowOnPlayer[block.Lava](in); lavaHeight > 0{
		in.flow = flow
		in.travelLava()
	}else if in.pose == Gliding{
		in.travelGlide()
	}else{
		in.travelAir()
	}
	in.slowOnCobweb()
	in.stopOnEdge()
	
	maxDt := in.collide()
	in.Position = in.Position.Add(maxDt)
	if maxDt[0] != in.Velocity[0]{in.Velocity[0] = 0}
	yCollision, oldY := maxDt[1] != in.Velocity[1], in.Velocity[1]
	if yCollision{in.Velocity[1] = 0}
	if yCollision && oldY < 0{
		in.OnGround = true
		if _, ok := in.Tx().Block(cube.PosFromVec3(in.Position.Sub(mgl64.Vec3{0, 0.5, 0}))).(block.Slime); 
		ok && !in.isSneak() && !in.isJump(){
			in.Velocity[1] = -oldY
		}
	}else{
		in.setOnGround()
	}
	if maxDt[2] != in.Velocity[2]{in.Velocity[2] = 0}
	if in.OnGround{
		in.lastSlipperiness = in.currSlippernessWithBlockUnder()
	}else{
		in.lastSlipperiness = AirSlipperness
	}

	return MovementResult{
		Position: in.Position,
		Velocity: in.Velocity,
		OnGround: in.OnGround,
		Pose: in.pose,
	}
}

func (in *MovementInput) speed() float64{
	if in.isSprint(){
		return SprintMovementMul * DefaultPlayerSpeed
	}
	return DefaultPlayerSpeed
}

func (in *MovementInput) currSlippernessWithBlockUnder() float64{
	return dioblocks.DFblockToBlock(in.blockUnder).Slipperiness()
}

func (in *MovementInput) setBlockUnder(){
	in.blockUnder = in.Tx().Block(cube.PosFromVec3(in.Position.Sub(mgl64.Vec3{0, 0.5, 0})))
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
	probeBBox := in.feetUnderBBox().ExtendTowards(cube.FaceDown, MaxStepHeight)
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

func (in *MovementInput) maxDelta(aabb cube.BBox, dt mgl64.Vec3) mgl64.Vec3{
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
	sin, cos := in.yawSinCos()
	in.Velocity[0] += speed * (in.RawMoveVector[0]*cos - in.RawMoveVector[1]*sin)
	in.Velocity[2] += speed * (in.RawMoveVector[1]*cos + in.RawMoveVector[0]*sin)
}

func (in *MovementInput) bboxWithPose(pose int) cube.BBox{
	s := in.Scale()
	switch pose{
	case Gliding, Swimming, Crawling:
		return cube.Box(-0.3*s, 0, -0.3*s, 0.3*s, 0.6*s, 0.3*s).Translate(in.Position)
	case Sneaking:
		return cube.Box(-0.3*s, 0, -0.3*s, 0.3*s, 1.49*s, 0.3*s).Translate(in.Position)
	default:
		return cube.Box(-0.3*s, 0, -0.3*s, 0.3*s, 1.8*s, 0.3*s).Translate(in.Position)
	}
}

func (in *MovementInput) bbox() cube.BBox{
	return in.bboxWithPose(in.pose)
}

func (in *MovementInput) feetUnderBBox() cube.BBox{
	s := in.Scale()
	return cube.Box(-0.275*s, -ProbeOffset, -0.275*s, 0.275*s, 0, 0.275*s).Translate(in.Position)
}

func (in *MovementInput) setOnGround(){
	in.OnGround = false
	if in.Velocity[1] == 0 && utils.BBoxIntersectsSolid(in.Tx(), in.feetUnderBBox()) {
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
	in.Velocity[0] *= friction
	in.Velocity[2] *= friction
}

func (in *MovementInput) appliedLevitation() bool{
	if l, ok := in.Effect(effect.Levitation); ok{
		in.Velocity[1] += (LevitationMul * float64(l.Level()) - in.Velocity[1]) * LevitationDrag
		return true
	}
	return false
}

func (in *MovementInput) yawSinCos() (float64, float64){
	yawRad := in.Yaw() * (math.Pi / 180)
	return math.Sin(yawRad), math.Cos(yawRad)
}