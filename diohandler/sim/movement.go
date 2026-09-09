package diosim

import (
	"math"

	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
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
    position              mgl64.Vec3
    onClimb               bool
}

type MovementResult struct{
	Position mgl64.Vec3
	Velocity mgl64.Vec3
	OnGround bool
	JumpCooldown uint
	Slipperiness float64
}

func SimMovement(in *MovementInput) MovementResult{ 
	if fiuldHeightOnPlayer[block.Water](in) > 0{
		in.travelWater()
	}else if fiuldHeightOnPlayer[block.Lava](in) > 0{
		in.travelLava()
	}else{
		in.travelAir()
	}
	in.position = in.position.Add(in.Velocity)
	return MovementResult{
		Position: in.position,
		Velocity: in.Velocity,
		OnGround: in.OnGround,
		JumpCooldown: in.JumpCooldown,
	}
}

func (in *MovementInput) travelWater(){
	oldY := in.position[1]
	slowDown := WaterDefaultMul
	if in.isSprint(){
		slowDown = WaterSprintMul
	}
	if en, ok := in.Armour().Boots().Enchantment(enchantment.DepthStrider); ok{
		depthStriderMul := min(float64(en.Level()), 3)
		if !in.OnGround{
			depthStriderMul *= 0.5
		}
		speedMul += (in.Speed() - speedMul) * depthStriderMul * 0.3
	}
	

}

func (in *MovementInput) travelLava(){

}

func fiuldHeightOnPlayer[T world.Liquid](in *MovementInput) float64{
	aabb := in.bbox()
	highest := 0.0
	for pos := range utils.CubePosWithInBBox(aabb){
		if fiuld, ok := in.Tx().Liquid(pos); ok{
			if _, ok := fiuld.(T); ok{
				fluidHeight := float64(pos[1]) + dioblocks.DFfiuldToFiuld(fiuld).Height()
				if fluidHeight > aabb.Min()[1]{
					highest = max(highest, fluidHeight - aabb.Min()[1])
				}
			}
		}
	}
	return highest
}

func (in *MovementInput) travelAir(){
	in.position = in.Position()
	in.setJumpCooldown()
	in.onClimb = dioblocks.DFblockToBlock(in.Tx().Block(cube.PosFromVec3(in.position))).Climbable()
	in.applyHorizontalMovement()
	if in.Space{
		in.jump()
	}
	if !in.isStop(){
		in.run()
	}
	in.applyGravity()
	aabb := in.bbox()
	for pos := range utils.CubePosWithInBBox(aabb){
		if _, ok := in.Tx().Block(pos).(block.Cobweb); ok && 
		aabb.IntersectsWith(utils.Box(pos.Vec3(), pos.Vec3().Add(mgl64.Vec3{1, 1, 1}))){
			in.Velocity[1] *= CobwebVerticalSpeed
			in.Velocity[0] *= CobwebHorizontalSpeed
			in.Velocity[2] *= CobwebHorizontalSpeed
			break
		}
	}
	in.stopOnEdge()
	in.collide()
}

func (in *MovementInput) setOnGround(){
	in.OnGround = false
	tinyBBox := utils.BBoxOnBBoxFaceWithThreshold(in.bbox(), cube.FaceDown, utils.ProbeOffset)
	if in.Velocity[1] == 0 && utils.BBoxIntersectsSolid(in.Tx(), tinyBBox) {
		in.OnGround = true
	}
}

func (in *MovementInput) applyGravity() {
	if !in.OnGround && !in.onClimb {
		in.Velocity[1] = (in.Velocity[1] - 0.08) * 0.98
		return
	}
	if in.onClimb && !in.Space && !in.isSneak(){
		in.Velocity[1] = ClimbSpeed * -1
	}
}

func (in *MovementInput) collide(){
	if in.Velocity == (mgl64.Vec3{}){
		return
	}
	maxDt := in.maxDelta(in.bbox(), in.Velocity)
	defer func(){in.Velocity = maxDt}()
	xCollision := in.Velocity[0] != maxDt[0]
	yCollision := in.Velocity[1] != maxDt[1]
	zCollision := in.Velocity[2] != maxDt[2]
	onGroundAfterCollision := yCollision && maxDt[1] < 0.0
	// need to be on ground to do step assist 
	// video on step assist: https://www.youtube.com/watch?v=Awa9mZQwVi8
	if !((onGroundAfterCollision || in.OnGround) && (xCollision || zCollision)){
		return
	}
	horizenalVelocity := utils.SetVec3AxisTo(in.Velocity, 1, 0)
	stepDt := in.maxDelta(in.bbox().Translate(mgl64.Vec3{0, MaxStepHeight}), horizenalVelocity)
	stepDt = in.maxDelta(in.bbox().Translate(utils.SetVec3AxisTo(stepDt, 1, -MaxStepHeight)), mgl64.Vec3{0, -MaxStepHeight})
	if stepDt.LenSqr() > maxDt.LenSqr(){
		maxDt = stepDt
		return
	}
	// there might be a ceil
	stepAABB := in.bbox().Extend(horizenalVelocity)
	stepDt = in.maxDelta(stepAABB, mgl64.Vec3{0, MaxStepHeight})
	stepAABB = in.bbox().Translate(mgl64.Vec3{0, stepDt[1]})
	stepDt = in.maxDelta(stepAABB, horizenalVelocity)
	stepDt = in.maxDelta(stepAABB.Translate(stepDt), mgl64.Vec3{0, -MaxStepHeight})
	if stepDt.LenSqr() > maxDt.LenSqr(){
		maxDt = stepDt
	}
}

