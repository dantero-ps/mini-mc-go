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
)

// Block data
const (
	BlockSize = 1.0
)

var (
	// Cube vertices with position and normal attributes
	CubeVertices = []float32{
		// NORTH
		-0.5, -0.5, 0.5, 0, 0, 1,
		0.5, -0.5, 0.5, 0, 0, 1,
		0.5, 0.5, 0.5, 0, 0, 1,
		0.5, 0.5, 0.5, 0, 0, 1,
		-0.5, 0.5, 0.5, 0, 0, 1,
		-0.5, -0.5, 0.5, 0, 0, 1,

		// SOUTH
		0.5, -0.5, -0.5, 0, 0, -1,
		-0.5, -0.5, -0.5, 0, 0, -1,
		-0.5, 0.5, -0.5, 0, 0, -1,
		-0.5, 0.5, -0.5, 0, 0, -1,
		0.5, 0.5, -0.5, 0, 0, -1,
		0.5, -0.5, -0.5, 0, 0, -1,

		// WEST
		-0.5, -0.5, -0.5, -1, 0, 0,
		-0.5, -0.5, 0.5, -1, 0, 0,
		-0.5, 0.5, 0.5, -1, 0, 0,
		-0.5, 0.5, 0.5, -1, 0, 0,
		-0.5, 0.5, -0.5, -1, 0, 0,
		-0.5, -0.5, -0.5, -1, 0, 0,

		// EAST
		0.5, -0.5, 0.5, 1, 0, 0,
		0.5, -0.5, -0.5, 1, 0, 0,
		0.5, 0.5, -0.5, 1, 0, 0,
		0.5, 0.5, -0.5, 1, 0, 0,
		0.5, 0.5, 0.5, 1, 0, 0,
		0.5, -0.5, 0.5, 1, 0, 0,

		// TOP
		-0.5, 0.5, 0.5, 0, 1, 0,
		0.5, 0.5, 0.5, 0, 1, 0,
		0.5, 0.5, -0.5, 0, 1, 0,
		0.5, 0.5, -0.5, 0, 1, 0,
		-0.5, 0.5, -0.5, 0, 1, 0,
		-0.5, 0.5, 0.5, 0, 1, 0,

		// BOTTOM
		-0.5, -0.5, -0.5, 0, -1, 0,
		0.5, -0.5, -0.5, 0, -1, 0,
		0.5, -0.5, 0.5, 0, -1, 0,
		0.5, -0.5, 0.5, 0, -1, 0,
		-0.5, -0.5, 0.5, 0, -1, 0,
		-0.5, -0.5, -0.5, 0, -1, 0,
	}

	// Wireframe cube edges for highlighting
	CubeWireframeVertices = []float32{
		-0.5, -0.5, -0.5, 0.5, -0.5, -0.5,
		0.5, -0.5, -0.5, 0.5, -0.5, 0.5,
		0.5, -0.5, 0.5, -0.5, -0.5, 0.5,
		-0.5, -0.5, 0.5, -0.5, -0.5, -0.5,
		-0.5, 0.5, -0.5, 0.5, 0.5, -0.5,
		0.5, 0.5, -0.5, 0.5, 0.5, 0.5,
		0.5, 0.5, 0.5, -0.5, 0.5, 0.5,
		-0.5, 0.5, 0.5, -0.5, 0.5, -0.5,
		-0.5, -0.5, -0.5, -0.5, 0.5, -0.5,
		0.5, -0.5, -0.5, 0.5, 0.5, -0.5,
		0.5, -0.5, 0.5, 0.5, 0.5, 0.5,
		-0.5, -0.5, 0.5, -0.5, 0.5, 0.5,
	}
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
