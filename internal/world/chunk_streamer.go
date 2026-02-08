package world

import (
	"math"
	"mini-mc/internal/profiling"
	"runtime"
	"sync"
)

// ChunkStreamer manages asynchronous chunk generation and loading.
type ChunkStreamer struct {
	jobs       chan ChunkCoord
	pending    map[ChunkCoord]struct{}
	pendingMu  sync.Mutex
	maxPending int

	maxJobsPerCall int

	// Cached terrain heights per column (chunkX, chunkZ) -> maxChunkY
	heightCache   map[[2]int]int
	heightCacheMu sync.RWMutex

	// Dependencies
	store *ChunkStore
	gen   TerrainGenerator
}

// NewChunkStreamer creates a new chunk streamer.
func NewChunkStreamer(store *ChunkStore, gen TerrainGenerator) *ChunkStreamer {
	cs := &ChunkStreamer{
		jobs:           make(chan ChunkCoord, 4096),
		pending:        make(map[ChunkCoord]struct{}),
		maxJobsPerCall: 2048,
		maxPending:     16384,
		heightCache:    make(map[[2]int]int),
		store:          store,
		gen:            gen,
	}

	workers := max(runtime.NumCPU(), 1)
	for i := 0; i < workers; i++ {
		go cs.worker()
	}

	return cs
}

// Close stops the background generation workers.
func (cs *ChunkStreamer) Close() {
	close(cs.jobs)
}

func (cs *ChunkStreamer) worker() {
	for coord := range cs.jobs {
		cs.generateChunkSync(coord)
		cs.pendingMu.Lock()
		delete(cs.pending, coord)
		cs.pendingMu.Unlock()
	}
}

// generateChunkSync builds and installs a chunk if missing.
func (cs *ChunkStreamer) generateChunkSync(coord ChunkCoord) {
	if cs.store.HasChunk(coord) {
		return
	}

	chunk := NewChunk(coord.X, coord.Y, coord.Z)
	cs.gen.PopulateChunk(chunk)

	cs.store.AddChunk(coord, chunk)
}

// StreamChunksAroundSync leads chunks synchronously.
func (cs *ChunkStreamer) StreamChunksAroundSync(x, z float32, radius int) {
	defer profiling.Track("world.StreamChunksAroundSync")()
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			chunkX := cx + dx
			chunkZ := cz + dz
			worldX := chunkX*ChunkSizeX + ChunkSizeX/2
			worldZ := chunkZ*ChunkSizeZ + ChunkSizeZ/2
			h := cs.gen.HeightAt(worldX, worldZ)
			maxChunkY := max(floorDiv(h, ChunkSizeY), 0)
			for cy := 0; cy <= maxChunkY; cy++ {
				cs.generateChunkSync(ChunkCoord{X: chunkX, Y: cy, Z: chunkZ})
			}
		}
	}
}

// StreamChunksAroundAsync queues chunks for async loading.
func (cs *ChunkStreamer) StreamChunksAroundAsync(x, z float32, radius int) {
	defer profiling.Track("world.StreamChunksAroundAsync")()
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)

	jobsPushed := 0

	for r := 0; r <= radius; r++ {
		if jobsPushed >= cs.maxJobsPerCall {
			break
		}

		if r == 0 {
			jobsPushed += cs.enqueueColumn(cx, cz)
			continue
		}

		x0 := cx - r
		x1 := cx + r
		z0 := cz - r
		z1 := cz + r

		for xk := x0; xk <= x1; xk++ {
			jobsPushed += cs.enqueueColumn(xk, z0)
			if jobsPushed >= cs.maxJobsPerCall {
				return
			}
		}
		for zk := z0 + 1; zk <= z1-1; zk++ {
			jobsPushed += cs.enqueueColumn(x1, zk)
			if jobsPushed >= cs.maxJobsPerCall {
				return
			}
		}
		for xk := x1; xk >= x0; xk-- {
			jobsPushed += cs.enqueueColumn(xk, z1)
			if jobsPushed >= cs.maxJobsPerCall {
				return
			}
		}
		for zk := z1 - 1; zk >= z0+1; zk-- {
			jobsPushed += cs.enqueueColumn(x0, zk)
			if jobsPushed >= cs.maxJobsPerCall {
				return
			}
		}
	}
}

// enqueueColumn enqueues all needed Y-chunks for a column.
func (cs *ChunkStreamer) enqueueColumn(chunkX, chunkZ int) int {
	// check pending cap
	cs.pendingMu.Lock()
	if cs.maxPending > 0 && len(cs.pending) >= cs.maxPending {
		cs.pendingMu.Unlock()
		return 0
	}
	cs.pendingMu.Unlock()

	// Use cached column height to avoid repeated noise work
	key := [2]int{chunkX, chunkZ}
	cs.heightCacheMu.RLock()
	cached, ok := cs.heightCache[key]
	cs.heightCacheMu.RUnlock()
	maxChunkY := -1
	if ok {
		maxChunkY = cached
	} else {
		worldX := chunkX*ChunkSizeX + ChunkSizeX/2
		worldZ := chunkZ*ChunkSizeZ + ChunkSizeZ/2
		h := cs.gen.HeightAt(worldX, worldZ)
		maxChunkY = floorDiv(h, ChunkSizeY)
		cs.heightCacheMu.Lock()
		cs.heightCache[key] = maxChunkY
		cs.heightCacheMu.Unlock()
	}
	if maxChunkY < 0 {
		maxChunkY = 0
	}

	enq := 0
	for cy := 0; cy <= maxChunkY; cy++ {
		if cs.requestChunkLimited(ChunkCoord{X: chunkX, Y: cy, Z: chunkZ}) {
			enq++
		}
	}
	return enq
}

// requestChunkLimited respects pending cap and returns true if enqueued.
func (cs *ChunkStreamer) requestChunkLimited(coord ChunkCoord) bool {
	// already present?
	if cs.store.HasChunk(coord) {
		return false
	}

	// pending check + cap
	cs.pendingMu.Lock()
	if _, ok := cs.pending[coord]; ok {
		cs.pendingMu.Unlock()
		return false
	}
	if cs.maxPending > 0 && len(cs.pending) >= cs.maxPending {
		cs.pendingMu.Unlock()
		return false
	}
	cs.pending[coord] = struct{}{}
	cs.pendingMu.Unlock()

	select {
	case cs.jobs <- coord:
		return true
	default:
		// queue full: rollback
		cs.pendingMu.Lock()
		delete(cs.pending, coord)
		cs.pendingMu.Unlock()
		return false
	}
}

// EvictFarChunks removes chunks outside the given radius.
func (cs *ChunkStreamer) EvictFarChunks(x, z float32, radius int) int {
	cx := floorDiv(int(math.Floor(float64(x))), ChunkSizeX)
	cz := floorDiv(int(math.Floor(float64(z))), ChunkSizeZ)

	// Delegate physical removal to Store
	removed := cs.store.EvictFarChunks(cx, cz, radius)

	// Prune height cache entries outside radius
	cs.heightCacheMu.Lock()
	for key := range cs.heightCache {
		dx := key[0] - cx
		dz := key[1] - cz
		if dx*dx+dz*dz > radius*radius {
			delete(cs.heightCache, key)
		}
	}
	cs.heightCacheMu.Unlock()

	return removed
}