func (in *MovementInput) maxDelta(aabb cube.BBox, dt mgl64.Vec3) mgl64.Vec3{
	if dt[1] != 0{
		blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 1), math.Abs(dt[1]))
		for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn){
			dt[1] = aabb.YOffset(bbox, dt[1])	
		}
		aabb = aabb.Translate(mgl64.Vec3{0, dt[1]})
	}
	minX := func (){
		if dt[0] != 0{
			blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 0), math.Abs(dt[0]))
			for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn){
				dt[0] = aabb.XOffset(bbox, dt[0])
			}
			aabb = aabb.Translate(mgl64.Vec3{dt[0]})
		}
	}
	minZ := func (){
		if dt[2] != 0{
			blocksIn := aabb.ExtendTowards(utils.FaceOnDeltaAxis(dt, 2), math.Abs(dt[2]))
			for bbox := range utils.BBoxesInBBox(in.Tx(), blocksIn){
				dt[2] = aabb.ZOffset(bbox, dt[2])
			}
			aabb = aabb.Translate(mgl64.Vec3{0, 0, dt[2]})
		}
	}

	if math.Abs(dt[2]) > math.Abs(dt[0]){
		minX()
		minZ()
	}else{
		minZ()
		minX()
	}
	return dt
}

func (in *MovementInput) run() {
	yawRad := in.Yaw() * (math.Pi / 180)
	sinF := -math.Sin(yawRad)
	cosF := math.Cos(yawRad)
	dirRad := (in.keyOffset() + in.Yaw()) * (math.Pi / 180)
	sinD := -math.Sin(dirRad)
	cosD := math.Cos(dirRad)
	if in.OnGround{
		bl := dioblocks.DFblockToBlock(in.Tx().Block(cube.PosFromVec3(in.position.Sub(mgl64.Vec3{0, 0.5, 0}))))
		speedMul := 1.0
		if e, ok := in.Effect(effect.Speed); ok{
			speedMul = 1 + 0.2 * float64(e.Level())
		}
		if e, ok := in.Effect(effect.Slowness); ok{
			speedMul *= 1 - 0.15 * float64(e.Level())
		}
		accel := in.Speed() * in.movementMultiplier() * max(speedMul, 0) * math.Pow(0.6/bl.Friction(), 3)
		in.Velocity[0] += accel * sinD
		in.Velocity[2] += accel * cosD

		if in.Space && !in.onClimb && in.isSprint(){
			in.Velocity[0] += SprintJumpBoost * sinF
			in.Velocity[2] += SprintJumpBoost * cosF
		}
	} else {
		in.Velocity[0] += AirborneAccelration * 0.98 * sinD
		in.Velocity[2] += AirborneAccelration * 0.98 * cosD
	}
}

func (in *MovementInput) applyHorizontalMovement() {
	friction := in.LastSlipperiness * SlipperinessToFriction
	mx := in.Velocity[0] * friction
	mz := in.Velocity[2] * friction
	if math.Abs(mx) < MomentumThreshold {
		mx = 0
	}
	if math.Abs(mz) < MomentumThreshold {
		mz = 0
	}
	in.Velocity[0] = mx
	in.Velocity[2] = mz
}

func (in *MovementInput) jump() {
	if in.onClimb{
		in.Velocity[1] = ClimbSpeed
		return
	}
	if in.OnGround && in.JumpCooldown == 0 {
		in.Velocity[1] = max(in.Velocity[1], JumpSpeed)
		in.JumpCooldown = PlayerJumpCooldown
	}
}

func (in *MovementInput) setJumpCooldown(){
	if !in.Space{
		in.JumpCooldown = 0
		return
	}else{
		in.JumpCooldown = max(0, in.JumpCooldown-1)
	}
}

func (in *MovementInput) stopOnEdge(){
	if !(in.isSneak() && in.OnGround && in.Velocity[1] <= 0){
		return
	}
	in.Velocity[1] = 0
	probeOnEdge := func(axis int){
		planeSign := math.Abs(in.Velocity[axis])/in.Velocity[axis]
		planeFinal := in.Velocity[axis]
		planeFinal -= planeSign*0.05
		if planeFinal != 0{
			if math.Abs(planeFinal)/planeFinal != planeSign{
				planeFinal = 0
			}
		}
		in.Velocity[axis] = planeFinal
	}
	probeBBox := utils.BBoxOnBBoxFaceWithThreshold(in.bbox().Grow(-SneakProbeBBoxShrinks), 
	cube.FaceDown, 
	MaxStepHeight+utils.ProbeOffset+SneakProbeBBoxShrinks)
	for in.Velocity[0] != 0{
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(utils.SetVec3AxisTo(in.Velocity, 2, 0))){
			break
		}
		probeOnEdge(0)
	}
	for in.Velocity[2] != 0{
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(utils.SetVec3AxisTo(in.Velocity, 0, 0))){
			break
		}
		probeOnEdge(2)
	}
	for in.Velocity[0] != 0 && in.Velocity[2] != 0{
		if utils.BBoxIntersectsSolid(in.Tx(), probeBBox.Translate(in.Velocity)){
			break
		}
		probeOnEdge(0)
		probeOnEdge(2)
	}
}

func (in *MovementInput) bbox() cube.BBox{
	return in.H().Type().BBox(in.Player).Translate(in.position)
}