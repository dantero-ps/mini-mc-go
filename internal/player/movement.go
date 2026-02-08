package player

import (
	"fmt"
	"math"
	"mini-mc/internal/input"
	"mini-mc/internal/physics"
	"mini-mc/internal/profiling"
	"time"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	Gravity          = 32.0
	TerminalVelocity = -78.4

	WalkSpeed        = 0.1
	SprintMultiplier = 1.3
	SneakMultiplier  = 0.3

	JumpVelocity    = 9.4
	AirAcceleration = 0.02 // jumpMovementFactor
	AirDrag         = 0.98 // Default air drag per tick
	GroundDrag      = 0.91 // Default ground drag (before friction)
	BlockFriction   = 0.6  // Default block friction
)

func (p *Player) UpdatePosition(dt float64, im *input.InputManager) {
	start := time.Now()
	defer func() {
		d := time.Since(start)
		if d > 10*time.Millisecond {
			fmt.Println(d)
		}
	}()
	defer profiling.Track("player.Update.Position")()
	// Update flight mode double-tap timer
	if p.lastSpacePressTime >= 0 {
		p.lastSpacePressTime += dt
		// Reset after 0.5 seconds (longer than double-tap window)
		if p.lastSpacePressTime > 0.5 {
			p.lastSpacePressTime = -1
		}
	}

	// Handle double-tap space for flight mode
	if p.GameMode == GameModeCreative {
		spacePressed := im.IsActive(input.ActionJump)
		spaceJustPressed := im.JustPressed(input.ActionJump)

		if spaceJustPressed {
			if p.lastSpacePressTime >= 0 && p.lastSpacePressTime < 0.3 {
				// Double tap detected - toggle flight mode
				p.IsFlying = !p.IsFlying
				if p.IsFlying {
					p.Velocity[1] = 0 // Stop vertical velocity when entering flight
				}
				p.lastSpacePressTime = -1 // Reset after double tap
			} else {
				p.lastSpacePressTime = 0 // Start timing from first press
			}
		}
		p.lastSpaceState = spacePressed
	} else {
		// Ensure flying is off in survival
		p.IsFlying = false
	}

	// Handle forward double-tap detection for sprint
	if p.lastForwardPressTime >= 0 {
		p.lastForwardPressTime += dt
		// Reset after 0.5 seconds
		if p.lastForwardPressTime > 0.5 {
			p.lastForwardPressTime = -1
		}
	}

	// Handle sprint and sneak
	if !p.IsInventoryOpen {
		forwardJustPressed := im.JustPressed(input.ActionMoveForward)

		// Sprint toggle: either press sprint key or double-tap forward
		if im.IsActive(input.ActionSprint) {
			p.IsSprinting = true
		}

		// Double-tap forward to sprint
		if forwardJustPressed {
			if p.lastForwardPressTime >= 0 && p.lastForwardPressTime < 0.3 {
				// Double tap detected
				p.IsSprinting = true
				p.lastForwardPressTime = -1
			} else {
				p.lastForwardPressTime = 0
			}
		}

		if im.IsActive(input.ActionSneak) {
			p.IsSneaking = true
			p.IsSprinting = false
		} else {
			p.IsSneaking = false
		}
	} else {
		p.IsSprinting = false
		p.IsSneaking = false
	}

	p.PrevPosition = p.Position
	p.PrevDistanceWalkedModified = p.DistanceWalkedModified

	// Get input direction
	forward := float32(0)
	strafe := float32(0)

	if !p.IsInventoryOpen {
		if im.IsActive(input.ActionMoveForward) {
			forward += 1
		}
		if im.IsActive(input.ActionMoveBackward) {
			forward -= 1
		}
		if im.IsActive(input.ActionMoveLeft) {
			strafe -= 1
		}
		if im.IsActive(input.ActionMoveRight) {
			strafe += 1
		}

		// Stop sprinting if not moving forward
		if forward <= 0 {
			p.IsSprinting = false
		}
	}

	// Calculate movement based on camera direction
	yaw := float32(p.CamYaw)
	yawRad := float64(mgl32.DegToRad(yaw))

	// Calculate forward/backward movement vector
	frontX := float32(math.Cos(yawRad))
	frontZ := float32(math.Sin(yawRad))

	// Calculate strafe movement vector (Right vector)
	strafeX := float32(math.Cos(yawRad + math.Pi/2))
	strafeZ := float32(math.Sin(yawRad + math.Pi/2))

	modeDistance := float32(dt * 20.0) // For Drag scaling (time dilation)
	accelScale := modeDistance * 20.0  // For Acceleration scaling (force)

	// Helper to apply movement
	applyMovement := func(s, f, friction float32) {
		dist := s*s + f*f
		if dist >= 0.0001 {
			dist = float32(math.Sqrt(float64(dist)))
			if dist < 1 {
				dist = 1
			}
			dist = friction / dist
			s *= dist
			f *= dist

			// Apply to velocity using our calculated vectors
			// strafe (s) acts along strafe vector
			// forward (f) acts along front vector

			speedX := (s*strafeX + f*frontX) * accelScale
			speedZ := (s*strafeZ + f*frontZ) * accelScale

			p.Velocity[0] += speedX
			p.Velocity[2] += speedZ
		}
	}

	if p.IsFlying {
		// Flight mode physics

		// Vertical input
		if !p.IsInventoryOpen {
			if im.IsActive(input.ActionJump) {
				p.Velocity[1] += 3.0 * modeDistance // Flight accel (matches MC 0.15 blocks/tick)
			} else if im.IsActive(input.ActionSneak) {
				p.Velocity[1] -= 3.0 * modeDistance
			}
		}

		// Horizontal friction in air is different
		friction := float32(1.05) // Flying speed
		if p.IsSprinting {
			friction *= 1.0
		}

		applyMovement(strafe, forward, friction)

		// Drag is applied after position update to match MC behavior
	} else {
		// Normal walking/jumping physics

		// Calculate ground friction factor
		// In MC: f4 = 0.91; if onGround: f4 = blockFriction * 0.91
		currentFriction := float32(GroundDrag)
		if p.OnGround {
			currentFriction = BlockFriction * GroundDrag
		}

		// Calculate acceleration (f5 in MC)
		accel := float32(0.0)
		if p.OnGround {
			// f = 0.16277136 / (currentFriction^3)
			f := 0.16277136 / (currentFriction * currentFriction * currentFriction)

			// Speed attribute
			speed := float32(WalkSpeed)
			if p.IsSprinting {
				speed *= SprintMultiplier
			} else if p.IsSneaking {
				speed *= SneakMultiplier
			}

			accel = speed * f
		} else {
			// Air acceleration
			accel = AirAcceleration
			if p.IsSprinting {
				accel = AirAcceleration + 0.006 // Slight air sprint boost (MC 1.9+ feature but feels good)
			}
		}

		// Apply input acceleration
		correction := float32(1.0)
		if currentFriction < 0.999 { // Avoid divide by zero
			lnD := math.Log(float64(currentFriction))
			correction = float32(-lnD / (1.0 - float64(currentFriction)))
		}

		applyMovement(strafe, forward, accel*correction)

		// Jump
		if !p.IsInventoryOpen && im.IsActive(input.ActionJump) && p.OnGround {
			p.Velocity[1] = JumpVelocity
			p.OnGround = false
			p.JumpStartY = p.Position[1]
			p.MaxJumpHeight = 0

			// Sprint jump boost
			if p.IsSprinting {
				jumpBoost := float32(0.125 * 20.0) // Reduced to 2.5 m/s to match feel
				p.Velocity[0] += frontX * jumpBoost
				p.Velocity[2] += frontZ * jumpBoost
			}
		}
	}

	// Stop completely if very slow (epsilon check)
	if math.Abs(float64(p.Velocity[0])) < 0.005 {
		p.Velocity[0] = 0
	}
	if math.Abs(float64(p.Velocity[2])) < 0.005 {
		p.Velocity[2] = 0
	}

	// Calculate new position using CURRENT velocity
	newPos := p.Position.Add(p.Velocity.Mul(float32(dt)))
	pWidth, pHeight := p.GetBounds()

	if p.IsFlying {
		testPosY := mgl32.Vec3{p.Position[0], newPos[1], p.Position[2]}
		if !physics.Collides(testPosY, pWidth, pHeight, p.World) {
			p.Position[1] = newPos[1]
		} else {
			p.Velocity[1] = 0
		}
	} else {
		// Resolve Y first to avoid stepping up vertical walls
		testPosY := mgl32.Vec3{p.Position[0], newPos[1], p.Position[2]}
		if !physics.Collides(testPosY, pWidth, pHeight, p.World) {
			p.Position[1] = newPos[1]
			if p.Velocity[1] > 0 {
				// Clamp to ceiling if head intersects
				ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, pWidth, pHeight, p.World)
				if p.Position[1]+pHeight > ceiling {
					p.Position[1] = ceiling - pHeight
					p.Velocity[1] = 0
					p.OnGround = false
				}
			} else {
				p.OnGround = false
			}
		} else {
			if p.Velocity[1] <= 0 {
				// Landing on ground
				groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, pWidth, pHeight, p.World)
				if !float32IsInfNeg(groundLevel) {
					p.OnGround = true
					p.Velocity[1] = 0 // Reset vertical velocity on landing
					p.Position[1] = groundLevel
				} else {
					p.OnGround = false
				}
			} else {
				// Hitting ceiling
				ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, pWidth, pHeight, p.World)
				p.Position[1] = ceiling - pHeight
				p.OnGround = false
				p.Velocity[1] = 0
			}
		}
	}

	// Then resolve X at updated Y
	testPosX := mgl32.Vec3{newPos[0], p.Position[1], p.Position[2]}
	if !physics.Collides(testPosX, pWidth, pHeight, p.World) {
		if p.IsSneaking && p.Velocity[1] == 0 && physics.FindGroundLevel(newPos[0], p.Position[2], p.Position, pWidth, pHeight, p.World) < p.Position[1]-0.1 {
			p.Velocity[0] = 0
		} else {
			p.Position[0] = newPos[0]
		}
	} else {
		p.Velocity[0] = 0
		p.IsSprinting = false
	}

	// Finally resolve Z at updated Y
	testPosZ := mgl32.Vec3{p.Position[0], p.Position[1], newPos[2]}
	if !physics.Collides(testPosZ, pWidth, pHeight, p.World) {
		if p.IsSneaking && p.Velocity[1] == 0 && physics.FindGroundLevel(p.Position[0], newPos[2], p.Position, pWidth, pHeight, p.World) < p.Position[1]-0.1 {
			p.Velocity[2] = 0
		} else {
			p.Position[2] = newPos[2]
		}
	} else {
		p.Velocity[2] = 0
		p.IsSprinting = false
	}

	// Final ground settle
	if !p.IsFlying {
		groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, pWidth, pHeight, p.World)
		if !float32IsInfNeg(groundLevel) {
			delta := p.Position[1] - groundLevel
			if p.Velocity[1] <= 0 {
				if delta < -0.001 {
					// We're slightly inside ground due to numerical issues
					p.Position[1] = groundLevel
					p.Velocity[1] = 0
					p.OnGround = true
				} else if delta <= 0.08 {
					// Close enough to ground: stick
					p.Position[1] = groundLevel
					p.Velocity[1] = 0
					p.OnGround = true
				}
			} else if delta > 0.1 {
				p.OnGround = false
			}
		}

		// Double check if on ground
		if p.OnGround {
			checkPos := mgl32.Vec3{p.Position[0], p.Position[1] - 0.01, p.Position[2]}
			if !physics.Collides(checkPos, pWidth, pHeight, p.World) {
				p.OnGround = false
			}
		}
	} else {
		p.OnGround = false
	}

	// Apply flight drag at the end of the tick
	if p.IsFlying {
		groundDrag := float32(0.91)
		groundDragFactor := float32(math.Pow(float64(groundDrag), float64(modeDistance)))
		p.Velocity[0] *= groundDragFactor
		p.Velocity[2] *= groundDragFactor

		verticalDrag := float32(0.6)
		verticalDragFactor := float32(math.Pow(float64(verticalDrag), float64(modeDistance)))
		p.Velocity[1] *= verticalDragFactor
	} else {
		// Apply gravity
		p.Velocity[1] -= Gravity * float32(dt)
		if p.Velocity[1] < TerminalVelocity {
			p.Velocity[1] = TerminalVelocity
		}

		// Recalculate friction for drag
		currentFriction := float32(GroundDrag)
		if p.OnGround { // OnGround updated by physics
			currentFriction = BlockFriction * GroundDrag
		}

		// Apply Drag
		dragFactor := float32(math.Pow(float64(currentFriction), float64(modeDistance)))
		p.Velocity[0] *= dragFactor
		p.Velocity[2] *= dragFactor

		// Vertical drag
		p.Velocity[1] *= float32(math.Pow(float64(AirDrag), float64(modeDistance)))
	}

	positionChange := p.Position.Sub(p.PrevPosition)
	distanceMoved := math.Sqrt(float64(positionChange.X()*positionChange.X() + positionChange.Z()*positionChange.Z()))
	p.DistanceWalkedModified = p.DistanceWalkedModified + distanceMoved*0.6

	// Update fall state
	dy := p.Position.Y() - p.PrevPosition[1]
	p.UpdateFallState(float64(dy), p.OnGround)

	// Update MaxJumpHeight if in air
	if !p.OnGround {
		currentHeight := p.Position[1] - p.JumpStartY
		if currentHeight > p.MaxJumpHeight {
			p.MaxJumpHeight = currentHeight
		}
	}
}

func (p *Player) UpdateFallState(dy float64, onGround bool) {
	if p.IsFlying {
		p.FallDistance = 0
		return
	}

	if onGround {
		if p.FallDistance > 0 {
			// Apply fall damage
			p.Fall(p.FallDistance, 1.0)
			p.FallDistance = 0
		}
	} else if dy < 0 {
		// Falling down
		p.FallDistance -= float32(dy)
	}
}

func (p *Player) Fall(distance float32, damageMultiplier float32) {
	// Jump boost reduction (placeholder logic for now)
	jumpBoostReduction := float32(0.0)
	// TODO: Get jump boost effect amplifier if implemented

	mcDistance := float64(distance)*0.82 + 0.2
	damage := int(math.Ceil((mcDistance - 3.0 - float64(jumpBoostReduction)) * float64(damageMultiplier)))

	if damage > 0 {
		p.ApplyDamage(float32(damage))
		// TODO: Play fall sound
	}
}

// float32IsInfNeg reports whether v is negative infinity.
func float32IsInfNeg(v float32) bool {
	return math.IsInf(float64(v), -1)
}
