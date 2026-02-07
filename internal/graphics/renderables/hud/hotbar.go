package hud

import (
	"fmt"

	"mini-mc/internal/player"
	"mini-mc/internal/registry"

	"github.com/go-gl/mathgl/mgl32"
)

func (h *HUD) renderHotbar(p *player.Player) {
	if p.Inventory == nil {
		return
	}

	// Screen dimensions
	screenWidth := h.width
	screenHeight := h.height

	// Hotbar dimensions (widgets.png)
	// Original: 182x22 pixels. Scaled x2 for visibility -> 364x44
	scale := float32(2.0)
	hbW := 182 * scale
	hbH := 22 * scale

	x := (screenWidth - hbW) / 2
	y := screenHeight - hbH - 10 // 10px padding from bottom

	// Draw Hotbar Background
	// UVs: 0,0 to 182/256, 22/256
	u1 := float32(182) / 256.0
	v1 := float32(22) / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.uiRenderer.DrawTexturedRect(x, y, hbW, hbH, h.widgetsTexture, 0, 0, u1, v1, color, 1.0)

	// Draw Selected Slot
	// Selector is 24x24 pixels in texture at 0,22
	selW := 24 * scale
	selH := 24 * scale

	// Slot index (0-8)
	slotIdx := p.Inventory.CurrentItem
	// Each slot is 20px wide in texture grid (approx)
	// Actually:
	// Slot 0 starts at x=3 (relative to 182).
	// Spacing is 20px.
	// Slot x pos = 3 + 20*i
	// But the selector is centered on the slot.
	// Selector texture is at 0, 22. Size 24x24.
	// Selector pos relative to hotbar: (slotX - 1, -1)
	// Let's calculate screen pos directly.

	// Offset for selector logic: -1px in texture space relative to slot start
	slotXTex := 3 + 20*slotIdx
	selXTex := slotXTex - 2 // -2 pixels adjustment to align 24px box over 20px slot?
	// Actually Minecraft logic:
	// Hotbar: 182 wide.
	// Slot 0: x=6 (in 182 width coords? No, let's look at texture)
	// Texture check:
	// First slot seems to be at x=3 (border is 3px).
	// Slot inner is 16x16.
	// Selector is 24x24.
	// Selector draws OVER the hotbar.

	selXScreen := x + float32(selXTex)*scale - float32(1)*scale // Fine tune alignment
	selYScreen := y - float32(1)*scale                          // -1px up

	// Selector UVs
	selU0 := float32(0.0)
	selV0 := float32(22) / 256.0
	selU1 := float32(24) / 256.0
	selV1 := float32(22+24) / 256.0

	h.uiRenderer.DrawTexturedRect(selXScreen, selYScreen, selW, selH, h.widgetsTexture, selU0, selV0, selU1, selV1, color, 1.0)

	// IMPORTANT: UI quads are FIFO-batched; flush backgrounds before rendering items/text so they don't draw over items.
	h.uiRenderer.Flush()

	// Draw Items
	for i := range 9 {
		stack := p.Inventory.MainInventory[i]
		if stack != nil {
			// Calculate slot position
			// Slot is 16x16 items
			// x = 3 + 20*i + 3 (padding to center 16 in 20? No)
			// Let's approximate: 3 + 20*i is left edge of slot area.
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
		ty := y - 30
		h.fontRenderer.Render(name, tx, ty, 0.4, mgl32.Vec3{1, 1, 1})
	}
}
