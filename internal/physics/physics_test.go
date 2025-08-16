package physics

import (
	"testing"

	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

func BenchmarkRaycast(b *testing.B) {
	w := world.NewEmpty()
	// Build a simple wall
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			w.Set(x, y, 5, world.BlockTypeGrass)
		}
	}
	start := mgl32.Vec3{0, 8, 0}
	dir := mgl32.Vec3{0, 0, 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Raycast(start, dir, 0.1, 10.0, w)
	}
}
