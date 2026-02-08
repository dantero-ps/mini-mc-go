package entity

import (
	"math"
	"math/rand"
	"mini-mc/internal/item"

	"github.com/go-gl/mathgl/mgl32"
)

// Stacking constants matching Minecraft 1.8.9
const (
	// Item entity dimensions
	ItemEntityWidth  = 0.25
	ItemEntityHeight = 0.25

	// Stacking search range expansion
	StackSearchExpandX = 0.5
	StackSearchExpandY = 0.0
	StackSearchExpandZ = 0.5

	// Special values
	InfinitePickupDelay = -1.0  // Equivalent to MC's 32767 (never pick up)
	NoDespawnAge        = -6000 // Age that prevents despawn (300-(-6000)=6300 seconds = 105 minutes)

	// Timing
	StackSearchInterval = 25   // ticks (1.25 seconds)
	TickDuration        = 0.05 // seconds per tick (1/20)
)

// NearbyItemsFunc is a callback function to get nearby item entities.
// Takes center position and range, returns slice of interface{} that can be cast to *ItemEntity.
type NearbyItemsFunc func(centerX, centerY, centerZ, rangeX, rangeY, rangeZ float32) []interface{}

type ItemEntity struct {
	Stack       item.ItemStack
	Pos         mgl32.Vec3
	Vel         mgl32.Vec3
	World       WorldSource
	Age         float64
	HoverStart  float64
	RotationYaw float64
	OnGround    bool
	Dead        bool
	PickupDelay float64
	Owner       string // Owner name for pickup delay (empty if no owner)

	// Stacking trigger fields (Minecraft 1.8.9 behavior)
	ticksExisted                       int
	prevBlockX, prevBlockY, prevBlockZ int
	noDespawn                          bool // If true, item never despawns

	// GetNearbyItems is a callback function for stacking queries (set by world after creation)
	GetNearbyItems NearbyItemsFunc

	// Pickup animation (visual only, not physical movement)
	IsPickingUp     bool
	PickupProgress  float64 // 0.0 to 1.0
	PickupStartPos  mgl32.Vec3
	PickupTargetPos mgl32.Vec3
}

func NewItemEntity(w WorldSource, pos mgl32.Vec3, stack item.ItemStack) *ItemEntity {
	// Random velocity based on Minecraft logic
	vx := (rand.Float64() * 0.2) - 0.1
	vz := (rand.Float64() * 0.2) - 0.1
	vy := 0.4

	return &ItemEntity{
		Stack:       stack,
		Pos:         pos,
		Vel:         mgl32.Vec3{float32(vx), float32(vy), float32(vz)},
		World:       w,
		HoverStart:  rand.Float64() * math.Pi * 2.0,
		RotationYaw: rand.Float64() * 360.0,
		PickupDelay: 0.5, // 0.5 second delay (10 ticks = 0.5 second at 20 ticks/s, Minecraft default)
		Owner:       "",  // No owner by default
		// Initialize previous block positions to current
		prevBlockX: int(math.Floor(float64(pos.X()))),
		prevBlockY: int(math.Floor(float64(pos.Y()))),
		prevBlockZ: int(math.Floor(float64(pos.Z()))),
	}
}

