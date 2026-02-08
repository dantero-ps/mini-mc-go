package inventory

import (
	"mini-mc/internal/item"
)

// Slot represents a single slot in a container
type Slot struct {
	inventory *Inventory
	index     int
	X, Y      int
}

// NewSlot creates a new slot
func NewSlot(inv *Inventory, index, x, y int) *Slot {
	return &Slot{
		inventory: inv,
		index:     index,
		X:         x,
		Y:         y,
	}
}

// GetStack returns the item stack in this slot.
// It delegates to the inventory's GetItem method which handles
// mapping global indices to specific internal arrays (Main/Armor).
func (s *Slot) GetStack() *item.ItemStack {
	if s.inventory == nil {
		return nil
	}
	return s.inventory.GetItem(s.index)
}

// PutStack places an item stack into this slot
func (s *Slot) PutStack(stack *item.ItemStack) {
	if s.inventory == nil {
		return
	}
	s.inventory.SetItem(s.index, stack)
}

// OnSlotChanged can be called when slot content changes
func (s *Slot) OnSlotChanged() {
	// Trigger updates if needed
}

// GetMaxStackSize returns max stack size for this slot
func (s *Slot) GetMaxStackSize() int {
	return 64 // Standard max stack size
}
