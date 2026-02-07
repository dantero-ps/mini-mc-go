package player

import (
	"mini-mc/internal/inventory"
	"mini-mc/internal/item"
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	PlayerEyeHeight = 1.62
	PlayerHeight    = 1.8
)

type GameMode int

const (
	GameModeSurvival GameMode = iota
	GameModeCreative
)

type Player struct {
	GameMode     GameMode
	PrevPosition mgl32.Vec3
	Position     mgl32.Vec3
	Velocity     mgl32.Vec3
	OnGround     bool
	IsSprinting  bool
	IsSneaking   bool
	IsFlying     bool

	PrevHeadBobYaw   float64
	HeadBobYaw       float64
	PrevHeadBobPitch float64
	HeadBobPitch     float64
	CamYaw           float64
	CamPitch         float64
	LastMouseX       float64
	LastMouseY       float64
	MouseX           float64 // Current absolute mouse X
	MouseY           float64 // Current absolute mouse Y
	FirstMouse       bool

	DistanceWalkedModified     float64
	PrevDistanceWalkedModified float64

	// View bobbing animation
	PrevCameraYaw   float32
	CameraYaw       float32
	PrevCameraPitch float32
	CameraPitch     float32

	// Render arm sway (sway when turning head)
	PrevRenderArmYaw   float32
	RenderArmYaw       float32
	PrevRenderArmPitch float32
	RenderArmPitch     float32

	// Interaction
	HoveredBlock    [3]int
	HasHoveredBlock bool

	// Mining state
	IsBreaking    bool
	BreakingBlock [3]int
	BreakProgress float32

	World *world.World

	// Inventory
	Inventory       *inventory.Inventory
	IsInventoryOpen bool

	// Hand animation state
	handSwingTimer    float64
	handSwingDuration float64
	HandSwingProgress float32
	EquipProgress     float32
	EquippedItem      *item.ItemStack

	// Block breaking cooldown
	breakCooldown float64

	// Flight mode double-tap detection
	lastSpacePressTime float64
	lastSpaceState     bool

	// Forward double-tap detection for sprint
	lastForwardPressTime float64

	Health       float32
	MaxHealth    float32
	FoodLevel    float32
	MaxFoodLevel float32
	FallDistance float32

	// Jump diagnostics
	JumpStartY    float32
	MaxJumpHeight float32
}

func New(world *world.World, mode GameMode) *Player {
	return &Player{
		GameMode:             mode,
		Position:             mgl32.Vec3{0, 2.8, 0},
		Velocity:             mgl32.Vec3{0, 0, 0},
		OnGround:             false,
		IsSprinting:          false,
		IsSneaking:           false,
		IsFlying:             false,
		CamYaw:               0.0,
		CamPitch:             0.0,
		LastMouseX:           0,
		LastMouseY:           0,
		FirstMouse:           true,
		World:                world,
		Inventory:            inventory.New(),
		handSwingTimer:       0,
		handSwingDuration:    0.25,
		HandSwingProgress:    0,
		EquipProgress:        0,
		EquippedItem:         nil,
		lastSpacePressTime:   -1,
		lastSpaceState:       false,
		lastForwardPressTime: -1,
		PrevCameraYaw:        0,
		CameraYaw:            0,
		PrevCameraPitch:      0,
		CameraPitch:          0,
		PrevRenderArmYaw:     0,
		RenderArmYaw:         0,
		PrevRenderArmPitch:   0,
		RenderArmPitch:       0,
		Health:               20.0,
		MaxHealth:            20.0,
		FoodLevel:            20.0,
		MaxFoodLevel:         20.0,
		FallDistance:         0,
		JumpStartY:           0,
		MaxJumpHeight:        0,
	}
}

func (p *Player) GetEyePosition() mgl32.Vec3 {
	eyeOffset := PlayerEyeHeight
	if p.IsSneaking {
		eyeOffset -= 0.08
	}
	return p.Position.Add(mgl32.Vec3{0, float32(eyeOffset), 0})
}

func (p *Player) ApplyDamage(amount float32) {
	if p.GameMode == GameModeCreative {
		return
	}

	p.Health -= amount
	if p.Health < 0 {
		p.Health = 0
		// TODO: Handle death (respawn, etc)
	}
}
