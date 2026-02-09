package world

import (
	"math"
	"math/rand"
)

// ChunkProvider189 implements TerrainGenerator using Minecraft 1.8.9 authentic noise.
// Thread-safe: all noise buffers are allocated per-call, generators are read-only after init.
type ChunkProvider189 struct {
	seed int64

	minLimitNoise *AuthenticNoiseGeneratorOctaves // 16 octaves (field_147431_j)
	maxLimitNoise *AuthenticNoiseGeneratorOctaves // 16 octaves (field_147432_k)
	mainNoise     *AuthenticNoiseGeneratorOctaves // 8 octaves  (field_147429_l)
	surfaceNoise  *AuthenticNoiseGeneratorOctaves // 4 octaves  (field_147430_m - actually Perlin, simplified here)
	scaleNoise    *AuthenticNoiseGeneratorOctaves // 10 octaves (noiseGen5 → depthRegion)
	depthNoise    *AuthenticNoiseGeneratorOctaves // 16 octaves (noiseGen6 → scaleRegion)

	// MC default settings
	coordinateScale  float64 // 684.412
	heightScale      float64 // 684.412
	mainNoiseScaleX  float64 // 80.0
	mainNoiseScaleY  float64 // 160.0
	mainNoiseScaleZ  float64 // 80.0
	lowerLimitScale  float64 // 512.0
	upperLimitScale  float64 // 512.0
	baseSize         float64 // 8.5
	stretchY         float64 // 12.0
	depthNoiseScaleX float64 // 200.0
	depthNoiseScaleZ float64 // 200.0
	depthNoiseExpo   float64 // 0.5
	biomeDepthWeight float64 // 1.0
	biomeDepthOffset float64 // 0.0
	biomeScaleWeight float64 // 1.0
	biomeScaleOffset float64 // 0.0

	// Precomputed parabolic blending weights (5x5 grid)
	parabolicField [25]float64
}

func NewChunkProvider189(seed int64) *ChunkProvider189 {
	rnd := rand.New(rand.NewSource(seed))

	cp := &ChunkProvider189{
		seed:          seed,
		minLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		maxLimitNoise: NewAuthenticNoiseGeneratorOctaves(rnd, 16),
		mainNoise:     NewAuthenticNoiseGeneratorOctaves(rnd, 8),
		surfaceNoise:  NewAuthenticNoiseGeneratorOctaves(rnd, 4),
		scaleNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 10),
		depthNoise:    NewAuthenticNoiseGeneratorOctaves(rnd, 16),

		coordinateScale:  684.412,
		heightScale:      684.412,
		mainNoiseScaleX:  80.0,
		mainNoiseScaleY:  160.0,
		mainNoiseScaleZ:  80.0,
		lowerLimitScale:  512.0,
		upperLimitScale:  512.0,
		baseSize:         8.5,
		stretchY:         12.0,
		depthNoiseScaleX: 200.0,
		depthNoiseScaleZ: 200.0,
		depthNoiseExpo:   0.5,
		biomeDepthWeight: 1.0,
		biomeDepthOffset: 0.0,
		biomeScaleWeight: 1.0,
		biomeScaleOffset: 0.0,
	}

	// Precompute parabolic field: 10.0 / sqrt(dx^2 + dz^2 + 0.2)
	for dx := -2; dx <= 2; dx++ {
		for dz := -2; dz <= 2; dz++ {
			cp.parabolicField[(dx+2)+(dz+2)*5] = 10.0 / math.Sqrt(float64(dx*dx+dz*dz)+0.2)
		}
	}

	return cp
}

// HeightAt returns a constant upper bound for the streamer.
// ChunkProvider189 generates full 256-height chunks in a single call.
func (cp *ChunkProvider189) HeightAt(_, _ int) int {
	return 128
}

const (
	noiseGridX = 5
	noiseGridZ = 5
	noiseGridY = 33
	seaLevel   = 63
)

