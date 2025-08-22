package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	// Chunk dimensions
	ChunkSizeX = 16
	ChunkSizeY = 256
	ChunkSizeZ = 16
)

// Chunk represents a 16x256x16 section of the world
type Chunk struct {
	// Position of the chunk in chunk coordinates (not block coordinates)
	X, Y, Z int
	// 3D array of block types
	blocks [ChunkSizeX][ChunkSizeY][ChunkSizeZ]BlockType
	// Flag to track if the chunk has been modified since last render
	dirty bool
}

// NewChunk creates a new chunk at the specified chunk coordinates
func NewChunk(x, y, z int) *Chunk {
	return &Chunk{
		X:     x,
		Y:     y,
		Z:     z,
		dirty: true,
	}
}

// GetBlock returns the block type at the specified local coordinates
func (c *Chunk) GetBlock(x, y, z int) BlockType {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return BlockTypeAir
	}
	return c.blocks[x][y][z]
}

// SetBlock sets the block type at the specified local coordinates
func (c *Chunk) SetBlock(x, y, z int, blockType BlockType) {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return
	}
	c.blocks[x][y][z] = blockType
	c.dirty = true
}

// IsAir checks if the block at the specified local coordinates is air
func (c *Chunk) IsAir(x, y, z int) bool {
	return c.GetBlock(x, y, z) == BlockTypeAir
}

// IsDirty returns whether the chunk has been modified since last render
func (c *Chunk) IsDirty() bool {
	return c.dirty
}

// SetClean marks the chunk as clean (not modified)
func (c *Chunk) SetClean() {
	c.dirty = false
}

// GetActiveBlocks returns a list of positions of non-air blocks in this chunk
func (c *Chunk) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3

	// Calculate world position offset for this chunk
	worldOffsetX := c.X * ChunkSizeX
	worldOffsetY := c.Y * ChunkSizeY
	worldOffsetZ := c.Z * ChunkSizeZ

	for x := 0; x < ChunkSizeX; x++ {
		for y := 0; y < ChunkSizeY; y++ {
			for z := 0; z < ChunkSizeZ; z++ {
				if c.blocks[x][y][z] != BlockTypeAir {
					worldX := float32(worldOffsetX + x)
					worldY := float32(worldOffsetY + y)
					worldZ := float32(worldOffsetZ + z)
					positions = append(positions, mgl32.Vec3{worldX, worldY, worldZ})
				}
			}
		}
	}

	return positions
}
