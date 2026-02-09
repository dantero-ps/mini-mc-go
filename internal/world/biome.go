package world

import (
	"math"
)

// Biome defines the properties of a terrain type
type Biome struct {
	ID          int
	Name        string
	MinHeight   float64   // Base height
	MaxHeight   float64   // Height variation
	TopBlock    BlockType // e.g. Grass
	FillerBlock BlockType // e.g. Dirt (under grass)
}

var (
	BiomePlains = &Biome{
		ID:          1,
		Name:        "Plains",
		MinHeight:   0.1,
		MaxHeight:   0.2,
		TopBlock:    BlockTypeGrass,
		FillerBlock: BlockTypeDirt,
	}
	BiomeForest = &Biome{
		ID:          4,
		Name:        "Forest",
		MinHeight:   0.1,
		MaxHeight:   0.2,
		TopBlock:    BlockTypeGrass,
		FillerBlock: BlockTypeDirt,
	}
	BiomeHills = &Biome{
		ID:          3,
		Name:        "Extreme Hills",
		MinHeight:   0.3,
		MaxHeight:   1.5,
		TopBlock:    BlockTypeGrass,
		FillerBlock: BlockTypeDirt, // Sometimes Stone, but let's stick to Dirt for top
	}
	BiomeMountains = &Biome{
		ID:          3, // Reusing ID for now
		Name:        "Mountains",
		MinHeight:   1.0,
		MaxHeight:   1.0,
		TopBlock:    BlockTypeStone, // Mountains often exposed stone
		FillerBlock: BlockTypeStone,
	}
	BiomeOcean = &Biome{
		ID:          0,
		Name:        "Ocean",
		MinHeight:   -1.0,
		MaxHeight:   0.1,
		TopBlock:    BlockTypeDirt, // Underwater dirt/gravel/sand usually
		FillerBlock: BlockTypeDirt,
	}
)

var Biomes = []*Biome{BiomePlains, BiomeForest, BiomeHills, BiomeMountains, BiomeOcean}

// GetBiomeForCoords returns a deterministic biome for a given world coordinate.
// This is a simplified version using Perlin noise.
// Real MC uses a much more complex layered noise system (Voronoi/Cells).
func GetBiomeForCoords(x, z float64, seed int64) *Biome {
	// Simple Perlin noise to pick biome
	// Scale needs to be very large for biomes (e.g. 1/200 or 1/400)
	scale := 1.0 / 400.0
	val := octaveNoise2D(x*scale, z*scale, seed, 2, 0.5, 2.0)

	// Map [-1, 1] (approx) noise to biomes
	// Since octaveNoise2D returns [0, 1] approximately, let's map it.
	// Actually octaveNoise2D as implemented in noise.go returns [0,1]?
	// Let's check noise.go again. Yes, returns [0,1].

	// 0.0 - 0.3 : Ocean
	// 0.3 - 0.6 : Plains/Forest
	// 0.6 - 0.8 : Hills
	// 0.8 - 1.0 : Mountains

	if val < 0.35 {
		return BiomeOcean
	} else if val < 0.6 {
		if math.Sin(x*0.01) > 0 {
			return BiomePlains
		}
		return BiomeForest
	} else if val < 0.8 {
		return BiomeHills
	} else {
		return BiomeMountains
	}
}
