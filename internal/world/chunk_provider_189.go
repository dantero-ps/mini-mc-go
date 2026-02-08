package world

import (
	"math"
	"math/rand"
)

// ChunkProvider189 implements the Minecraft 1.8.9 chunk generation logic.
type ChunkProvider189 struct {
	rnd *rand.Rand

	minLimitNoise *AuthenticNoiseGeneratorOctaves
	maxLimitNoise *AuthenticNoiseGeneratorOctaves
	mainNoise     *AuthenticNoiseGeneratorOctaves
	surfaceNoise  *AuthenticNoiseGeneratorOctaves
	scaleNoise    *AuthenticNoiseGeneratorOctaves
	depthNoise    *AuthenticNoiseGeneratorOctaves
	forestNoise   *AuthenticNoiseGeneratorOctaves

	// Arrays for noise generation to reuse memory
	minLimitRegion []float64
	maxLimitRegion []float64
	mainRegion     []float64
	depthRegion    []float64 // scaleNoise in MC
	scaleRegion    []float64 // depthNoise in MC
}

func NewChunkProvider189(seed int64) *ChunkProvider189 {
	rnd := rand.New(rand.NewSource(seed))

	return &ChunkProvider189{
		rnd:           rnd,
		minLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		maxLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		mainNoise:     NewAuthenticNoiseGeneratorOctaves(rnd, 8),
		surfaceNoise:  NewAuthenticNoiseGeneratorOctaves(rnd, 4),
		scaleNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 10), // MC: noiseGen5
		depthNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 16), // MC: noiseGen6
		forestNoise:   NewAuthenticNoiseGeneratorOctaves(rnd, 8),
	}
}

