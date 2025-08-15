package player

import (
	"fmt"
	"math"

	"mini-mc/internal/physics"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	PlayerEyeHeight  = 1.62
	PlayerHeight     = 1.8 // Player collision height
	BaseMoveSpeed    = 4.317
	SprintMultiplier = 1.5
	SneakMultiplier  = 0.3
	Gravity          = 32.0
	JumpVelocity     = 9.4
	TerminalVelocity = -78.4 // Maximum fall speed
)

type Player struct {
	Position    mgl32.Vec3
	Velocity    mgl32.Vec3
	OnGround    bool
	IsSprinting bool
	IsSneaking  bool
	CamYaw      float64
	CamPitch    float64
	LastMouseX  float64
	LastMouseY  float64
	FirstMouse  bool

	// Interaction
	HoveredBlock    [3]int
	HasHoveredBlock bool

	World *world.World
}

func New(world *world.World) *Player {
	return &Player{
		Position:    mgl32.Vec3{0, 2.8, 0},
		Velocity:    mgl32.Vec3{0, 0, 0},
		OnGround:    false,
		IsSprinting: false,
		IsSneaking:  false,
		CamYaw:      -90.0,
		CamPitch:    -20.0,
		LastMouseX:  0,
		LastMouseY:  0,
		FirstMouse:  true,
		World:       world,
	}
}

func (p *Player) HandleMouseMovement(w *glfw.Window, xpos, ypos float64) {
	if p.FirstMouse {
		p.LastMouseX = xpos
		p.LastMouseY = ypos
		p.FirstMouse = false
		return
	}

	xoffset := xpos - p.LastMouseX
	yoffset := p.LastMouseY - ypos
	p.LastMouseX = xpos
	p.LastMouseY = ypos

	sensitivity := 0.1
	xoffset *= sensitivity
	yoffset *= sensitivity

	p.CamYaw += xoffset
	p.CamPitch += yoffset

	// Constrain pitch
	if p.CamPitch > 89.0 {
		p.CamPitch = 89.0
	}
	if p.CamPitch < -89.0 {
		p.CamPitch = -89.0
	}
}

func (p *Player) HandleMouseButton(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press && p.HasHoveredBlock {
		if button == glfw.MouseButtonLeft {
			// Break block
			p.World.Set(p.HoveredBlock[0], p.HoveredBlock[1], p.HoveredBlock[2], world.BlockTypeAir)
		}
		if button == glfw.MouseButtonRight {
			// Place block
			front := p.GetFrontVector()
			rayStart := p.GetEyePosition()
			result := physics.Raycast(rayStart, front, physics.MinReachDistance, physics.MaxReachDistance, p.World)
			if result.Hit {
				// Clamp Y placement into [0,255]
				if result.AdjacentPosition[1] >= 0 && result.AdjacentPosition[1] <= 255 {
					ax, ay, az := result.AdjacentPosition[0], result.AdjacentPosition[1], result.AdjacentPosition[2]
					// Only place if empty and does not intersect player
					if p.World.IsAir(ax, ay, az) && !physics.IntersectsBlock(p.Position, PlayerHeight, ax, ay, az) {
						p.World.Set(ax, ay, az, world.BlockTypeGrass)
					}
				}
			}
		}
	}
}

func (p *Player) Update(dt float64, window *glfw.Window) {
	// Update hovered block
	p.UpdateHoveredBlock()

	// Process movement
	p.UpdatePosition(dt, window)
}

func (p *Player) UpdateHoveredBlock() {
	front := p.GetFrontVector()
	rayStart := p.GetEyePosition()
	result := physics.Raycast(rayStart, front, physics.MinReachDistance, physics.MaxReachDistance, p.World)

	p.HasHoveredBlock = result.Hit
	if result.Hit {
		p.HoveredBlock = result.HitPosition
	}
}