func (e *ItemEntity) Update(dt float64) {
	if e.Dead {
		return
	}

	// Update pickup animation
	if e.IsPickingUp {
		e.PickupProgress += dt * 10.0 // Animation duration: 0.1 seconds (2 ticks at 20 ticks/s)
		if e.PickupProgress >= 1.0 {
			e.PickupProgress = 1.0
			e.Dead = true // Item is fully picked up, remove it
			return
		}
		// Don't update physics during pickup animation
		return
	}

	e.Age += dt
	if e.PickupDelay > 0 {
		e.PickupDelay -= dt
	}

	// Minecraft: items despawn after 6000 ticks (300 seconds at 20 ticks/s)
	// Age is in seconds, so 300 seconds = 5 minutes
	// Unless noDespawn is set
	if !e.noDespawn && e.Age >= 300.0 {
		e.Dead = true
		return
	}

	// Apply Gravity
	// Minecraft gravity is 0.04 blocks/tick^2 (20 ticks/s) = 16 blocks/s^2
	// But let's tune it to feel right with our dt
	gravity := float32(18.0)
	e.Vel = e.Vel.Sub(mgl32.Vec3{0, gravity * float32(dt), 0})

	// Drag
	drag := float32(0.98) // Per tick approx
	// Adjust for dt (pow(0.98, dt*20))
	dragFactor := float32(math.Pow(float64(drag), dt*20))

	e.Vel = e.Vel.Mul(dragFactor)

	// Predict next position
	delta := e.Vel.Mul(float32(dt))

	// Collision Handling (Simple Axis separation)
	// Item size is 0.25x0.25x0.25. Half-width 0.125

	// X axis
	if e.checkCollision(e.Pos.X()+delta.X(), e.Pos.Y(), e.Pos.Z()) {
		e.Vel = mgl32.Vec3{0, e.Vel.Y(), e.Vel.Z()}
	} else {
		e.Pos = mgl32.Vec3{e.Pos.X() + delta.X(), e.Pos.Y(), e.Pos.Z()}
	}

	// Y axis
	if e.checkCollision(e.Pos.X(), e.Pos.Y()+delta.Y(), e.Pos.Z()) {
		if e.Vel.Y() < 0 {
			e.OnGround = true
		}
		e.Vel = mgl32.Vec3{e.Vel.X(), 0, e.Vel.Z()}
	} else {
		e.Pos = mgl32.Vec3{e.Pos.X(), e.Pos.Y() + delta.Y(), e.Pos.Z()}
		e.OnGround = false
	}

	// Z axis
	if e.checkCollision(e.Pos.X(), e.Pos.Y(), e.Pos.Z()+delta.Z()) {
		e.Vel = mgl32.Vec3{e.Vel.X(), e.Vel.Y(), 0}
	} else {
		e.Pos = mgl32.Vec3{e.Pos.X(), e.Pos.Y(), e.Pos.Z() + delta.Z()}
	}

	// Ground Friction
	if e.OnGround {
		friction := float32(0.6)
		frictionFactor := float32(math.Pow(float64(friction), dt*20))
		e.Vel = mgl32.Vec3{e.Vel.X() * frictionFactor, e.Vel.Y(), e.Vel.Z() * frictionFactor}
	}

	// Increment tick counter (convert dt to ticks and accumulate)
	e.ticksExisted++

	// Check if position crossed block boundary (Minecraft stacking trigger)
	currentBlockX := int(math.Floor(float64(e.Pos.X())))
	currentBlockY := int(math.Floor(float64(e.Pos.Y())))
	currentBlockZ := int(math.Floor(float64(e.Pos.Z())))

	crossedBlockBoundary := currentBlockX != e.prevBlockX ||
		currentBlockY != e.prevBlockY ||
		currentBlockZ != e.prevBlockZ

	// Update previous position for next frame
	e.prevBlockX = currentBlockX
	e.prevBlockY = currentBlockY
	e.prevBlockZ = currentBlockZ

	// Search for nearby items to merge with (Minecraft 1.8.9 behavior)
	// Trigger when crossing block boundary OR every 25 ticks
	if e.GetNearbyItems != nil && (crossedBlockBoundary || e.ticksExisted%StackSearchInterval == 0) {
		e.searchForOtherItemsNearby()
	}
}

// searchForOtherItemsNearby finds and attempts to combine with nearby items
// Matching Minecraft 1.8.9 EntityItem.searchForOtherItemsNearby()
func (e *ItemEntity) searchForOtherItemsNearby() {
	if e.Dead || e.GetNearbyItems == nil {
		return
	}

	// Calculate search range (expand by 0.5 on X/Z, 0.0 on Y)
	halfWidth := float32(ItemEntityWidth / 2)
	halfHeight := float32(ItemEntityHeight / 2)

	rangeX := halfWidth + StackSearchExpandX
	rangeY := halfHeight // No Y expansion
	rangeZ := halfWidth + StackSearchExpandZ

	nearbyItems := e.GetNearbyItems(
		e.Pos.X(), e.Pos.Y(), e.Pos.Z(),
		rangeX, rangeY, rangeZ,
	)

	for _, other := range nearbyItems {
		if otherItem, ok := other.(*ItemEntity); ok {
			if e.combineItems(otherItem) {
				// This item is now dead, stop searching
				return
			}
		}
	}
}

