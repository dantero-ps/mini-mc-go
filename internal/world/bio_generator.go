package world

import (
	"math"
)

// BioGenerator implements surface generation inspired by Minecraft 1.8.9 but with corrected scales.
// It uses three noise layers (Main, Min, Max) and biome-based height blending.
type BioGenerator struct {
	seed       int64
	baseHeight int

	// Noise Scales
	coordinateScale    float64
	heightScale        float64
	depthNoiseScaleX   float64
	depthNoiseScaleZ   float64
	depthNoiseExponent float64
	biomeDepthWeight   float64
	biomeScaleWeight   float64
	biomeScaleOffset   float64
	biomeDepthOffset   float64
	stretchY           float64
	baseSize           float64

	parabolicField []float32
}

func NewBioGenerator(seed int64) TerrainGenerator {
	g := &BioGenerator{
		seed:       seed,
		baseHeight: 64,
		// Adjusted scales for "Earth-like" terrain features
		coordinateScale:    0.01,  // Was 684.412 -> 0.01 (1/100 blocks)
		heightScale:        0.01,  // Was 684.412
		depthNoiseScaleX:   200.0, // Used for depth noise (not implemented fully yet, kept as placeholder)
		depthNoiseScaleZ:   200.0,
		depthNoiseExponent: 0.5,
		biomeDepthWeight:   1.0,
		biomeScaleWeight:   1.0,
		biomeScaleOffset:   0.0,
		biomeDepthOffset:   0.0,
		stretchY:           12.0,
		baseSize:           8.5,
	}

	// Calculate parabolic field for biome blending (10.0 / sqrt(i^2 + j^2 + 0.2))
	g.parabolicField = make([]float32, 25)
	for i := -2; i <= 2; i++ {
		for j := -2; j <= 2; j++ {
			f := 10.0 / float32(math.Sqrt(float64(i*i+j*j)+0.2))
			g.parabolicField[(i+2)+(j+2)*5] = f
		}
	}
	return g
}

// clamp helper
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// denormalizeClamp similar to MC's MathHelper
func denormalizeClamp(min, max, ratio float64) float64 {
	if ratio < 0.0 {
		return min
	}
	if ratio > 1.0 {
		return max
	}
	return min + (max-min)*ratio
}

// computeDensity calculates the density value at a world coordinate.
func (g *BioGenerator) computeDensity(worldX, worldY, worldZ int) float64 {
	x := float64(worldX)
	y := float64(worldY)
	z := float64(worldZ)

	// 1. Biome Blending
	avgScale := 0.0
	avgDepth := 0.0
	totalWeight := 0.0

	centerBiome := GetBiomeForCoords(x, z, g.seed)

	for i := -2; i <= 2; i++ {
		for j := -2; j <= 2; j++ {
			// Sample biomes every 16 blocks (Chunk granularity approximation)
			sampleX := x + float64(i*16)
			sampleZ := z + float64(j*16)
			biome := GetBiomeForCoords(sampleX, sampleZ, g.seed)

			depth := g.biomeDepthOffset + biome.MinHeight*g.biomeDepthWeight
			scale := g.biomeScaleOffset + biome.MaxHeight*g.biomeScaleWeight

			// Parabolic weight
			weight := float64(g.parabolicField[(i+2)+(j+2)*5]) / (depth + 2.0)

			if biome.MinHeight > centerBiome.MinHeight {
				weight /= 2.0
			}

			avgScale += scale * weight
			avgDepth += depth * weight
			totalWeight += weight
		}
	}

	avgScale /= totalWeight
	avgDepth /= totalWeight
	avgScale = avgScale*0.9 + 0.1
	avgDepth = (avgDepth*4.0 - 1.0) / 8.0

	// 2. Main Generation Logic
	// d0 = baseSize + avgDepth * 4.0D
	densityOffset := (float64(g.baseSize) + avgDepth*4.0)

	// Scale factor affects how quickly density falls off with height
	// Larger scaleFactor -> sharper terrain
	scaleFactor := (g.stretchY * 128.0 / 256.0) / avgScale

	// Normalize Y to roughly [0, 33] assuming Y=0..256
	mcY := y / 8.0

	// Base density from height gradient
	// "How far are we from the 'ideal' surface height?"
	heightDensity := (mcY - densityOffset) * scaleFactor

	// 3. Noise Sampling
	// Use simpler scales for visual coherence
	// Main structural noise (Large features)
	// Min/Max noise provide variation
	minNoise := octaveNoise3D(x*g.coordinateScale, y*g.heightScale, z*g.coordinateScale, g.seed, 4, 0.5, 2.0)
	maxNoise := octaveNoise3D(x*g.coordinateScale, y*g.heightScale, z*g.coordinateScale, g.seed+1000, 4, 0.5, 2.0)

	// Interpolation control noise (finer detail)
	mainNoise := octaveNoise3D(x*(g.coordinateScale*2.0), y*(g.heightScale*2.0), z*(g.coordinateScale*2.0), g.seed+2000, 2, 0.5, 2.0)

	// Map random [0,1] to signed range [-1, 1]
	minNoise = minNoise*2.0 - 1.0
	maxNoise = maxNoise*2.0 - 1.0
	mainNoise = mainNoise*2.0 - 1.0

	// Interpolate
	vol := (mainNoise/10.0 + 1.0) / 2.0
	vol = clamp(vol, 0.0, 1.0)

	// Combine noise with height gradient
	// density > 0 means solid
	density := denormalizeClamp(minNoise, maxNoise, vol) - heightDensity

	// Top/Bottom limits to ensure bedrock and sky
	if y > 250 {
		return -1.0 // Force air
	}
	if y < 1 {
		return 10.0 // Force bedrock/solid
	}

	return density
}

