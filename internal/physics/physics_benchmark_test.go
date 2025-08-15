package physics

import (
	"testing"

	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

func makeWorldForPhysics() *world.World {
	w := world.New()
	w.SetRenderDistance(6)
	w.StreamAround(0, 64, 0)
	return w
}

func BenchmarkCollides(b *testing.B) {
	w := makeWorldForPhysics()
	pos := mgl32.Vec3{0, 70, 0}
	height := float32(1.8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Collides(pos, height, w)
	}
}

func BenchmarkRaycast(b *testing.B) {
	w := makeWorldForPhysics()
	start := mgl32.Vec3{0, 70, 0}
	dir := mgl32.Vec3{1, -0.2, 0}.Normalize()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Raycast(start, dir, MinReachDistance, MaxReachDistance, w)
	}
}
