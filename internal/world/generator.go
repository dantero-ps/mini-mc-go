package world

import (
	"math"
)

// Generator handles terrain generation logic.
type Generator struct {
	seed        int64
	scale       float64
	baseHeight  int
	amp         float64
	octaves     int
	persistence float64
	lacunarity  float64
}

// NewGenerator creates a new generator with default settings.
// Seed can be parameterized here.
func NewGenerator(seed int64) *Generator {
	return &Generator{
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
func (g *Generator) HeightAt(worldX, worldZ int) int {
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
func (g *Generator) PopulateChunk(c *Chunk) {
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
