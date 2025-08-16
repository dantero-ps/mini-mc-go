package meshing

import (
	"testing"

	"mini-mc/internal/world"
)

func TestSingleBlockMesh(t *testing.T) {
	w := world.NewEmpty()
	// Place single block
	w.Set(0, 0, 0, world.BlockTypeGrass)
	ch := w.GetChunkFromBlockCoords(0, 0, 0, false)
	verts := BuildGreedyMeshForChunk(w, ch)
	expectedFloats := 36 * 6 // 12 triangles * 3 verts * 6 floats
	if len(verts) != expectedFloats {
		t.Fatalf("single block: got %d floats, want %d", len(verts), expectedFloats)
	}
}

func TestTwoBlocksSeparated(t *testing.T) {
	w := world.NewEmpty()
	// Two blocks with a gap (non-touching)
	w.Set(0, 0, 0, world.BlockTypeGrass)
	w.Set(2, 0, 0, world.BlockTypeGrass)
	ch := w.GetChunkFromBlockCoords(0, 0, 0, false)
	verts := BuildGreedyMeshForChunk(w, ch)
	expectedFloats := 72 * 6 // 24 triangles * 3 verts * 6 floats
	if len(verts) != expectedFloats {
		t.Fatalf("two separated blocks: got %d floats, want %d", len(verts), expectedFloats)
	}
}

func TestTwoBlocksTouchingGreedy(t *testing.T) {
	w := world.NewEmpty()
	// Two adjacent blocks along X
	w.Set(0, 0, 0, world.BlockTypeGrass)
	w.Set(1, 0, 0, world.BlockTypeGrass)
	ch := w.GetChunkFromBlockCoords(0, 0, 0, false)
	verts := BuildGreedyMeshForChunk(w, ch)
	// Union is a 2x1x1 cuboid => 12 triangles
	expectedFloats := 36 * 6
	if len(verts) != expectedFloats {
		t.Fatalf("two touching blocks (greedy merge): got %d floats, want %d", len(verts), expectedFloats)
	}
}

func TestCrossChunkFaceCulling(t *testing.T) {
	w := world.NewEmpty()
	// Place one block at the +X edge of chunk (15,*,*) and neighbor in next chunk at x=16
	w.Set(world.ChunkSizeX-1, 0, 0, world.BlockTypeGrass) // local x=15 in chunk (0,0,0)
	w.Set(world.ChunkSizeX, 0, 0, world.BlockTypeGrass)   // neighbor chunk (1,0,0)
	ch := w.GetChunk(0, 0, 0, false)
	verts := BuildGreedyMeshForChunk(w, ch)
	// One face hidden due to neighbor => 10 triangles = 30 verts
	expectedFloats := 30 * 6
	if len(verts) != expectedFloats {
		t.Fatalf("cross-chunk culling: got %d floats, want %d", len(verts), expectedFloats)
	}
}

func BenchmarkBuildGreedyMeshForChunk_FullSurface(b *testing.B) {
	w := world.NewEmpty()
	ch := world.NewChunk(0, 0, 0)
	// Fill a full top surface
	for x := 0; x < world.ChunkSizeX; x++ {
		for z := 0; z < world.ChunkSizeZ; z++ {
			ch.SetBlock(x, world.ChunkSizeY-1, z, world.BlockTypeGrass)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGreedyMeshForChunk(w, ch)
	}
}
