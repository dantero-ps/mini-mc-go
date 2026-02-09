package player

import (
	"math/rand"
	"mini-mc/internal/entity"
	"mini-mc/internal/item"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

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
			// Determine drops
			dropType := blockType
			dropCount := 1

			def, ok := registry.Blocks[blockType]
			if ok {
				dropType = def.GetItemDropped()
				dropCount = def.QuantityDropped()
			}

			if dropCount > 0 {
				// Create item entity in the world
				// Start slightly above the bottom of the block, with random horizontal offset
				offsetX := (rand.Float64() * 0.7) + 0.15
				offsetY := 0.8
				offsetZ := (rand.Float64() * 0.7) + 0.15

				pos := mgl32.Vec3{float32(x) + float32(offsetX), float32(y) + float32(offsetY), float32(z) + float32(offsetZ)}
				itemEnt := entity.NewItemEntity(p.World, pos, item.NewItemStack(dropType, dropCount))
				p.World.AddEntity(itemEnt)
			}
		}

		// Reset mining
		p.ResetMining()
	}
}
