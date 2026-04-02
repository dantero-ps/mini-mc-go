package meshing

import (
	"os"
	"testing"

	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

func TestMain(m *testing.M) {
	if err := os.Chdir("../.."); err != nil {
		panic("cannot chdir to project root: " + err.Error())
	}
	registry.InitRegistry()
	os.Exit(m.Run())
}

// BenchmarkGreedyDirection isolates single-direction greedy meshing
// independent of the worker pool and custom model pass (MESH-01).
func BenchmarkGreedyDirection(b *testing.B) {
	b.Run("+X", func(b *testing.B) {
		const seed = int64(8675309)
		w := world.New()
		defer w.Close()
		c := w.GetChunk(0, 0, 0, true)
		world.NewChunkProvider189(seed).PopulateChunk(c)

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := buildGreedyForDirection(w, c, 1, 0, 0, nil)
			lastVertCount = len(verts) / 2
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})

	b.Run("+Y", func(b *testing.B) {
		const seed = int64(8675309)
		w := world.New()
		defer w.Close()
		c := w.GetChunk(0, 0, 0, true)
		world.NewChunkProvider189(seed).PopulateChunk(c)

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := buildGreedyForDirection(w, c, 0, 1, 0, nil)
			lastVertCount = len(verts) / 2
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})
}

// BenchmarkFluidMesh measures fluid mesh generation for three water-density
// configurations (MESH-02).
func BenchmarkFluidMesh(b *testing.B) {
	b.Run("source_only", func(b *testing.B) {
		w := world.NewEmpty()
		c := w.GetChunk(0, 0, 0, true)
		for x := 0; x < world.ChunkSizeX; x++ {
			for z := 0; z < world.ChunkSizeZ; z++ {
				w.Set(x, 32, z, world.BlockTypeWater)
				w.SetMeta(x, 32, z, 0)
			}
		}

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := BuildFluidMesh(w, c)
			lastVertCount = len(verts) / 10
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})

	b.Run("flowing", func(b *testing.B) {
		w := world.NewEmpty()
		c := w.GetChunk(0, 0, 0, true)
		for x := 0; x < world.ChunkSizeX; x++ {
			for z := 0; z < world.ChunkSizeZ; z++ {
				w.Set(x, 32, z, world.BlockTypeWater)
				w.SetMeta(x, 32, z, 3)
			}
		}

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := BuildFluidMesh(w, c)
			lastVertCount = len(verts) / 10
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})

	b.Run("mixed", func(b *testing.B) {
		w := world.NewEmpty()
		c := w.GetChunk(0, 0, 0, true)
		for x := 0; x < world.ChunkSizeX; x++ {
			for z := 0; z < world.ChunkSizeZ; z++ {
				w.Set(x, 32, z, world.BlockTypeWater)
				if (x+z)%2 == 0 {
					w.SetMeta(x, 32, z, 0) // source
				} else {
					w.SetMeta(x, 32, z, 3) // flowing
				}
			}
		}

		var lastVertCount int

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			verts := BuildFluidMesh(w, c)
			lastVertCount = len(verts) / 10
		}

		b.ReportMetric(float64(lastVertCount), "vertices/op")
	})
}

// BenchmarkCustomModel benchmarks transparent/complex block meshing through
// the non-greedy custom model path (MESH-03).
func BenchmarkCustomModel(b *testing.B) {
	var customBlockType world.BlockType
	var def *registry.BlockDefinition

	for bt, d := range registry.BlockDefs {
		if d == nil {
			continue
		}
		if !d.IsSolid || d.IsTransparent || len(d.Elements) > 1 {
			customBlockType = world.BlockType(bt)
			def = d
			break
		}
	}

	if def == nil {
		b.Skip("no transparent/multi-element block found in registry")
		return
	}

	const seed = int64(8675309)
	w := world.New()
	defer w.Close()
	c := w.GetChunk(0, 0, 0, true)
	world.NewChunkProvider189(seed).PopulateChunk(c)
	w.Set(8, 32, 8, customBlockType)

	var verts []uint32

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		verts = verts[:0]
		meshCustomBlock(&verts, w, c, 8, 32, 8, def)
	}

	b.ReportMetric(float64(len(verts)/2), "vertices/op")
}
