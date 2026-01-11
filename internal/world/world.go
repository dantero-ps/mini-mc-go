package world

import (
	"math"
	"mini-mc/internal/profiling"
	"runtime"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

// Ticker interface for updating entities (avoids circular dependency with entity package)
type Ticker interface {
	Update(dt float64)
	IsDead() bool
	SetDead()
	Position() mgl32.Vec3
}

// World represents the game world, composed of chunks
type World struct {
	// Map of chunks indexed by their coordinates
	chunks map[ChunkCoord]*Chunk
	mu     sync.RWMutex

	// Entities in the world
	Entities   []Ticker
	EntitiesMu sync.RWMutex

	// Per-column index for fast XZ radius queries: (chunkX,chunkZ) -> slice indexed by chunkY
	colIndex map[[2]int][]*Chunk

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

	// Production caps
	maxJobsPerCall int
	maxPending     int

	// Cached terrain heights per column (chunkX, chunkZ) -> maxChunkY
	heightCache   map[[2]int]int
	heightCacheMu sync.RWMutex
}

// ChunkCoord is a unique identifier for a chunk based on its position
type ChunkCoord struct {
	X, Y, Z int
}

// New creates a new world with async noise generation workers
func New() *World {
	w := &World{
		chunks:         make(map[ChunkCoord]*Chunk),
		Entities:       make([]Ticker, 0),
		jobs:           make(chan ChunkCoord, 4096),
		pending:        make(map[ChunkCoord]struct{}),
		colIndex:       make(map[[2]int][]*Chunk),
		seed:           1337,
		scale:          1.0 / 64.0,
		baseHeight:     32,
		amp:            32,
		octaves:        4,
		persistence:    0.5,
		lacunarity:     2.0,
		maxJobsPerCall: 2048,
		maxPending:     16384,
		heightCache:    make(map[[2]int]int),
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

// Close stops the background generation workers
func (w *World) Close() {
	close(w.jobs)
}

// AddEntity adds an entity to the world
func (w *World) AddEntity(e Ticker) {
	w.EntitiesMu.Lock()
	defer w.EntitiesMu.Unlock()
	w.Entities = append(w.Entities, e)
}

// UpdateEntities updates all entities and removes dead ones
func (w *World) UpdateEntities(dt float64) {
	defer profiling.Track("world.UpdateEntities")()
	w.EntitiesMu.Lock()
	defer w.EntitiesMu.Unlock()

	activeCount := 0
	for i := 0; i < len(w.Entities); i++ {
		e := w.Entities[i]
		if !e.IsDead() {
			e.Update(dt)
			if !e.IsDead() {
				// Keep alive
				w.Entities[activeCount] = e
				activeCount++
			}
		}
	}
	// Trim slice to remove dead entities
	w.Entities = w.Entities[:activeCount]
}

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
		// maintain column index
		key := [2]int{chunkX, chunkZ}
		col := w.colIndex[key]
		if chunkY >= 0 {
			if len(col) <= chunkY {
				n := make([]*Chunk, chunkY+1)
				copy(n, col)
				col = n
			}
			col[chunkY] = chunk
			w.colIndex[key] = col
		}
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

	// Mark neighbor chunks dirty if we touched a border block
	if localX == 0 {
		if nb := w.GetChunkFromBlockCoords(x-1, y, z, false); nb != nil {
			nb.dirty = true
		}
	} else if localX == ChunkSizeX-1 {
		if nb := w.GetChunkFromBlockCoords(x+1, y, z, false); nb != nil {
			nb.dirty = true
		}
	}
	if localY == 0 {
		if nb := w.GetChunkFromBlockCoords(x, y-1, z, false); nb != nil {
			nb.dirty = true
		}
	} else if localY == ChunkSizeY-1 {
		if nb := w.GetChunkFromBlockCoords(x, y+1, z, false); nb != nil {
			nb.dirty = true
		}
	}
	if localZ == 0 {
		if nb := w.GetChunkFromBlockCoords(x, y, z-1, false); nb != nil {
			nb.dirty = true
		}
	} else if localZ == ChunkSizeZ-1 {
		if nb := w.GetChunkFromBlockCoords(x, y, z+1, false); nb != nil {
			nb.dirty = true
		}
	}
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
	defer profiling.Track("world.StreamChunksAroundSync")()
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
	defer profiling.Track("world.StreamChunksAroundAsync")()
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)

	jobsPushed := 0

	for r := 0; r <= radius; r++ {
		if jobsPushed >= w.maxJobsPerCall {
			break
		}

		if r == 0 {
			jobsPushed += w.enqueueColumn(cx, cz)
			continue
		}

		x0 := cx - r
		x1 := cx + r
		z0 := cz - r
		z1 := cz + r

		for xk := x0; xk <= x1; xk++ {
			jobsPushed += w.enqueueColumn(xk, z0)
			if jobsPushed >= w.maxJobsPerCall {
				return
			}
		}
		for zk := z0 + 1; zk <= z1-1; zk++ {
			jobsPushed += w.enqueueColumn(x1, zk)
			if jobsPushed >= w.maxJobsPerCall {
				return
			}
		}
		for xk := x1; xk >= x0; xk-- {
			jobsPushed += w.enqueueColumn(xk, z1)
			if jobsPushed >= w.maxJobsPerCall {
				return
			}
		}
		for zk := z1 - 1; zk >= z0+1; zk-- {
			jobsPushed += w.enqueueColumn(x0, zk)
			if jobsPushed >= w.maxJobsPerCall {
				return
			}
		}
	}
}

// enqueueColumn enqueues all needed Y-chunks for a column, respecting pending cap
func (w *World) enqueueColumn(chunkX, chunkZ int) int {
	// check pending cap
	w.pendingMu.Lock()
	if w.maxPending > 0 && len(w.pending) >= w.maxPending {
		w.pendingMu.Unlock()
		return 0
	}
	w.pendingMu.Unlock()

	// Use cached column height to avoid repeated noise work
	key := [2]int{chunkX, chunkZ}
	w.heightCacheMu.RLock()
	cached, ok := w.heightCache[key]
	w.heightCacheMu.RUnlock()
	maxChunkY := -1
	if ok {
		maxChunkY = cached
	} else {
		worldX := chunkX*ChunkSizeX + ChunkSizeX/2
		worldZ := chunkZ*ChunkSizeZ + ChunkSizeZ/2
		h := w.heightAt(worldX, worldZ)
		maxChunkY = floorDiv(h, ChunkSizeY)
		w.heightCacheMu.Lock()
		w.heightCache[key] = maxChunkY
		w.heightCacheMu.Unlock()
	}
	if maxChunkY < 0 {
		maxChunkY = 0
	}

	enq := 0
	for cy := 0; cy <= maxChunkY; cy++ {
		if w.requestChunkLimited(ChunkCoord{X: chunkX, Y: cy, Z: chunkZ}) {
			enq++
		}
	}
	return enq
}

// EvictFarChunks removes chunks outside the given radius (in chunks) from the center (world x,z).
// It also clears related height cache entries. Returns number of evicted chunks.
func (w *World) EvictFarChunks(x, z float32, radius int) int {
	defer profiling.Track("world.EvictFarChunks")()
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)

	removed := 0
	w.mu.Lock()
	for coord := range w.chunks {
		dx := coord.X - cx
		dz := coord.Z - cz
		if dx*dx+dz*dz > radius*radius {
			delete(w.chunks, coord)
			// maintain column index
			key := [2]int{coord.X, coord.Z}
			if col, ok := w.colIndex[key]; ok {
				if coord.Y >= 0 && coord.Y < len(col) {
					col[coord.Y] = nil
					// trim trailing nils
					end := len(col)
					for end > 0 && col[end-1] == nil {
						end--
					}
					if end == 0 {
						delete(w.colIndex, key)
					} else {
						w.colIndex[key] = col[:end]
					}
				}
			}
			removed++
		}
	}
	w.mu.Unlock()

	// Prune height cache entries outside radius as well
	w.heightCacheMu.Lock()
	for key := range w.heightCache {
		dx := key[0] - cx
		dz := key[1] - cz
		if dx*dx+dz*dz > radius*radius {
			delete(w.heightCache, key)
		}
	}
	w.heightCacheMu.Unlock()

	return removed
}

