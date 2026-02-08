package world

import (
	"math"
)

// TerrainGenerator handles terrain generation for a chunk.
type TerrainGenerator interface {
	// HeightAt computes world surface height (block Y) at world X,Z.
	HeightAt(x, z int) int
	// PopulateChunk fills the given chunk with blocks based on the generation logic.
	PopulateChunk(c *Chunk)
}

// StandardGenerator handles terrain generation logic using Perlin noise.
type StandardGenerator struct {
	seed        int64
	scale       float64
	baseHeight  int
	amp         float64
	octaves     int
	persistence float64
	lacunarity  float64
}

// NewGenerator creates a new generator with default settings.
// Returns a TerrainGenerator interface.
func NewGenerator(seed int64) TerrainGenerator {
	return &StandardGenerator{
		seed:        seed,
		scale:       1.0 / 64.0,
		baseHeight:  32,
		amp:         32,
		octaves:     4,
		persistence: 0.5,
		lacunarity:  2.0,
	}
}

// HeightAt computes world surface height (block Y) at world X,Z.
func (g *StandardGenerator) HeightAt(worldX, worldZ int) int {
	x := float64(worldX) * g.scale
	z := float64(worldZ) * g.scale
	n := octaveNoise2D(x, z, g.seed, g.octaves, g.persistence, g.lacunarity)
	height := float64(g.baseHeight) + n*g.amp
	if height < 0 {
		height = 0
	}
	return int(math.Floor(height))
}

// PopulateChunk fills a chunk using noise heightmap.
func (g *StandardGenerator) PopulateChunk(c *Chunk) {
	chunkBaseY := c.Y * ChunkSizeY
	for lx := range ChunkSizeX {
		for lz := range ChunkSizeZ {
			worldX := c.X*ChunkSizeX + lx
			worldZ := c.Z*ChunkSizeZ + lz
			height := g.HeightAt(worldX, worldZ)
			topLocal := height - chunkBaseY
			if topLocal < 0 {
				continue
			}
			if topLocal >= ChunkSizeY {
				topLocal = ChunkSizeY - 1
			}
			for ly := 0; ly < topLocal; ly++ {
				if chunkBaseY+ly == 0 {
					c.SetBlock(lx, ly, lz, BlockTypeBedrock)
				} else {
					c.SetBlock(lx, ly, lz, BlockTypeDirt)
				}
			}
			if chunkBaseY+topLocal == 0 {
				c.SetBlock(lx, topLocal, lz, BlockTypeBedrock)
			} else {
				c.SetBlock(lx, topLocal, lz, BlockTypeGrass)
			}
		}
	}
	c.dirty = true
}

// FlatGenerator generates a flat world at a specific height.
type FlatGenerator struct {
	Height int
}

// NewFlatGenerator creates a new flat world generator.
func NewFlatGenerator(height int) TerrainGenerator {
	return &FlatGenerator{
		Height: height,
	}
}

func (g *FlatGenerator) HeightAt(x, z int) int {
	return g.Height
}

func (g *FlatGenerator) PopulateChunk(c *Chunk) {
	chunkBaseY := c.Y * ChunkSizeY
	flatHeight := g.Height

	for lx := range ChunkSizeX {
		for lz := range ChunkSizeZ {
			maxY := flatHeight - chunkBaseY

			// If the entire chunk is above the flat height, do nothing (air)
			if maxY < 0 {
				continue
			}

			// Clamp to chunk top
			limit := maxY
			if limit >= ChunkSizeY {
				limit = ChunkSizeY - 1
			}

			for ly := 0; ly <= limit; ly++ {
				y := chunkBaseY + ly
				if y == 0 {
					c.SetBlock(lx, ly, lz, BlockTypeBedrock)
				} else if y < flatHeight {
					c.SetBlock(lx, ly, lz, BlockTypeDirt)
				} else {
					// y == flatHeight
					c.SetBlock(lx, ly, lz, BlockTypeGrass)
				}
			}
		}
	}
	c.dirty = true
}
