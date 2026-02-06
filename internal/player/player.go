package player

import (
	"math"
	"math/rand"

	"mini-mc/internal/config"
	"mini-mc/internal/entity"
	"mini-mc/internal/input"
	"mini-mc/internal/inventory"
	"mini-mc/internal/item"
	"mini-mc/internal/physics"
	"mini-mc/internal/profiling"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	PlayerEyeHeight = 1.62
	PlayerHeight    = 1.8

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
	// ... keeping this structure, just editing the constants above and the logic below ...
	// Wait, replace_file_content replaces a chunk. I need two chunks or one big one.
	// Let's rely on line numbers from previous view.
	// Constants are at lines 28-39 (approx). Logic is around 758.
	// I will split this into 2 chunks using multi_replace_file_content if I could,
	// but I am restricted to replace_file_content for single contiguous block?
	// No, I can use multi_replace.

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
		Health:               20.0,
		MaxHealth:            20.0,
		FoodLevel:            20.0,
		MaxFoodLevel:         20.0,
		FallDistance:         0,
		JumpStartY:           0,
		MaxJumpHeight:        0,
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
			// Left click logic moved to Update for continuous breaking
		}
		if button == glfw.MouseButtonRight {
			// Place block
			front := p.GetFrontVector()
			rayStart := p.GetEyePosition()
			result := physics.Raycast(rayStart, front, physics.MinReachDistance, physics.MaxReachDistance, p.World)
			if result.Hit {
				// Get selected item from inventory
				selectedStack := p.Inventory.GetCurrentItem()
				if selectedStack != nil && selectedStack.Count > 0 && selectedStack.Type != world.BlockTypeAir {
					// Clamp Y placement into [0,255]
					if result.AdjacentPosition[1] >= 0 && result.AdjacentPosition[1] <= 255 {
						ax, ay, az := result.AdjacentPosition[0], result.AdjacentPosition[1], result.AdjacentPosition[2]
						// Allow placement if empty and either not intersecting player
						// or the block's top is at/below the player's feet (pillar-up case)
						targetTop := float32(ay)
						placingUnderFeet := targetTop <= p.Position[1]+0.001
						if p.World.IsAir(ax, ay, az) && (placingUnderFeet || !physics.IntersectsBlock(p.Position, PlayerHeight, ax, ay, az)) {
							// Place the selected block type
							p.World.Set(ax, ay, az, selectedStack.Type)
							p.TriggerHandSwing()
							// Consume item if not in creative mode
							if p.GameMode != GameModeCreative {
								selectedStack.Count--
								if selectedStack.Count <= 0 {
									p.Inventory.MainInventory[p.Inventory.CurrentItem] = nil
								}
							}
						}
					}
				}
			}
		}
	}
}

func (p *Player) HandleScroll(w *glfw.Window, xoff, yoff float64) {
	// Scroll to change inventory slot
	// yoff > 0 is up, yoff < 0 is down
	if yoff > 0 {
		p.Inventory.ChangeCurrentItem(1)
	} else if yoff < 0 {
		p.Inventory.ChangeCurrentItem(-1)
	}
}

func (p *Player) HandleNumKey(slot int) {
	if slot >= 0 && slot < 9 {
		p.Inventory.SetCurrentItem(slot)
	}
}

