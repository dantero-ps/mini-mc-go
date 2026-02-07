package hud

import (
	"fmt"

	"mini-mc/internal/player"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func (h *HUD) renderInventory(p *player.Player) {
	if p.Inventory == nil {
		return
	}

	// Screen dimensions
	screenWidth := h.width
	screenHeight := h.height

	scale := float32(2.0)
	invW := 176 * scale
	invH := 166 * scale

	x := (screenWidth - invW) / 2
	y := (screenHeight - invH) / 2

	// Draw Background
	// UVs: 0,0 to 176/256, 166/256
	u1 := float32(176) / 256.0
	v1 := float32(166) / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}

	// Use inventoryTexture
	h.uiRenderer.DrawTexturedRect(x, y, invW, invH, h.inventoryTexture, 0, 0, u1, v1, color, 1.0)

	// Flush background UI first so items/text draw on top
	h.uiRenderer.Flush()

	// Render "Crafting" text (on top of background)
	craftingX := x + 86*scale
	craftingY := y + 16*scale
	h.fontRenderer.Render("Crafting", craftingX, craftingY, 0.35, mgl32.Vec3{0.3, 0.3, 0.3})

	// Render Player Entity
	// Position in MC: x + 51, y + 75. Scale 30.
	// Our HUD is scaled by 2.0.
	// 51*2 = 102. 75*2 = 150. Scale 30*2 = 60?
	// Let's rely on standard MC values scaled by UI scale.
	playerX := x + 51*scale
	playerY := y + 75*scale
	playerScale := 30.0 * scale

	// Mouse relative to player center
	// Note: RenderInventoryPlayer function handles calculation of relative mouse pos if we pass raw mouse pos?
	// The function I wrote accepts raw mouseX/Y and computes relative.

	h.playerModel.RenderInventoryPlayer(p, playerX, playerY, playerScale, float32(p.MouseX), float32(p.MouseY), h.width, h.height, glfw.GetTime())

	// Draw Items
	itemSize := 16 * scale

	// Reset hover slot
	h.HoveredSlot = -1

	// Helper to check hover and draw overlay
	drawHoverOverlay := func(slotIdx int, slotX, slotY float32) {
		mx := float32(p.MouseX)
		my := float32(p.MouseY)
		if mx >= slotX && mx < slotX+itemSize && my >= slotY && my < slotY+itemSize {
			// Draw semi-transparent white overlay
			// 0x80FFFFFF in ARGB is 1,1,1,0.5
			h.uiRenderer.DrawFilledRect(slotX, slotY, itemSize, itemSize, mgl32.Vec3{1, 1, 1}, 0.5)
			h.HoveredSlot = slotIdx
		}
	}

	// 1. Main Inventory (Indices 9-35)
	// Java: x=8, y=84
	for i := 9; i < 36; i++ {
		stack := p.Inventory.MainInventory[i]
		// Grid: 9 cols, 3 rows
		// i-9 to normalize to 0-26
		normIdx := i - 9
		col := normIdx % 9
		row := normIdx / 9

		slotX := x + float32(8+col*18)*scale
		slotY := y + float32(84+row*18)*scale

		if stack != nil {
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
		// Draw hover overlay after item (so it's on top)
		drawHoverOverlay(i, slotX, slotY)
	}

	// 2. Hotbar (Indices 0-8)
	// Java: x=8, y=142
	for i := range 9 {
		stack := p.Inventory.MainInventory[i]
		slotX := x + float32(8+i*18)*scale
		slotY := y + float32(142)*scale

		if stack != nil {
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
		drawHoverOverlay(i, slotX, slotY)
	}

	// 3. Armor (Indices 0-3 in ArmorInventory)
	// Java: x=8, y=8 start, vertical step 18
	for i := range 4 {
		stack := p.Inventory.ArmorInventory[i]
		if stack != nil {
			slotX := x + float32(8)*scale
			slotY := y + float32(8+i*18)*scale

			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)
		}
	}

	// Flush hover overlays (and any other filled UI drawn during inventory) above items.
	h.uiRenderer.Flush()

	// Render Cursor Stack
	if p.Inventory.CursorStack != nil {
		mx := float32(p.MouseX)
		my := float32(p.MouseY)
		// Center item on cursor
		h.itemRenderer.RenderGUI(p.Inventory.CursorStack, mx-itemSize/2, my-itemSize/2, itemSize)

		if p.Inventory.CursorStack.Count > 1 {
			countText := fmt.Sprintf("%d", p.Inventory.CursorStack.Count)
			tx := mx + itemSize/4
			ty := my + itemSize/4
			h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
		}
	}
}
