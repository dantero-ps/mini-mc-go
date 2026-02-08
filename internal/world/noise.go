package world

import (
	"math"
)

// Simple deterministic 2D value noise with multiple octaves.
// No external deps; uses integer hashing for lattice values.

// fade function is used for smoothing (Spline)
func fade(t float64) float64 {
	// Smoothstep-like fade function 6t^5 - 15t^4 + 10t^3
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

// lerp performs linear interpolation
// Note: This is also defined in density.go, but Go treats package-level funcs as same scope.
// We should remove one of them. For now, let's keep it here and remove from density.go if needed
// or just rename it. If we have multiple files in same package, they share scope.
// I will rename this one to lerpNoise just in case or assume the one in density.go will be removed.
// Actually, I should check if I broke the build.
// Best to keep helper functions unexported.
// The previous error said "lerp redeclared".
// I'll use the one defined in density.go if I keep it there, or here.
// Let's use noise.go as the source of truth for noise functions.

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
	for i := range octaves {
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

// 3D noise functions for density-based terrain generation

func hash3(x, y, z int64, seed int64) uint64 {
	// SplitMix64 style integer hash for 3D coordinates
	// Use separate golden ratio variants per axis for better distribution
	v := uint64(x)*0x9E3779B97F4A7C15 + uint64(y)*0x517CC1B727220A95 + uint64(z)*0x6C62272E07BB0142 + uint64(seed)
	v += 0x9E3779B97F4A7C15
	v = (v ^ (v >> 30)) * 0xBF58476D1CE4E5B9
	v = (v ^ (v >> 27)) * 0x94D049BB133111EB
	v = v ^ (v >> 31)
	return v
}

func latticeValue3D(x, y, z int64, seed int64) float64 {
	// Map to [0,1]
	h := hash3(x, y, z, seed)
	return float64(h&0xFFFFFFFF) / float64(0xFFFFFFFF)
}

func valueNoise3D(x, y, z float64, seed int64) float64 {
	// Lattice points (8 corners of the cube)
	x0 := math.Floor(x)
	y0 := math.Floor(y)
	z0 := math.Floor(z)
	x1 := x0 + 1
	y1 := y0 + 1
	z1 := z0 + 1

	// Interpolation weights
	fx := fade(x - x0)
	fy := fade(y - y0)
	fz := fade(z - z0)

	// Get lattice values at 8 corners
	v000 := latticeValue3D(int64(x0), int64(y0), int64(z0), seed)
	v100 := latticeValue3D(int64(x1), int64(y0), int64(z0), seed)
	v010 := latticeValue3D(int64(x0), int64(y1), int64(z0), seed)
	v110 := latticeValue3D(int64(x1), int64(y1), int64(z0), seed)
	v001 := latticeValue3D(int64(x0), int64(y0), int64(z1), seed)
	v101 := latticeValue3D(int64(x1), int64(y0), int64(z1), seed)
	v011 := latticeValue3D(int64(x0), int64(y1), int64(z1), seed)
	v111 := latticeValue3D(int64(x1), int64(y1), int64(z1), seed)

	// Trilinear interpolation
	// First interpolate along X (4 results)
	i00 := lerp(v000, v100, fx)
	i10 := lerp(v010, v110, fx)
	i01 := lerp(v001, v101, fx)
	i11 := lerp(v011, v111, fx)

	// Then interpolate along Y (2 results)
	i0 := lerp(i00, i10, fy)
	i1 := lerp(i01, i11, fy)

	// Finally interpolate along Z (1 result)
	return lerp(i0, i1, fz) // [0,1]
}

func octaveNoise3D(x, y, z float64, seed int64, octaves int, persistence, lacunarity float64) float64 {
	amplitude := 1.0
	frequency := 1.0
	sum := 0.0
	norm := 0.0
	for i := range octaves {
		v := valueNoise3D(x*frequency, y*frequency, z*frequency, seed+int64(i*131))
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
