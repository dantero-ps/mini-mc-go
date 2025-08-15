package meshing

import (
	"testing"

	"mini-mc/internal/world"
)

func makeChunk(w *world.World) *world.Chunk {
	// Ensure chunks exist around origin
	w.SetRenderDistance(2)
	w.StreamAround(0, 64, 0)
	// Pick central chunk
	return w.GetChunk(0, 0, 0, true)
}

func BenchmarkBuildGreedyMeshForChunk(b *testing.B) {
	w := world.New()
	ch := makeChunk(w)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGreedyMeshForChunk(w, ch)
	}
}