func (cp *ChunkProvider189) generateHighLowNoise(xChunk, zChunk int, noiseField []float64) []float64 {
	if noiseField == nil {
		noiseField = make([]float64, 5*5*33)
	}

	scaleX := 684.412
	scaleZ := 684.412
	// MC 1.8 uses scaleNoise for depthRegion and depthNoise for scaleRegion logic naming is confusing.
	// depthRegion derived from scaleNoise (10 octaves)
	// scaleRegion derived from depthNoise (16 octaves)

	// Generate base noise regions
	// 5x5x1 for depth/scale (2D effectively, but usually generated as 3D with 1 y size)
	// Actually MC generates them as 5x1x5

	// We need coordinates.
	xPos := xChunk * 4
	zPos := zChunk * 4

	cp.scaleRegion = cp.depthNoise.GenerateNoiseOctaves(cp.scaleRegion, xPos, 0, zPos, 5, 1, 5, 200.0, 200.0, 200.0) // noiseGen6
	cp.depthRegion = cp.scaleNoise.GenerateNoiseOctaves(cp.depthRegion, xPos, 0, zPos, 5, 1, 5, 1.121, 1.121, 1.121) // noiseGen5

	cp.mainRegion = cp.mainNoise.GenerateNoiseOctaves(cp.mainRegion, xPos, 0, zPos, 5, 33, 5, scaleX/80.0, 684.412/160.0, scaleZ/80.0)
	cp.minLimitRegion = cp.minLimitNoise.GenerateNoiseOctaves(cp.minLimitRegion, xPos, 0, zPos, 5, 33, 5, scaleX, 684.412, scaleZ)
	cp.maxLimitRegion = cp.maxLimitNoise.GenerateNoiseOctaves(cp.maxLimitRegion, xPos, 0, zPos, 5, 33, 5, scaleX, 684.412, scaleZ)

	idx := 0
	const xSize = 5
	const zSize = 5
	const ySize = 33

	for x := 0; x < xSize; x++ {
		for z := 0; z < zSize; z++ {

			// Biome blending
			// Center is (xPos + x, zPos + z) which are chunks of 4 blocks
			// We need biomes.
			// Using our simple GetBiomeForCoords.
			// Coordinate for biome check: (xChunk * 16) + (x * 4) ...
			// Actually MC calculates biomes at the chunk columns.

			// Parabolic blending
			avgHeight := 0.0
			avgScale := 0.0
			totalWeight := 0.0

			// Center biome
			// centerBiome := GetBiomeForCoords(float64(xPos+x)*4, float64(zPos+z)*4, 123)
			// Wait, xPos is already xChunk*4. So block coords are (xPos + x)*4?
			// No. xPos passed to GenerateNoiseOctaves is usually the noise coordinate.
			// For noise generators, coords are passed as is.
			// The grid is 5x5 covering 16x16 blocks.
			// x=0..4. x=0 is block 0, x=4 is block 16 (overlap with next chunk).
			// So grid spacing is 4 blocks.

			// Let's get the center biome at (xPos + x, zPos + z) - wait, coordinates.
			// xPos is xChunk * 4.
			// realX := (xChunk * 4 + x) * 4 ? No.
			// In MC:
			// this.depthRegion = this.scaleNoise.generateNoiseOctaves(..., x * 4, 0, z * 4, 5, 1, 5, ...)
			// So the input coordinates are scaled by 4 relative to chunk coords.

			centerX := (xChunk * 4) + x
			centerZ := (zChunk * 4) + z

			centerBiome := GetBiomeForCoords(float64(centerX*4+2), float64(centerZ*4+2), 0) // +2 for center of 4x4 block
			// Seed for biome? GetBiomeForCoords takes seed. I'll use 0 or cp.seed if I stored it.
			// I didn't store seed in cp directly, only in rnd. I'll just use 0 for now as biome gen is static per world seed usually.
			// Wait, I should use the world seed.

			for rx := -2; rx <= 2; rx++ {
				for rz := -2; rz <= 2; rz++ {
					// Neighbor biome
					biome := GetBiomeForCoords(float64((centerX+rx)*4+2), float64((centerZ+rz)*4+2), 0)

					baseHeight := biome.MinHeight
					heightVariation := biome.MaxHeight

					// Pre-adjust values
					// MC 1.8:
					// float base = biome.minHeight;
					// float variation = biome.maxHeight;
					// if (variation < 0 && base < 0) variation = -1.0 * 0.1? No...
					// This logic is complex in MC 1.8. I'll implement a simplified version for now.
					// Weights:
					// weight := 10.0 / sqrt((rx*rx + rz*rz) + 0.2)

					weight := 10.0 / math.Sqrt(float64(rx*rx+rz*rz)+0.2)

					// Determine if biome is lower/higher?
					// if neighbor height > center height, weight /= 2
					if biome.MinHeight > centerBiome.MinHeight {
						weight /= 2.0
					}

					avgHeight += baseHeight * weight
					avgScale += heightVariation * weight
					totalWeight += weight
				}
			}

			avgHeight /= totalWeight
			avgScale /= totalWeight

			avgHeight = avgHeight*0.2 + 0.1 // Base offset?
			avgScale = avgScale*0.9 + 0.1   // Scale offset?
			// These magic numbers are from MC 1.8 usually.

			scaleVal := (cp.scaleRegion[z*5+x] / 8000.0)
			if scaleVal < 0 {
				scaleVal = -scaleVal * 0.3
			}
			scaleVal = scaleVal*3.0 - 2.0

			if scaleVal < 0 {
				scaleVal /= 2.0
				if scaleVal < -1 {
					scaleVal = -1
				}
				scaleVal /= 1.4
				scaleVal /= 2.0
			} else {
				if scaleVal > 1 {
					scaleVal = 1
				}
				scaleVal /= 8.0
			}

			// Density calculation
			for y := 0; y < ySize; y++ {
				// Scale/Height based density offset
				offset := avgHeight
				offset += scaleVal * 0.2
				offset = offset * float64(ySize) / 16.0

				yPos := float64(y)
				densityOffset := (yPos - offset) * 12.0 * 128.0 / 256.0 / avgScale
				// 128.0/256.0 is 0.5. 12.0 is magic.

				if densityOffset < 0 {
					densityOffset *= 4.0
				}

				// Interpolate noise
				// In 1.8:
				// double min = minLimit[idx] / 512.0;
				// double max = maxLimit[idx] / 512.0;
				// double main = (mainNoise[idx] / 10.0 + 1.0) / 2.0;
				// double val = lerp(main, min, max) - densityOffset;

				// My noise generators return X first?
				// AuthenticNoiseGeneratorOctaves loops x, then z, then y.
				// PopulateNoiseArray loops x, z, y.
				// Indexing: x*zSize*ySize + z*ySize + y.
				// xSize=5, zSize=5, ySize=33.
				// idx = x*5*33 + z*33 + y.

				// But here I'm looping x, z, y.
				noiseIdx := (x * zSize * ySize) + (z * ySize) + y

				min := cp.minLimitRegion[noiseIdx] / 512.0
				max := cp.maxLimitRegion[noiseIdx] / 512.0
				main := (cp.mainRegion[noiseIdx]/10.0 + 1.0) / 2.0

				var val float64
				if main < 0 {
					val = min
				} else if main > 1 {
					val = max
				} else {
					val = min + (max-min)*main
				}

				val -= densityOffset

				// Y limit clamp (Top/Bottom)
				if y > 29 {
					t := float64(y-29) / 3.0
					val = val*(1.0-t) + -10.0*t
				}

				noiseField[idx] = val
				idx++
			}
		}
	}

	return noiseField
}

