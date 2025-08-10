package player

import (
	"fmt"
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"mini-mc/internal/physics"
	"mini-mc/internal/world"
)

const (
	PlayerHeight    = 2.5
	PlayerEyeHeight = 1.5
	PlayerSpeed     = 5.0
	PlayerJumpSpeed = 6.0
	Gravity         = 15.0
)

type Player struct {
	Position   mgl32.Vec3
	VelocityY  float32
	OnGround   bool
	CamYaw     float64
	CamPitch   float64
	LastMouseX float64
	LastMouseY float64
	FirstMouse bool

	// Interaction
	HoveredBlock    [3]int
	HasHoveredBlock bool

	// Reference to world
	World *world.World
}

func New(world *world.World) *Player {
	return &Player{
		Position:   mgl32.Vec3{0, 2.8, 0},
		VelocityY:  0,
		OnGround:   false,
		CamYaw:     -90.0,
		CamPitch:   -20.0,
		LastMouseX: 0,
		LastMouseY: 0,
		FirstMouse: true,
		World:      world,
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
			// Place block at the adjacent position
			front := p.GetFrontVector()
			rayStart := p.GetEyePosition()
			result := physics.Raycast(rayStart, front, physics.MinReachDistance, physics.MaxReachDistance, p.World)
			if result.Hit {
				p.World.Set(result.AdjacentPosition[0], result.AdjacentPosition[1], result.AdjacentPosition[2], world.BlockTypeGrass)
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
	speed := float32(PlayerSpeed * dt)
	front := p.GetFrontVector()

	// Project front vector to horizontal plane for consistent movement speed
	frontHorizontal := mgl32.Vec3{front.X(), 0, front.Z()}.Normalize()
	right := frontHorizontal.Cross(mgl32.Vec3{0, 1, 0}).Normalize()

	// Calculate movement direction based on input
	moveDir := mgl32.Vec3{0, 0, 0}
	if window.GetKey(glfw.KeyW) == glfw.Press {
		moveDir = moveDir.Add(frontHorizontal)
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		moveDir = moveDir.Sub(frontHorizontal)
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		moveDir = moveDir.Sub(right)
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		moveDir = moveDir.Add(right)
	}

	// Normalize and scale movement
	if moveDir.Len() > 0 {
		moveDir = moveDir.Normalize().Mul(speed)
	}

	// Handle jumping
	if p.OnGround && window.GetKey(glfw.KeySpace) == glfw.Press {
		p.VelocityY = PlayerJumpSpeed
		p.OnGround = false
	}

	// Apply gravity
	p.VelocityY -= Gravity * float32(dt)
	moveY := p.VelocityY * float32(dt)

	// Try horizontal movement first
	newPosH := p.Position.Add(mgl32.Vec3{moveDir.X(), 0, moveDir.Z()})
	if !physics.Collides(newPosH, PlayerHeight, p.World) {
		p.Position = newPosH
	}

	// Try vertical movement
	newPosV := p.Position.Add(mgl32.Vec3{0, moveY, 0})
	if !physics.Collides(newPosV, PlayerHeight, p.World) {
		p.Position = newPosV
		p.OnGround = false
	} else {
		// Collision on vertical movement
		if p.VelocityY < 0 {
			// Falling - find ground level
			groundY := physics.FindGroundLevel(p.Position.X(), p.Position.Z(), p.Position, p.World)
			p.Position = mgl32.Vec3{p.Position.X(), groundY, p.Position.Z()}
			p.OnGround = true
			p.VelocityY = 0
		} else {
			// Hitting ceiling
			p.VelocityY = 0
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
	return p.Position.Add(mgl32.Vec3{0, PlayerEyeHeight, 0})
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
