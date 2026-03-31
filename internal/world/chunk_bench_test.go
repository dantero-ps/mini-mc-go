package world

// BenchmarkChunk measures GetBlock/SetBlock throughput and section-aware
// iteration with IsSectionEmpty fast-path validation.
//
// Note: populateChunk and BenchmarkChunk live in package world (internal test)
// because they only use world-internal types and do not need the registry or
// meshing packages. The meshing-dependent BenchmarkChunkNeighborMeshing lives
// in chunk_neighbor_bench_test.go (package world_test) to avoid import cycles.

import (
	"testing"
)

// populateChunk fills a chunk with BlockTypeStone at the given density.
// density 1.0 = all blocks filled; 0.5 = every other block (checkerboard).
func populateChunk(c *Chunk, density float64) {
	for x := range ChunkSizeX {
		for y := range ChunkSizeY {
			for z := range ChunkSizeZ {
				if density == 1.0 || (density < 1.0 && (x+y+z)%2 == 0) {
					c.SetBlockFast(x, y, z, BlockTypeStone)
				}
			}
		}
	}
}

func BenchmarkChunk(b *testing.B) {
	b.Run("GetBlock/all_sections_full", func(b *testing.B) {
		c := NewChunk(0, 0, 0)
		populateChunk(c, 1.0)

		// Fixed read pattern: every 4th block in x/z, all y layers.
		// 16/4 * 256 * 16/4 = 4 * 256 * 4 = 4096 reads per iteration.
		const totalReads = ChunkSizeX / 4 * ChunkSizeY * ChunkSizeZ / 4

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for x := 0; x < ChunkSizeX; x += 4 {
				for y := range ChunkSizeY {
					for z := 0; z < ChunkSizeZ; z += 4 {
						_ = c.GetBlock(x, y, z)
					}
				}
			}
		}

		b.ReportMetric(float64(totalReads), "blocks/op")
	})

	b.Run("SetBlock/all_sections_full", func(b *testing.B) {
		// Pre-allocate a template chunk outside the loop so we can cheaply
		// reset state after each write iteration.
		template := NewChunk(0, 0, 0)
		populateChunk(template, 1.0)

		// Write pattern: alternate between Dirt and Air for every other block.
		// Exercises the O(4096) zero-scan on block removal path.
		const totalWrites = ChunkSizeX * ChunkSizeY * ChunkSizeZ / 2

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			c := NewChunk(0, 0, 0)
			// Copy template sections into fresh chunk for a fair starting state.
			for secIdx := range NumSections {
				srcSec := template.sections[secIdx]
				if srcSec != nil && srcSec.basePtr != nil {
					c.sections[secIdx] = srcSec
				}
			}

			for x := range ChunkSizeX {
				for y := range ChunkSizeY {
					for z := range ChunkSizeZ {
						if (x+y+z)%2 == 0 {
							c.SetBlock(x, y, z, BlockTypeDirt)
						} else {
							c.SetBlock(x, y, z, BlockTypeAir)
						}
					}
				}
			}
		}

		b.ReportMetric(float64(totalWrites), "blocks/op")
	})

	b.Run("GetBlock/mostly_empty", func(b *testing.B) {
		c := NewChunk(0, 0, 0)
		// Populate ONLY section 4 (y=64-79) with stone.
		for x := range ChunkSizeX {
			for y := 64; y < 80; y++ {
				for z := range ChunkSizeZ {
					c.SetBlockFast(x, y, z, BlockTypeStone)
				}
			}
		}

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			for secIdx := range NumSections {
				_ = c.IsSectionEmpty(secIdx)
			}
		}

		b.ReportMetric(float64(NumSections), "sections_scanned/op")
	})
}
