package hud

import (
	"fmt"
	"mini-mc/internal/graphics"
	"mini-mc/internal/registry"

	"mini-mc/internal/player"

	"github.com/go-gl/mathgl/mgl32"
)

func (h *HUD) renderHotbar(p *player.Player) {
	if p.Inventory == nil {
		return
	}

	screenWidth := h.width
	screenHeight := h.height

	scale := float32(2.0)
	hbW := 182 * scale
	hbH := 22 * scale

	x := (screenWidth - hbW) / 2
	y := screenHeight - hbH

	// Draw Hotbar Background
	// UVs: 0,0 to 182/256, 22/256
	u1 := float32(182) / 256.0
	v1 := float32(22) / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	texWidgets, err := graphics.GetTexture("assets/textures/gui/widgets.png")
	if err != nil {
		// Log error or fallback?
		fmt.Printf("Error loading widgets texture: %v\n", err)
	}

	h.uiRenderer.DrawTexturedRect(x, y, hbW, hbH, texWidgets, 0, 0, u1, v1, color, 1.0)

	// Draw Selected Slot
	// Selector is 24x24 pixels in texture at 0,22
	selW := 24 * scale
	selH := 24 * scale

	// Slot index (0-8)
	slotIdx := p.Inventory.CurrentItem

	// Offset for selector logic: -1px in texture space relative to slot start
	slotXTex := 3 + 20*slotIdx
	selXTex := slotXTex - 2

	selXScreen := x + float32(selXTex)*scale - float32(1)*scale // Fine tune alignment
	selYScreen := y - float32(1)*scale                          // -1px up

	// Selector UVs
	selU0 := float32(0.0)
	selV0 := float32(22) / 256.0
	selU1 := float32(24) / 256.0
	selV1 := float32(22+24) / 256.0

	h.uiRenderer.DrawTexturedRect(selXScreen, selYScreen, selW, selH, texWidgets, selU0, selV0, selU1, selV1, color, 1.0)

	// IMPORTANT: UI quads are FIFO-batched; flush backgrounds before rendering items/text so they don't draw over items.
	h.uiRenderer.Flush()

	// Draw Items
	for i := range 9 {
		stack := p.Inventory.MainInventory[i]
		if stack != nil {
			// Calculate slot position
			slotX := x + float32(3+20*i)*scale
			slotY := y + float32(3)*scale

			// Render Item
			itemSize := 16 * scale
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			// Render Count if > 1
			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				// Bottom right of slot
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
	}

	// Draw item name text above hotbar if selected
	selItem := p.Inventory.GetCurrentItem()
	if selItem != nil {
		name := "Unknown"
		if def, ok := registry.Blocks[selItem.Type]; ok {
			name = def.Name
		}
		// Center text
		w, _ := h.fontRenderer.Measure(name, 0.4)
		tx := (screenWidth - w) / 2
		ty := y - 60
		h.fontRenderer.Render(name, tx, ty, 0.4, mgl32.Vec3{1, 1, 1})
	}
}
