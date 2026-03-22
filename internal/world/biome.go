package world

// TreeType specifies what kind of trees to generate in a biome.
type TreeType uint8

const (
	TreeNone   TreeType = iota
	TreeOak             // Standard oak tree (forest, jungle)
	TreeSpruce          // Conical spruce/pine tree (taiga)
)

// Biome defines terrain generation and surface properties.
type Biome struct {
	ID          int
	Name        string
	MinHeight   float64   // Base terrain height (MC: biome depth). Higher = taller terrain.
	MaxHeight   float64   // Terrain height variation (MC: biome scale). Higher = rougher.
	TopBlock    BlockType // Surface top block
	FillerBlock BlockType // Sub-surface filler block (3-4 layers)
	Temperature float64   // 0.0=cold, 2.0=hot (MC convention)
	Rainfall    float64   // 0.0=dry, 1.0=wet
	Trees       TreeType  // Tree type to place during decoration
	TreeCount   uint8     // Trees attempted per chunk
}

// All biome definitions use MC 1.8.9 authentic MinHeight/MaxHeight parameters.
var (
	BiomeOcean = &Biome{
		ID: 0, Name: "Ocean",
		MinHeight: -1.0, MaxHeight: 0.1,
		TopBlock: BlockTypeSand, FillerBlock: BlockTypeSand,
		Temperature: 0.5, Rainfall: 0.5,
	}
	BiomePlains = &Biome{
		ID: 1, Name: "Plains",
		MinHeight: 0.125, MaxHeight: 0.05,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.8, Rainfall: 0.4,
	}
	BiomeDesert = &Biome{
		ID: 2, Name: "Desert",
		MinHeight: 0.125, MaxHeight: 0.05,
		TopBlock: BlockTypeSand, FillerBlock: BlockTypeSand,
		Temperature: 2.0, Rainfall: 0.0,
	}
	BiomeExtremeHills = &Biome{
		ID: 3, Name: "Extreme Hills",
		MinHeight: 1.0, MaxHeight: 0.5,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.2, Rainfall: 0.3,
		Trees: TreeOak, TreeCount: 3,
	}
	BiomeForest = &Biome{
		ID: 4, Name: "Forest",
		MinHeight: 0.1, MaxHeight: 0.2,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.7, Rainfall: 0.8,
		Trees: TreeOak, TreeCount: 10,
	}
	BiomeTaiga = &Biome{
		ID: 5, Name: "Taiga",
		MinHeight: 0.2, MaxHeight: 0.2,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.25, Rainfall: 0.8,
		Trees: TreeSpruce, TreeCount: 10,
	}
	BiomeSwamp = &Biome{
		ID: 6, Name: "Swampland",
		MinHeight: -0.2, MaxHeight: 0.1,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.8, Rainfall: 0.9,
		Trees: TreeOak, TreeCount: 2,
	}
	BiomeSavanna = &Biome{
		ID: 35, Name: "Savanna",
		MinHeight: 0.125, MaxHeight: 0.05,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 1.2, Rainfall: 0.0,
	}
	BiomeJungle = &Biome{
		ID: 21, Name: "Jungle",
		MinHeight: 0.1, MaxHeight: 0.2,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.95, Rainfall: 0.9,
		Trees: TreeOak, TreeCount: 50,
	}
	BiomeBirchForest = &Biome{
		ID: 27, Name: "Birch Forest",
		MinHeight: 0.1, MaxHeight: 0.2,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.6, Rainfall: 0.6,
		Trees: TreeOak, TreeCount: 10,
	}
	BiomeForestHills = &Biome{
		ID: 18, Name: "Forest Hills",
		MinHeight: 0.45, MaxHeight: 0.3,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.7, Rainfall: 0.8,
		Trees: TreeOak, TreeCount: 10,
	}
	BiomeTaigaHills = &Biome{
		ID: 19, Name: "Taiga Hills",
		MinHeight: 0.45, MaxHeight: 0.3,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.25, Rainfall: 0.8,
		Trees: TreeSpruce, TreeCount: 10,
	}
	BiomeColdTaiga = &Biome{
		ID: 30, Name: "Cold Taiga",
		MinHeight: 0.2, MaxHeight: 0.2,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: -0.5, Rainfall: 0.4,
		Trees: TreeSpruce, TreeCount: 10,
	}
	BiomeIcePlains = &Biome{
		ID: 12, Name: "Ice Plains",
		MinHeight: 0.125, MaxHeight: 0.05,
		TopBlock: BlockTypeGrass, FillerBlock: BlockTypeDirt,
		Temperature: 0.0, Rainfall: 0.5,
	}
	BiomeDeepOcean = &Biome{
		ID: 24, Name: "Deep Ocean",
		MinHeight: -1.8, MaxHeight: 0.1,
		TopBlock: BlockTypeSand, FillerBlock: BlockTypeSand,
		Temperature: 0.5, Rainfall: 0.5,
	}
)

// GetBiomeForCoords returns the biome at the given world coordinates.
// Uses three noise layers (continent, temperature, rainfall) to produce
// a Minecraft-style climate map — the simplest faithful approximation of
// the MC 1.8.9 GenLayer system without the 40+ layer stack.
func GetBiomeForCoords(x, z float64, seed int64) *Biome {
	// 1. Continent noise: island/ocean separation (large scale ~1800 blocks)
	continent := octaveNoise2D(x/1800.0, z/1800.0, seed, 3, 0.5, 2.0)

	// Deep ocean threshold: very low continent value
	if continent < 0.32 {
		return BiomeDeepOcean
	}
	if continent < 0.42 {
		return BiomeOcean
	}

	// 2. Temperature noise (scale ~1600 blocks, different seed)
	temp := octaveNoise2D(x/1600.0, z/1600.0, seed+71, 4, 0.5, 2.0) // [0, 1]

	// 3. Rainfall noise (scale ~1400 blocks, independent seed)
	rain := octaveNoise2D(x/1400.0, z/1400.0, seed+213, 4, 0.5, 2.0) // [0, 1]

	// 4. Hilliness noise: drives Extreme Hills placement (scale ~900 blocks)
	hilliness := octaveNoise2D(x/900.0, z/900.0, seed+389, 2, 0.5, 2.0) // [0, 1]

	// Extreme Hills override: high terrain, cool climate, inland
	if hilliness > 0.72 && temp < 0.6 && continent > 0.52 {
		return BiomeExtremeHills
	}

	// Climate table mapped from (temp, rain) → biome.
	// temp: 0=cold (arctic), 1=hot (tropical)
	// rain: 0=dry (desert), 1=wet (rainforest)

	switch {
	// Hot climate (temp > 0.68)
	case temp > 0.68:
		if rain < 0.28 {
			return BiomeDesert
		} else if rain < 0.65 {
			return BiomeSavanna
		}
		return BiomeJungle

	// Warm climate (temp 0.42–0.68)
	case temp > 0.42:
		if rain < 0.28 {
			return BiomePlains
		} else if rain < 0.52 {
			return BiomeBirchForest
		} else if rain < 0.72 {
			if hilliness > 0.58 {
				return BiomeForestHills
			}
			return BiomeForest
		}
		return BiomeSwamp

	// Cool climate (temp 0.22–0.42)
	case temp > 0.22:
		if rain < 0.35 {
			return BiomePlains
		} else if hilliness > 0.60 {
			return BiomeTaigaHills
		}
		return BiomeTaiga

	// Cold climate (temp ≤ 0.22)
	default:
		if rain < 0.45 {
			return BiomeIcePlains
		}
		return BiomeColdTaiga
	}
}
