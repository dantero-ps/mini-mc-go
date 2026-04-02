package world_test

import (
	"testing"

	"mini-mc/internal/world"
)

// BenchmarkFluidTick measures cascading water simulation throughput,
// reporting flow_dirs/op to detect FluidTick performance regressions.
// Lives in internal/world/ per D-03 (co-located with FluidTick implementation).
// TestMain is provided by chunk_neighbor_bench_test.go — do NOT add one here.
func BenchmarkFluidTick(b *testing.B) {
	b.Run("cascading_column", func(b *testing.B) {
		const colDepth = 8
		b.ReportAllocs()
		var flowDirsEvaluated int
		for b.Loop() {
			b.StopTimer()
			w := world.NewEmpty()
			// Plant source water blocks in a column: y=40..40+colDepth
			for dy := 0; dy <= colDepth; dy++ {
				w.Set(8, 40+dy, 8, world.BlockTypeWater)
				w.SetMeta(8, 40+dy, 8, 0) // 0=source
			}
			b.StartTimer()
			// Tick each position top-down
			for dy := colDepth; dy >= 0; dy-- {
				world.FluidTick(w, 8, 40+dy, 8)
			}
			flowDirsEvaluated = (colDepth + 1) * 4 // 4 horizontal directions checked per block
		}
		b.ReportMetric(float64(flowDirsEvaluated), "flow_dirs/op")
	})
}
