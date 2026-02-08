package inventory

import (
	"mini-mc/internal/item"
)

// MouseButton represents a mouse button click
type MouseButton int

const (
	MouseButtonLeft   MouseButton = 0
	MouseButtonRight  MouseButton = 1
	MouseButtonMiddle MouseButton = 2
)

// Container manages a collection of slots and handles item interactions.
// It represents the logical side of an inventory interface (e.g. Player Inventory, Chest, Furnace).
type Container struct {
	Slots       []*Slot
	CursorStack *item.ItemStack
}

// NewContainer create a new container
func NewContainer() *Container {
	return &Container{
		Slots: make([]*Slot, 0),
	}
}

func (c *Container) AddSlot(s *Slot) *Slot {
	c.Slots = append(c.Slots, s)
	return s
}

func (c *Container) GetSlot(index int) *Slot {
	if index >= 0 && index < len(c.Slots) {
		return c.Slots[index]
	}
	return nil
}

// SlotClick handles interactions with a slot
// It returns true if something happened
func (c *Container) SlotClick(slotIndex int, button MouseButton, isDoubleClick bool, playerInventory *Inventory) bool {
	if slotIndex < 0 || slotIndex >= len(c.Slots) {
		return false
	}

	slot := c.GetSlot(slotIndex)
	cursor := playerInventory.CursorStack
	itemInSlot := slot.GetStack()

	// Handle double-click: collect all items of same type
	if isDoubleClick {
		handleClickDoubleClick(c, slotIndex, playerInventory)
		return true
	}

	// Right click logic
	if button == MouseButtonRight {
		if cursor != nil {
			// Place one item from cursor stack into slot
			if itemInSlot == nil {
				// Empty slot: place one item
				newStack := item.NewItemStack(cursor.Type, 1)
				slot.PutStack(&newStack)
				cursor.Count--
				if cursor.Count <= 0 {
					playerInventory.CursorStack = nil
				}
			} else if itemInSlot.IsItemEqual(*cursor) {
				// Same item: merge one item if there's space
				if itemInSlot.Count < slot.GetMaxStackSize() {
					itemInSlot.Count++
					cursor.Count--
					if cursor.Count <= 0 {
						playerInventory.CursorStack = nil
					}
				}
			}
			return true
		} else if itemInSlot != nil {
			// Pick up half stack (Minecraft behavior)
			// But our simple logic: pick up half?
			// Let's implement standard pick up half
			half := (itemInSlot.Count + 1) / 2
			newCursor := item.NewItemStack(itemInSlot.Type, half)
			playerInventory.CursorStack = &newCursor

			itemInSlot.Count -= half
			if itemInSlot.Count == 0 {
				slot.PutStack(nil)
			}
			return true
		}
	} else if button == MouseButtonLeft {
		if cursor == nil {
			if itemInSlot != nil {
				// Pick up entire stack
				playerInventory.CursorStack = itemInSlot
				slot.PutStack(nil)
			}
		} else {
			if itemInSlot == nil {
				// Place entire stack into empty slot
				slot.PutStack(cursor)
				playerInventory.CursorStack = nil
			} else if itemInSlot.IsItemEqual(*cursor) {
				// Same item type: merge stacks
				space := slot.GetMaxStackSize() - itemInSlot.Count
				if space > 0 {
					toAdd := min(cursor.Count, space)
					itemInSlot.Count += toAdd
					cursor.Count -= toAdd
					if cursor.Count <= 0 {
						playerInventory.CursorStack = nil
					}
				}
			} else {
				// Different items: swap
				slot.PutStack(cursor)
				playerInventory.CursorStack = itemInSlot
			}
		}
		return true
	}

	return false
}

func handleClickDoubleClick(c *Container, clickedSlotIndex int, playerInventory *Inventory) {
	cursor := playerInventory.CursorStack

	// If cursor has an item, collect all matching items from inventory
	if cursor != nil {
		totalCount := cursor.Count

		// Collect matching items from other slots to cursor
		for i, slot := range c.Slots {
			if i == clickedSlotIndex {
				continue // Skip the source slot (already handled by previous click or current cursor)
			}

			itemInSlot := slot.GetStack()
			if itemInSlot != nil && itemInSlot.IsItemEqual(*cursor) {
				// Calculate space remaining in cursor stack
				space := cursor.GetMaxStackSize() - totalCount
				if space <= 0 {
					break
				}

				toTake := min(itemInSlot.Count, space)
				totalCount += toTake
				itemInSlot.Count -= toTake
				if itemInSlot.Count == 0 {
					slot.PutStack(nil)
				}
			}
		}

		// Update cursor with total
		cursor.Count = totalCount
	}
}
