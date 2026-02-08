package item

import "mini-mc/internal/world"

// ItemStack represents a stack of items
type ItemStack struct {
	Type  world.BlockType
	Count int

	// AnimationsToGo is the number of animation frames remaining.
	// Set to 5 when item is picked up, decremented each tick.
	AnimationsToGo int
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

// CanStackWith returns true if this stack can be merged with another.
// Checks same item type. In the future, could also check NBT data.
func (s ItemStack) CanStackWith(other ItemStack) bool {
	// Must be same item type
	if s.Type != other.Type {
		return false
	}
	// Future: Add NBT/metadata comparison here if needed
	return true
}

// Animation constants
const (
	// PickupAnimationTicks is how many ticks the pickup animation lasts
	// Higher value = slower animation (12 ticks ≈ 200ms at 60fps)
	PickupAnimationTicks = 20
)

// TriggerAnimation starts the pickup animation for this item stack
func (s *ItemStack) TriggerAnimation() {
	s.AnimationsToGo = PickupAnimationTicks
}

// UpdateAnimation decrements the animation counter if active.
// Should be called once per tick.
func (s *ItemStack) UpdateAnimation() {
	if s.AnimationsToGo > 0 {
		s.AnimationsToGo--
	}
}

// GetAnimationScale returns the scale factors for the pickup animation.
// Returns scaleX, scaleY. If no animation is active, returns 1.0, 1.0.
// The animation squeezes horizontally and stretches vertically, then returns to normal.
func (s *ItemStack) GetAnimationScale() (scaleX, scaleY float32) {
	if s.AnimationsToGo <= 0 {
		return 1.0, 1.0
	}

	// f1 = 1.0 + animationsToGo / 5.0
	// At tick 5: f1 = 2.0, scaleX = 0.5, scaleY = 1.5
	// At tick 1: f1 = 1.2, scaleX = 0.83, scaleY = 1.1
	f := float32(s.AnimationsToGo)
	f1 := 1.0 + f/float32(PickupAnimationTicks)

	scaleX = 1.0 / f1
	scaleY = (f1 + 1.0) / 2.0

	return scaleX, scaleY
}
