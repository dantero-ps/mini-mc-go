package world

import (
	"math"
	"math/rand"
)

// Gradient lookup tables from MC 1.8.9 NoiseGeneratorImproved.java
var (
	gradX = [16]float64{1, -1, 1, -1, 1, -1, 1, -1, 0, 0, 0, 0, 1, 0, -1, 0}
	gradY = [16]float64{1, 1, -1, -1, 0, 0, 0, 0, 1, -1, 1, -1, 1, -1, 1, -1}
	gradZ = [16]float64{0, 0, 0, 0, 1, 1, -1, -1, 1, 1, -1, -1, 0, 1, 0, -1}
	// 2D gradient tables (used when ySize==1)
	grad2X = [16]float64{1, -1, 1, -1, 1, -1, 1, -1, 0, 0, 0, 0, 1, 0, -1, 0}
	grad2Z = [16]float64{0, 0, 0, 0, 1, 1, -1, -1, 1, 1, -1, -1, 0, 1, 0, -1}
)

// AuthenticNoiseGeneratorImproved ports MC 1.8.9's NoiseGeneratorImproved.java exactly.
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
		n.permutations[i], n.permutations[j] = n.permutations[j], n.permutations[i]
		n.permutations[i+256] = n.permutations[i]
	}

	return n
}

func (n *AuthenticNoiseGeneratorImproved) lerpN(t, a, b float64) float64 {
	return a + t*(b-a)
}

func (n *AuthenticNoiseGeneratorImproved) grad3d(hash int, x, y, z float64) float64 {
	i := hash & 15
	return gradX[i]*x + gradY[i]*y + gradZ[i]*z
}

func (n *AuthenticNoiseGeneratorImproved) grad2d(hash int, x, z float64) float64 {
	i := hash & 15
	return grad2X[i]*x + grad2Z[i]*z
}

// floorToInt matches MC's (int)d behavior: floor then cast
func floorToInt(d float64) int {
	i := int(d)
	if d < float64(i) {
		i--
	}
	return i
}

// PopulateNoiseArray is a 1:1 port of MC 1.8.9 NoiseGeneratorImproved.populateNoiseArray.
// Hash order: X→Y→Z (MC uses permutations[permX]+permY, then +permZ).
func (n *AuthenticNoiseGeneratorImproved) PopulateNoiseArray(
	noiseArray []float64,
	xOffset, yOffset, zOffset float64,
	xSize, ySize, zSize int,
	xScale, yScale, zScale, noiseScale float64,
) {
	scaleInv := 1.0 / noiseScale

	if ySize == 1 {
		// 2D mode (used for depth noise)
		idx := 0
		for ix := 0; ix < xSize; ix++ {
			fx := xOffset + float64(ix)*xScale + n.xCoord
			flx := floorToInt(fx)
			permX := flx & 255
			fx -= float64(flx)
			fadeX := fx * fx * fx * (fx*(fx*6.0-15.0) + 10.0)

			for iz := 0; iz < zSize; iz++ {
				fz := zOffset + float64(iz)*zScale + n.zCoord
				flz := floorToInt(fz)
				permZ := flz & 255
				fz -= float64(flz)
				fadeZ := fz * fz * fz * (fz*(fz*6.0-15.0) + 10.0)

				// MC 2D hash: perm[permX]+0, then +permZ
				i5 := n.permutations[permX] + 0
				j5 := n.permutations[i5] + permZ
				j := n.permutations[permX+1] + 0
				k5 := n.permutations[j] + permZ

				d14 := n.lerpN(fadeX, n.grad2d(n.permutations[j5], fx, fz), n.grad3d(n.permutations[k5], fx-1.0, 0.0, fz))
				d15 := n.lerpN(fadeX, n.grad3d(n.permutations[j5+1], fx, 0.0, fz-1.0), n.grad3d(n.permutations[k5+1], fx-1.0, 0.0, fz-1.0))
				val := n.lerpN(fadeZ, d14, d15)

				noiseArray[idx] += val * scaleInv
				idx++
			}
		}
		return
	}

	// 3D mode
	idx := 0
	prevPermY := -1

	var d1, d2, d3, d4 float64
	var l, i1, j1, k1, l1, i2 int

	for ix := 0; ix < xSize; ix++ {
		fx := xOffset + float64(ix)*xScale + n.xCoord
		flx := floorToInt(fx)
		permX := flx & 255
		fx -= float64(flx)
		fadeX := fx * fx * fx * (fx*(fx*6.0-15.0) + 10.0)

		for iz := 0; iz < zSize; iz++ {
			fz := zOffset + float64(iz)*zScale + n.zCoord
			flz := floorToInt(fz)
			permZ := flz & 255
			fz -= float64(flz)
			fadeZ := fz * fz * fz * (fz*(fz*6.0-15.0) + 10.0)

			for iy := 0; iy < ySize; iy++ {
				fy := yOffset + float64(iy)*yScale + n.yCoord
				fly := floorToInt(fy)
				permY := fly & 255
				fy -= float64(fly)
				fadeY := fy * fy * fy * (fy*(fy*6.0-15.0) + 10.0)

				// MC optimization: only recompute hashes when permY changes
				if iy == 0 || permY != prevPermY {
					prevPermY = permY

					// Hash order: X → Y → Z (MC's exact order)
					l = n.permutations[permX] + permY
					i1 = n.permutations[l] + permZ
					j1 = n.permutations[l+1] + permZ
					k1 = n.permutations[permX+1] + permY
					l1 = n.permutations[k1] + permZ
					i2 = n.permutations[k1+1] + permZ

					// Lerp along X for all 4 edges (recomputed when Y changes)
					d1 = n.lerpN(fadeX,
						n.grad3d(n.permutations[i1], fx, fy, fz),
						n.grad3d(n.permutations[l1], fx-1.0, fy, fz))
					d2 = n.lerpN(fadeX,
						n.grad3d(n.permutations[j1], fx, fy-1.0, fz),
						n.grad3d(n.permutations[i2], fx-1.0, fy-1.0, fz))
					d3 = n.lerpN(fadeX,
						n.grad3d(n.permutations[i1+1], fx, fy, fz-1.0),
						n.grad3d(n.permutations[l1+1], fx-1.0, fy, fz-1.0))
					d4 = n.lerpN(fadeX,
						n.grad3d(n.permutations[j1+1], fx, fy-1.0, fz-1.0),
						n.grad3d(n.permutations[i2+1], fx-1.0, fy-1.0, fz-1.0))
				}

				// Lerp Y then Z
				d11 := n.lerpN(fadeY, d1, d2)
				d12 := n.lerpN(fadeY, d3, d4)
				val := n.lerpN(fadeZ, d11, d12)

				noiseArray[idx] += val * scaleInv
				idx++
			}
		}
	}
}

