package world

import (
	"testing"
)

func TestChunkProvider189_GenerateChunk(t *testing.T) {
	seed := int64(12345)
	cp := NewChunkProvider189(seed)

	chunk := NewChunk(0, 0, 0)
	cp.PopulateChunk(chunk)

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
}

func TestChunkProvider189_Determinism(t *testing.T) {
	seed := int64(98765)
	cp1 := NewChunkProvider189(seed)
	cp2 := NewChunkProvider189(seed)

	chunk1 := NewChunk(10, 0, 10)
	chunk2 := NewChunk(10, 0, 10)
	cp1.PopulateChunk(chunk1)
	cp2.PopulateChunk(chunk2)

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
