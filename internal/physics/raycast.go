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

	// Initial block position
	bx := int(math.Floor(float64(start.X())))
	by := int(math.Floor(float64(start.Y())))
	bz := int(math.Floor(float64(start.Z())))

	// Direction signs and steps
	stepX := 1
	stepY := 1
	stepZ := 1

	if direction.X() < 0 {
		stepX = -1
	}
	if direction.Y() < 0 {
		stepY = -1
	}
	if direction.Z() < 0 {
		stepZ = -1
	}

	// tDelta is the distance along the ray to travel one unit in each direction
	// avoid division by zero
	var tDeltaX, tDeltaY, tDeltaZ float32
	if direction.X() == 0 {
		tDeltaX = float32(math.Inf(1))
	} else {
		tDeltaX = float32(math.Abs(float64(1.0 / direction.X())))
	}
	if direction.Y() == 0 {
		tDeltaY = float32(math.Inf(1))
	} else {
		tDeltaY = float32(math.Abs(float64(1.0 / direction.Y())))
	}
	if direction.Z() == 0 {
		tDeltaZ = float32(math.Inf(1))
	} else {
		tDeltaZ = float32(math.Abs(float64(1.0 / direction.Z())))
	}

	// tMax is the distance along the ray to the next boundary
	var tMaxX, tMaxY, tMaxZ float32

	// Calculate initial tMax
	if direction.X() > 0 {
		tMaxX = (float32(bx+1) - start.X()) * tDeltaX
	} else if direction.X() < 0 {
		tMaxX = (start.X() - float32(bx)) * tDeltaX
	} else {
		tMaxX = float32(math.Inf(1))
	}

	if direction.Y() > 0 {
		tMaxY = (float32(by+1) - start.Y()) * tDeltaY
	} else if direction.Y() < 0 {
		tMaxY = (start.Y() - float32(by)) * tDeltaY
	} else {
		tMaxY = float32(math.Inf(1))
	}

	if direction.Z() > 0 {
		tMaxZ = (float32(bz+1) - start.Z()) * tDeltaZ
	} else if direction.Z() < 0 {
		tMaxZ = (start.Z() - float32(bz)) * tDeltaZ
	} else {
		tMaxZ = float32(math.Inf(1))
	}

	result := RaycastResult{Hit: false}

	// DDA Loop
	// We loop until we exceed maxDist or find a hit
	for {
		var dist float32
		var axis int // 0:x, 1:y, 2:z

		// Find the smallest tMax to determine the next step
		if tMaxX < tMaxY {
			if tMaxX < tMaxZ {
				axis = 0
			} else {
				axis = 2
			}
		} else {
			if tMaxY < tMaxZ {
				axis = 1
			} else {
				axis = 2
			}
		}

		// Step and update tMax
		if axis == 0 {
			dist = tMaxX
			tMaxX += tDeltaX
			bx += stepX
		} else if axis == 1 {
			dist = tMaxY
			tMaxY += tDeltaY
			by += stepY
		} else {
			dist = tMaxZ
			tMaxZ += tDeltaZ
			bz += stepZ
		}

		if dist > maxDist {
			break
		}

		// Check bounds (0-255 Y)
		if by < 0 || by > 255 {
			continue
		}

		// Check if block is not air
		if !world.IsAir(bx, by, bz) {
			if dist < minDist {
				continue
			}

			result.HitPosition = [3]int{bx, by, bz}

			// Adjacent position is the one we stepped from
			adj := [3]int{bx, by, bz}
			if axis == 0 {
				adj[0] -= stepX
			} else if axis == 1 {
				adj[1] -= stepY
			} else {
				adj[2] -= stepZ
			}
			result.AdjacentPosition = adj

			result.Distance = dist
			result.Hit = true
			return result
		}
	}

	return result
}
