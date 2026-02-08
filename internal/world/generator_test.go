package world

import (
	"testing"
)

func TestStandardGeneratorImplementsInterface(t *testing.T) {
	var _ TerrainGenerator = NewGenerator(123)
}

func TestFlatGeneratorImplementsInterface(t *testing.T) {
	var _ TerrainGenerator = NewFlatGenerator(10)
}

func TestFlatGeneratorHeight(t *testing.T) {
	g := NewFlatGenerator(10)
	if h := g.HeightAt(0, 0); h != 10 {
		t.Errorf("Expected height 10, got %d", h)
	}
	if h := g.HeightAt(100, -50); h != 10 {
		t.Errorf("Expected height 10, got %d", h)
	}
}

func TestFlatGeneratorPopulate(t *testing.T) {
	c := NewChunk(0, 0, 0)
	g := NewFlatGenerator(5) // Height 5

	g.PopulateChunk(c)

	// Check block at 0,0,0 (Bedrock)
	if b := c.GetBlock(0, 0, 0); b != BlockTypeBedrock {
		t.Errorf("Expected Bedrock at 0,0,0, got %v", b)
	}

	// Check block at 0,1,0 to 0,4,0 (Dirt)
	for y := 1; y < 5; y++ {
		if b := c.GetBlock(0, y, 0); b != BlockTypeDirt {
			t.Errorf("Expected Dirt at 0,%d,0, got %v", y, b)
		}
	}

	// Check block at 0,5,0 (Grass)
	if b := c.GetBlock(0, 5, 0); b != BlockTypeGrass {
		t.Errorf("Expected Grass at 0,5,0, got %v", b)
	}

	// Check block at 0,6,0 (Air)
	if b := c.GetBlock(0, 6, 0); b != BlockTypeAir {
		t.Errorf("Expected Air at 0,6,0, got %v", b)
	}
}
