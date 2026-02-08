package world

import (
	"math/rand"
	"testing"
)

func TestAuthenticNoiseGeneratorImproved_PopulateNoiseArray(t *testing.T) {
	rnd := rand.New(rand.NewSource(12345))
	gen := NewAuthenticNoiseGeneratorImproved(rnd)

	xSize, ySize, zSize := 5, 5, 5
	noiseArray := make([]float64, xSize*ySize*zSize)

	gen.PopulateNoiseArray(noiseArray, 0, 0, 0, xSize, ySize, zSize, 0.5, 0.5, 0.5, 1.0)

	// Check for non-zero values (probabilistic but likely)
	hasNonZero := false
	for _, v := range noiseArray {
		if v != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		t.Errorf("Expected noise array to have non-zero values")
	}
}

func TestAuthenticNoiseGeneratorOctaves_GenerateNoiseOctaves(t *testing.T) {
	rnd := rand.New(rand.NewSource(12345))
	gen := NewAuthenticNoiseGeneratorOctaves(rnd, 4)

	xSize, ySize, zSize := 5, 5, 5
	noiseArray := gen.GenerateNoiseOctaves(nil, 0, 0, 0, xSize, ySize, zSize, 0.5, 0.5, 0.5)

	if len(noiseArray) != xSize*ySize*zSize {
		t.Errorf("Expected array length %d, got %d", xSize*ySize*zSize, len(noiseArray))
	}

	hasNonZero := false
	for _, v := range noiseArray {
		if v != 0 {
			hasNonZero = true
			break
		}
	}

	if !hasNonZero {
		t.Errorf("Expected noise octaves array to have non-zero values")
	}
}
