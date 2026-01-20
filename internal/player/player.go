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
	PlayerEyeHeight  = 1.62
	PlayerHeight     = 1.8 // Player collision height
	BaseMoveSpeed    = 3.637
	SprintMultiplier = 1.5
	SneakMultiplier  = 0.3
	Gravity          = 32.0
	JumpVelocity     = 9.4
	TerminalVelocity = -78.4 // Maximum fall speed
	FlySpeed         = 10.0  // Flight mode movement speed
)

type GameMode int

const (
	GameModeSurvival GameMode = iota
	GameModeCreative
)

type Player struct {
	GameMode         GameMode
	PrevPosition     mgl32.Vec3
	Position         mgl32.Vec3
	Velocity         mgl32.Vec3
	OnGround         bool
	IsSprinting      bool
	IsSneaking       bool
	IsFlying         bool
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
		if im.JustPressed(input.ActionSprint) {
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

	// Flight mode movement
	if p.IsFlying {
		// Vertical movement in flight mode
		verticalSpeed := float32(FlySpeed)
		if !p.IsInventoryOpen {
			if im.IsActive(input.ActionJump) {
				p.Velocity[1] = verticalSpeed
			} else if im.IsActive(input.ActionSneak) {
				p.Velocity[1] = -verticalSpeed
			} else {
				// No vertical input - stop vertical movement
				p.Velocity[1] = 0
			}
		} else {
			p.Velocity[1] = 0
		}

		// Horizontal movement in flight mode (similar to ground movement)
		if forward != 0 || strafe != 0 {
			flightSpeed := float32(FlySpeed)
			if p.IsSprinting {
				flightSpeed *= SprintMultiplier
			}
			targetVelX := (frontX*forward + strafeX*strafe) * flightSpeed
			targetVelZ := (frontZ*forward + strafeZ*strafe) * flightSpeed

			blendFactor := float32(0.3)
			p.Velocity[0] = p.Velocity[0]*(1-blendFactor) + targetVelX*blendFactor
			p.Velocity[2] = p.Velocity[2]*(1-blendFactor) + targetVelZ*blendFactor
		} else {
			// Apply friction when no input
			frictionFactor := float32(math.Pow(0.7, dt*20))
			p.Velocity[0] *= frictionFactor
			p.Velocity[2] *= frictionFactor

			if math.Abs(float64(p.Velocity[0])) < 0.005 {
				p.Velocity[0] = 0
			}
			if math.Abs(float64(p.Velocity[2])) < 0.005 {
				p.Velocity[2] = 0
			}
		}
	} else {
		// Normal ground/air movement
		// Jump
		if !p.IsInventoryOpen && im.IsActive(input.ActionJump) && p.OnGround {
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
	}

	// Calculate new position
	newPos := p.Position.Add(p.Velocity.Mul(float32(dt)))

	if p.IsFlying {
		// In flight mode, check collisions but allow movement through blocks
		// Only prevent movement if we would collide
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
					p.Position[1] = groundLevel
				} else {
					p.OnGround = false
				}
			} else {
				// Hitting ceiling
				ceiling := physics.FindCeilingLevel(p.Position[0], p.Position[2], p.Position, PlayerHeight, p.World)
				p.Position[1] = ceiling - PlayerHeight
				p.OnGround = false
			}
			p.Velocity[1] = 0
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

	positionChange := p.Position.Sub(p.PrevPosition)
	distanceMoved := math.Sqrt(float64(positionChange.X()*positionChange.X() + positionChange.Z()*positionChange.Z()))
	p.DistanceWalkedModified = p.DistanceWalkedModified + distanceMoved*0.6
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
