package physics_test

import (
	"mini-mc/internal/physics"
	"mini-mc/internal/world"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestRaycast(t *testing.T) {
	// Create an empty world
	w := world.NewEmpty()

	// Place a block at (5, 0, 0)
	w.Set(5, 0, 0, world.BlockTypeStone)

	// Test 1: Raycast hitting the block
	start := mgl32.Vec3{0.5, 0.5, 0.5}
	dir := mgl32.Vec3{1, 0, 0}
	minDist := float32(0.1)
	maxDist := float32(10.0)

	result := physics.Raycast(start, dir, minDist, maxDist, w)

	if !result.Hit {
		t.Fatalf("Expected hit, got miss")
	}
	if result.HitPosition != [3]int{5, 0, 0} {
		t.Errorf("Expected hit at {5,0,0}, got %v", result.HitPosition)
	}
	if result.AdjacentPosition != [3]int{4, 0, 0} {
		t.Errorf("Expected adjacent at {4,0,0}, got %v", result.AdjacentPosition)
	}
	// Expected distance:
	// Ray starts at X=0.5. Hits X=5.0 boundary. Dist = 4.5.
	// Allow small float error
	if result.Distance < 4.49 || result.Distance > 4.51 {
		t.Errorf("Expected distance 4.5, got %f", result.Distance)
	}

	// Test 2: Raycast missing (max dist)
	maxDistShort := float32(4.0)
	resultShort := physics.Raycast(start, dir, minDist, maxDistShort, w)
	if resultShort.Hit {
		t.Errorf("Expected miss due to maxDist, got hit at %v", resultShort.HitPosition)
	}

	// Test 3: Raycast missing (wrong direction)
	dirWrong := mgl32.Vec3{0, 1, 0}
	resultWrong := physics.Raycast(start, dirWrong, minDist, maxDist, w)
	if resultWrong.Hit {
		t.Errorf("Expected miss, got hit")
	}

	// Test 4: Raycast hitting with mixed direction
	// Start at (0.5, 0.5, 0.5). Target at (2, 2, 2).
	// Ray direction should be normalized (1,1,1).
	// Distance to hit:
	// We hit (2,2,2).
	// Boundary is x=2, y=2, z=2.
	// We hit x=2 at t = (2-0.5)/dirX.
	// dir = (1,1,1) normalized = (1/sqrt(3), 1/sqrt(3), 1/sqrt(3)).
	// dx = 0.577...
	// t = 1.5 / 0.577 = 1.5 * sqrt(3) = 2.598...

	w.Set(2, 2, 2, world.BlockTypeStone)
	dirDiag := mgl32.Vec3{1, 1, 1}.Normalize()
	resultDiag := physics.Raycast(start, dirDiag, minDist, maxDist, w)

	if !resultDiag.Hit {
		t.Errorf("Expected hit at {2,2,2}, got miss")
	} else {
		if resultDiag.HitPosition != [3]int{2, 2, 2} {
			t.Errorf("Expected hit at {2,2,2}, got %v", resultDiag.HitPosition)
		}
	}
}
