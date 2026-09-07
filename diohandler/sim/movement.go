package diosim

import (
	"math"

	dioblocks "github.com/deferio/diohandler/blocks"
	"github.com/deferio/diohandler/utils"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/go-gl/mathgl/mgl64"
)

type MovementInput struct{
	*player.Player
	cube.Rotation
    Up, Down, Left, Right bool
    Shift, Space, Ctrl    bool
    onGround              bool
    Velocity              mgl64.Vec3           
    JumpCooldown          uint
    position              mgl64.Vec3
    onClimb               bool
    slipperiness          float64
}

type MovementResult struct{
	Position mgl64.Vec3
	Velocity mgl64.Vec3
	OnGround bool
	JumpCooldown uint
}

func SimMovement(in *MovementInput) MovementResult{
	in.setOnGround()
	in.position = in.Position()
	in.setJumpCooldown()
	in.onClimb = dioblocks.DFblockToBlock(in.Tx().Block(cube.PosFromVec3(in.position))).Climbable()
	if in.onGround {
		bl := dioblocks.DFblockToBlock(in.Tx().Block(cube.PosFromVec3(in.position.Sub(mgl64.Vec3{0, 0.5, 0}))))
		in.slipperiness = bl.Friction()
	}else{
		in.slipperiness = AirborneSlipperiness
	}

	in.applyHorizontalMovement()
	if in.Space{
		in.jump()
	}
	if !in.isStop(){
		in.run()
	}
	in.applyGravity()
	in.stopOnEdge()
	in.collide()
	in.setOnGround()

	return MovementResult{
		Position: in.position,
		Velocity: in.Velocity,
		OnGround: in.onGround,
		JumpCooldown: in.JumpCooldown,
	}
}

func (in *MovementInput) setOnGround(){
	in.onGround = false
	tinyBBox := utils.BBoxOnBBoxFaceWithThreshold(in.bbox(), cube.FaceDown, utils.ProbeOffset)
	if in.Velocity[1] == 0 && utils.BBoxIntersectsSolid(in.Tx(), tinyBBox) {
		in.onGround = true
	}
}

func (in *MovementInput) applyGravity() {
	if !in.onGround && !in.onClimb {
		in.Velocity[1] = (in.Velocity[1] - 0.08) * 0.98
		return
	}
	if in.onClimb && !in.Space && !in.isSneak(){
		in.Velocity[1] = ClimbSpeed * -1
	}
}

func (in *MovementInput) collide(){
	// TODO
}

func (in *MovementInput) run() {
	yawRad := in.Yaw() * (math.Pi / 180)
	sinF := -math.Sin(yawRad)
	cosF := math.Cos(yawRad)
	dirRad := (in.keyOffset() + in.Yaw()) * (math.Pi / 180)
	sinD := -math.Sin(dirRad)
	cosD := math.Cos(dirRad)

	if in.onGround {
		accel := in.Speed() * in.movementMultiplier() * math.Pow(0.6/in.slipperiness, 3)
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
	friction := in.slipperiness * SlipperinessToFriction
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
	if in.onGround && in.JumpCooldown == 0 {
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
	if !(in.isSneak() && in.onGround && in.Velocity[1] <= 0){
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