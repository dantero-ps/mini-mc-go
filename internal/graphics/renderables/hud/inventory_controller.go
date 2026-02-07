package hud

import (
	"time"

	"mini-mc/internal/inventory"
	"mini-mc/internal/item"
	"mini-mc/internal/player"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// HandleInventoryClick handles mouse clicks in the inventory
func (h *HUD) HandleInventoryClick(p *player.Player, x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	if action != glfw.Press {
		return false
	}

	// Coordinates match renderInventory
	scale := float32(2.0)
	invW := 176 * scale
	invH := 166 * scale

	// Screen dimensions
	screenW := h.width
	screenH := h.height

	startX := (screenW - invW) / 2
	startY := (screenH - invH) / 2

	mouseX := float32(x)
	mouseY := float32(y)

	checkSlot := func(idx int, slotX, slotY float32) {
		// slotX, slotY are relative to startX, startY
		sx := startX + slotX
		sy := startY + slotY
		size := 16 * scale

		if mouseX >= sx && mouseX < sx+size && mouseY >= sy && mouseY < sy+size {
			h.handleSlotClick(p, idx, button)
		}
	}

	// Main Inventory (9-35)
	for i := 9; i < 36; i++ {
		col := (i - 9) % 9
		row := (i - 9) / 9

		// Java coords: x=8 + col*18, y=84 + row*18
		checkSlot(i, float32(8+col*18)*scale, float32(84+row*18)*scale)
	}

	// Hotbar (0-8)
	for i := range 9 {
		// Java coords: x=8 + i*18, y=142
		checkSlot(i, float32(8+i*18)*scale, float32(142)*scale)
	}

	return true
}

func (h *HUD) handleSlotClick(p *player.Player, slotIdx int, button glfw.MouseButton) {
	cursor := p.Inventory.CursorStack
	slot := p.Inventory.MainInventory[slotIdx]

	// Double-click detection (within 300ms on the same slot)
	isDoubleClick := false
	if button == glfw.MouseButtonLeft && slotIdx == h.lastClickSlot && time.Since(h.lastClickTime) < 300*time.Millisecond {
		isDoubleClick = true
	}
	h.lastClickSlot = slotIdx
	h.lastClickTime = time.Now()

	// Handle double-click: collect all items of same type
	if isDoubleClick {
		// If cursor has an item, collect all matching items from inventory
		if cursor != nil {
			totalCount := cursor.Count

			// If clicked slot has same item, add it
			if slot != nil && slot.IsItemEqual(*cursor) {
				totalCount += slot.Count
				p.Inventory.MainInventory[slotIdx] = nil
			}

			// Loop through all slots and collect matching items
			for i := range len(p.Inventory.MainInventory) {
				if i == slotIdx {
					continue // Already handled above
				}

				otherSlot := p.Inventory.MainInventory[i]
				if otherSlot != nil && otherSlot.IsItemEqual(*cursor) {
					totalCount += otherSlot.Count
					p.Inventory.MainInventory[i] = nil // Clear the slot
				}
			}

			// Update cursor with total
			cursor.Count = totalCount
			return
		} else if slot != nil {
			// Cursor empty, slot has item: collect all matching items
			totalCount := slot.Count

			// Loop through all slots and collect matching items
			for i := range len(p.Inventory.MainInventory) {
				if i == slotIdx {
					continue // Skip the slot we're clicking
				}

				otherSlot := p.Inventory.MainInventory[i]
				if otherSlot != nil && otherSlot.IsItemEqual(*slot) {
					totalCount += otherSlot.Count
					p.Inventory.MainInventory[i] = nil // Clear the slot
				}
			}

			// Put all collected items in cursor
			if totalCount > 0 {
				cursorStack := item.NewItemStack(slot.Type, totalCount)
				p.Inventory.CursorStack = &cursorStack
				p.Inventory.MainInventory[slotIdx] = nil
			}
			return
		}
	}

	if button == glfw.MouseButtonRight && cursor != nil {
		// Right click: place one item from cursor stack into slot
		if slot == nil {
			// Empty slot: place one item
			newStack := item.NewItemStack(cursor.Type, 1)
			p.Inventory.MainInventory[slotIdx] = &newStack
			cursor.Count--
			if cursor.Count <= 0 {
				p.Inventory.CursorStack = nil
			}
		} else if slot.IsItemEqual(*cursor) {
			// Same item: merge one item if there's space
			if slot.Count < slot.GetMaxStackSize() {
				slot.Count++
				cursor.Count--
				if cursor.Count <= 0 {
					p.Inventory.CursorStack = nil
				}
			}
		}
		return
	}

	// Left click (normal behavior)
	if cursor == nil {
		if slot != nil {
			// Pick up
			p.Inventory.CursorStack = slot
			p.Inventory.MainInventory[slotIdx] = nil
		}
	} else {
		if slot == nil {
			// Place into empty slot
			p.Inventory.MainInventory[slotIdx] = cursor
			p.Inventory.CursorStack = nil
		} else if slot.IsItemEqual(*cursor) {
			// Same item type: merge stacks
			space := slot.GetMaxStackSize() - slot.Count
			if space > 0 {
				toAdd := min(cursor.Count, space)
				slot.Count += toAdd
				cursor.Count -= toAdd
				if cursor.Count <= 0 {
					p.Inventory.CursorStack = nil
				}
			}
		} else {
			// Different items: swap
			p.Inventory.MainInventory[slotIdx] = cursor
			p.Inventory.CursorStack = slot
		}
	}
}

// MoveHoveredItemToHotbar moves the hovered item to the specified hotbar slot
func (h *HUD) MoveHoveredItemToHotbar(p *player.Player, hotbarSlot int) {
	if h.HoveredSlot < 0 || h.HoveredSlot >= inventory.MainInventorySize {
		return
	}

	hoveredStack := p.Inventory.MainInventory[h.HoveredSlot]
	if hoveredStack == nil {
		return
	}

	// If hotbar slot has something, move hovered to cursor instead
	hotbarStack := p.Inventory.MainInventory[hotbarSlot]
	if hotbarStack != nil {
		p.Inventory.CursorStack = hoveredStack
		p.Inventory.MainInventory[h.HoveredSlot] = hotbarStack
		p.Inventory.MainInventory[hotbarSlot] = nil
	} else {
		// Hotbar slot is empty: move hovered item there
		p.Inventory.MainInventory[hotbarSlot] = hoveredStack
		p.Inventory.MainInventory[h.HoveredSlot] = nil
	}
}
