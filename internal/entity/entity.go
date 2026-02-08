package entity

import (
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

// WorldSource interface abstracts the World for entities
type WorldSource interface {
	IsAir(x, y, z int) bool
	Get(x, y, z int) world.BlockType
}

// Entity interface
type Entity interface {
	Update(dt float64)
	Position() mgl32.Vec3
	IsDead() bool
	SetDead()
	GetBounds() (width, height float32)
}
