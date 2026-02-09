package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

// Ticker interface for updating entities (avoids circular dependency with entity package)
type Ticker interface {
	Update(dt float64)
	IsDead() bool
	SetDead()
	Position() mgl32.Vec3
}

// ItemEntityConfigurator is called when adding ItemEntity to configure it with world services
// This avoids import cycles by using a function reference set from the entity package
var ItemEntityConfigurator func(item Ticker, world interface{})

// World represents the game world, composed of chunks
type World struct {
	// Components
	store    *ChunkStore
	entities *EntityManager
	gen      TerrainGenerator
	streamer *ChunkStreamer
}

// ChunkCoord is a unique identifier for a chunk based on its position
type ChunkCoord struct {
	X, Y, Z int
}

// New creates a new world.
func New() *World {
	store := NewChunkStore()
	entities := NewEntityManager()
	gen := NewChunkProvider189(1337)
	streamer := NewChunkStreamer(store, gen)

	return &World{
		store:    store,
		entities: entities,
		gen:      gen,
		streamer: streamer,
	}
}

// NewEmpty creates an empty world.
func NewEmpty() *World {
	return New()
}

// Close stops the background generation workers
func (w *World) Close() {
	w.streamer.Close()
}

// AddEntity adds an entity to the world
func (w *World) AddEntity(e Ticker) {
	// If the configurator is set and this is an ItemEntity, configure it
	if ItemEntityConfigurator != nil {
		ItemEntityConfigurator(e, w)
	}
	w.entities.Add(e)
}

// UpdateEntities updates all entities and removes dead ones
func (w *World) UpdateEntities(dt float64) {
	w.entities.Update(dt)
}

// GetEntities returns a safe copy of the current entities in the world
func (w *World) GetEntities() []Ticker {
	return w.entities.GetAll()
}

// GetNearbyEntities returns entities within a box centered at (cx, cy, cz) with ranges (rx, ry, rz).
// Returns []interface{} to avoid import cycles - callers type-assert to specific entity types.
func (w *World) GetNearbyEntities(cx, cy, cz, rx, ry, rz float32) []interface{} {
	minX := cx - rx
	maxX := cx + rx
	minY := cy - ry
	maxY := cy + ry
	minZ := cz - rz
	maxZ := cz + rz

	tickers := w.entities.GetEntitiesInAABB(minX, minY, minZ, maxX, maxY, maxZ)

	result := make([]interface{}, len(tickers))
	for i, t := range tickers {
		result[i] = t
	}
	return result
}

// GetChunk returns the chunk at the specified chunk coordinates
func (w *World) GetChunk(chunkX, chunkY, chunkZ int, create bool) *Chunk {
	return w.store.GetChunk(chunkX, chunkY, chunkZ, create)
}

// GetChunkFromBlockCoords returns the chunk containing the block at the specified world coordinates
func (w *World) GetChunkFromBlockCoords(x, y, z int, create bool) *Chunk {
	return w.store.GetChunkFromBlockCoords(x, y, z, create)
}

// Get returns the block type at the specified world coordinates
func (w *World) Get(x, y, z int) BlockType {
	return w.store.Get(x, y, z)
}

// IsAir checks if the block at the specified world coordinates is air
func (w *World) IsAir(x, y, z int) bool {
	return w.store.IsAir(x, y, z)
}

// Set sets the block type at the specified world coordinates
func (w *World) Set(x, y, z int, val BlockType) {
	w.store.Set(x, y, z, val)
}

// GetActiveBlocks returns a list of positions of all non-air blocks in the world
func (w *World) GetActiveBlocks() []mgl32.Vec3 {
	return w.store.GetActiveBlocks()
}

// ChunkWithCoord pairs a chunk with its coordinates
type ChunkWithCoord struct {
	Chunk *Chunk
	Coord ChunkCoord
}

// GetAllChunks returns a slice of all chunks in the world with their coordinates
func (w *World) GetAllChunks() []ChunkWithCoord {
	return w.store.GetAllChunks()
}

// StreamChunksAroundSync synchronously generates chunks around a world position (x,z) within radius
func (w *World) StreamChunksAroundSync(x, z float32, radius int) {
	w.streamer.StreamChunksAroundSync(x, z, radius)
}

// StreamChunksAroundAsync enqueues async generation around a world position (x,z) within radius
func (w *World) StreamChunksAroundAsync(x, z float32, radius int) {
	w.streamer.StreamChunksAroundAsync(x, z, radius)
}

// EvictFarChunks removes chunks outside the given radius (in chunks) from the center (world x,z).
func (w *World) EvictFarChunks(x, z float32, radius int) int {
	return w.streamer.EvictFarChunks(x, z, radius)
}

// SurfaceHeightAt exposes the terrain surface height used for generation at world (x,z).
func (w *World) SurfaceHeightAt(x, z int) int {
	return w.gen.HeightAt(x, z)
}

// AppendChunksInRadiusXZ appends all loaded chunks within a radius
func (w *World) AppendChunksInRadiusXZ(cx, cz, radius int, dst []ChunkWithCoord) []ChunkWithCoord {
	return w.store.AppendChunksInRadiusXZ(cx, cz, radius, dst)
}

// GetModCount returns the current modification count of the chunk map
func (w *World) GetModCount() uint64 {
	return w.store.GetModCount()
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
