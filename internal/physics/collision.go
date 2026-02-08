package physics

import (
	"fmt"
	"math"
	"time"

	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

// Checks if a position collides with any block in the world
func Collides(pos mgl32.Vec3, width, height float32, world *world.World) bool {
	now := time.Now()

	defer profiling.Track("physics.Collides")()
	minX := int(math.Floor(float64(pos.X() - width/2)))
	maxX := int(math.Floor(float64(pos.X() + width/2)))
	// Y uses bottom-at-integer mapping (Standard)
	minY := int(math.Floor(float64(pos.Y())))
	maxY := int(math.Floor(float64(pos.Y() + height)))
	minZ := int(math.Floor(float64(pos.Z() - width/2)))
	maxZ := int(math.Floor(float64(pos.Z() + width/2)))

	iterations := 0

	for x := minX - 1; x <= maxX+1; x++ {
		for y := minY - 1; y <= maxY+1; y++ {
			for z := minZ - 1; z <= maxZ+1; z++ {
				if !world.IsAir(x, y, z) {
					iterations++
					blockMinX := float32(x)
					blockMaxX := float32(x) + 1.0
					// Standard mapping: Y range is [y, y+1)
					blockMinY := float32(y)
					blockMaxY := float32(y) + 1.0
					blockMinZ := float32(z)
					blockMaxZ := float32(z) + 1.0

					isCollidingMaxX := pos.X()-width/2 < blockMaxX
					isCollidingMinX := pos.X()+width/2 > blockMinX
					isCollidingMaxY := pos.Y() < blockMaxY
					isCollidingMinY := pos.Y()+height > blockMinY
					isCollidingMaxZ := pos.Z()-width/2 < blockMaxZ
					isCollidingMinZ := pos.Z()+width/2 > blockMinZ
					if isCollidingMaxX && isCollidingMinX &&
						isCollidingMaxY && isCollidingMinY &&
						isCollidingMaxZ && isCollidingMinZ {
						return true
					}
				}
			}
		}
	}

	d := time.Since(now)
	if d > 10*time.Millisecond {
		fmt.Println(maxX, maxY, maxZ, iterations)
	}

	return false
}

// FindGroundLevel finds the highest block below the player
func FindGroundLevel(x, z float32, playerPos mgl32.Vec3, width, height float32, world *world.World) float32 {
	defer profiling.Track("physics.FindGroundLevel")()
	minX := int(math.Floor(float64(x - width/2)))
	maxX := int(math.Floor(float64(x + width/2)))
	minZ := int(math.Floor(float64(z - width/2)))
	maxZ := int(math.Floor(float64(z + width/2)))

	// Player horizontal AABB (XZ)
	playerMinX := x - width/2
	playerMaxX := x + width/2
	playerMinZ := z - width/2
	playerMaxZ := z + width/2

	// Use -Inf to indicate "no ground found"
	maxGroundY := float32(math.Inf(-1))
	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			// Only consider blocks that overlap horizontally with player footprint
			blockMinX := float32(bx)
			blockMaxX := float32(bx) + 1.0
			blockMinZ := float32(bz)
			blockMaxZ := float32(bz) + 1.0
			if !(playerMinX < blockMaxX && playerMaxX > blockMinX && playerMinZ < blockMaxZ && playerMaxZ > blockMinZ) {
				continue
			}
			// Search from player feet downwards
			for by := int(math.Floor(float64(playerPos.Y()))); by >= 0; by-- {
				if !world.IsAir(bx, by, bz) {
					// Top of block is at y+1
					groundY := float32(by) + 1.0
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
func IntersectsBlock(playerPos mgl32.Vec3, width, height float32, bx, by, bz int) bool {
	blockMinX := float32(bx)
	blockMaxX := float32(bx) + 1.0
	// Standard mapping: Y range is [y, y+1)
	blockMinY := float32(by)
	blockMaxY := float32(by) + 1.0
	blockMinZ := float32(bz)
	blockMaxZ := float32(bz) + 1.0

	// Player half-width around X/Z and height along Y
	playerMinX := playerPos.X() - width/2
	playerMaxX := playerPos.X() + width/2
	playerMinY := playerPos.Y()
	playerMaxY := playerPos.Y() + height
	playerMinZ := playerPos.Z() - width/2
	playerMaxZ := playerPos.Z() + width/2

	return playerMinX < blockMaxX && playerMaxX > blockMinX &&
		playerMinY < blockMaxY && playerMaxY > blockMinY &&
		playerMinZ < blockMaxZ && playerMaxZ > blockMinZ
}

// FindCeilingLevel finds the lowest ceiling (bottom face of a block) above the player's head
func FindCeilingLevel(x, z float32, playerPos mgl32.Vec3, width, height float32, world *world.World) float32 {
	defer profiling.Track("physics.FindCeilingLevel")()
	minX := int(math.Floor(float64(x - width/2)))
	maxX := int(math.Floor(float64(x + width/2)))
	minZ := int(math.Floor(float64(z - width/2)))
	maxZ := int(math.Floor(float64(z + width/2)))

	// Player horizontal AABB (XZ)
	playerMinX := x - width/2
	playerMaxX := x + width/2
	playerMinZ := z - width/2
	playerMaxZ := z + width/2

	minCeilingY := float32(256)
	// Check from player head upwards
	startY := int(math.Floor(float64(playerPos.Y() + height)))
	startY = max(min(startY, 255), 0)

	for bx := minX; bx <= maxX; bx++ {
		for bz := minZ; bz <= maxZ; bz++ {
			// Only consider blocks that overlap horizontally with player footprint
			blockMinX := float32(bx)
			blockMaxX := float32(bx) + 1.0
			blockMinZ := float32(bz)
			blockMaxZ := float32(bz) + 1.0
			if !(playerMinX < blockMaxX && playerMaxX > blockMinX && playerMinZ < blockMaxZ && playerMaxZ > blockMinZ) {
				continue
			}
			for by := startY; by <= 255; by++ {
				if !world.IsAir(bx, by, bz) {
					// Bottom of block is at by
					ceilingY := float32(by)
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
