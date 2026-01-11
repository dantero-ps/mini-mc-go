package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	// Chunk dimensions
	ChunkSizeX = 16
	ChunkSizeY = 256
	ChunkSizeZ = 16

	// Section dimensions
	SectionHeight = 16
	NumSections   = ChunkSizeY / SectionHeight
	SectionVolume = ChunkSizeX * SectionHeight * ChunkSizeZ
)

// Section represents a 16x16x16 sub-volume of a chunk
type Section struct {
	blocks     []BlockType
	blockCount int
}

// Chunk represents a 16x256x16 section of the world
type Chunk struct {
	// Position of the chunk in chunk coordinates (not block coordinates)
	X, Y, Z int
	// Sections dividing the vertical space
	sections [NumSections]*Section
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

// indexInSection converts local 3D coordinates (x,y,z) relative to section
func indexInSection(x, y, z int) int {
	return x*SectionHeight*ChunkSizeZ + y*ChunkSizeZ + z
}

// GetBlock returns the block type at the specified local coordinates
func (c *Chunk) GetBlock(x, y, z int) BlockType {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return BlockTypeAir
	}

	secIdx := y / SectionHeight
	sec := c.sections[secIdx]
	if sec == nil || sec.blocks == nil {
		return BlockTypeAir
	}

	localY := y % SectionHeight
	return sec.blocks[indexInSection(x, localY, z)]
}

// SetBlock sets the block type at the specified local coordinates
func (c *Chunk) SetBlock(x, y, z int, blockType BlockType) {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return
	}

	secIdx := y / SectionHeight
	localY := y % SectionHeight

	sec := c.sections[secIdx]

	if blockType == BlockTypeAir {
		if sec != nil && sec.blocks != nil {
			idx := indexInSection(x, localY, z)
			old := sec.blocks[idx]
			if old != BlockTypeAir {
				sec.blocks[idx] = BlockTypeAir
				sec.blockCount--
				c.dirty = true

				if sec.blockCount == 0 {
					sec.blocks = nil
					c.sections[secIdx] = nil // Free the section struct too
				}
			}
		}
		return
	}

	if sec == nil {
		sec = &Section{
			blocks: make([]BlockType, SectionVolume),
		}
		c.sections[secIdx] = sec
	} else if sec.blocks == nil {
		sec.blocks = make([]BlockType, SectionVolume)
	}

	idx := indexInSection(x, localY, z)
	old := sec.blocks[idx]
	if old != blockType {
		sec.blocks[idx] = blockType
		if old == BlockTypeAir {
			sec.blockCount++
		}
		c.dirty = true
	}
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

	for secIdx, sec := range c.sections {
		if sec == nil || sec.blocks == nil {
			continue
		}

		offsetY := secIdx * SectionHeight

		for x := 0; x < ChunkSizeX; x++ {
			for y := 0; y < SectionHeight; y++ {
				for z := 0; z < ChunkSizeZ; z++ {
					if sec.blocks[indexInSection(x, y, z)] != BlockTypeAir {
						worldX := float32(worldOffsetX + x)
						worldY := float32(worldOffsetY + offsetY + y)
						worldZ := float32(worldOffsetZ + z)
						positions = append(positions, mgl32.Vec3{worldX, worldY, worldZ})
					}
				}
			}
		}
	}

	return positions
}