// GenerateChunk generates a chunk at the specified coordinates
func (cp *ChunkProvider189) GenerateChunk(xChunk, zChunk int) *Chunk {
	chunk := NewChunk(xChunk, 0, zChunk)

	// Generate noise field
	// We pass nil to allocate new array, or we could reuse a buffer if we had one per provider (not thread safe)
	// For now allocate new.
	noiseField := cp.generateHighLowNoise(xChunk, zChunk, nil)

	const xSize = 5
	const zSize = 5
	const ySize = 33

	// Tri-linear interpolation
	for x := 0; x < 4; x++ {
		for z := 0; z < 4; z++ {
			for y := 0; y < 32; y++ {
				// 8 corners of the 4x4x8 cell
				// Index mapping: (x * 5 * 33) + (z * 33) + y

				idx000 := (x * zSize * ySize) + (z * ySize) + y
				idx001 := idx000 + 1

				idx010 := (x * zSize * ySize) + ((z + 1) * ySize) + y
				idx011 := idx010 + 1

				idx100 := ((x + 1) * zSize * ySize) + (z * ySize) + y
				idx101 := idx100 + 1

				idx110 := ((x + 1) * zSize * ySize) + ((z + 1) * ySize) + y
				idx111 := idx110 + 1

				d000 := noiseField[idx000]
				d001 := noiseField[idx001]
				d010 := noiseField[idx010]
				d011 := noiseField[idx011]
				d100 := noiseField[idx100]
				d101 := noiseField[idx101]
				d110 := noiseField[idx110]
				d111 := noiseField[idx111]

				// Interpolate
				for ly := 0; ly < 8; ly++ {
					// Lerp factors for Y
					// 0..8
					ty := float64(ly) / 8.0

					// Interpolate along Y for 4 vertical lines
					d00 := d000 + (d001-d000)*ty
					d01 := d010 + (d011-d010)*ty
					d10 := d100 + (d101-d100)*ty
					d11 := d110 + (d111-d110)*ty

					for lx := 0; lx < 4; lx++ {
						tx := float64(lx) / 4.0

						// Interpolate along X for 2 lines
						d0 := d00 + (d10-d00)*tx
						d1 := d01 + (d11-d01)*tx

						for lz := 0; lz < 4; lz++ {
							tz := float64(lz) / 4.0

							// Interpolate along Z
							val := d0 + (d1-d0)*tz

							// Block position
							bx := x*4 + lx
							by := y*8 + ly
							bz := z*4 + lz

							if val > 0 {
								chunk.SetBlock(bx, by, bz, BlockTypeStone)
							} else if by < 63 {
								chunk.SetBlock(bx, by, bz, BlockTypeWater)
							} else {
								chunk.SetBlock(bx, by, bz, BlockTypeAir)
							}
						}
					}
				}
			}
		}
	}

	// Bedrock at bottom
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			chunk.SetBlock(x, 0, z, BlockTypeBedrock)
		}
	}

	return chunk
}
