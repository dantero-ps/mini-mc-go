package player

import (
	"mini-mc/internal/entity"
	"mini-mc/internal/item"
	"mini-mc/internal/physics"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func (p *Player) HandleMouseButton(button glfw.MouseButton, action glfw.Action) {
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
						width, height := p.GetBounds()
						if p.World.IsAir(ax, ay, az) && (placingUnderFeet || !physics.IntersectsBlock(p.Position, width, height, ax, ay, az)) {
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

func (p *Player) HandleScroll(yoff float64) {
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
	defer profiling.Track("player.Update.collisionChecks.total")()
	// Minecraft-style pickup: only when player collides with item
	// No magnet effect - items don't move towards player
	entities := p.World.GetEntities()

	// Player collision box: 0.6 width, 1.8 height
	// Item collision box: 0.25x0.25x0.25
	// Check if item is within pickup range (based on collision boxes)
	// Using larger collision box for easier pickup
	playerPos := p.Position
	playerHalfWidth := float32(1.0) // Half of 2.0 (very large for easier pickup)
	playerHeight := float32(1.8)
	itemHalfSize := float32(0.125) // Half of 0.25

	for _, e := range entities {
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
			// Allow tolerance (1.5) for items below feet (e.g. in holes or lower blocks)
			if itemPos.Y() < playerPos.Y()-1.5 {
				continue
			}

			// AABB collision check between player and item
			// Player AABB: (pos.x - 0.3, pos.y, pos.z - 0.3) to (pos.x + 0.3, pos.y + 1.8, pos.z + 0.3)
			// Item AABB: (item.x - 0.125, item.y, item.z - 0.125) to (item.x + 0.125, item.y + 0.25, item.z + 0.125)
			// Extend player Y range upward to pick up items on blocks above
			playerMinX := playerPos.X() - playerHalfWidth
			playerMaxX := playerPos.X() + playerHalfWidth
			playerMinY := playerPos.Y() - 1.0                // Extend downward to pick up items below
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
						targetPos := p.GetEyePosition().Sub(mgl32.Vec3{0, 0, 0})
						itemEnt.StartPickupAnimation(targetPos)
					}
				}
			}
		}
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
	pos := p.GetEyePosition().Add(mgl32.Vec3{0, -0.35, 0}).Add(front.Mul(0.3))

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
