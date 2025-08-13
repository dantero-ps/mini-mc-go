package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

// World represents the game world, composed of chunks
type World struct {
	// Map of chunks indexed by their coordinates
	chunks map[ChunkCoord]*Chunk
}

// ChunkCoord is a unique identifier for a chunk based on its position
type ChunkCoord struct {
	X, Y, Z int
}

// New creates a new world
func New() *World {
	w := &World{
		chunks: make(map[ChunkCoord]*Chunk),
	}

	// Initialize a more interesting world with multiple chunks

	// Create a flat grass layer
	for x := -16; x <= 16; x++ {
		for z := -16; z <= 16; z++ {
			w.Set(x, 0, z, BlockTypeGrass)
		}
	}

	// Create some hills
	for x := -8; x <= 8; x++ {
		for z := -8; z <= 8; z++ {
			// Calculate height based on distance from center
			height := 3 - (abs(x)+abs(z))/4
			if height > 0 {
				for y := 1; y <= height; y++ {
					w.Set(x, y, z, BlockTypeGrass)
				}
			}
		}
	}

	// Create a tower
	for y := 1; y <= 8; y++ {
		w.Set(5, y, 5, BlockTypeGrass)
		w.Set(5, y, 6, BlockTypeGrass)
		w.Set(6, y, 5, BlockTypeGrass)
		w.Set(6, y, 6, BlockTypeGrass)
	}

	return w
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetChunk returns the chunk at the specified chunk coordinates
// If the chunk doesn't exist and create is true, it will be created
func (w *World) GetChunk(chunkX, chunkY, chunkZ int, create bool) *Chunk {
	coord := ChunkCoord{X: chunkX, Y: chunkY, Z: chunkZ}
	chunk, exists := w.chunks[coord]

	if !exists && create {
		chunk = NewChunk(chunkX, chunkY, chunkZ)
		w.chunks[coord] = chunk
	}

	return chunk
}

// GetChunkFromBlockCoords returns the chunk containing the block at the specified world coordinates
func (w *World) GetChunkFromBlockCoords(x, y, z int, create bool) *Chunk {
	// Convert world coordinates to chunk coordinates
	chunkX := floorDiv(x, ChunkSizeX)
	chunkY := floorDiv(y, ChunkSizeY)
	chunkZ := floorDiv(z, ChunkSizeZ)

	return w.GetChunk(chunkX, chunkY, chunkZ, create)
}

// Get returns the block type at the specified world coordinates
func (w *World) Get(x, y, z int) BlockType {
	chunk := w.GetChunkFromBlockCoords(x, y, z, false)
	if chunk == nil {
		return BlockTypeAir
	}

	// Convert world coordinates to local chunk coordinates
	localX := mod(x, ChunkSizeX)
	localY := mod(y, ChunkSizeY)
	localZ := mod(z, ChunkSizeZ)

	return chunk.GetBlock(localX, localY, localZ)
}

// IsAir checks if the block at the specified world coordinates is air
func (w *World) IsAir(x, y, z int) bool {
	return w.Get(x, y, z) == BlockTypeAir
}

// Set sets the block type at the specified world coordinates
func (w *World) Set(x, y, z int, val BlockType) {
	chunk := w.GetChunkFromBlockCoords(x, y, z, true)

	// Convert world coordinates to local chunk coordinates
	localX := mod(x, ChunkSizeX)
	localY := mod(y, ChunkSizeY)
	localZ := mod(z, ChunkSizeZ)

	chunk.SetBlock(localX, localY, localZ, val)
}

// GetActiveBlocks returns a list of positions of all non-air blocks in the world
func (w *World) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3

	// Collect active blocks from all chunks
	for _, chunk := range w.chunks {
		positions = append(positions, chunk.GetActiveBlocks()...)
	}

	return positions
}

// ChunkWithCoord pairs a chunk with its coordinates
type ChunkWithCoord struct {
	Chunk *Chunk
	Coord ChunkCoord
}

// GetAllChunks returns a slice of all chunks in the world with their coordinates
func (w *World) GetAllChunks() []ChunkWithCoord {
	chunks := make([]ChunkWithCoord, 0, len(w.chunks))
	for coord, chunk := range w.chunks {
		chunks = append(chunks, ChunkWithCoord{
			Chunk: chunk,
			Coord: coord,
		})
	}
	return chunks
}

// Helper functions for coordinate conversion

// floorDiv performs integer division that rounds down for negative numbers
func floorDiv(a, b int) int {
	if a < 0 {
		return (a - b + 1) / b
	}
	return a / b
}

// mod returns the remainder of a/b, always positive
func mod(a, b int) int {
	result := a % b
	if result < 0 {
		result += b
	}
	return result
}
