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
)

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
