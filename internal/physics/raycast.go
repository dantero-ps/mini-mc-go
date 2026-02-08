package physics

import (
	"math"
	"mini-mc/internal/profiling"

	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	MinReachDistance = 0.1
	MaxReachDistance = 5.0
)

// RaycastResult stores the result of a raycast operation
type RaycastResult struct {
	HitPosition      [3]int
	AdjacentPosition [3]int
	Distance         float32
	Hit              bool
}

// Raycast performs a ray casting operation from a starting point in a given direction
func Raycast(start mgl32.Vec3, direction mgl32.Vec3, minDist, maxDist float32, world *world.World) RaycastResult {
	defer profiling.Track("physics.Raycast")()
	stepSize := float32(0.05) // Small enough to not miss thin blocks, but larger for perf
	steps := int(maxDist / stepSize)

	result := RaycastResult{Hit: false}

	// Keep track of the last block position to determine adjacent face
	// Initialize with the block containing the start position
	startBlockX := int(math.Floor(float64(start.X())))
	startBlockY := int(math.Floor(float64(start.Y())))
	startBlockZ := int(math.Floor(float64(start.Z())))
	lastBlockPos := [3]int{startBlockX, startBlockY, startBlockZ}

	for i := 0; i <= steps; i++ {
		dist := float32(i) * stepSize
		if dist < minDist {
			continue
		}

		pos := start.Add(direction.Mul(dist))

		// Convert to block coordinates (Standard: Bottom-Left corner is integer)
		bx := int(math.Floor(float64(pos.X())))
		by := int(math.Floor(float64(pos.Y())))
		bz := int(math.Floor(float64(pos.Z())))

		// Check bounds (optional, but good for safety)
		if by < 0 || by > 255 {
			lastBlockPos = [3]int{bx, by, bz}
			continue
		}

		// Check if block is not air
		if !world.IsAir(bx, by, bz) {
			// Hit found!
			result.HitPosition = [3]int{bx, by, bz}
			result.AdjacentPosition = lastBlockPos
			result.Distance = dist
			result.Hit = true
			return result
		}

		lastBlockPos = [3]int{bx, by, bz}
	}

	return result
}
