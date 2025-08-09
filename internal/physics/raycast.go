package physics

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"mini-mc/internal/world"
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
	stepSize := float32(0.02)
	steps := int(maxDist / stepSize)

	var lastEmptyPos [3]int
	result := RaycastResult{Hit: false}

	for i := 0; i <= steps; i++ {
		dist := float32(i) * stepSize
		if dist < minDist {
			continue
		}

		pos := start.Add(direction.Mul(dist))

		blockPos := [3]int{
			int(math.Floor(float64(pos.X()) + 0.5)),
			int(math.Floor(float64(pos.Y()) + 0.5)),
			int(math.Floor(float64(pos.Z()) + 0.5)),
		}

		if !world.IsAir(blockPos[0], blockPos[1], blockPos[2]) {
			bx, by, bz := float32(blockPos[0]), float32(blockPos[1]), float32(blockPos[2])
			if pos.X() >= bx-0.5 && pos.X() < bx+0.5 &&
				pos.Y() >= by-0.5 && pos.Y() < by+0.5 &&
				pos.Z() >= bz-0.5 && pos.Z() < bz+0.5 {

				result.HitPosition = blockPos
				result.AdjacentPosition = lastEmptyPos
				result.Distance = dist
				result.Hit = true
				return result
			}
		}

		lastEmptyPos = blockPos
	}

	return result
}