func (p *Player) CheckEntityCollisions(dt float64) {
	// Minecraft-style pickup: only when player collides with item
	// No magnet effect - items don't move towards player
	p.World.EntitiesMu.RLock()
	defer p.World.EntitiesMu.RUnlock()

	// Player collision box: 0.6 width, 1.8 height
	// Item collision box: 0.25x0.25x0.25
	// Check if item is within pickup range (based on collision boxes)
	// Using larger collision box for easier pickup
	playerPos := p.Position
	playerHalfWidth := float32(1.0) // Half of 2.0 (very large for easier pickup)
	playerHeight := float32(1.8)
	itemHalfSize := float32(0.125) // Half of 0.25

	for _, e := range p.World.Entities {
		if itemEnt, ok := e.(*entity.ItemEntity); ok {
			if itemEnt.IsDead() {
				continue
			}

			// Check pickup delay (must be 0 or less)
			if itemEnt.PickupDelay > 0 {
				continue
			}

			// Owner check: if item has owner, only allow pickup if:
			// - owner is empty (no owner)
			// - age >= 5800 (200 ticks before despawn at 6000)
			// - owner matches player name (we don't have player names, so skip this)
			if itemEnt.Owner != "" {
				// Age in ticks (assuming 20 ticks per second)
				ageInTicks := itemEnt.Age * 20.0
				if ageInTicks < 5800 {
					// Item was recently dropped by owner, can't pick up yet
					continue
				}
			}

			itemPos := itemEnt.Position()

			// Don't pick up items that are below the player's feet
			// Player Y position is at feet level, so if item Y is lower, skip it
			if itemPos.Y() < playerPos.Y()+0.5 {
				continue
			}

			// AABB collision check between player and item
			// Player AABB: (pos.x - 0.3, pos.y, pos.z - 0.3) to (pos.x + 0.3, pos.y + 1.8, pos.z + 0.3)
			// Item AABB: (item.x - 0.125, item.y, item.z - 0.125) to (item.x + 0.125, item.y + 0.25, item.z + 0.125)
			// Extend player Y range upward to pick up items on blocks above
			playerMinX := playerPos.X() - playerHalfWidth
			playerMaxX := playerPos.X() + playerHalfWidth
			playerMinY := playerPos.Y()                      // Don't extend downward - can't pick up items below
			playerMaxY := playerPos.Y() + playerHeight + 1.0 // Extend upward to pick up items on blocks above
			playerMinZ := playerPos.Z() - playerHalfWidth
			playerMaxZ := playerPos.Z() + playerHalfWidth

			itemMinX := itemPos.X() - itemHalfSize
			itemMaxX := itemPos.X() + itemHalfSize
			itemMinY := itemPos.Y()
			itemMaxY := itemPos.Y() + 0.25
			itemMinZ := itemPos.Z() - itemHalfSize
			itemMaxZ := itemPos.Z() + itemHalfSize

			// Check AABB overlap
			if playerMaxX >= itemMinX && playerMinX <= itemMaxX &&
				playerMaxY >= itemMinY && playerMinY <= itemMaxY &&
				playerMaxZ >= itemMinZ && playerMinZ <= itemMaxZ {

				// Try to add item to inventory
				if p.Inventory.AddItem(&itemEnt.Stack) {
					// Item was successfully added (or partially added)
					// If stack is now empty, start pickup animation (visual only)
					if itemEnt.Stack.Count <= 0 && !itemEnt.IsPickingUp {
						// Target position: player's body center (eye level - 0.3)
						targetPos := p.GetEyePosition().Sub(mgl32.Vec3{0, 0.3, 0})
						itemEnt.StartPickupAnimation(targetPos)
					}
					// The animation will remove the entity when complete
					// Note: In Minecraft, onItemPickup is called here with original count
					// We don't have that callback, so we skip it
				}
			}
		}
	}
}

