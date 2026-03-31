package world

import (
	"math"
	"math/rand"
	"testing"
)

// TestHash3Deterministic verifies hash3 produces identical results for same inputs
func TestHash3Deterministic(t *testing.T) {
	var results [100]uint64
	for i := range results {
		results[i] = hash3(10, 20, 30, 42)
	}

	// All results must be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		if results[i] != first {
			t.Errorf("hash3 not deterministic: results[0]=%d, results[%d]=%d", first, i, results[i])
		}
	}
}

// TestHash3DifferentInputs verifies hash3 produces different values for different inputs
func TestHash3DifferentInputs(t *testing.T) {
	seed := int64(42)

	// Different X
	h1 := hash3(1, 0, 0, seed)
	h2 := hash3(2, 0, 0, seed)
	if h1 == h2 {
		t.Errorf("hash3 should differ for different X: hash3(1,0,0,seed)=%d == hash3(2,0,0,seed)=%d", h1, h2)
	}

	// Different Y
	h1 = hash3(0, 1, 0, seed)
	h2 = hash3(0, 2, 0, seed)
	if h1 == h2 {
		t.Errorf("hash3 should differ for different Y: hash3(0,1,0,seed)=%d == hash3(0,2,0,seed)=%d", h1, h2)
	}

	// Different Z
	h1 = hash3(0, 0, 1, seed)
	h2 = hash3(0, 0, 2, seed)
	if h1 == h2 {
		t.Errorf("hash3 should differ for different Z: hash3(0,0,1,seed)=%d == hash3(0,0,2,seed)=%d", h1, h2)
	}

	// Different seed
	h1 = hash3(1, 1, 1, 100)
	h2 = hash3(1, 1, 1, 200)
	if h1 == h2 {
		t.Errorf("hash3 should differ for different seed: hash3(1,1,1,100)=%d == hash3(1,1,1,200)=%d", h1, h2)
	}

	// Axis swap (ensures axes aren't interchangeable)
	h1 = hash3(1, 2, 3, seed)
	h2 = hash3(3, 2, 1, seed)
	if h1 == h2 {
		t.Errorf("hash3 should differ for axis swap: hash3(1,2,3,seed)=%d == hash3(3,2,1,seed)=%d", h1, h2)
	}
}

// TestValueNoise3DRange verifies valueNoise3D outputs are in [0,1]
func TestValueNoise3DRange(t *testing.T) {
	rng := rand.New(rand.NewSource(12345)) // deterministic test RNG
	seed := int64(42)

	for i := 0; i < 1000; i++ {
		x := rng.Float64()*200 - 100 // [-100, 100]
		y := rng.Float64()*200 - 100
		z := rng.Float64()*200 - 100

		v := valueNoise3D(x, y, z, seed)

		if v < 0.0 || v > 1.0 {
			t.Errorf("valueNoise3D(%f, %f, %f, %d) = %f, expected in [0,1]", x, y, z, seed, v)
		}
	}
}

// TestValueNoise3DDeterministic verifies valueNoise3D produces identical results
func TestValueNoise3DDeterministic(t *testing.T) {
	var results [100]float64
	for i := range results {
		results[i] = valueNoise3D(1.5, 2.7, 3.3, 42)
	}

	// All results must be identical (exact float64 match)
	first := results[0]
	for i := 1; i < len(results); i++ {
		if results[i] != first {
			t.Errorf("valueNoise3D not deterministic: results[0]=%f, results[%d]=%f", first, i, results[i])
		}
	}
}

// TestValueNoise3DContinuity verifies smooth interpolation (no random jumps)
func TestValueNoise3DContinuity(t *testing.T) {
	seed := int64(42)

	// Sample at two nearby points
	v1 := valueNoise3D(1.0, 1.0, 1.0, seed)
	v2 := valueNoise3D(1.01, 1.0, 1.0, seed)

	diff := math.Abs(v1 - v2)

	// Difference should be small (< 0.1 for 0.01 distance)
	if diff >= 0.1 {
		t.Errorf("valueNoise3D not continuous: valueNoise3D(1.0,1.0,1.0)=%f, valueNoise3D(1.01,1.0,1.0)=%f, diff=%f >= 0.1",
			v1, v2, diff)
	}
}

// TestOctaveNoise3DRange verifies octaveNoise3D outputs are in [0,1]
func TestOctaveNoise3DRange(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	seed := int64(42)
	octaves := 4

	for i := 0; i < 1000; i++ {
		x := rng.Float64()*200 - 100
		y := rng.Float64()*200 - 100
		z := rng.Float64()*200 - 100

		v := octaveNoise3D(x, y, z, seed, octaves, 0.5, 2.0)

		if v < 0.0 || v > 1.0 {
			t.Errorf("octaveNoise3D(%f, %f, %f, %d, %d, 0.5, 2.0) = %f, expected in [0,1]",
				x, y, z, seed, octaves, v)
		}
	}
}