// requestChunkLimited respects pending cap and returns true if enqueued
func (w *World) requestChunkLimited(coord ChunkCoord) bool {
	// already present?
	w.mu.RLock()
	_, exists := w.chunks[coord]
	w.mu.RUnlock()
	if exists {
		return false
	}

	// pending check + cap
	w.pendingMu.Lock()
	if _, ok := w.pending[coord]; ok {
		w.pendingMu.Unlock()
		return false
	}
	if w.maxPending > 0 && len(w.pending) >= w.maxPending {
		w.pendingMu.Unlock()
		return false
	}
	w.pending[coord] = struct{}{}
	w.pendingMu.Unlock()

	select {
	case w.jobs <- coord:
		return true
	default:
		// queue full: rollback
		w.pendingMu.Lock()
		delete(w.pending, coord)
		w.pendingMu.Unlock()
		return false
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
	defer profiling.Track("world.generateChunkSync")()
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
		// maintain column index
		key := [2]int{coord.X, coord.Z}
		col := w.colIndex[key]
		if coord.Y >= 0 {
			if len(col) <= coord.Y {
				n := make([]*Chunk, coord.Y+1)
				copy(n, col)
				col = n
			}
			col[coord.Y] = chunk
			w.colIndex[key] = col
		}
	}
	w.mu.Unlock()
}

