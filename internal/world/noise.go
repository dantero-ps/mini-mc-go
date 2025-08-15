package world

import "math"

// Simple deterministic 2D value noise with multiple octaves.
// No external deps; uses integer hashing for lattice values.

func fade(t float64) float64 {
	// Smoothstep-like fade function 6t^5 - 15t^4 + 10t^3
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func hash2(x int64, z int64, seed int64) uint64 {
	// SplitMix64 style integer hash, stable across runs for same inputs
	v := uint64(x) + (uint64(z) << 1) + uint64(seed)*0x9E3779B97F4A7C15
	v += 0x9E3779B97F4A7C15
	v = (v ^ (v >> 30)) * 0xBF58476D1CE4E5B9
	v = (v ^ (v >> 27)) * 0x94D049BB133111EB
	v = v ^ (v >> 31)
	return v
}

func latticeValue(x int64, z int64, seed int64) float64 {
	// Map to [0,1]
	h := hash2(x, z, seed)
	return float64(h&0xFFFFFFFF) / float64(0xFFFFFFFF)
}

func valueNoise2D(x float64, z float64, seed int64) float64 {
	// Lattice points
	x0 := math.Floor(x)
	z0 := math.Floor(z)
	x1 := x0 + 1
	z1 := z0 + 1

	// Interpolation weights
	fx := fade(x - x0)
	fz := fade(z - z0)

	v00 := latticeValue(int64(x0), int64(z0), seed)
	v10 := latticeValue(int64(x1), int64(z0), seed)
	v01 := latticeValue(int64(x0), int64(z1), seed)
	v11 := latticeValue(int64(x1), int64(z1), seed)

	i0 := lerp(v00, v10, fx)
	i1 := lerp(v01, v11, fx)
	return lerp(i0, i1, fz) // [0,1]
}

func octaveNoise2D(x float64, z float64, seed int64, octaves int, persistence, lacunarity float64) float64 {
	amplitude := 1.0
	frequency := 1.0
	sum := 0.0
	norm := 0.0
	for i := 0; i < octaves; i++ {
		v := valueNoise2D(x*frequency, z*frequency, seed+int64(i*131))
		sum += v * amplitude
		norm += amplitude
		amplitude *= persistence
		frequency *= lacunarity
	}
	if norm == 0 {
		return 0
	}
	return sum / norm // [0,1]
}