// TestOctaveNoise3DDeterministic verifies octaveNoise3D produces identical results
func TestOctaveNoise3DDeterministic(t *testing.T) {
	var results [100]float64
	for i := range results {
		results[i] = octaveNoise3D(1.5, 2.7, 3.3, 42, 4, 0.5, 2.0)
	}

	// All results must be identical
	first := results[0]
	for i := 1; i < len(results); i++ {
		if results[i] != first {
			t.Errorf("octaveNoise3D not deterministic: results[0]=%f, results[%d]=%f", first, i, results[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: NOISE-01, NOISE-02, NOISE-03
// ---------------------------------------------------------------------------

// BenchmarkValueNoise measures per-sample cost of value noise at different
// coordinate ranges and dimensions (implements NOISE-01).
func BenchmarkValueNoise(b *testing.B) {
	const seed = int64(8675309)

	b.Run("2D/origin", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			valueNoise2D(float64(i%64), float64(i/64%64), seed)
			i++
		}
		b.ReportMetric(1.0, "samples/op")
	})

	b.Run("2D/mid_range", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			valueNoise2D(100.5+float64(i%64), 200.3+float64(i/64%64), seed)
			i++
		}
		b.ReportMetric(1.0, "samples/op")
	})

	b.Run("3D/origin", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			valueNoise3D(float64(i%16), float64(i/16%16), float64(i/256%16), seed)
			i++
		}
		b.ReportMetric(1.0, "samples/op")
	})

	b.Run("3D/mid_range", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			valueNoise3D(100.5+float64(i%16), 64.0+float64(i/16%16), 200.3+float64(i/256%16), seed)
			i++
		}
		b.ReportMetric(1.0, "samples/op")
	})
}

// BenchmarkOctaveNoise measures per-sample cost of octave noise with varying
// octave counts in 2D and 3D (implements NOISE-02).
func BenchmarkOctaveNoise(b *testing.B) {
	const seed = int64(8675309)

	b.Run("2D", func(b *testing.B) {
		for _, oc := range []struct {
			name    string
			octaves int
		}{{"2", 2}, {"4", 4}, {"6", 6}} {
			b.Run(oc.name, func(b *testing.B) {
				b.ReportAllocs()
				i := 0
				b.ResetTimer()
				for b.Loop() {
					octaveNoise2D(50.5+float64(i%64), 50.5+float64(i/64%64), seed, oc.octaves, 0.5, 2.0)
					i++
				}
				b.ReportMetric(1.0, "samples/op")
			})
		}
	})

	b.Run("3D", func(b *testing.B) {
		for _, oc := range []struct {
			name    string
			octaves int
		}{{"2", 2}, {"4", 4}, {"6", 6}} {
			b.Run(oc.name, func(b *testing.B) {
				b.ReportAllocs()
				i := 0
				b.ResetTimer()
				for b.Loop() {
					octaveNoise3D(50.5+float64(i%16), 30.0, 50.5+float64(i/16%16), seed, oc.octaves, 0.5, 2.0)
					i++
				}
				b.ReportMetric(1.0, "samples/op")
			})
		}
	})
}

// BenchmarkNoiseArray measures the cost of authentic MC noise array generation
// with PopulateNoiseArray and GenerateNoiseOctaves (implements NOISE-03).
func BenchmarkNoiseArray(b *testing.B) {
	const (
		xSize   = 5
		ySize   = 33
		zSize   = 5
		samples = xSize * ySize * zSize // 825
	)

	b.Run("PopulateNoiseArray", func(b *testing.B) {
		rnd := rand.New(rand.NewSource(8675309))
		gen := NewAuthenticNoiseGeneratorImproved(rnd)
		noiseArray := make([]float64, samples)

		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			for j := range noiseArray {
				noiseArray[j] = 0
			}
			gen.PopulateNoiseArray(
				noiseArray,
				float64(i%64), 0.0, float64(i/64%64),
				xSize, ySize, zSize,
				0.0625, 0.0625, 0.0625, 1.0,
			)
			i++
		}
		b.ReportMetric(float64(samples), "samples/op")
	})

	b.Run("GenerateNoiseOctaves", func(b *testing.B) {
		rnd := rand.New(rand.NewSource(8675309))
		gen := NewAuthenticNoiseGeneratorOctaves(rnd, 4)

		b.ReportAllocs()
		i := 0
		b.ResetTimer()
		for b.Loop() {
			gen.GenerateNoiseOctaves(
				nil,
				i%64, 0, i/64%64,
				xSize, ySize, zSize,
				684.412, 684.412, 684.412,
			)
			i++
		}
		b.ReportMetric(float64(samples), "samples/op")
	})
}
