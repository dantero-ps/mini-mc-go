package registry

import "mini-mc/internal/world"

// BlockDefinition defines the properties of a block type
type BlockDefinition struct {
	ID            world.BlockType
	Name          string
	TextureTop    string
	TextureSide   string
	TextureBot    string
	IsSolid       bool
	IsTransparent bool
	TintColor     uint32                   // 0xRRGGBB (0 = No Tint)
	TintFaces     map[world.BlockFace]bool // Set of faces to apply tint
	Hardness      float32                  // Seconds to break (approximate)
}

// Global registry
var (
	Blocks       = make(map[world.BlockType]*BlockDefinition)
	BlockNames   = make(map[string]world.BlockType)
	TextureNames []string
	TextureMap   = make(map[string]int) // Texture Name -> Layer Index
)

func RegisterBlock(def *BlockDefinition) {
	Blocks[def.ID] = def
	BlockNames[def.Name] = def.ID

	// Collect unique textures
	registerTexture(def.TextureTop)
	registerTexture(def.TextureSide)
	registerTexture(def.TextureBot)
}

func registerTexture(name string) {
	if name == "" {
		return
	}
	if _, exists := TextureMap[name]; !exists {
		TextureMap[name] = len(TextureNames)
		TextureNames = append(TextureNames, name)
	}
}

func InitRegistry() {
	// Pre-register critical textures to ensure specific ID order
	// This guarantees grass_top is ID 0, grass_side is ID 1, dirt is ID 2
	registerTexture("grass_top.png")
	registerTexture("grass_side.png")
	registerTexture("dirt.png")

	// Air
	RegisterBlock(&BlockDefinition{
		ID:            world.BlockTypeAir,
		Name:          "air",
		IsSolid:       false,
		IsTransparent: true,
	})

	// Grass
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeGrass,
		Name:        "grass",
		TextureTop:  "grass_top.png",
		TextureSide: "grass_side.png",
		TextureBot:  "dirt.png",
		IsSolid:     true,
		TintColor:   0x7DFF5C, // Grass Green
		TintFaces:   map[world.BlockFace]bool{world.FaceTop: true},
		Hardness:    0.6,
	})

	// Dirt
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeDirt,
		Name:        "dirt",
		TextureTop:  "dirt.png",
		TextureSide: "dirt.png",
		TextureBot:  "dirt.png",
		IsSolid:     true,
		Hardness:    0.5,
	})

	// Stone
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeStone,
		Name:        "stone",
		TextureTop:  "stone.png",
		TextureSide: "stone.png",
		TextureBot:  "stone.png",
		IsSolid:     true,
		Hardness:    1.5,
	})

	// Bedrock
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeBedrock,
		Name:        "bedrock",
		TextureTop:  "bedrock.png",
		TextureSide: "bedrock.png",
		TextureBot:  "bedrock.png",
		IsSolid:     true,
		Hardness:    -1.0, // Unbreakable
	})

	// Stone Brick
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeStoneBrick,
		Name:        "stonebrick",
		TextureTop:  "stonebrick.png",
		TextureSide: "stonebrick.png",
		TextureBot:  "stonebrick.png",
		IsSolid:     true,
		Hardness:    1.5,
	})

	// Oak Planks
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypePlanksOak,
		Name:        "planks_oak",
		TextureTop:  "planks_oak.png",
		TextureSide: "planks_oak.png",
		TextureBot:  "planks_oak.png",
		IsSolid:     true,
		Hardness:    2.0,
	})

	// Birch Planks
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypePlanksBirch,
		Name:        "planks_birch",
		TextureTop:  "planks_birch.png",
		TextureSide: "planks_birch.png",
		TextureBot:  "planks_birch.png",
		IsSolid:     true,
		Hardness:    2.0,
	})

	// Spruce Planks
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypePlanksSpruce,
		Name:        "planks_spruce",
		TextureTop:  "planks_spruce.png",
		TextureSide: "planks_spruce.png",
		TextureBot:  "planks_spruce.png",
		IsSolid:     true,
		Hardness:    2.0,
	})

	// Jungle Planks
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypePlanksJungle,
		Name:        "planks_jungle",
		TextureTop:  "planks_jungle.png",
		TextureSide: "planks_jungle.png",
		TextureBot:  "planks_jungle.png",
		IsSolid:     true,
		Hardness:    2.0,
	})

	// Acacia Planks
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypePlanksAcacia,
		Name:        "planks_acacia",
		TextureTop:  "planks_acacia.png",
		TextureSide: "planks_acacia.png",
		TextureBot:  "planks_acacia.png",
		IsSolid:     true,
		Hardness:    2.0,
	})
}

// GetTextureLayer returns the texture layer index for a given block and face
func GetTextureLayer(blockType world.BlockType, face world.BlockFace) int {
	def, ok := Blocks[blockType]
	if !ok {
		return 0 // Fallback/Error texture
	}

	var texName string
	switch face {
	case world.FaceTop:
		texName = def.TextureTop
	case world.FaceBottom:
		texName = def.TextureBot
	default:
		texName = def.TextureSide
	}

	if idx, ok := TextureMap[texName]; ok {
		return idx
	}
	return 0
}
