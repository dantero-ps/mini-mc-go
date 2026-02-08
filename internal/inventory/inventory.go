package inventory

import (
	"mini-mc/internal/item"
)

// GetItem returns the item stack at the given global index
// 0-35: Main Inventory (including hotbar)
// 36-39: Armor Inventory
func (inv *Inventory) GetItem(index int) *item.ItemStack {
	if index >= 0 && index < MainInventorySize {
		return inv.MainInventory[index]
	}
	if index >= MainInventorySize && index < MainInventorySize+ArmorInventorySize {
		return inv.ArmorInventory[index-MainInventorySize]
	}
	return nil
}

// SetItem sets the item stack at the given global index
func (inv *Inventory) SetItem(index int, stack *item.ItemStack) {
	if index >= 0 && index < MainInventorySize {
		inv.MainInventory[index] = stack
	} else if index >= MainInventorySize && index < MainInventorySize+ArmorInventorySize {
		inv.ArmorInventory[index-MainInventorySize] = stack
	}
}

const (
	MainInventorySize  = 36
	ArmorInventorySize = 4
	HotbarSize         = 9
)

type Inventory struct {
	// Main inventory includes hotbar (indices 0-8) and main storage (9-35)
	MainInventory  [MainInventorySize]*item.ItemStack
	ArmorInventory [ArmorInventorySize]*item.ItemStack
	CurrentItem    int             // Index 0-8
	CursorStack    *item.ItemStack // Item held by mouse cursor
}

func New() *Inventory {
	return &Inventory{
		CurrentItem: 0,
	}
}

// GetCurrentItem returns the currently selected item in the hotbar
func (inv *Inventory) GetCurrentItem() *item.ItemStack {
	if inv.CurrentItem >= 0 && inv.CurrentItem < HotbarSize {
		return inv.MainInventory[inv.CurrentItem]
	}
	return nil
}

// AddItem attempts to add an item stack to the inventory.
// Returns true if successful (fully added), false if failed (inventory full).
// Updates the passed stack's count if partially added.
func (inv *Inventory) AddItem(stack *item.ItemStack) bool {
	if stack == nil || stack.Count == 0 {
		return false
	}

	// 1. Try to merge with existing stacks
	// Loop through main inventory to find matching items
	if stack.IsStackable() {
		for i := range len(inv.MainInventory) {
			existing := inv.MainInventory[i]
			if existing != nil && existing.IsItemEqual(*stack) {
				j := existing.Count

				// Calculate how much we can add
				maxStack := existing.GetMaxStackSize()
				if j < maxStack {
					space := maxStack - j
					toAdd := min(stack.Count, space)

					existing.Count += toAdd
					stack.Count -= toAdd

					if stack.Count == 0 {
						return true
					}
				}
			}
		}
	}

	// 2. Place in empty slots
	// Loop until stack is empty or no more room
	for stack.Count > 0 {
		emptySlot := inv.GetFirstEmptyStack()
		if emptySlot >= 0 {
			// Fill empty slot
			maxStack := stack.GetMaxStackSize()
			toAdd := min(stack.Count, maxStack)

			// Create new stack in slot
			newItem := item.NewItemStack(stack.Type, toAdd)
			inv.MainInventory[emptySlot] = &newItem

			stack.Count -= toAdd
		} else {
			// No more room
			return false
		}
	}

	return true
}

// GetFirstEmptyStack returns the index of the first empty slot in main inventory
func (inv *Inventory) GetFirstEmptyStack() int {
	for i := range len(inv.MainInventory) {
		if inv.MainInventory[i] == nil {
			return i
		}
	}
	return -1
}

// SetCurrentItem sets the selected hotbar slot directly (0-8)
func (inv *Inventory) SetCurrentItem(index int) {
	if index >= 0 && index < HotbarSize {
		inv.CurrentItem = index
	}
}

// ChangeCurrentItem changes the selected hotbar slot
func (inv *Inventory) ChangeCurrentItem(direction int) {
	if direction > 0 {
		direction = 1
	} else if direction < 0 {
		direction = -1
	}

	inv.CurrentItem -= direction
	for inv.CurrentItem < 0 {
		inv.CurrentItem += HotbarSize
	}
	for inv.CurrentItem >= HotbarSize {
		inv.CurrentItem -= HotbarSize
	}
}

// HasItem checks if the inventory contains a specific item type
func (inv *Inventory) HasItem(t item.ItemStack) bool {
	for _, slot := range inv.MainInventory {
		if slot != nil && slot.IsItemEqual(t) {
			return true
		}
	}
	return false
}
