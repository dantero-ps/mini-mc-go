package physics

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"mini-mc/internal/world"
)

// Checks if a position collides with any block in the world
func Collides(pos mgl32.Vec3, playerHeight float32, world *world.World) bool {
	minX := int(math.Floor(float64(pos.X() - 0.3 + 0.5)))
	maxX := int(math.Floor(float64(pos.X() + 0.3 + 0.5)))
	minY := int(math.Floor(float64(pos.Y() + 0.5)))
	maxY := int(math.Floor(float64(pos.Y() + playerHeight + 0.5)))
	minZ := int(math.Floor(float64(pos.Z() - 0.3 + 0.5)))
	maxZ := int(math.Floor(float64(pos.Z() + 0.3 + 0.5)))

	for x := minX - 1; x <= maxX+1; x++ {
		for y := minY - 1; y <= maxY+1; y++ {
			for z := minZ - 1; z <= maxZ+1; z++ {
				if !world.IsAir(x, y, z) {
					blockMinX := float32(x) - 0.5
					blockMaxX := float32(x) + 0.5
					blockMinY := float32(y) - 0.5
					blockMaxY := float32(y) + 0.5
					blockMinZ := float32(z) - 0.5
					blockMaxZ := float32(z) + 0.5

					if pos.X()-0.3 < blockMaxX && pos.X()+0.3 > blockMinX &&
						pos.Y() < blockMaxY && pos.Y()+playerHeight > blockMinY &&
						pos.Z()-0.3 < blockMaxZ && pos.Z()+0.3 > blockMinZ {
						return true
					}
				}
			}
		}
	}
	return false
}

// FindGroundLevel finds the highest block below the player
func FindGroundLevel(x, z float32, playerPos mgl32.Vec3, world *world.World) float32 {
	minX := int(math.Floor(float64(x - 0.3 + 0.5)))
	maxX := int(math.Floor(float64(x + 0.3 + 0.5)))
	minZ := int(math.Floor(float64(z - 0.3 + 0.5)))
	maxZ := int(math.Floor(float64(z + 0.3 + 0.5)))

	maxGroundY := float32(-10)
	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			for by := int(math.Floor(float64(playerPos.Y()) + 0.5)); by >= -10; by-- {
				if !world.IsAir(bx, by, bz) {
					groundY := float32(by) + 0.5 // Top of block
					if groundY > maxGroundY {
						maxGroundY = groundY
					}
					break
				}
			}
		}
	}
	if maxGroundY == -10 {
		return 1.0 // Safe default
	}
	return maxGroundY
}