// HeightAt computes the approximate surface height.
func (g *BioGenerator) HeightAt(worldX, worldZ int) int {
	// Raycast down from max reasonable height
	for y := 250; y >= 0; y-- {
		d := g.computeDensity(worldX, y, worldZ)
		if d > 0 {
			return y
		}
	}
	return 0
}

// PopulateChunk fills the given chunk with blocks.
func (g *BioGenerator) PopulateChunk(c *Chunk) {
	chunkBaseY := c.Y * ChunkSizeY

	if chunkBaseY > 256 || chunkBaseY < 0 {
		return
	}

	for lx := 0; lx < ChunkSizeX; lx++ {
		for lz := 0; lz < ChunkSizeZ; lz++ {
			worldX := c.X*ChunkSizeX + lx
			worldZ := c.Z*ChunkSizeZ + lz

			// Use chunk-center biome for the whole column to avoid "jittery" biome changes in short distances
			// (Or stick to per-column logic which is fine if blended properly)
			biome := GetBiomeForCoords(float64(worldX), float64(worldZ), g.seed)

			// Column filling
			fillerRemaining := -1
			topBlock := biome.TopBlock
			fillerBlock := biome.FillerBlock

			// Iterate Top-Down
			for ly := ChunkSizeY - 1; ly >= 0; ly-- {
				worldY := chunkBaseY + ly

				if worldY <= 0 {
					c.SetBlock(lx, ly, lz, BlockTypeBedrock)
					continue
				}

				if worldY >= 256 {
					continue
				}

				density := g.computeDensity(worldX, worldY, worldZ)

				if density > 0 {
					// Solid block
					if fillerRemaining == -1 {
						// We just hit the surface
						fillerRemaining = 3 // Typically 3-4 blocks of dirt
						c.SetBlock(lx, ly, lz, topBlock)
					} else if fillerRemaining > 0 {
						// Sub-surface layer
						fillerRemaining--
						c.SetBlock(lx, ly, lz, fillerBlock)
					} else {
						// Deep stone
						c.SetBlock(lx, ly, lz, BlockTypeStone)
					}
				} else {
					// Air (or water if we implement SeaLevel)
					fillerRemaining = -1
					// If Ocean biome and below sea level (63), place water?
					// For now leave as air.
				}
			}
		}
	}
	c.dirty = true
}
