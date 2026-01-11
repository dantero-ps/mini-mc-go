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
	})

	// Dirt
	RegisterBlock(&BlockDefinition{
		ID:          world.BlockTypeDirt,
		Name:        "dirt",
		TextureTop:  "dirt.png",
		TextureSide: "dirt.png",
		TextureBot:  "dirt.png",
		IsSolid:     true,
	})

	// Example: Stone (Future)
	/*
		RegisterBlock(&BlockDefinition{
			ID: 3,
			Name: "stone",
			TextureTop: "stone.png", ...
		})
	*/
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