// generateDensityField computes the 5x33x5 density field for a chunk.
// This is a 1:1 port of MC's func_147423_a.
// MC field_147434_q layout: [825] indexed as (x*5+z)*33+y (iterating k=x, l=z, l1=y).
func (cp *ChunkProvider189) generateDensityField(xChunk, zChunk int) []float64 {
	field := make([]float64, noiseGridX*noiseGridZ*noiseGridY)

	xPos := xChunk * 4
	zPos := zChunk * 4

	// Depth noise: 2D (5x5), MC uses noiseGen6 with 2D bouncer
	depthNoiseArray := cp.depthNoise.GenerateNoiseOctaves2D(nil, xPos, zPos, 5, 5,
		cp.depthNoiseScaleX, cp.depthNoiseScaleZ, cp.depthNoiseExpo)

	// 3D noise arrays
	f := cp.coordinateScale
	f1 := cp.heightScale

	mainRegion := cp.mainNoise.GenerateNoiseOctaves(nil, xPos, 0, zPos, 5, 33, 5,
		f/cp.mainNoiseScaleX, f1/cp.mainNoiseScaleY, f/cp.mainNoiseScaleZ)
	minLimitRegion := cp.minLimitNoise.GenerateNoiseOctaves(nil, xPos, 0, zPos, 5, 33, 5,
		f, f1, f)
	maxLimitRegion := cp.maxLimitNoise.GenerateNoiseOctaves(nil, xPos, 0, zPos, 5, 33, 5,
		f, f1, f)

	// MC iterates: k (x 0..4), l (z 0..4), l1 (y 0..32)
	// field_147434_q index i increments linearly
	// depthNoiseArray index j increments per (x,z) column
	noiseIdx := 0 // i in MC (3D noise index)
	depthIdx := 0 // j in MC (2D depth noise index)

	for k := 0; k < 5; k++ {
		for l := 0; l < 5; l++ {
			// Biome blending over 5x5 neighborhood
			var f2, f3, f4 float64

			// Center biome at grid position
			centerBiome := GetBiomeForCoords(
				float64((xPos+k)*4+2),
				float64((zPos+l)*4+2),
				cp.seed,
			)

			for j1 := -2; j1 <= 2; j1++ {
				for k1 := -2; k1 <= 2; k1++ {
					biome := GetBiomeForCoords(
						float64((xPos+k+j1)*4+2),
						float64((zPos+l+k1)*4+2),
						cp.seed,
					)

					f5 := cp.biomeDepthOffset + biome.MinHeight*cp.biomeDepthWeight
					f6 := cp.biomeScaleOffset + biome.MaxHeight*cp.biomeScaleWeight

					// Parabolic weight: field / (depth + 2.0)
					f7 := cp.parabolicField[(j1+2)+(k1+2)*5] / (f5 + 2.0)
					if biome.MinHeight > centerBiome.MinHeight {
						f7 /= 2.0
					}

					f2 += f6 * f7 // scale accumulator
					f3 += f5 * f7 // depth accumulator
					f4 += f7      // weight accumulator
				}
			}

			f2 /= f4
			f3 /= f4
			f2 = f2*0.9 + 0.1
			f3 = (f3*4.0 - 1.0) / 8.0

			// Depth noise processing
			d7 := depthNoiseArray[depthIdx] / 8000.0
			depthIdx++

			if d7 < 0 {
				d7 = -d7 * 0.3
			}
			d7 = d7*3.0 - 2.0

			if d7 < 0 {
				d7 /= 2.0
				if d7 < -1 {
					d7 = -1
				}
				d7 /= 1.4
				d7 /= 2.0
			} else {
				if d7 > 1 {
					d7 = 1
				}
				d7 /= 8.0
			}

			d8 := f3
			d9 := f2
			d8 += d7 * 0.2
			d8 = d8 * cp.baseSize / 8.0
			d0 := cp.baseSize + d8*4.0

			for l1 := 0; l1 < 33; l1++ {
				d1 := (float64(l1) - d0) * cp.stretchY * 128.0 / 256.0 / d9

				if d1 < 0 {
					d1 *= 4.0
				}

				d2 := minLimitRegion[noiseIdx] / cp.lowerLimitScale
				d3 := maxLimitRegion[noiseIdx] / cp.upperLimitScale
				d4 := (mainRegion[noiseIdx]/10.0 + 1.0) / 2.0

				var d5 float64
				if d4 < 0 {
					d5 = d2
				} else if d4 > 1 {
					d5 = d3
				} else {
					d5 = d2 + (d3-d2)*d4
				}

				d5 -= d1

				// Top fade: force air above y=29
				if l1 > 29 {
					d6 := float64(l1-29) / 3.0
					d5 = d5*(1.0-d6) + -10.0*d6
				}

				field[noiseIdx] = d5
				noiseIdx++
			}
		}
	}

	return field
}

