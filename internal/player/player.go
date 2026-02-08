package player

import (
	"mini-mc/internal/input"
	"mini-mc/internal/profiling"
)

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

	// Process movement (handles flight timer as well)
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

	// Update render arm sway
	p.UpdateRenderArm(dt)

	// Update inventory item animations (for pickup pop effect)
	if p.Inventory != nil {
		p.Inventory.UpdateAnimations()
	}
}
