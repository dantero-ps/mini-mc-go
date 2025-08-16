package world

import "testing"

func BenchmarkPopulateChunk(b *testing.B) {
	w := NewEmpty()
	ch := NewChunk(0, 0, 0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.populateChunk(ch)
	}
}

func BenchmarkHeightAt(b *testing.B) {
	w := NewEmpty()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.SurfaceHeightAt(i%1024, (i*31)%1024)
	}
}
