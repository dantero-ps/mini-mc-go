package main

import (
	"fmt"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func TestRaycast(t *testing.T) {
	// Test blocks - create a simple world
	blocks := map[string]bool{
		"0,0,0": true,  // Ground block
		"1,0,0": true,  // Adjacent block
		"0,1,0": true,  // Block above ground
		"0,0,1": true,  // Another ground block
	}

	tests := []struct {
		name      string
		start     mgl32.Vec3
		direction mgl32.Vec3
		maxDist   float32
		expectHit bool
		expectHitPos [3]int
		expectPlacePos [3]int
	}{
		{
			name:      "Look at ground block from above",
			start:     mgl32.Vec3{0, 2, 0},
			direction: mgl32.Vec3{0, -1, 0}, // Look down
			maxDist:   3.0,
			expectHit: true,
			expectHitPos: [3]int{0, 1, 0}, // Should hit the block at (0,1,0)
			expectPlacePos: [3]int{0, 2, 0}, // Should place at (0,2,0)
		},
		{
			name:      "Look at side of block",
			start:     mgl32.Vec3{-1, 0.5, 0},
			direction: mgl32.Vec3{1, 0, 0}, // Look right
			maxDist:   2.0,
			expectHit: true,
			expectHitPos: [3]int{0, 0, 0}, // Should hit ground block
			expectPlacePos: [3]int{-1, 0, 0}, // Should place to the left
		},
		{
			name:      "Look into empty space",
			start:     mgl32.Vec3{5, 5, 5},
			direction: mgl32.Vec3{1, 0, 0}, // Look into empty space
			maxDist:   2.0,
			expectHit: false,
		},
		{
			name:      "Look at distant block",
			start:     mgl32.Vec3{-1, 0.6, 0}, // Player eye level, away from blocks
			direction: mgl32.Vec3{1, 0, 0}, // Look right toward blocks
			maxDist:   2.5,
			expectHit: true,
			expectHitPos: [3]int{0, 0, 0}, // Should hit ground block
			expectPlacePos: [3]int{-1, 0, 0}, // Should place at last empty position
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hitPos, placePos, hit := raycast(tt.start, tt.direction, tt.maxDist, blocks)
			
			if hit != tt.expectHit {
				t.Errorf("Expected hit=%v, got hit=%v", tt.expectHit, hit)
				return
			}
			
			if tt.expectHit {
				if hitPos != tt.expectHitPos {
					t.Errorf("Expected hitPos=%v, got hitPos=%v", tt.expectHitPos, hitPos)
				}
				if placePos != tt.expectPlacePos {
					t.Errorf("Expected placePos=%v, got placePos=%v", tt.expectPlacePos, placePos)
				}
				
				fmt.Printf("✓ Test '%s':\n", tt.name)
				fmt.Printf("  Hit block at: %v\n", hitPos)
				fmt.Printf("  Would place at: %v\n", placePos)
				fmt.Printf("  Start: %v, Direction: %v\n\n", tt.start, tt.direction)
			} else {
				fmt.Printf("✓ Test '%s': No hit (as expected)\n\n", tt.name)
			}
		})
	}
}

func TestRaycastManual(t *testing.T) {
	// Manual test - simulate actual gameplay scenario
	blocks := map[string]bool{}
	
	// Create a simple ground
	for x := -2; x <= 2; x++ {
		for z := -2; z <= 2; z++ {
			blocks[key(x, 0, z)] = true
		}
	}
	
	// Add a wall
	blocks[key(2, 1, 0)] = true
	blocks[key(2, 2, 0)] = true
	
	playerPos := mgl32.Vec3{0, 1, 0}
	rayStart := playerPos.Add(mgl32.Vec3{0, 0.6, 0}) // Eye level
	
	fmt.Println("=== Manual Raycast Test ===")
	fmt.Printf("Player at: %v\n", playerPos)
	fmt.Printf("Ray starts at: %v (eye level)\n", rayStart)
	
	// Test different directions
	directions := map[string]mgl32.Vec3{
		"Forward (Z+)": {0, 0, 1},
		"Right (X+)":   {1, 0, 0},
		"Down":         {0, -1, 0},
		"Up":           {0, 1, 0},
		"Diagonal":     {1, 0, 1},
	}
	
	for name, dir := range directions {
		hitPos, placePos, hit := raycast(rayStart, dir.Normalize(), 2.5, blocks)
		
		fmt.Printf("\nLooking %s (%v):\n", name, dir)
		if hit {
			fmt.Printf("  ✓ Hit block at: %v\n", hitPos)
			fmt.Printf("  ✓ Would place block at: %v\n", placePos)
		} else {
			fmt.Printf("  ✗ No hit\n")
		}
	}
}
