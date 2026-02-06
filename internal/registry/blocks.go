package registry

import (
	"fmt"
	"mini-mc/internal/world"
	"mini-mc/pkg/blockmodel"
	"os"
	"path/filepath"
	"sort"
)

// BlockDefinition defines the properties of a block type
type BlockDefinition struct {
	ID            world.BlockType
	Name          string
	TextureTop    string
	TextureSide   string
	TextureBot    string
	IsSolid       bool
	IsTransparent bool
	TintColor     uint32
	TintFaces     map[world.BlockFace]bool
	Hardness      float32
	Elements      []blockmodel.Element
}

var (
	Blocks       = make(map[world.BlockType]*BlockDefinition)
	BlockNames   = make(map[string]world.BlockType)
	TextureNames []string
	TextureMap   = make(map[string]int)
	ModelLoader  *blockmodel.Loader
)

func RegisterBlock(def *BlockDefinition) {
	if ModelLoader != nil && def.Name != "air" {
		loadTexturesFromModel(def)
	}

	Blocks[def.ID] = def
	BlockNames[def.Name] = def.ID

	registerTexture(def.TextureTop)
	registerTexture(def.TextureSide)
	registerTexture(def.TextureBot)
}

func loadTexturesFromModel(def *BlockDefinition) {
	bs, err := ModelLoader.LoadBlockState(def.Name)
	if err != nil {
		fmt.Printf("Warning: Failed to load blockstate for %s: %v\n", def.Name, err)
		return
	}

	var modelName string
	if v, ok := bs.Variants["normal"]; ok && len(v) > 0 {
		modelName = v[0].Model
	} else if v, ok := bs.Variants[""]; ok && len(v) > 0 {
		modelName = v[0].Model
	} else {
		// Pick first alphabetically to ensure deterministic behavior (fix race condition)
		var keys []string
		for k := range bs.Variants {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		if len(keys) > 0 {
			if v, ok := bs.Variants[keys[0]]; ok && len(v) > 0 {
				modelName = v[0].Model
			}
		}
	}

	if modelName == "" {
		return
	}

	model, err := ModelLoader.LoadModel(modelName)
	if err != nil {
		fmt.Printf("Warning: Failed to load model %s for block %s: %v\n", modelName, def.Name, err)
		return
	}

	// Store the elements for rendering usage
	def.Elements = model.Elements

	// Infer physical properties from elements
	// If ANY element is a full opaque cube, we consider the block solid.
	isFullBlock := false
	if len(model.Elements) > 0 {
		epsilon := float32(0.001)
		isZero := func(v [3]float32) bool {
			return v[0] > -epsilon && v[0] < epsilon &&
				v[1] > -epsilon && v[1] < epsilon &&
				v[2] > -epsilon && v[2] < epsilon
		}
		isSixteen := func(v [3]float32) bool {
			return v[0] > 16.0-epsilon && v[0] < 16.0+epsilon &&
				v[1] > 16.0-epsilon && v[1] < 16.0+epsilon &&
				v[2] > 16.0-epsilon && v[2] < 16.0+epsilon
		}

		for _, e := range model.Elements {
			if isZero(e.From) && isSixteen(e.To) {
				isFullBlock = true
				break
			}
		}
	} else if def.IsSolid {
		// No elements parsed (maybe manual block?) but labeled IsSolid. Keep it.
		isFullBlock = true
	}

	// If it's not a full block, force non-solid/transparent.
	// But if it IS a full block (like Grass which has 1 full element + 1 overlay),
	// we keep the IsSolid=true from registration.
	if !isFullBlock {
		def.IsSolid = false
		def.IsTransparent = true
	}

	for _, elem := range model.Elements {
		if face, ok := elem.Faces["up"]; ok {
			def.TextureTop = resolveTextureName(face.Texture, model)
		}
		if face, ok := elem.Faces["down"]; ok {
			def.TextureBot = resolveTextureName(face.Texture, model)
		}

		sideFaces := []string{"north", "south", "east", "west"}
		for _, side := range sideFaces {
			if face, ok := elem.Faces[side]; ok {
				def.TextureSide = resolveTextureName(face.Texture, model)
				if def.TextureSide != "" {
					break
				}
			}
		}

		if def.TextureTop != "" && def.TextureBot != "" && def.TextureSide != "" {
			break
		}
	}
}

func resolveTextureName(ref string, model *blockmodel.Model) string {
	resolved := ModelLoader.ResolveTexture(ref, model)
	base := filepath.Base(resolved)
	if base == "." || base == "/" {
		return ""
	}
	return base + ".png"
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
	cwd, _ := os.Getwd()
	assetsDir := filepath.Join(cwd, "assets")
	ModelLoader = blockmodel.NewLoader(assetsDir)

	registerTexture("grass_top.png")
	registerTexture("grass_side.png")
	registerTexture("dirt.png")

	RegisterBlock(&BlockDefinition{
		ID:            world.BlockTypeAir,
		Name:          "air",
		IsSolid:       false,
		IsTransparent: true,
	})

	RegisterBlock(&BlockDefinition{
		ID:        world.BlockTypeGrass,
		Name:      "grass",
		IsSolid:   true,
		TintColor: 0x7DFF5C,
		TintFaces: map[world.BlockFace]bool{world.FaceTop: true},
		Hardness:  0.6,
	})

	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypeDirt,
		Name:     "dirt",
		IsSolid:  true,
		Hardness: 0.5,
	})

	// Stone
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypeStone,
		Name:     "stone",
		IsSolid:  true,
		Hardness: 1.5,
	})

	// Bedrock
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypeBedrock,
		Name:     "bedrock",
		IsSolid:  true,
		Hardness: -1.0, // Unbreakable
	})

	// Stone Brick
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypeStoneBrick,
		Name:     "stonebrick",
		IsSolid:  true,
		Hardness: 1.5,
	})

	// Oak Planks
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypePlanksOak,
		Name:     "oak_planks", // Changed from "planks_oak" to match json?
		IsSolid:  true,
		Hardness: 2.0,
	})
	// ... need to fix naming for other planks too

	// Birch Planks
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypePlanksBirch,
		Name:     "birch_planks",
		IsSolid:  true,
		Hardness: 2.0,
	})

	// Spruce Planks
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypePlanksSpruce,
		Name:     "spruce_planks",
		IsSolid:  true,
		Hardness: 2.0,
	})

	// Jungle Planks
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypePlanksJungle,
		Name:     "jungle_planks",
		IsSolid:  true,
		Hardness: 2.0,
	})

	// Acacia Planks
	RegisterBlock(&BlockDefinition{
		ID:       world.BlockTypePlanksAcacia,
		Name:     "acacia_planks",
		IsSolid:  true,
		Hardness: 2.0,
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
