package world

import (
	"math"
	"math/rand"
	"sync"
)

// chunkGenBuffers holds pre-allocated noise and biome buffers reused across chunk generation calls.
type chunkGenBuffers struct {
	densityField  []float64   // 825 = 5*33*5
	depthNoise    []float64   // 25  = 5*1*5
	mainNoise     []float64   // 825 = 5*33*5
	minNoise      []float64   // 825 = 5*33*5
	maxNoise      []float64   // 825 = 5*33*5
	biomeGrid     [81]*Biome  // 9×9 grid covering the density blending neighbourhood
	surfaceBiomes [256]*Biome // 16×16 grid for replaceSurface
	heightMap     [256]int16  // per-column max Y with non-air block, indexed [lx*16+lz]
}

var genBufferPool = sync.Pool{
	New: func() interface{} {
		return &chunkGenBuffers{
			densityField: make([]float64, noiseGridX*noiseGridZ*noiseGridY),
			depthNoise:   make([]float64, noiseGridX*noiseGridZ),
			mainNoise:    make([]float64, noiseGridX*noiseGridZ*noiseGridY),
			minNoise:     make([]float64, noiseGridX*noiseGridZ*noiseGridY),
			maxNoise:     make([]float64, noiseGridX*noiseGridZ*noiseGridY),
		}
	},
}

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

