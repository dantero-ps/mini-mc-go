package world

import (
	"crypto/sha256"
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

// hashChunkBlocks computes a SHA-256 hash of all blocks in a chunk
func hashChunkBlocks(c *Chunk) [32]byte {
	h := sha256.New()
	for ly := 0; ly < ChunkSizeY; ly++ {
		for lx := 0; lx < ChunkSizeX; lx++ {
			for lz := 0; lz < ChunkSizeZ; lz++ {
				b := byte(c.GetBlock(lx, ly, lz))
				h.Write([]byte{b})
			}
		}
	}
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// TestDensityGeneratorImplementsInterface verifies compile-time interface compliance
func TestDensityGeneratorImplementsInterface(t *testing.T) {
	var _ TerrainGenerator = NewDensityGenerator(123)
}

// TestDensityDeterminism verifies same seed produces identical terrain
func TestDensityDeterminism(t *testing.T) {
	seed := int64(12345)
	var hashes [100][32]byte

	for i := range hashes {
		g := NewDensityGenerator(seed)
		c := NewChunk(0, 0, 0)
		g.PopulateChunk(c)
		hashes[i] = hashChunkBlocks(c)
	}

	// All hashes must be identical
	first := hashes[0]
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != first {
			t.Errorf("Chunk generation not deterministic: hash[0] != hash[%d]", i)
		}
	}
}

// TestDensityDeterminismMultipleChunks verifies world coordinates used correctly
func TestDensityDeterminismMultipleChunks(t *testing.T) {
	seed := int64(12345)
	chunkPositions := [][3]int{
		{0, 0, 0},
		{1, 0, 0},
		{0, 0, 1},
		{-1, 0, -1},
	}

	for _, pos := range chunkPositions {
		g1 := NewDensityGenerator(seed)
		c1 := NewChunk(pos[0], pos[1], pos[2])
		g1.PopulateChunk(c1)
		hash1 := hashChunkBlocks(c1)

		g2 := NewDensityGenerator(seed)
		c2 := NewChunk(pos[0], pos[1], pos[2])
		g2.PopulateChunk(c2)
		hash2 := hashChunkBlocks(c2)

		if hash1 != hash2 {
			t.Errorf("Chunk at (%d,%d,%d) not deterministic", pos[0], pos[1], pos[2])
		}
	}
}

// TestDensityTerrainNotEmpty verifies terrain exists (not all air)
func TestDensityTerrainNotEmpty(t *testing.T) {
	g := NewDensityGenerator(1337)
	c := NewChunk(0, 0, 0)
	g.PopulateChunk(c)

	nonAirCount := 0
	for ly := 0; ly < ChunkSizeY; ly++ {
		for lx := 0; lx < ChunkSizeX; lx++ {
			for lz := 0; lz < ChunkSizeZ; lz++ {
				if c.GetBlock(lx, ly, lz) != BlockTypeAir {
					nonAirCount++
				}
			}
		}
	}

	if nonAirCount == 0 {
		t.Errorf("Expected terrain to have non-air blocks, got all air")
	}
}

// TestDensityTerrainNotSolid verifies terrain has air (not all solid)
func TestDensityTerrainNotSolid(t *testing.T) {
	g := NewDensityGenerator(1337)
	c := NewChunk(0, 0, 0)
	g.PopulateChunk(c)

	airCount := 0
	for ly := 0; ly < ChunkSizeY; ly++ {
		for lx := 0; lx < ChunkSizeX; lx++ {
			for lz := 0; lz < ChunkSizeZ; lz++ {
				if c.GetBlock(lx, ly, lz) == BlockTypeAir {
					airCount++
				}
			}
		}
	}

	if airCount == 0 {
		t.Errorf("Expected terrain to have air blocks, got all solid")
	}
}

// TestDensityBedrockAtZero verifies y=0 is always bedrock
func TestDensityBedrockAtZero(t *testing.T) {
	g := NewDensityGenerator(1337)
	c := NewChunk(0, 0, 0)
	g.PopulateChunk(c)

	// Check center block at y=0
	if b := c.GetBlock(8, 0, 8); b != BlockTypeBedrock {
		t.Errorf("Expected Bedrock at (8,0,8), got %v", b)
	}
}

// TestDensityHighAltitudeAir verifies high altitude chunks are all air
func TestDensityHighAltitudeAir(t *testing.T) {
	g := NewDensityGenerator(1337)
	// Chunk Y=1 means worldY 256-511, far above baseHeight=64
	c := NewChunk(0, 1, 0)
	g.PopulateChunk(c)

	// All blocks should be air
	for ly := 0; ly < ChunkSizeY; ly++ {
		for lx := 0; lx < ChunkSizeX; lx++ {
			for lz := 0; lz < ChunkSizeZ; lz++ {
				if b := c.GetBlock(lx, ly, lz); b != BlockTypeAir {
					t.Errorf("Expected Air at (%d,%d,%d) in high altitude chunk, got %v",
						lx, ly, lz, b)
					return
				}
			}
		}
	}
}

// TestDensityHeightAt verifies HeightAt returns reasonable value
func TestDensityHeightAt(t *testing.T) {
	g := NewDensityGenerator(1337)
	h := g.HeightAt(0, 0)

	// Height should be > 0 and <= baseHeight + gradientStrength (roughly 64+32=96)
	if h <= 0 {
		t.Errorf("HeightAt returned %d, expected > 0", h)
	}
	if h > 96 {
		t.Errorf("HeightAt returned %d, expected <= 96 (baseHeight + gradientStrength)", h)
	}
}

// BenchmarkDensityPopulateChunk measures chunk generation performance
func BenchmarkDensityPopulateChunk(b *testing.B) {
	g := NewDensityGenerator(12345)
	c := NewChunk(0, 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset chunk to empty state for fair benchmark
		c = NewChunk(0, 0, 0)
		g.PopulateChunk(c)
	}
}
