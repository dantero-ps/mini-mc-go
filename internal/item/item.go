package item

import "mini-mc/internal/world"

// ItemStack represents a stack of items
type ItemStack struct {
	Type  world.BlockType
	Count int
}

// NewItemStack creates a new item stack
func NewItemStack(t world.BlockType, count int) ItemStack {
	return ItemStack{
		Type:  t,
		Count: count,
	}
}

// GetMaxStackSize returns the maximum stack size for this item
func (s ItemStack) GetMaxStackSize() int {
	return 64
}

// IsStackable returns if the item can be stacked
func (s ItemStack) IsStackable() bool {
	return true
}

// IsItemEqual checks if two stacks contain the same item type
func (s ItemStack) IsItemEqual(other ItemStack) bool {
	return s.Type == other.Type
}
