package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

type BlockType uint16

const (
	BlockTypeAir BlockType = iota
	BlockTypeGrass
	BlockTypeDirt
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

// GetBlockColor returns the color for a specific block face
func GetBlockColor(face BlockFace) mgl32.Vec3 {
	// Different colors for each face
	switch face {
	case FaceNorth:
		return mgl32.Vec3{1.0, 0.0, 0.0} // Red
	case FaceSouth:
		return mgl32.Vec3{0.0, 1.0, 0.0} // Green
	case FaceEast:
		return mgl32.Vec3{0.0, 0.0, 1.0} // Blue
	case FaceWest:
		return mgl32.Vec3{1.0, 1.0, 0.0} // Yellow
	case FaceTop:
		return mgl32.Vec3{1.0, 0.0, 1.0} // Magenta
	case FaceBottom:
		return mgl32.Vec3{0.0, 1.0, 1.0} // Cyan
	default:
		return mgl32.Vec3{0.5, 0.5, 0.5} // Gray (fallback)
	}
}