func (p *Player) UpdatePosition(dt float64, window *glfw.Window) {
	// Handle sprint and sneak
	if window.GetKey(glfw.KeyLeftControl) == glfw.Press {
		p.IsSprinting = true
		p.IsSneaking = false
	} else if window.GetKey(glfw.KeyLeftShift) == glfw.Press {
		p.IsSneaking = true
		p.IsSprinting = false
	} else {
		p.IsSprinting = false
		p.IsSneaking = false
	}

	// Get input direction
	forward := float32(0)
	strafe := float32(0)

	if window.GetKey(glfw.KeyW) == glfw.Press {
		forward += 1
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		forward -= 1
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		strafe -= 1
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		strafe += 1
	}

	// Normalize diagonal movement
	if forward != 0 && strafe != 0 {
		forward *= 0.98
		strafe *= 0.98
	}

	// Calculate movement based on camera direction
	yaw := mgl32.DegToRad(float32(p.CamYaw))

	// Calculate forward/backward movement vector
	frontX := float32(math.Cos(float64(yaw)))
	frontZ := float32(math.Sin(float64(yaw)))

	// Calculate strafe movement vector (perpendicular to front)
	strafeX := float32(math.Cos(float64(yaw + math.Pi/2)))
	strafeZ := float32(math.Sin(float64(yaw + math.Pi/2)))

	moveSpeed := float32(BaseMoveSpeed)

	if p.OnGround {
		if p.IsSprinting {
			moveSpeed *= SprintMultiplier
		} else if p.IsSneaking {
			moveSpeed *= SneakMultiplier
		}

		// Calculate target velocity
		targetVelX := (frontX*forward + strafeX*strafe) * moveSpeed
		targetVelZ := (frontZ*forward + strafeZ*strafe) * moveSpeed

		// Smooth acceleration
		blendFactor := float32(0.2)

		p.Velocity[0] = p.Velocity[0]*(1-blendFactor) + targetVelX*blendFactor
		p.Velocity[2] = p.Velocity[2]*(1-blendFactor) + targetVelZ*blendFactor

		// Apply friction when no input
		if forward == 0 && strafe == 0 {
			frictionFactor := float32(math.Pow(0.6, dt*20))
			p.Velocity[0] *= frictionFactor
			p.Velocity[2] *= frictionFactor

			// Stop completely if very slow
			if math.Abs(float64(p.Velocity[0])) < 0.005 {
				p.Velocity[0] = 0
			}
			if math.Abs(float64(p.Velocity[2])) < 0.005 {
				p.Velocity[2] = 0
			}
		}
	} else {
		// In air - only apply small air control, don't change existing velocity much
		if forward != 0 || strafe != 0 {
			// Air control acceleration (much smaller than ground)
			airControl := float32(0.8)
			airAccelX := (frontX*forward + strafeX*strafe) * airControl
			airAccelZ := (frontZ*forward + strafeZ*strafe) * airControl

			p.Velocity[0] += airAccelX * float32(dt)
			p.Velocity[2] += airAccelZ * float32(dt)
		}

		// Very light air drag
		airDragFactor := float32(math.Pow(0.98, dt*20))
		p.Velocity[0] *= airDragFactor
		p.Velocity[2] *= airDragFactor
	}

	// Jump
	if window.GetKey(glfw.KeySpace) == glfw.Press && p.OnGround {
		p.Velocity[1] = JumpVelocity
		p.OnGround = false

		// Sprint jump boost
		if p.IsSprinting && (forward != 0 || strafe != 0) {
			jumpBoostX := (frontX*forward + strafeX*strafe) * 2.0
			jumpBoostZ := (frontZ*forward + strafeZ*strafe) * 2.0
			p.Velocity[0] += jumpBoostX
			p.Velocity[2] += jumpBoostZ
		}
	}

	// Apply gravity
	if !p.OnGround {
		p.Velocity[1] -= Gravity * float32(dt)
		if p.Velocity[1] < TerminalVelocity {
			p.Velocity[1] = TerminalVelocity
		}
	}

	// Calculate new position
	newPos := p.Position.Add(p.Velocity.Mul(float32(dt)))

	// Resolve Y first to avoid stepping up vertical walls
	testPosY := mgl32.Vec3{p.Position[0], newPos[1], p.Position[2]}
	if !physics.Collides(testPosY, PlayerHeight, p.World) {
		p.Position[1] = newPos[1]
		if p.Velocity[1] > 0 {
			// Clamp to ceiling if head intersects
			ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, PlayerHeight, p.World)
			if p.Position[1]+PlayerHeight > ceiling {
				p.Position[1] = ceiling - PlayerHeight
				p.Velocity[1] = 0
				p.OnGround = false
			}
		} else {
			p.OnGround = false
		}
	} else {
		if p.Velocity[1] <= 0 {
			// Landing on ground
			p.OnGround = true
			groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, p.World)
			p.Position[1] = groundLevel
		} else {
			// Hitting ceiling
			ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, PlayerHeight, p.World)
			p.Position[1] = ceiling - PlayerHeight
			p.OnGround = false
		}
		p.Velocity[1] = 0
	}

	// Then resolve X at updated Y
	testPosX := mgl32.Vec3{newPos[0], p.Position[1], p.Position[2]}
	if !physics.Collides(testPosX, PlayerHeight, p.World) {
		p.Position[0] = newPos[0]
	} else {
		p.Velocity[0] = 0
	}

	// Finally resolve Z at updated Y
	testPosZ := mgl32.Vec3{p.Position[0], p.Position[1], newPos[2]}
	if !physics.Collides(testPosZ, PlayerHeight, p.World) {
		p.Position[2] = newPos[2]
	} else {
		p.Velocity[2] = 0
	}

	// Final ground settle to avoid micro fall/jitter when hugging walls
	groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, p.World)
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

	// Double check if on ground
	if p.OnGround {
		checkPos := mgl32.Vec3{p.Position[0], p.Position[1] - 0.01, p.Position[2]}
		if !physics.Collides(checkPos, PlayerHeight, p.World) {
			p.OnGround = false
		}
	}
}

func (p *Player) GetFrontVector() mgl32.Vec3 {
	y := mgl32.DegToRad(float32(p.CamYaw))
	pt := mgl32.DegToRad(float32(p.CamPitch))
	fx := float32(math.Cos(float64(y)) * math.Cos(float64(pt)))
	fy := float32(math.Sin(float64(pt)))
	fz := float32(math.Sin(float64(y)) * math.Cos(float64(pt)))
	return mgl32.Vec3{fx, fy, fz}.Normalize()
}

func (p *Player) GetEyePosition() mgl32.Vec3 {
	eyeOffset := PlayerEyeHeight
	if p.IsSneaking {
		eyeOffset -= 0.08
	}
	return p.Position.Add(mgl32.Vec3{0, float32(eyeOffset), 0})
}

func (p *Player) GetViewMatrix() mgl32.Mat4 {
	eyePos := p.GetEyePosition()
	front := p.GetFrontVector()
	target := eyePos.Add(front)
	return mgl32.LookAtV(eyePos, target, mgl32.Vec3{0, 1, 0})
}

var WireframeMode bool = false

// ToggleWireframeMode, wireframe modunu açıp kapatır
func (p *Player) ToggleWireframeMode() {
	WireframeMode = !WireframeMode
	fmt.Printf("Wireframe mode: %v\n", WireframeMode)
}