// populateChunk fills a chunk using noise heightmap
func (w *World) populateChunk(c *Chunk) {
	defer profiling.Track("world.populateChunk")()
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
			for ly := 0; ly < topLocal; ly++ {
				if chunkBaseY+ly == 0 {
					c.SetBlock(lx, ly, lz, BlockTypeBedrock)
				} else {
					c.SetBlock(lx, ly, lz, BlockTypeDirt)
				}
			}
			if chunkBaseY+topLocal == 0 {
				c.SetBlock(lx, topLocal, lz, BlockTypeBedrock)
			} else {
				c.SetBlock(lx, topLocal, lz, BlockTypeGrass)
			}
		}
	}
	c.dirty = true
}

// heightAt computes world surface height (block Y) at world X,Z
func (w *World) heightAt(worldX, worldZ int) int {
	defer profiling.Track("world.heightAt")()
	x := float64(worldX) * w.scale
	z := float64(worldZ) * w.scale
	n := octaveNoise2D(x, z, w.seed, w.octaves, w.persistence, w.lacunarity)
	height := float64(w.baseHeight) + n*w.amp
	if height < 0 {
		height = 0
	}
	return int(math.Floor(height))
}

// SurfaceHeightAt exposes the terrain surface height used for generation at world (x,z).
// This is useful for estimating occupancy when adjacent chunks are not yet generated.
func (w *World) SurfaceHeightAt(x, z int) int {
	return w.heightAt(x, z)
}

// AppendChunksInRadiusXZ appends all loaded chunks within a radius (in chunks)
// around a center chunk coordinate (cx, cz) into dst and returns the resulting slice.
// This avoids scanning the entire chunk map each frame.
func (w *World) AppendChunksInRadiusXZ(cx, cz, radius int, dst []ChunkWithCoord) []ChunkWithCoord {
	defer profiling.Track("world.AppendChunksInRadiusXZ")()
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Ensure capacity roughly for typical columns; we cannot know Y count, so grow as needed
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			if dx*dx+dz*dz > radius*radius {
				continue
			}
			xk := cx + dx
			zk := cz + dz
			if col, ok := w.colIndex[[2]int{xk, zk}]; ok {
				for y := 0; y < len(col); y++ {
					ch := col[y]
					if ch == nil {
						continue
					}
					dst = append(dst, ChunkWithCoord{Chunk: ch, Coord: ChunkCoord{X: xk, Y: y, Z: zk}})
				}
			}
		}
	}
	return dst
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