// PopulateChunk fills a chunk using the MC 1.8.9 density field + trilinear interpolation.
// This is a 1:1 port of MC's setBlocksInChunk.
func (cp *ChunkProvider189) PopulateChunk(c *Chunk) {
	xChunk := c.X
	zChunk := c.Z

	noiseField := cp.generateDensityField(xChunk, zChunk)

	// MC's interpolation loop:
	// for i (x 0..3): j = i*5, k = (i+1)*5
	//   for l (z 0..3): i1=(j+l)*33, j1=(j+l+1)*33, k1=(k+l)*33, l1=(k+l+1)*33
	//     for i2 (y 0..31):
	//       8 corners from noiseField[i1+i2], etc.
	//       interpolate 8x4x4 sub-block

	for i := 0; i < 4; i++ {
		j := i * 5
		k := (i + 1) * 5

		for l := 0; l < 4; l++ {
			i1 := (j + l) * 33
			j1 := (j + l + 1) * 33
			k1 := (k + l) * 33
			l1 := (k + l + 1) * 33

			for i2 := 0; i2 < 32; i2++ {
				d1 := noiseField[i1+i2]
				d2 := noiseField[j1+i2]
				d3 := noiseField[k1+i2]
				d4 := noiseField[l1+i2]
				d5 := (noiseField[i1+i2+1] - d1) * 0.125
				d6 := (noiseField[j1+i2+1] - d2) * 0.125
				d7 := (noiseField[k1+i2+1] - d3) * 0.125
				d8 := (noiseField[l1+i2+1] - d4) * 0.125

				for j2 := 0; j2 < 8; j2++ {
					d10 := d1
					d11 := d2
					d12 := (d3 - d1) * 0.25
					d13 := (d4 - d2) * 0.25

					for k2 := 0; k2 < 4; k2++ {
						d16 := (d11 - d10) * 0.25
						lvt := d10 - d16

						for l2 := 0; l2 < 4; l2++ {
							lvt += d16

							bx := i*4 + k2
							by := i2*8 + j2
							bz := l*4 + l2

							if lvt > 0 {
								c.SetBlock(bx, by, bz, BlockTypeStone)
							} else if by < seaLevel {
								c.SetBlock(bx, by, bz, BlockTypeWater)
							}
							// else: air (default)
						}

						d10 += d12
						d11 += d13
					}

					d1 += d5
					d2 += d6
					d3 += d7
					d4 += d8
				}
			}
		}
	}

	// Phase 2: Surface replacement (grass/dirt) + bedrock
	cp.replaceSurface(c, xChunk, zChunk)

	c.dirty = true
}

// replaceSurface replaces top stone with grass/dirt layers and adds bedrock.
func (cp *ChunkProvider189) replaceSurface(c *Chunk, xChunk, zChunk int) {
	for lx := 0; lx < ChunkSizeX; lx++ {
		for lz := 0; lz < ChunkSizeZ; lz++ {
			worldX := xChunk*ChunkSizeX + lx
			worldZ := zChunk*ChunkSizeZ + lz
			biome := GetBiomeForCoords(float64(worldX), float64(worldZ), cp.seed)

			topBlock := biome.TopBlock
			fillerBlock := biome.FillerBlock
			fillerDepth := -1

			for y := 255; y >= 0; y-- {
				// Bedrock layer (y 0-4)
				if y <= 4 {
					if y == 0 {
						c.SetBlock(lx, y, lz, BlockTypeBedrock)
						continue
					}
					hash := uint64(worldX)*0x9E3779B9 + uint64(worldZ)*0x517CC1B7 + uint64(y)*0x6C622723
					hash = (hash ^ (hash >> 16)) * 0x45D9F3B
					if int(hash%5) <= (4 - y) {
						c.SetBlock(lx, y, lz, BlockTypeBedrock)
						continue
					}
				}

				block := c.GetBlock(lx, y, lz)

				if block == BlockTypeAir || block == BlockTypeWater {
					fillerDepth = -1
					continue
				}

				if block != BlockTypeStone {
					continue
				}

				if fillerDepth == -1 {
					fillerDepth = 3
					if y >= seaLevel-1 {
						c.SetBlock(lx, y, lz, topBlock)
					} else {
						c.SetBlock(lx, y, lz, fillerBlock)
					}
				} else if fillerDepth > 0 {
					fillerDepth--
					c.SetBlock(lx, y, lz, fillerBlock)
				}
			}
		}
	}
}
