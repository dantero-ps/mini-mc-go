package entity

import (
	"math"
	"math/rand"
	"mini-mc/internal/item"

	"github.com/go-gl/mathgl/mgl32"
)

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
	if e.Age >= 300.0 {
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
