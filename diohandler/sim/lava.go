package diosim

func (in *MovementInput) travelLava(){
	// friction and drag...
	in.applyFriction(LavaDrag)
	in.Velocity[1] *= LavaDrag
	
	// gravity...
	in.applyFiuldGravity()

	// move...
	in.fiuldSinkNJump()
	in.fiuldFlowPush(LavaFlowPushForce)
	if !in.isStop(){
		in.moveRelative(LavaSpeed)
	}
}