// generateDensityField computes the 5x33x5 density field for a chunk into bufs.densityField.
// This is a 1:1 port of MC's func_147423_a.
// MC field_147434_q layout: [825] indexed as (x*5+z)*33+y (iterating k=x, l=z, l1=y).
// bufs.biomeGrid must be pre-populated before calling this function.
func (cp *ChunkProvider189) generateDensityField(xChunk, zChunk int, bufs *chunkGenBuffers) []float64 {
	field := bufs.densityField

	xPos := xChunk * 4
	zPos := zChunk * 4

	// 3D noise arrays — pass pre-allocated slices to avoid 3 × 825 allocations per chunk.
	f := cp.coordinateScale
	f1 := cp.heightScale

	// Run the 3 expensive 3D noise computations in parallel goroutines.
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		cp.mainNoise.GenerateNoiseOctaves(bufs.mainNoise, xPos, 0, zPos, 5, 33, 5,
			f/cp.mainNoiseScaleX, f1/cp.mainNoiseScaleY, f/cp.mainNoiseScaleZ)
	}()
	go func() {
		defer wg.Done()
		cp.minLimitNoise.GenerateNoiseOctaves(bufs.minNoise, xPos, 0, zPos, 5, 33, 5, f, f1, f)
	}()
	go func() {
		defer wg.Done()
		cp.maxLimitNoise.GenerateNoiseOctaves(bufs.maxNoise, xPos, 0, zPos, 5, 33, 5, f, f1, f)
	}()

	// Run cheap depth noise on current goroutine while the 3D noises run in parallel.
	// Depth noise: 2D (5x5), MC uses noiseGen6 with 2D bouncer
	// Pass pre-allocated slice; GenerateNoiseOctaves2D zeroes it before use.
	depthNoiseArray := cp.depthNoise.GenerateNoiseOctaves2D(bufs.depthNoise, xPos, zPos, 5, 5,
		cp.depthNoiseScaleX, cp.depthNoiseScaleZ, cp.depthNoiseExpo)

	wg.Wait()

	mainRegion := bufs.mainNoise
	minLimitRegion := bufs.minNoise
	maxLimitRegion := bufs.maxNoise

	// MC iterates: k (x 0..4), l (z 0..4), l1 (y 0..32)
	// field_147434_q index i increments linearly
	// depthNoiseArray index j increments per (x,z) column
	noiseIdx := 0 // i in MC (3D noise index)
	depthIdx := 0 // j in MC (2D depth noise index)

	for k := 0; k < 5; k++ {
		for l := 0; l < 5; l++ {
			// Biome blending over 5x5 neighborhood.
			// biomeGrid covers (k-2..k+2) × (l-2..l+2) already pre-computed as a 9×9 grid
			// where grid[gx*9+gz] maps to world grid column (xPos+k-2+gx, zPos+l-2+gz).
			// For the current (k,l) the center is at gx=k+2, gz=l+2 in that 9×9.
			var f2, f3, f4 float64

			centerBiome := bufs.biomeGrid[(k+2)*9+(l+2)]

			for j1 := -2; j1 <= 2; j1++ {
				for k1 := -2; k1 <= 2; k1++ {
					biome := bufs.biomeGrid[(k+j1+2)*9+(l+k1+2)]

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

	// Acquire per-call reusable buffers from pool.
	bufs := genBufferPool.Get().(*chunkGenBuffers)
	defer genBufferPool.Put(bufs)

	// Zero heightMap: -1 means no non-air block seen in this column yet.
	for i := range bufs.heightMap {
		bufs.heightMap[i] = -1
	}

	xPos := xChunk * 4
	zPos := zChunk * 4

	// Pre-compute 9×9 biome grid covering all density columns and their 5×5 neighbourhoods.
	// Grid origin is (xPos-2, zPos-2); index = gx*9 + gz.
	for gx := 0; gx < 9; gx++ {
		for gz := 0; gz < 9; gz++ {
			bufs.biomeGrid[gx*9+gz] = GetBiomeForCoords(
				float64((xPos-2+gx)*4+2),
				float64((zPos-2+gz)*4+2),
				cp.seed,
			)
		}
	}

	// Pre-compute 16×16 surface biomes for replaceSurface.
	for lx := 0; lx < ChunkSizeX; lx++ {
		for lz := 0; lz < ChunkSizeZ; lz++ {
			worldX := xChunk*ChunkSizeX + lx
			worldZ := zChunk*ChunkSizeZ + lz
			bufs.surfaceBiomes[lx*16+lz] = GetBiomeForCoords(float64(worldX), float64(worldZ), cp.seed)
		}
	}

	noiseField := cp.generateDensityField(xChunk, zChunk, bufs)

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
								c.SetBlockFast(bx, by, bz, BlockTypeStone)
								if int16(by) > bufs.heightMap[bx*16+bz] {
									bufs.heightMap[bx*16+bz] = int16(by)
								}
							} else if by < seaLevel {
								c.SetBlockFast(bx, by, bz, BlockTypeWater)
								if int16(by) > bufs.heightMap[bx*16+bz] {
									bufs.heightMap[bx*16+bz] = int16(by)
								}
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

	// Phase 2: Surface replacement (grass/dirt/sand) + bedrock
	cp.replaceSurface(c, xChunk, zChunk, &bufs.surfaceBiomes, &bufs.heightMap)

	// Phase 3: Vegetation (trees)
	cp.generateTrees(c, xChunk, zChunk, &bufs.surfaceBiomes)

	c.dirty = true
}

// absInt returns the absolute value of an integer.
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// generateTrees places trees after surface generation.
// Uses the center biome of the chunk to pick tree type and count,
// matching the MC 1.8.9 BiomeDecorator approach (treesPerChunk attempts).
func (cp *ChunkProvider189) generateTrees(c *Chunk, xChunk, zChunk int, surfaceBiomes *[256]*Biome) {
	// Determine tree parameters from the chunk's center biome.
	biome := surfaceBiomes[7*16+7]
	if biome.Trees == TreeNone || biome.TreeCount == 0 {
		return
	}

	// Seeded RNG for deterministic, chunk-local decoration.
	// XOR-mix of world seed and chunk coords — matches MC's per-chunk decoration seed pattern.
	rngSeed := cp.seed ^ (int64(xChunk) * 0x4F9939F508) ^ (int64(zChunk) * 0x1EF1565BD5)
	rng := rand.New(rand.NewSource(rngSeed))

	count := int(biome.TreeCount)
	if rng.Intn(10) == 0 { // MC adds 10% chance of +1 tree
		count++
	}

	for i := 0; i < count; i++ {
		lx := 1 + rng.Intn(14) // stay off edges so canopy has room
		lz := 1 + rng.Intn(14)

		// Find the surface block in this column (scan down from max expected height).
		surfaceY := -1
		for y := 120; y >= seaLevel; y-- {
			b := c.GetBlock(lx, y, lz)
			if b != BlockTypeAir && b != BlockTypeWater {
				surfaceY = y
				break
			}
		}
		if surfaceY < seaLevel {
			continue
		}
		// Trees only grow on grass.
		if c.GetBlock(lx, surfaceY, lz) != BlockTypeGrass {
			continue
		}

		switch biome.Trees {
		case TreeOak:
			cp.placeOakTree(c, lx, surfaceY+1, lz, rng)
		case TreeSpruce:
			cp.placeSpruceTree(c, lx, surfaceY+1, lz, rng)
		}
	}
}

// placeOakTree generates a standard oak tree matching WorldGenTrees exactly.
// baseY is the Y of the first trunk block (one above ground).
func (cp *ChunkProvider189) placeOakTree(c *Chunk, x, baseY, z int, rng *rand.Rand) {
	i := rng.Intn(3) + 4 // 4-6: trunk height (WorldGenTrees minTreeHeight=4)

	// Abort if trunk space is obstructed.
	for y := 0; y < i; y++ {
		if c.GetBlock(x, baseY+y, z) != BlockTypeAir {
			return
		}
	}

	// topLeafY: one above the last trunk block (position.getY() + i in MC).
	topLeafY := baseY + i

	// Leaf layers: 4 layers from topLeafY-3 to topLeafY.
	// Radius formula: j1 = 1 - i4/2 where i4 = leafY - topLeafY
	// i4=-3 → j1=2, i4=-2 → j1=2, i4=-1 → j1=1, i4=0 → j1=1
	for leafY := topLeafY - 3; leafY <= topLeafY; leafY++ {
		i4 := leafY - topLeafY // -3..0
		j1 := 1 - i4/2         // radius: 2,2,1,1
		for dx := -j1; dx <= j1; dx++ {
			for dz := -j1; dz <= j1; dz++ {
				isCorner := absInt(dx) == j1 && absInt(dz) == j1
				// MC: drop corner if at top layer (i4==0) OR with 50% chance elsewhere.
				if isCorner && (i4 == 0 || rng.Intn(2) == 0) {
					continue
				}
				if c.GetBlock(x+dx, leafY, z+dz) == BlockTypeAir {
					c.SetBlock(x+dx, leafY, z+dz, BlockTypeOakLeaves)
				}
			}
		}
	}

	// Trunk: placed after leaves to overwrite any leaf at trunk position.
	for y := 0; y < i; y++ {
		b := c.GetBlock(x, baseY+y, z)
		if b == BlockTypeAir || b == BlockTypeOakLeaves {
			c.SetBlock(x, baseY+y, z, BlockTypeOakLog)
		}
	}
}

// placeSpruceTree generates a spruce tree matching WorldGenTaiga2 exactly.
// baseY is the Y of the first trunk block (one above ground).
func (cp *ChunkProvider189) placeSpruceTree(c *Chunk, x, baseY, z int, rng *rand.Rand) {
	i := rng.Intn(4) + 6 // 6-9: total height
	j := 1 + rng.Intn(2) // 1-2: bare trunk blocks at bottom (no leaves below)
	k := i - j           // number of leaf coverage layers
	l := 2 + rng.Intn(2) // 2-3: max leaf radius

	// Abort if trunk space is obstructed.
	for y := 0; y < i; y++ {
		if c.GetBlock(x, baseY+y, z) != BlockTypeAir {
			return
		}
	}

	// Leaf placement using MC's staircase radius pattern (WorldGenTaiga2).
	// Iterates from top (baseY+i) downward k+1 times.
	// The radius zig-zags: 0,1,0,1,2,1,2,3,... giving the classic stepped look.
	i3 := rng.Intn(2) // starting radius: 0 or 1
	j3 := 1           // threshold at which radius resets and step grows
	k3 := 0           // value i3 resets to

	for l3 := 0; l3 <= k; l3++ {
		leafY := baseY + i - l3

		for dx := -i3; dx <= i3; dx++ {
			for dz := -i3; dz <= i3; dz++ {
				// MC uses hollow squares: skip the 4 exact corners.
				if absInt(dx) == i3 && absInt(dz) == i3 && i3 > 0 {
					continue
				}
				if c.GetBlock(x+dx, leafY, z+dz) == BlockTypeAir {
					c.SetBlock(x+dx, leafY, z+dz, BlockTypeSpruceLeaves)
				}
			}
		}

		// Advance radius: staircase pattern.
		if i3 >= j3 {
			i3 = k3
			k3 = 1
			j3++
			if j3 > l {
				j3 = l
			}
		} else {
			i3++
		}
	}

	// Trunk placed after leaves (may be slightly shorter than i), overwriting leaves.
	trunkH := i - rng.Intn(3)
	for y := 0; y < trunkH; y++ {
		b := c.GetBlock(x, baseY+y, z)
		if b == BlockTypeAir || b == BlockTypeSpruceLeaves {
			c.SetBlock(x, baseY+y, z, BlockTypeSpruceLog)
		}
	}
}

// replaceSurface replaces top stone with grass/dirt layers and adds bedrock.
// surfaceBiomes is a pre-computed 16×16 array indexed [lx*16+lz].
// heightMap tracks the maximum Y with a non-air block per column, allowing the Y loop
// to start from the actual terrain surface rather than always scanning from y=255.
func (cp *ChunkProvider189) replaceSurface(c *Chunk, xChunk, zChunk int, surfaceBiomes *[256]*Biome, heightMap *[256]int16) {
	for lx := 0; lx < ChunkSizeX; lx++ {
		for lz := 0; lz < ChunkSizeZ; lz++ {
			worldX := xChunk*ChunkSizeX + lx
			worldZ := zChunk*ChunkSizeZ + lz
			biome := surfaceBiomes[lx*16+lz]

			topBlock := biome.TopBlock
			fillerBlock := biome.FillerBlock
			fillerDepth := -1

			startY := int(heightMap[lx*16+lz])
			if startY < 0 {
				startY = 4 // no terrain in this column; still process bedrock layer
			}
			for y := startY; y >= 0; y-- {
				// Bedrock layer (y 0-4)
				if y <= 4 {
					if y == 0 {
						c.SetBlockFast(lx, y, lz, BlockTypeBedrock)
						continue
					}
					hash := uint64(worldX)*0x9E3779B9 + uint64(worldZ)*0x517CC1B7 + uint64(y)*0x6C622723
					hash = (hash ^ (hash >> 16)) * 0x45D9F3B
					if int(hash%5) <= (4 - y) {
						c.SetBlockFast(lx, y, lz, BlockTypeBedrock)
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
						c.SetBlockFast(lx, y, lz, topBlock)
					} else {
						c.SetBlockFast(lx, y, lz, fillerBlock)
					}
				} else if fillerDepth > 0 {
					fillerDepth--
					c.SetBlockFast(lx, y, lz, fillerBlock)
				}
			}
		}
	}
}