// AuthenticNoiseGeneratorOctaves ports MC 1.8.9's NoiseGeneratorOctaves.java
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

// GenerateNoiseOctaves is a 1:1 port of MC's NoiseGeneratorOctaves.generateNoiseOctaves (3D variant).
// Includes the modulo 16777216 wrap for float precision.
func (g *AuthenticNoiseGeneratorOctaves) GenerateNoiseOctaves(
	noiseArray []float64,
	xOffset, yOffset, zOffset int,
	xSize, ySize, zSize int,
	xScale, yScale, zScale float64,
) []float64 {
	if noiseArray == nil {
		noiseArray = make([]float64, xSize*ySize*zSize)
	} else {
		for i := range noiseArray {
			noiseArray[i] = 0
		}
	}

	d3 := 1.0

	for j := 0; j < g.octaves; j++ {
		d0 := float64(xOffset) * d3 * xScale
		d1 := float64(yOffset) * d3 * yScale
		d2 := float64(zOffset) * d3 * zScale

		// MC wraps large coords to avoid floating-point precision loss
		k := int64(math.Floor(d0))
		l := int64(math.Floor(d2))
		d0 -= float64(k)
		d2 -= float64(l)
		k %= 16777216
		l %= 16777216
		d0 += float64(k)
		d2 += float64(l)

		g.generators[j].PopulateNoiseArray(
			noiseArray,
			d0, d1, d2,
			xSize, ySize, zSize,
			xScale*d3, yScale*d3, zScale*d3,
			d3,
		)
		d3 /= 2.0
	}

	return noiseArray
}

// GenerateNoiseOctaves2D is the 2D bouncer (used for depth noise in MC).
// MC: generateNoiseOctaves(array, xOffset, zOffset, xSize, zSize, xScale, zScale, exponent)
// It calls the 3D variant with yOffset=10, ySize=1, yScale=1.0.
func (g *AuthenticNoiseGeneratorOctaves) GenerateNoiseOctaves2D(
	noiseArray []float64,
	xOffset, zOffset int,
	xSize, zSize int,
	xScale, zScale, exponent float64,
) []float64 {
	return g.GenerateNoiseOctaves(noiseArray, xOffset, 10, zOffset, xSize, 1, zSize, xScale, 1.0, zScale)
}