func (p *Player) Update(dt float64, im *input.InputManager) {
	defer profiling.Track("player.Update.total")()
	// Update hovered block
	if !p.IsInventoryOpen {
		p.UpdateHoveredBlock()
	} else {
		p.HasHoveredBlock = false
	}

	// Check collisions with items
	p.CheckEntityCollisions(dt)

	// Update flight mode double-tap timer
	if p.lastSpacePressTime >= 0 {
		p.lastSpacePressTime += dt
		// Reset after 0.5 seconds (longer than double-tap window)
		if p.lastSpacePressTime > 0.5 {
			p.lastSpacePressTime = -1
		}
	}

	// Process movement
	p.UpdatePosition(dt, im)

	// Mining logic
	justPressed := im.JustPressed(input.ActionMouseLeft)
	isHeld := im.IsActive(input.ActionMouseLeft)

	if !p.IsInventoryOpen && (justPressed || isHeld) {
		if p.HasHoveredBlock {
			p.UpdateMining(dt, justPressed && !isHeld)
		} else if justPressed {
			p.TriggerHandSwing()
		}
	} else {
		p.ResetMining()
	}

	// Update break cooldown
	if p.breakCooldown > 0 {
		p.breakCooldown -= dt
	}

	// Updates head bobbing animation based on player movement
	p.UpdateHeadBob()

	// Update camera bobbing (for view bobbing)
	p.UpdateCameraBob()

	// Update equipped item animation
	p.updateEquippedItem(float32(dt))

	// Update hand swing timer/progress
	if p.handSwingTimer > 0 {
		p.handSwingTimer -= dt
		if p.handSwingTimer < 0 {
			p.handSwingTimer = 0
		}
		if p.handSwingDuration > 0 {
			p.HandSwingProgress = float32(1.0 - p.handSwingTimer/p.handSwingDuration)
		} else {
			p.HandSwingProgress = 0
		}
	} else {
		p.HandSwingProgress = 0
	}
}

func (p *Player) ResetMining() {
	p.IsBreaking = false
	p.BreakProgress = 0
}

func (p *Player) UpdateMining(dt float64, justPressed bool) {
	if !p.HasHoveredBlock {
		p.ResetMining()
		return
	}

	// Creative mode instant break with cooldown logic
	if p.GameMode == GameModeCreative {
		// Single click: no cooldown, held: cooldown applies
		if justPressed || p.breakCooldown <= 0 {
			p.BreakingBlock = p.HoveredBlock
			p.IsBreaking = true
			p.TriggerHandSwing()
			p.BreakBlock()
			// Set cooldown only if held (not just pressed)
			if !justPressed {
				p.breakCooldown = 0.15
			}
		}
		return
	}

	// Check if targeting same block
	if p.IsBreaking {
		if p.BreakingBlock != p.HoveredBlock {
			// Target changed, reset progress but continue mining new block
			p.BreakProgress = 0
			p.BreakingBlock = p.HoveredBlock
		}
	} else {
		// Start mining
		p.IsBreaking = true
		p.BreakingBlock = p.HoveredBlock
		p.BreakProgress = 0
	}

	// Continuous hand swing
	if p.handSwingTimer <= 0 {
		p.TriggerHandSwing()
	}

	// Calculate hardness and progress
	blockType := p.World.Get(p.BreakingBlock[0], p.BreakingBlock[1], p.BreakingBlock[2])
	if blockType == world.BlockTypeAir {
		p.ResetMining()
		return
	}

	def, ok := registry.Blocks[blockType]
	hardness := float32(1.0) // Default
	if ok {
		hardness = def.Hardness
	}

	if hardness < 0 {
		// Unbreakable
		p.BreakProgress = 0
		return
	}

	// Break speed formula (simplified)
	breakSpeed := float32(1.0)
	if p.IsFlying {
		breakSpeed *= 5.0 // Flying breaks faster (if enabled)
	}

	// Increment progress
	p.BreakProgress += float32(dt) * breakSpeed / hardness

	if p.BreakProgress >= 1.0 {
		p.BreakBlock()
	}
}

func (p *Player) BreakBlock() {
	x, y, z := p.BreakingBlock[0], p.BreakingBlock[1], p.BreakingBlock[2]
	blockType := p.World.Get(x, y, z)

	if blockType != world.BlockTypeAir {
		p.World.Set(x, y, z, world.BlockTypeAir)

		if p.GameMode != GameModeCreative {
			// Create item entity in the world
			// Start slightly above the bottom of the block, with random horizontal offset
			offsetX := (rand.Float64() * 0.7) + 0.15
			offsetY := 0.8
			offsetZ := (rand.Float64() * 0.7) + 0.15

			pos := mgl32.Vec3{float32(x) + float32(offsetX), float32(y) + float32(offsetY), float32(z) + float32(offsetZ)}
			itemEnt := entity.NewItemEntity(p.World, pos, item.NewItemStack(blockType, 1))
			p.World.AddEntity(itemEnt)
		}

		// Reset mining
		p.ResetMining()
	}
}

