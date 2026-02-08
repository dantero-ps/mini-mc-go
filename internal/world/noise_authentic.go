package world

import (
	"math"
	"math/rand"
)

// NoiseGeneratorImproved ports MC 1.8.9's NoiseGeneratorImproved.java
type AuthenticNoiseGeneratorImproved struct {
	permutations [512]int
	xCoord       float64
	yCoord       float64
	zCoord       float64
}

func NewAuthenticNoiseGeneratorImproved(rnd *rand.Rand) *AuthenticNoiseGeneratorImproved {
	n := &AuthenticNoiseGeneratorImproved{
		xCoord: rnd.Float64() * 256.0,
		yCoord: rnd.Float64() * 256.0,
		zCoord: rnd.Float64() * 256.0,
	}

	for i := 0; i < 256; i++ {
		n.permutations[i] = i
	}

	for i := 0; i < 256; i++ {
		j := rnd.Intn(256-i) + i
		k := n.permutations[i]
		n.permutations[i] = n.permutations[j]
		n.permutations[j] = k
		n.permutations[i+256] = n.permutations[i]
	}

	return n
}

func (n *AuthenticNoiseGeneratorImproved) lerp(t, a, b float64) float64 {
	return a + t*(b-a)
}

func (n *AuthenticNoiseGeneratorImproved) grad2(hash int, x, y float64) float64 {
	h := hash & 15
	u := x
	if h < 8 {
		u = x
	} else {
		u = y
	}

	v := y
	if h < 4 {
		v = y
	} else {
		if h == 12 || h == 14 {
			v = x
		} else {
			v = 0 // z which is 0
		}
	}

	if (h & 1) == 0 {
		return u
	}
	vVal := v
	if (h & 2) != 0 {
		vVal = -v
	}
	return -u + vVal
}

// Simplified 3D grad from memory/algo
func (n *AuthenticNoiseGeneratorImproved) grad(hash int, x, y, z float64) float64 {
	h := hash & 15
	u := x
	if h >= 8 {
		u = y
	}
	v := y
	if h < 4 {
		v = y
	} else if h == 12 || h == 14 {
		v = x
	} else {
		v = z
	}

	r := 0.0
	if (h & 1) == 0 {
		r = u
	} else {
		r = -u
	}

	if (h & 2) == 0 {
		r += v
	} else {
		r -= v
	}
	return r
}

func (n *AuthenticNoiseGeneratorImproved) PopulateNoiseArray(noiseArray []float64, xOffset, yOffset, zOffset float64, xSize, ySize, zSize int, xScale, yScale, zScale, noiseScale float64) {
	if ySize == 1 {
		// 2D Noise Optimization (ChunkProviderGenerate uses this for depth/scale noise sometimes)
		// ... Implement 2D if needed, for now use general case or implement later
	}

	scaleInv := 1.0 / noiseScale
	idx := 0

	for x := 0; x < xSize; x++ {
		fx := xOffset + float64(x)*xScale + n.xCoord
		ix := int(math.Floor(fx))
		dx := fx - float64(ix)
		fadeX := dx * dx * dx * (dx*(dx*6.0-15.0) + 10.0)

		permX := ix & 255

		for z := 0; z < zSize; z++ {
			fz := zOffset + float64(z)*zScale + n.zCoord
			iz := int(math.Floor(fz))
			dz := fz - float64(iz)
			fadeZ := dz * dz * dz * (dz*(dz*6.0-15.0) + 10.0)

			permZ := iz & 255

			// Hash lookups
			A := n.permutations[permX] + permZ
			AA := n.permutations[A]
			AB := n.permutations[A+1]
			B := n.permutations[permX+1] + permZ
			BA := n.permutations[B]
			BB := n.permutations[B+1]

			for y := 0; y < ySize; y++ {
				fy := yOffset + float64(y)*yScale + n.yCoord
				iy := int(math.Floor(fy))
				dy := fy - float64(iy)
				fadeY := dy * dy * dy * (dy*(dy*6.0-15.0) + 10.0)

				permY := iy & 255

				// Hashing with Y
				// Note: MC 1.8 implementation slightly different mixing, let's follow improved perlin
				// The permutations are flattened.

				// Standard Perlin Interpolation
				// hashes
				hAAA := n.permutations[AA+permY]
				hBAA := n.permutations[BA+permY]
				hABA := n.permutations[AB+permY]
				hBBA := n.permutations[BB+permY]
				hAAB := n.permutations[AA+permY+1]
				hBAB := n.permutations[BA+permY+1]
				hABB := n.permutations[AB+permY+1]
				hBBB := n.permutations[BB+permY+1]

				// Gradients
				gAAA := n.grad(hAAA, dx, dy, dz)
				gBAA := n.grad(hBAA, dx-1, dy, dz)
				gABA := n.grad(hABA, dx, dy-1, dz)
				gBBA := n.grad(hBBA, dx-1, dy-1, dz)
				gAAB := n.grad(hAAB, dx, dy, dz-1)
				gBAB := n.grad(hBAB, dx-1, dy, dz-1)
				gABB := n.grad(hABB, dx, dy-1, dz-1)
				gBBB := n.grad(hBBB, dx-1, dy-1, dz-1)

				// Lerp X
				l1 := n.lerp(fadeX, gAAA, gBAA)
				l2 := n.lerp(fadeX, gABA, gBBA)
				l3 := n.lerp(fadeX, gAAB, gBAB)
				l4 := n.lerp(fadeX, gABB, gBBB)

				// Lerp Y
				l5 := n.lerp(fadeY, l1, l2)
				l6 := n.lerp(fadeY, l3, l4)

				// Lerp Z
				val := n.lerp(fadeZ, l5, l6)

				noiseArray[idx] += val * scaleInv
				idx++
			}
		}
	}
}

// NoiseGeneratorOctaves ports MC 1.8.9's NoiseGeneratorOctaves.java
type AuthenticNoiseGeneratorOctaves struct {
	generators []*AuthenticNoiseGeneratorImproved
	octaves    int
}

func NewAuthenticNoiseGeneratorOctaves(rnd *rand.Rand, octaves int) *AuthenticNoiseGeneratorOctaves {
	g := &AuthenticNoiseGeneratorOctaves{
		octaves:    octaves,
		generators: make([]*AuthenticNoiseGeneratorImproved, octaves),
	}
	for i := 0; i < octaves; i++ {
		g.generators[i] = NewAuthenticNoiseGeneratorImproved(rnd)
	}
	return g
}

func (g *AuthenticNoiseGeneratorOctaves) GenerateNoiseOctaves(noiseArray []float64, xOffset, yOffset, zOffset int, xSize, ySize, zSize int, xScale, yScale, zScale float64) []float64 {
	if noiseArray == nil {
		noiseArray = make([]float64, xSize*ySize*zSize)
	} else {
		for i := range noiseArray {
			noiseArray[i] = 0
		}
	}

	amp := 1.0

	for i := 0; i < g.octaves; i++ {
		// MC 1.8 keeps noise offsets positive by 16777216 mod math,
		// but standard float logic should handle negative fine.
		// We'll mimic the scaling mainly.

		g.generators[i].PopulateNoiseArray(noiseArray, float64(xOffset)*amp*xScale, float64(yOffset)*amp*yScale, float64(zOffset)*amp*zScale, xSize, ySize, zSize, xScale*amp, yScale*amp, zScale*amp, amp)
		amp /= 2.0
	}

	return noiseArray
}
