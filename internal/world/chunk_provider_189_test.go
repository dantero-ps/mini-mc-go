package world

import (
	"testing"
)

func TestChunkProvider189_GenerateChunk(t *testing.T) {
	seed := int64(12345)
	cp := NewChunkProvider189(seed)

	chunk := cp.GenerateChunk(0, 0)

	if chunk == nil {
		t.Fatal("GenerateChunk returned nil")
	}

	// Verify some blocks are set
	hasStone := false
	hasAir := false

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < 256; y++ {
				block := chunk.GetBlock(x, y, z)
				if block == BlockTypeStone {
					hasStone = true
				} else if block == BlockTypeAir {
					hasAir = true
				}
			}
		}
	}

	if !hasStone {
		t.Error("Chunk has no Stone")
	}
	if !hasAir {
		t.Error("Chunk has no Air")
	}
	// Note: Depending on seed, Water might not be present if land is high, but usually at (0,0) there should be some water or stone.
	// But let's check water exists below Y=63 if density <= 0.
}

func TestChunkProvider189_Determinism(t *testing.T) {
	seed := int64(98765)
	cp1 := NewChunkProvider189(seed)
	cp2 := NewChunkProvider189(seed)

	chunk1 := cp1.GenerateChunk(10, 10)
	chunk2 := cp2.GenerateChunk(10, 10)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			for y := 0; y < 256; y++ {
				b1 := chunk1.GetBlock(x, y, z)
				b2 := chunk2.GetBlock(x, y, z)
				if b1 != b2 {
					t.Fatalf("Chunks differ at (%d, %d, %d): %v != %v", x, y, z, b1, b2)
				}
			}
		}
	}
}