// DropCursorItem drops the item currently held by the cursor
func (p *Player) DropCursorItem() {
	if p.Inventory.CursorStack == nil {
		return
	}

	stack := p.Inventory.CursorStack
	p.Inventory.CursorStack = nil

	p.spawnItemEntity(*stack)
}

// DropHeldItem drops the item currently in the player's hand.
// If dropStack is true, it drops the entire stack, otherwise just one item.
func (p *Player) DropHeldItem(dropStack bool) {
	stack := p.Inventory.GetCurrentItem()
	if stack == nil {
		return
	}

	var dropped item.ItemStack
	if dropStack || stack.Count <= 1 {
		// Drop everything
		dropped = *stack
		p.Inventory.MainInventory[p.Inventory.CurrentItem] = nil
	} else {
		// Drop only one
		dropped = item.NewItemStack(stack.Type, 1)
		stack.Count--
	}

	p.spawnItemEntity(dropped)
	// Trigger a hand swing for visual feedback
	p.TriggerHandSwing()
}

func (p *Player) spawnItemEntity(stack item.ItemStack) {
	// Start slightly in front and at eye level
	front := p.GetFrontVector()
	pos := p.GetEyePosition().Add(mgl32.Vec3{0, 0.65, 0}).Add(front.Mul(0.3))

	// Add some velocity in look direction
	velocity := front.Mul(5.0)
	velocity[1] += 1.0 // Slight upward toss

	itemEnt := entity.NewItemEntity(p.World, pos, stack)
	itemEnt.Vel = velocity
	p.World.AddEntity(itemEnt)
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

func (p *Player) UpdatePosition(dt float64, im *input.InputManager) {
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

	// DEBUG: Trace inputs
	// fmt.Printf("Input: Fwd: %f, Strafe: %f, OnGround: %v\n", forward, strafe, p.OnGround)

	// Calculate movement based on camera direction
	yaw := float32(p.CamYaw)
	yawRad := float64(mgl32.DegToRad(yaw))

	// Calculate forward/backward movement vector
	frontX := float32(math.Cos(yawRad))
	frontZ := float32(math.Sin(yawRad))

	// Calculate strafe movement vector (Right vector)
	strafeX := float32(math.Cos(yawRad + math.Pi/2))
	strafeZ := float32(math.Sin(yawRad + math.Pi/2))

	// Apply input to acceleration
	// Using Minecraft's friction/acceleration logic but applied to our vectors

	// Scale factor for acceleration:
	// MC Accel is "velocity added per tick" in "blocks/tick" units.
	// To convert to "blocks/sec" added per second (m/s^2):
	// 1. Accel (m/tick) -> Accel * 20 (m/s impact per tick)
	// 2. Per tick (0.05s) -> (Accel * 20) / 0.05 = Accel * 400.
	//
	// Currently we multiply by modeDistance = dt * 20.
	// Velocity += Accel * dt * 20. (This results in Accel * 20 m/s^2)
	// We need 400. So we need another x20 factor for acceleration only.

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
		// CORRECTION: Minecraft applies drag once per tick (discrete). We apply it continuously.
		// Detailed math shows that discrete decay (V *= D) loses more momentum than continuous decay (V' = -k V)
		// specifically for the "freshly added" acceleration in that tick.
		// To match terminal velocity exactly: V_mc = A*D/(1-D) ? NO.
		// Detailed analysis shows MC movement speed is the PEAK velocity (V_move = V_base + A).
		// Discrete Equilibrium: V_base = (V_base + A) * D  => V_base = A*D/(1-D).
		// PEAK Velocity (Movement Speed) = V_base + A = A*D/(1-D) + A = A / (1-D).
		// Continuous Equilibrium: V_cont = A_cont / -ln(D).
		// We want V_cont = PEAK Velocity.
		// A_cont / -ln(D) = A / (1-D).
		// A_cont = A * (-ln(D)) / (1-D).

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
				jumpBoost := float32(0.125 * 20.0) // Reduced to 2.5 m/s to match feel (Logic: 4.0 was too strong vs 0.91 air drag timing)

				// Use the calculated front vector to ensure boost matches look direction
				p.Velocity[0] += frontX * jumpBoost
				p.Velocity[2] += frontZ * jumpBoost
			}
		}

		// Gravity and Drag moved to end of tick to match MC behavior
	}

	// Stop completely if very slow (epsilon check)
	if math.Abs(float64(p.Velocity[0])) < 0.005 {
		p.Velocity[0] = 0
	}
	if math.Abs(float64(p.Velocity[2])) < 0.005 {
		p.Velocity[2] = 0
	}

	// Calculate new position using CURRENT velocity
	// This was previously inside the else block but needs to be available for collision logic
	newPos := p.Position.Add(p.Velocity.Mul(float32(dt)))

	if p.IsFlying {
		testPosY := mgl32.Vec3{p.Position[0], newPos[1], p.Position[2]}
		if !physics.Collides(testPosY, PlayerHeight, p.World) {
			p.Position[1] = newPos[1]
		} else {
			p.Velocity[1] = 0
		}
	} else {
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
				// Landing on ground (only if ground exists)
				groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, p.World)
				if !float32IsInfNeg(groundLevel) {
					p.OnGround = true
					p.Velocity[1] = 0 // Reset vertical velocity on landing
					p.Position[1] = groundLevel
				} else {
					p.OnGround = false
				}
			} else {
				// Hitting ceiling
				ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, PlayerHeight, p.World)
				p.Position[1] = ceiling - PlayerHeight
				p.OnGround = false
				p.Velocity[1] = 0
			}
		}
	}

	// Then resolve X at updated Y
	testPosX := mgl32.Vec3{newPos[0], p.Position[1], p.Position[2]}
	if !physics.Collides(testPosX, PlayerHeight, p.World) {
		if p.IsSneaking && p.Velocity[1] == 0 && physics.FindGroundLevel(newPos[0], p.Position[2], p.Position, p.World) < p.Position[1]-0.1 {
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
	if !physics.Collides(testPosZ, PlayerHeight, p.World) {
		if p.IsSneaking && p.Velocity[1] == 0 && physics.FindGroundLevel(p.Position[0], newPos[2], p.Position, p.World) < p.Position[1]-0.1 {
			p.Velocity[2] = 0
		} else {
			p.Position[2] = newPos[2]
		}
	} else {
		p.Velocity[2] = 0
		p.IsSprinting = false
	}

	// Final ground settle to avoid micro fall/jitter when hugging walls (only when not flying)
	if !p.IsFlying {
		groundLevel := physics.FindGroundLevel(p.Position[0], p.Position[2], p.Position, p.World)
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
			if !physics.Collides(checkPos, PlayerHeight, p.World) {
				p.OnGround = false
			}
		}
	} else {
		// In flight mode, we're never on ground
		p.OnGround = false
	}

	// Apply flight drag at the end of the tick (matches MC behavior: Input -> Move -> Drag)
	if p.IsFlying {
		// Horizontal drag is 0.91 in flight mode (same as air drag)
		groundDrag := float32(0.91)
		groundDragFactor := float32(math.Pow(float64(groundDrag), float64(modeDistance)))
		p.Velocity[0] *= groundDragFactor
		p.Velocity[2] *= groundDragFactor

		// Vertical drag is 0.6 in flight mode
		verticalDrag := float32(0.6)
		verticalDragFactor := float32(math.Pow(float64(verticalDrag), float64(modeDistance)))
		p.Velocity[1] *= verticalDragFactor
	} else {
		// Apply gravity (matches MC: after movement)
		p.Velocity[1] -= Gravity * float32(dt)
		if p.Velocity[1] < TerminalVelocity {
			p.Velocity[1] = TerminalVelocity
		}

		// Recalculate friction for drag
		currentFriction := float32(GroundDrag)
		if p.OnGround { // OnGround updated by physics
			currentFriction = BlockFriction * GroundDrag
		}

		// Apply Drag (matches MC: after movement)
		dragFactor := float32(math.Pow(float64(currentFriction), float64(modeDistance)))
		p.Velocity[0] *= dragFactor
		p.Velocity[2] *= dragFactor

		// Vertical drag (air resistance)
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
	// if potionEffect != nil { jumpBoostReduction = float32(amplifier + 1) }

	mcDistance := float64(distance)*0.82 + 0.2
	damage := int(math.Ceil((mcDistance - 3.0 - float64(jumpBoostReduction)) * float64(damageMultiplier)))

	if damage > 0 {
		p.ApplyDamage(float32(damage))
		// TODO: Play fall sound
		// "game.neutral.hurt.fall.big" if damage > 4 else "game.neutral.hurt.fall.small"
	}
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

func (p *Player) UpdateHeadBob() {
	p.PrevHeadBobYaw = p.HeadBobYaw
	p.PrevHeadBobPitch = p.HeadBobPitch

	f := math.Sqrt(float64(p.Velocity.X()*p.Velocity.X() + p.Velocity.Z()*p.Velocity.Z()))
	f1 := math.Atan(float64(-p.Velocity.Y()*0.20000000298023224)) * 15.0

	if f > 0.1 {
		f = 0.1
	}

	// Smooth transition: when in air, gradually reduce instead of instantly setting to 0
	// Lower blend factors for smoother transitions
	if !p.OnGround {
		// Gradually reduce when in air (instead of instantly to 0)
		blendFactor := 0.1 // Lower blend factor for smoother transition
		p.HeadBobYaw += (0.0 - p.HeadBobYaw) * blendFactor
	} else {
		// Normal animation when on ground
		p.HeadBobYaw += (f - p.HeadBobYaw) * 0.1
	}

	if p.OnGround {
		f1 = 0.0
		// Smoothly reset pitch when on ground
		p.HeadBobPitch += (f1 - p.HeadBobPitch) * 0.3
	} else {
		// Smoothly apply pitch animation when in air
		p.HeadBobPitch += (f1 - p.HeadBobPitch) * 0.3
	}
}

// UpdateCameraBob updates cameraYaw and cameraPitch for view bobbing animation
// Based on Minecraft's EntityPlayer.onLivingUpdate()
func (p *Player) UpdateCameraBob() {
	p.PrevCameraYaw = p.CameraYaw
	p.PrevCameraPitch = p.CameraPitch

	// If view bobbing is disabled, smoothly reset to zero
	if !config.GetViewBobbing() {
		p.CameraYaw += (0.0 - p.CameraYaw) * 0.1
		p.CameraPitch += (0.0 - p.CameraPitch) * 0.1
		return
	}

	// Calculate horizontal speed magnitude
	f := float32(math.Sqrt(float64(p.Velocity.X()*p.Velocity.X() + p.Velocity.Z()*p.Velocity.Z())))
	// Calculate vertical speed angle (pitch bob) - reduced multiplier for less exaggerated effect
	f1 := float32(math.Atan(float64(-p.Velocity.Y()*0.20000000298023224)) * 1.5)

	// Clamp horizontal speed to 0.1
	if f > 0.1 {
		f = 0.1
	}

	// Reset to 0 if not on ground or health <= 0 (we don't have health, so just check onGround)
	if !p.OnGround {
		f = 0.0
	}

	// Reset pitch bob if on ground
	if p.OnGround {
		f1 = 0.0
	}

	// Smooth interpolation towards target values (Minecraft uses 0.4F and 0.8F)
	p.CameraYaw += (f - p.CameraYaw) * 0.03
	p.CameraPitch += (f1 - p.CameraPitch) * 0.1
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
	return p.GetViewMatrixWithPartialTicks(0.0)
}

func (p *Player) GetViewMatrixWithPartialTicks(partialTicks float32) mgl32.Mat4 {
	eyePos := p.GetEyePosition()
	front := p.GetFrontVector()
	target := eyePos.Add(front)

	// Base view matrix
	viewMatrix := mgl32.LookAtV(eyePos, target, mgl32.Vec3{0, 1, 0})

	// Apply view bobbing only if enabled
	if !config.GetViewBobbing() {
		return viewMatrix
	}

	// Calculate bobbing values
	f := float32(p.DistanceWalkedModified - p.PrevDistanceWalkedModified)
	f1 := float32(p.DistanceWalkedModified + float64(f)*float64(partialTicks))

	// Interpolate camera yaw and pitch
	f2 := p.PrevCameraYaw + (p.CameraYaw-p.PrevCameraYaw)*partialTicks
	f3 := p.PrevCameraPitch + (p.CameraPitch-p.PrevCameraPitch)*partialTicks

	// Apply view bobbing transformations (based on Minecraft's setupViewBobbing)
	// Calculate bobbing values
	translateX := float32(math.Sin(float64(f1*math.Pi))) * f2 * 0.5
	translateY := -float32(math.Abs(math.Cos(float64(f1*math.Pi))) * float64(f2))

	// Rotate around Z axis (roll)
	rotateZ := float32(math.Sin(float64(f1*math.Pi))) * f2 * 3.0

	// Rotate around X axis (pitch bob)
	rotateX := float32(math.Abs(math.Cos(float64(f1*math.Pi-0.2))*float64(f2))) * 5.0

	// In Minecraft, transformations are applied in this order:
	// 1. translate
	// 2. rotate Z (roll)
	// 3. rotate X (pitch bob)
	// 4. rotate X (camera pitch)
	// In OpenGL/mgl32, matrix multiplication is right-to-left, so we build:
	// translate * rotateZ * rotateX * cameraPitch
	translateMat := mgl32.Translate3D(translateX, translateY, 0.0)
	rotateZMat := mgl32.HomogRotate3D(mgl32.DegToRad(rotateZ), mgl32.Vec3{0, 0, 1})
	rotateXMat := mgl32.HomogRotate3D(mgl32.DegToRad(rotateX), mgl32.Vec3{1, 0, 0})
	cameraPitchMat := mgl32.HomogRotate3D(mgl32.DegToRad(f3), mgl32.Vec3{1, 0, 0})

	// Combine: translate * rotateZ * rotateX * cameraPitch
	bobbingMat := translateMat.Mul4(rotateZMat).Mul4(rotateXMat).Mul4(cameraPitchMat)

	// Apply bobbing to view matrix: bobbingMat * viewMatrix
	return bobbingMat.Mul4(viewMatrix)
}

// float32IsInfNeg reports whether v is negative infinity.
func float32IsInfNeg(v float32) bool {
	return math.IsInf(float64(v), -1)
}

// TriggerHandSwing starts a new right-hand swing animation.
func (p *Player) TriggerHandSwing() {
	p.handSwingTimer = p.handSwingDuration
}

// GetHandSwingProgress returns current right-hand swing progress in [0,1].
func (p *Player) GetHandSwingProgress() float32 {
	return p.HandSwingProgress
}

// GetHandEquipProgress returns current equip animation progress in [0,1].
// Reserved for future item switching.
func (p *Player) GetHandEquipProgress() float32 {
	return p.EquipProgress
}

func (p *Player) updateEquippedItem(dt float32) {
	itemstack := p.Inventory.GetCurrentItem()
	flag := false

	if p.EquippedItem != nil && itemstack != nil {
		if p.EquippedItem.Type != itemstack.Type {
			flag = true
		}
	} else if p.EquippedItem == nil && itemstack == nil {
		flag = false
	} else {
		flag = true
	}

	// Minecraft-like equip speed
	// f is the change amount per tick (0.4 in Java)
	// Our dt is in seconds, so we multiply by 20 to get ticks approx
	f := float32(0.4 * 20.0 * dt)
	f1 := float32(0.0)
	if !flag {
		f1 = 1.0
	}

	f2 := f1 - p.EquipProgress
	if f2 < -f {
		f2 = -f
	}
	if f2 > f {
		f2 = f
	}

	p.EquipProgress += f2

	if p.EquipProgress < 0.1 {
		p.EquippedItem = itemstack
	}
}
