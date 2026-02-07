package hud

import (
	"math"

	"mini-mc/internal/player"

	"github.com/go-gl/mathgl/mgl32"
)

func (h *HUD) renderHealth(p *player.Player) {
	screenWidth := h.width
	screenHeight := h.height
	scale := float32(2.0)

	hbH := 22.0 * scale
	yHotbar := screenHeight - hbH - 10.0
	y := yHotbar - 17.0*scale

	// Start X: Same as hotbar left
	hbW := 182.0 * scale
	startBaseX := (screenWidth - hbW) / 2

	// Textures
	texW := float32(256.0)
	texH := float32(256.0)

	heartW := float32(9.0) * scale
	heartH := float32(9.0) * scale

	uEmpty := float32(16.0) / texW
	vEmpty := float32(0.0) / texH
	uFull := float32(52.0) / texW
	uHalf := float32(61.0) / texW

	uWidth := float32(9.0) / texW
	vHeight := float32(9.0) / texH

	color := mgl32.Vec3{1.0, 1.0, 1.0}

	maxHearts := int(math.Ceil(float64(p.MaxHealth) / 2.0))
	currentHealth := int(math.Ceil(float64(p.Health)))

	for i := 0; i < maxHearts; i++ {
		x := startBaseX + float32(i*8)*scale

		// Draw Empty Heart (Background)
		h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uEmpty, vEmpty, uEmpty+uWidth, vEmpty+vHeight, color, 1.0)

		// Draw Full or Half
		if (i*2 + 1) < currentHealth {
			h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uFull, vEmpty, uFull+uWidth, vEmpty+vHeight, color, 1.0)
		} else if (i*2 + 1) == currentHealth {
			h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uHalf, vEmpty, uHalf+uWidth, vEmpty+vHeight, color, 1.0)
		}
	}
}

func (h *HUD) renderFood(p *player.Player) {
	screenWidth := h.width
	screenHeight := h.height
	scale := float32(2.0)

	hbH := 22.0 * scale
	yHotbar := screenHeight - hbH - 10.0
	y := yHotbar - 17.0*scale

	// Center point
	centerX := screenWidth / 2.0
	rightEdge := centerX + 91.0*scale

	// Textures
	texW := float32(256.0)
	texH := float32(256.0)

	iconW := float32(9.0) * scale
	iconH := float32(9.0) * scale

	// Icon UVs for Food (Y=27)
	vBase := float32(27.0) / texH
	uEmpty := float32(16.0) / texW
	uFull := float32(52.0) / texW
	uHalf := float32(61.0) / texW

	uWidth := float32(9.0) / texW
	vHeight := float32(9.0) / texH

	color := mgl32.Vec3{1.0, 1.0, 1.0}

	maxFood := int(math.Ceil(float64(p.MaxFoodLevel) / 2.0))
	currentFood := int(math.Ceil(float64(p.FoodLevel)))

	for i := 0; i < maxFood; i++ {
		// x = rightEdge - i*8*scale - 9*scale
		x := rightEdge - float32(i*8)*scale - 9.0*scale

		// Draw Empty Food (Background)
		h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uEmpty, vBase, uEmpty+uWidth, vBase+vHeight, color, 1.0)

		// Draw Full or Half
		if (i*2 + 1) < currentFood {
			h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uFull, vBase, uFull+uWidth, vBase+vHeight, color, 1.0)
		} else if (i*2 + 1) == currentFood {
			h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uHalf, vBase, uHalf+uWidth, vBase+vHeight, color, 1.0)
		}
	}
}
