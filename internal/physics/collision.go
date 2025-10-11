package physics

import (
	"math"

	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

// Checks if a position collides with any block in the world
func Collides(pos mgl32.Vec3, playerHeight float32, world *world.World) bool {
	defer profiling.Track("physics.Collides")()
	minX := int(math.Floor(float64(pos.X() - 0.3 + 0.5)))
	maxX := int(math.Floor(float64(pos.X() + 0.3 + 0.5)))
	// Y uses top-at-integer mapping
	minY := int(math.Ceil(float64(pos.Y())))
	maxY := int(math.Ceil(float64(pos.Y() + playerHeight)))
	minZ := int(math.Floor(float64(pos.Z() - 0.3 + 0.5)))
	maxZ := int(math.Floor(float64(pos.Z() + 0.3 + 0.5)))

	for x := minX - 1; x <= maxX+1; x++ {
		for y := minY - 1; y <= maxY+1; y++ {
			for z := minZ - 1; z <= maxZ+1; z++ {
				if !world.IsAir(x, y, z) {
					blockMinX := float32(x) - 0.5
					blockMaxX := float32(x) + 0.5
					// With top-at-integer mapping, Y range is (y-1, y]
					blockMinY := float32(y) - 1.0
					blockMaxY := float32(y)
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
	defer profiling.Track("physics.FindGroundLevel")()
	minX := int(math.Floor(float64(x - 0.3 + 0.5)))
	maxX := int(math.Floor(float64(x + 0.3 + 0.5)))
	minZ := int(math.Floor(float64(z - 0.3 + 0.5)))
	maxZ := int(math.Floor(float64(z + 0.3 + 0.5)))

	// Player horizontal AABB (XZ)
	playerMinX := x - 0.3
	playerMaxX := x + 0.3
	playerMinZ := z - 0.3
	playerMaxZ := z + 0.3

	// Use -Inf to indicate "no ground found"
	maxGroundY := float32(math.Inf(-1))
	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			// Only consider blocks that overlap horizontally with player footprint
			blockMinX := float32(bx) - 0.5
			blockMaxX := float32(bx) + 0.5
			blockMinZ := float32(bz) - 0.5
			blockMaxZ := float32(bz) + 0.5
			if !(playerMinX < blockMaxX && playerMaxX > blockMinX && playerMinZ < blockMaxZ && playerMaxZ > blockMinZ) {
				continue
			}
			for by := int(math.Ceil(float64(playerPos.Y()))); by >= 0; by-- {
				if !world.IsAir(bx, by, bz) {
					// Top of block is exactly at integer by
					groundY := float32(by)
					if groundY > maxGroundY {
						maxGroundY = groundY
					}
					break
				}
			}
		}
	}
	return maxGroundY
}

// IntersectsBlock checks if the player's AABB would intersect with the given block coordinates
func IntersectsBlock(playerPos mgl32.Vec3, playerHeight float32, bx, by, bz int) bool {
	blockMinX := float32(bx) - 0.5
	blockMaxX := float32(bx) + 0.5
	blockMinY := float32(by) - 0.5
	blockMaxY := float32(by) + 0.5
	blockMinZ := float32(bz) - 0.5
	blockMaxZ := float32(bz) + 0.5

	// Player half-width around X/Z and height along Y
	playerMinX := playerPos.X() - 0.3
	playerMaxX := playerPos.X() + 0.3
	playerMinY := playerPos.Y()
	playerMaxY := playerPos.Y() + playerHeight
	playerMinZ := playerPos.Z() - 0.3
	playerMaxZ := playerPos.Z() + 0.3

	return playerMinX < blockMaxX && playerMaxX > blockMinX &&
		playerMinY < blockMaxY && playerMaxY > blockMinY &&
		playerMinZ < blockMaxZ && playerMaxZ > blockMinZ
}

// FindCeilingLevel finds the lowest ceiling (bottom face of a block) above the player's head
func FindCeilingLevel(x, z float32, playerPos mgl32.Vec3, playerHeight float32, world *world.World) float32 {
	defer profiling.Track("physics.FindCeilingLevel")()
	minX := int(math.Floor(float64(x - 0.3 + 0.5)))
	maxX := int(math.Floor(float64(x + 0.3 + 0.5)))
	minZ := int(math.Floor(float64(z - 0.3 + 0.5)))
	maxZ := int(math.Floor(float64(z + 0.3 + 0.5)))

	// Player horizontal AABB (XZ)
	playerMinX := x - 0.3
	playerMaxX := x + 0.3
	playerMinZ := z - 0.3
	playerMaxZ := z + 0.3

	minCeilingY := float32(256)
	startY := int(math.Ceil(float64(playerPos.Y() + playerHeight)))
	if startY < 0 {
		startY = 0
	}
	if startY > 255 {
		startY = 255
	}
	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			// Only consider blocks that overlap horizontally with player footprint
			blockMinX := float32(bx) - 0.5
			blockMaxX := float32(bx) + 0.5
			blockMinZ := float32(bz) - 0.5
			blockMaxZ := float32(bz) + 0.5
			if !(playerMinX < blockMaxX && playerMaxX > blockMinX && playerMinZ < blockMaxZ && playerMaxZ > blockMinZ) {
				continue
			}
			for by := startY; by <= 255; by++ {
				if !world.IsAir(bx, by, bz) {
					// Bottom of block is at (by - 1)
					ceilingY := float32(by - 1)
					if ceilingY < minCeilingY {
						minCeilingY = ceilingY
					}
					break
				}
			}
		}
	}
	return minCeilingY
}
