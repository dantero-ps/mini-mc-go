package world_test

import (
	"os"
	"testing"

	"mini-mc/internal/meshing"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// TestMain initialises the block registry and sets the working directory to
// the project root so that benchmarks requiring registry lookups (meshing)
// can resolve assets correctly. This runs for all tests in the external test
// package (world_test) as well as the internal test package (world) since Go
// compiles them into a single test binary.
func TestMain(m *testing.M) {
	if err := os.Chdir("../.."); err != nil {
		panic("cannot chdir to project root: " + err.Error())
	}
	registry.InitRegistry()
	os.Exit(m.Run())
}

// BenchmarkChunkNeighborMeshing measures cross-border meshing overhead
// comparing a chunk with 0 neighbours vs a chunk with all 6 neighbour slots
// (4 populated horizontal + 2 nil vertical).
func BenchmarkChunkNeighborMeshing(b *testing.B) {
	const seed = int64(8675309)

	b.Run("NoNeighbors", func(b *testing.B) {
		w := world.New()
		defer w.Close()

		c := w.GetChunk(0, 0, 0, true)
		provider := world.NewChunkProvider189(seed)
		provider.PopulateChunk(c)

		pool := meshing.NewDirectionWorkerPool(6, 32)
		pool.Start()

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := meshing.BuildGreedyMeshForChunk(w, c, pool)
			lastVertCount = len(verts) / 2
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})

	b.Run("SixNeighbors", func(b *testing.B) {
		w := world.New()
		defer w.Close()

		provider := world.NewChunkProvider189(seed)

		// Create a 3×1×3 grid of chunks centered at (0,0,0).
		// This gives the center chunk 4 populated horizontal neighbours (±X, ±Z).
		// +Y/-Y neighbours are nil (Y=0 is ground level) — the code handles nil gracefully.
		for x := -1; x <= 1; x++ {
			for z := -1; z <= 1; z++ {
				c := w.GetChunk(x, 0, z, true)
				provider.PopulateChunk(c)
			}
		}

		center := w.GetChunk(0, 0, 0, false)
		if center == nil {
			b.Fatal("center chunk not found after world setup")
		}

		pool := meshing.NewDirectionWorkerPool(6, 32)
		pool.Start()

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := meshing.BuildGreedyMeshForChunk(w, center, pool)
			lastVertCount = len(verts) / 2
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})
}
