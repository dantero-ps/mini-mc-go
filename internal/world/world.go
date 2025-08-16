package world

import (
	"math"
	"runtime"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

// World represents the game world, composed of chunks
type World struct {
	// Map of chunks indexed by their coordinates
	chunks map[ChunkCoord]*Chunk
	mu     sync.RWMutex

	// Async generation
	jobs      chan ChunkCoord
	pending   map[ChunkCoord]struct{}
	pendingMu sync.Mutex

	// Noise params
	seed        int64
	scale       float64
	baseHeight  int
	amp         float64
	octaves     int
	persistence float64
	lacunarity  float64
}

// ChunkCoord is a unique identifier for a chunk based on its position
type ChunkCoord struct {
	X, Y, Z int
}

// New creates a new world with async noise generation workers
func New() *World {
	w := &World{
		chunks:      make(map[ChunkCoord]*Chunk),
		jobs:        make(chan ChunkCoord, 1024),
		pending:     make(map[ChunkCoord]struct{}),
		seed:        1337,
		scale:       1.0 / 64.0,
		baseHeight:  32,
		amp:         32,
		octaves:     4,
		persistence: 0.5,
		lacunarity:  2.0,
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go w.worker()
	}

	return w
}

// NewEmpty creates an empty world with no pre-filled terrain (for tests/tools).
func NewEmpty() *World { return New() }

// abs removed (no longer used)

// GetChunk returns the chunk at the specified chunk coordinates
// If the chunk doesn't exist and create is true, it will be created
func (w *World) GetChunk(chunkX, chunkY, chunkZ int, create bool) *Chunk {
	coord := ChunkCoord{X: chunkX, Y: chunkY, Z: chunkZ}
	w.mu.RLock()
	chunk, exists := w.chunks[coord]
	w.mu.RUnlock()
	if !exists && create {
		chunk = NewChunk(chunkX, chunkY, chunkZ)
		w.mu.Lock()
		w.chunks[coord] = chunk
		w.mu.Unlock()
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
	w.mu.RLock()
	for _, chunk := range w.chunks {
		positions = append(positions, chunk.GetActiveBlocks()...)
	}
	w.mu.RUnlock()
	return positions
}

// ChunkWithCoord pairs a chunk with its coordinates
type ChunkWithCoord struct {
	Chunk *Chunk
	Coord ChunkCoord
}

// GetAllChunks returns a slice of all chunks in the world with their coordinates
func (w *World) GetAllChunks() []ChunkWithCoord {
	w.mu.RLock()
	defer w.mu.RUnlock()
	chunks := make([]ChunkWithCoord, 0, len(w.chunks))
	for coord, chunk := range w.chunks {
		chunks = append(chunks, ChunkWithCoord{Chunk: chunk, Coord: coord})
	}
	return chunks
}

// StreamChunksAroundSync synchronously generates chunks around a world position (x,z) within radius (in chunks)
func (w *World) StreamChunksAroundSync(x, z float32, radius int) {
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			chunkX := cx + dx
			chunkZ := cz + dz
			worldX := chunkX*ChunkSizeX + ChunkSizeX/2
			worldZ := chunkZ*ChunkSizeZ + ChunkSizeZ/2
			h := w.heightAt(worldX, worldZ)
			maxChunkY := floorDiv(h, ChunkSizeY)
			if maxChunkY < 0 {
				maxChunkY = 0
			}
			for cy := 0; cy <= maxChunkY; cy++ {
				w.generateChunkSync(ChunkCoord{X: chunkX, Y: cy, Z: chunkZ})
			}
		}
	}
}

// StreamChunksAroundAsync enqueues async generation around a world position (x,z) within radius (in chunks)
func (w *World) StreamChunksAroundAsync(x, z float32, radius int) {
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			chunkX := cx + dx
			chunkZ := cz + dz
			worldX := chunkX*ChunkSizeX + ChunkSizeX/2
			worldZ := chunkZ*ChunkSizeZ + ChunkSizeZ/2
			h := w.heightAt(worldX, worldZ)
			maxChunkY := floorDiv(h, ChunkSizeY)
			if maxChunkY < 0 {
				maxChunkY = 0
			}
			for cy := 0; cy <= maxChunkY; cy++ {
				w.requestChunk(ChunkCoord{X: chunkX, Y: cy, Z: chunkZ})
			}
		}
	}
}

// requestChunk schedules a chunk for async generation if missing
func (w *World) requestChunk(coord ChunkCoord) {
	// skip if already present
	w.mu.RLock()
	_, exists := w.chunks[coord]
	w.mu.RUnlock()
	if exists {
		return
	}
	// skip if already pending
	w.pendingMu.Lock()
	if _, ok := w.pending[coord]; ok {
		w.pendingMu.Unlock()
		return
	}
	w.pending[coord] = struct{}{}
	w.pendingMu.Unlock()

	select {
	case w.jobs <- coord:
	default:
		// queue full: drop and clear pending marker
		w.pendingMu.Lock()
		delete(w.pending, coord)
		w.pendingMu.Unlock()
	}
}

func (w *World) worker() {
	for coord := range w.jobs {
		w.generateChunkSync(coord)
		w.pendingMu.Lock()
		delete(w.pending, coord)
		w.pendingMu.Unlock()
	}
}

// generateChunkSync builds and installs a chunk if missing
func (w *World) generateChunkSync(coord ChunkCoord) {
	// quick check
	w.mu.RLock()
	_, exists := w.chunks[coord]
	w.mu.RUnlock()
	if exists {
		return
	}

	chunk := NewChunk(coord.X, coord.Y, coord.Z)
	w.populateChunk(chunk)

	w.mu.Lock()
	if _, ok := w.chunks[coord]; !ok {
		w.chunks[coord] = chunk
	}
	w.mu.Unlock()
}

// populateChunk fills a chunk using noise heightmap
func (w *World) populateChunk(c *Chunk) {
	chunkBaseY := c.Y * ChunkSizeY
	for lx := 0; lx < ChunkSizeX; lx++ {
		for lz := 0; lz < ChunkSizeZ; lz++ {
			worldX := c.X*ChunkSizeX + lx
			worldZ := c.Z*ChunkSizeZ + lz
			height := w.heightAt(worldX, worldZ)
			topLocal := height - chunkBaseY
			if topLocal < 0 {
				continue
			}
			if topLocal >= ChunkSizeY {
				topLocal = ChunkSizeY - 1
			}
			for ly := 0; ly <= topLocal; ly++ {
				c.blocks[lx][ly][lz] = BlockTypeGrass
			}
		}
	}
	c.dirty = true
}

// heightAt computes world surface height (block Y) at world X,Z
func (w *World) heightAt(worldX, worldZ int) int {
	x := float64(worldX) * w.scale
	z := float64(worldZ) * w.scale
	n := octaveNoise2D(x, z, w.seed, w.octaves, w.persistence, w.lacunarity)
	height := float64(w.baseHeight) + n*w.amp
	if height < 0 {
		height = 0
	}
	return int(math.Floor(height))
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