// combineItems attempts to merge this item with another
// Returns true if merge was successful (this entity was killed)
// Matching Minecraft 1.8.9 EntityItem.combineItems()
func (e *ItemEntity) combineItems(other *ItemEntity) bool {
	// 1. Self-check
	if other == e {
		return false
	}

	// 2. Both must be alive
	if other.Dead || e.Dead {
		return false
	}

	// 3. Don't merge items currently being picked up
	if other.IsPickingUp || e.IsPickingUp {
		return false
	}

	thisStack := e.Stack
	otherStack := other.Stack

	// 4. Neither can have infinite pickup delay (represents items that can't be picked up)
	if e.PickupDelay == InfinitePickupDelay || other.PickupDelay == InfinitePickupDelay {
		return false
	}

	// 5. Neither can have "no despawn" flag (special items in MC have age = -32768)
	if e.noDespawn || other.noDespawn {
		return false
	}

	// 6. Check if stacks can be combined (same item type, metadata, etc.)
	if !thisStack.CanStackWith(otherStack) {
		return false
	}

	// 7. IMPORTANT: Smaller stack merges into larger stack
	// If this stack is larger, reverse the merge direction
	if otherStack.Count < thisStack.Count {
		return other.combineItems(e)
	}

	// 8. Combined size cannot exceed max stack size
	if otherStack.Count+thisStack.Count > thisStack.GetMaxStackSize() {
		return false
	}

	// 9. MERGE: Add this stack to other stack
	other.Stack.Count += thisStack.Count

	// 10. Pickup delay: Use the MAXIMUM of both delays (more restrictive)
	if e.PickupDelay > other.PickupDelay {
		other.PickupDelay = e.PickupDelay
	}

	// 11. Age: Use the MINIMUM age (younger survives longer, despawns later)
	if e.Age < other.Age {
		other.Age = e.Age
	}

	// 12. Kill this entity
	e.SetDead()

	return true
}

func (e *ItemEntity) checkCollision(x, y, z float32) bool {
	// AABB radius
	r := float32(0.125)

	minX := int(math.Floor(float64(x - r)))
	maxX := int(math.Floor(float64(x + r)))
	minY := int(math.Floor(float64(y)))
	maxY := int(math.Floor(float64(y + 0.2)))
	minZ := int(math.Floor(float64(z - r)))
	maxZ := int(math.Floor(float64(z + r)))

	for bx := minX; bx <= maxX; bx++ {
		for by := minY; by <= maxY; by++ {
			for bz := minZ; bz <= maxZ; bz++ {
				if !e.World.IsAir(bx, by, bz) {
					return true
				}
			}
		}
	}
	return false
}

func (e *ItemEntity) Position() mgl32.Vec3 {
	// During pickup animation, return interpolated position
	if e.IsPickingUp {
		// Interpolate from start to target (easing: f^2 for smooth acceleration)
		t := float32(e.PickupProgress)
		t = t * t // Quadratic easing
		return e.PickupStartPos.Add(e.PickupTargetPos.Sub(e.PickupStartPos).Mul(t))
	}
	return e.Pos
}

// StartPickupAnimation starts the visual pickup animation towards target position
func (e *ItemEntity) StartPickupAnimation(targetPos mgl32.Vec3) {
	e.IsPickingUp = true
	e.PickupProgress = 0.0
	e.PickupStartPos = e.Pos
	e.PickupTargetPos = targetPos
}

func (e *ItemEntity) IsDead() bool {
	return e.Dead
}

func (e *ItemEntity) SetDead() {
	e.Dead = true
}

// GetBounds returns the item entity dimensions
func (e *ItemEntity) GetBounds() (width, height float32) {
	return ItemEntityWidth, ItemEntityHeight
}

// SetNoDespawn marks this item as never despawning
func (e *ItemEntity) SetNoDespawn() {
	e.noDespawn = true
}

// SetInfinitePickupDelay marks this item as unpickable
func (e *ItemEntity) SetInfinitePickupDelay() {
	e.PickupDelay = InfinitePickupDelay
}
