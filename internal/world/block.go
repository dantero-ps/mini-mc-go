package world

type BlockType uint8

const (
	BlockTypeAir BlockType = iota
	BlockTypeGrass
	BlockTypeDirt
	BlockTypeStone
	BlockTypeBedrock
	BlockTypeStoneBrick
	BlockTypePlanksOak
	BlockTypePlanksBirch
	BlockTypePlanksSpruce
	BlockTypePlanksJungle
	BlockTypePlanksAcacia
	BlockTypeCobblestone
	BlockTypeWater
	BlockTypeLava
	BlockTypeObsidian
	BlockTypeSand
	BlockTypeOakLog
	BlockTypeOakLeaves
	BlockTypeSpruceLog
	BlockTypeSpruceLeaves
)

// BlockSolidTable is a flat lookup indexed by BlockType (uint8).
// true = block is solid (blocks fluid flow). Populated by the registry package
// after all blocks are registered, so that the world package does not need to
// import registry (which would create an import cycle).
var BlockSolidTable [256]bool

// BlockFluidTable is a flat lookup indexed by BlockType.
// true = block is a fluid (water or lava). Useful for fast checks in hot paths.
var BlockFluidTable [256]bool

// BlockFace identifies a face of a block
type BlockFace int

const (
	FaceNorth BlockFace = iota
	FaceSouth
	FaceEast
	FaceWest
	FaceTop
	FaceBottom
)
