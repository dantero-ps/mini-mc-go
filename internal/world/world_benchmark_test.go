package world

import (
	"testing"
)

// Benchmark streaming around a fixed point with configurable render distance
func BenchmarkStreamAround(b *testing.B) {
	w := New()
	// Keep RD small to avoid OOM in CI; adjust if needed
	w.SetRenderDistance(6)
	px, py, pz := float32(0), float32(64), float32(0)

	// Warm-up populate once
	w.StreamAround(px, py, pz)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate slight movement to exercise load/unload
		w.StreamAround(px+float32(i%3), py, pz+float32((i/3)%3))
	}
}
