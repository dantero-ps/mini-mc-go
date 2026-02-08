package world

import (
	"mini-mc/internal/profiling"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

// ChunkStore manages the storage and retrieval of chunks.
type ChunkStore struct {
	// Map of chunks indexed by their coordinates
	chunks   map[ChunkCoord]*Chunk
	mu       sync.RWMutex
	modCount uint64 // Increases on any chunk add/remove

	// Per-column index for fast XZ radius queries: (chunkX,chunkZ) -> slice indexed by chunkY
	colIndex map[[2]int][]*Chunk
}

// NewChunkStore creates a new chunk store.
func NewChunkStore() *ChunkStore {
	return &ChunkStore{
		chunks:   make(map[ChunkCoord]*Chunk),
		colIndex: make(map[[2]int][]*Chunk),
	}
}

// GetChunk returns the chunk at the specified chunk coordinates.
// If the chunk doesn't exist and create is true, it will be created (but NOT populated).
func (cs *ChunkStore) GetChunk(chunkX, chunkY, chunkZ int, create bool) *Chunk {
	coord := ChunkCoord{X: chunkX, Y: chunkY, Z: chunkZ}
	cs.mu.RLock()
	chunk, exists := cs.chunks[coord]
	cs.mu.RUnlock()
	if !exists && create {
		cs.mu.Lock()
		// Double-check locking: another goroutine might have created it while we were waiting for the lock
		if existing, ok := cs.chunks[coord]; ok {
			cs.mu.Unlock()
			return existing
		}

		chunk = NewChunk(chunkX, chunkY, chunkZ)
		cs.chunks[coord] = chunk
		cs.modCount++
		// maintain column index
		key := [2]int{chunkX, chunkZ}
		col := cs.colIndex[key]
		if chunkY >= 0 {
			if len(col) <= chunkY {
				n := make([]*Chunk, chunkY+1)
				copy(n, col)
				col = n
			}
			col[chunkY] = chunk
			cs.colIndex[key] = col
		}
		cs.mu.Unlock()
	}
	return chunk
}

// GetChunkFromBlockCoords returns the chunk containing the block at the specified world coordinates.
func (cs *ChunkStore) GetChunkFromBlockCoords(x, y, z int, create bool) *Chunk {
	// Convert world coordinates to chunk coordinates
	chunkX := floorDiv(x, ChunkSizeX)
	chunkY := floorDiv(y, ChunkSizeY)
	chunkZ := floorDiv(z, ChunkSizeZ)

	return cs.GetChunk(chunkX, chunkY, chunkZ, create)
}

// Get returns the block type at the specified world coordinates.
func (cs *ChunkStore) Get(x, y, z int) BlockType {
	chunk := cs.GetChunkFromBlockCoords(x, y, z, false)
	if chunk == nil {
		return BlockTypeAir
	}

	// Convert world coordinates to local chunk coordinates
	localX := mod(x, ChunkSizeX)
	localY := mod(y, ChunkSizeY)
	localZ := mod(z, ChunkSizeZ)

	return chunk.GetBlock(localX, localY, localZ)
}

// IsAir checks if the block at the specified world coordinates is air.
func (cs *ChunkStore) IsAir(x, y, z int) bool {
	return cs.Get(x, y, z) == BlockTypeAir
}

// Set sets the block type at the specified world coordinates.
func (cs *ChunkStore) Set(x, y, z int, val BlockType) {
	chunk := cs.GetChunkFromBlockCoords(x, y, z, true)

	// Convert world coordinates to local chunk coordinates
	localX := mod(x, ChunkSizeX)
	localY := mod(y, ChunkSizeY)
	localZ := mod(z, ChunkSizeZ)

	chunk.SetBlock(localX, localY, localZ, val)

	// Mark neighbor chunks dirty if we touched a border block
	if localX == 0 {
		if nb := cs.GetChunkFromBlockCoords(x-1, y, z, false); nb != nil {
			nb.dirty = true
		}
	} else if localX == ChunkSizeX-1 {
		if nb := cs.GetChunkFromBlockCoords(x+1, y, z, false); nb != nil {
			nb.dirty = true
		}
	}
	if localY == 0 {
		if nb := cs.GetChunkFromBlockCoords(x, y-1, z, false); nb != nil {
			nb.dirty = true
		}
	} else if localY == ChunkSizeY-1 {
		if nb := cs.GetChunkFromBlockCoords(x, y+1, z, false); nb != nil {
			nb.dirty = true
		}
	}
	if localZ == 0 {
		if nb := cs.GetChunkFromBlockCoords(x, y, z-1, false); nb != nil {
			nb.dirty = true
		}
	} else if localZ == ChunkSizeZ-1 {
		if nb := cs.GetChunkFromBlockCoords(x, y, z+1, false); nb != nil {
			nb.dirty = true
		}
	}
}

// GetActiveBlocks returns a list of positions of all non-air blocks in the world.
func (cs *ChunkStore) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3
	cs.mu.RLock()
	for _, chunk := range cs.chunks {
		positions = append(positions, chunk.GetActiveBlocks()...)
	}
	cs.mu.RUnlock()
	return positions
}

// GetAllChunks returns a slice of all chunks in the world with their coordinates.
func (cs *ChunkStore) GetAllChunks() []ChunkWithCoord {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	chunks := make([]ChunkWithCoord, 0, len(cs.chunks))
	for coord, chunk := range cs.chunks {
		chunks = append(chunks, ChunkWithCoord{Chunk: chunk, Coord: coord})
	}
	return chunks
}

// AppendChunksInRadiusXZ appends all loaded chunks within a radius (in chunks)
// around a center chunk coordinate (cx, cz) into dst and returns the resulting slice.
func (cs *ChunkStore) AppendChunksInRadiusXZ(cx, cz, radius int, dst []ChunkWithCoord) []ChunkWithCoord {
	defer profiling.Track("world.AppendChunksInRadiusXZ")()
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Ensure capacity roughly for typical columns; we cannot know Y count, so grow as needed
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			if dx*dx+dz*dz > radius*radius {
				continue
			}
			xk := cx + dx
			zk := cz + dz
			if col, ok := cs.colIndex[[2]int{xk, zk}]; ok {
				for y := range col {
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

// GetModCount returns the current modification count of the chunk map.
func (cs *ChunkStore) GetModCount() uint64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.modCount
}

// EvictFarChunks removes chunks outside the given radius from the store.
// Returns number of removed chunks.
func (cs *ChunkStore) EvictFarChunks(cx, cz, radius int) int {
	defer profiling.Track("world.EvictFarChunks")()
	removed := 0
	cs.mu.Lock()
	for coord := range cs.chunks {
		dx := coord.X - cx
		dz := coord.Z - cz
		if dx*dx+dz*dz > radius*radius {
			delete(cs.chunks, coord)
			cs.modCount++
			// maintain column index
			key := [2]int{coord.X, coord.Z}
			if col, ok := cs.colIndex[key]; ok {
				if coord.Y >= 0 && coord.Y < len(col) {
					col[coord.Y] = nil
					// trim trailing nils
					end := len(col)
					for end > 0 && col[end-1] == nil {
						end--
					}
					if end == 0 {
						delete(cs.colIndex, key)
					} else {
						cs.colIndex[key] = col[:end]
					}
				}
			}
			removed++
		}
	}
	cs.mu.Unlock()
	return removed
}

// HasChunk checks if a chunk exists without creating it (lite wrapper around RLock).
func (cs *ChunkStore) HasChunk(coord ChunkCoord) bool {
	cs.mu.RLock()
	_, exists := cs.chunks[coord]
	cs.mu.RUnlock()
	return exists
}

// AddChunk adds a pre-generated chunk to the store.
func (cs *ChunkStore) AddChunk(coord ChunkCoord, chunk *Chunk) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, ok := cs.chunks[coord]; !ok {
		cs.chunks[coord] = chunk
		cs.modCount++
		// maintain column index
		key := [2]int{coord.X, coord.Z}
		col := cs.colIndex[key]
		if coord.Y >= 0 {
			if len(col) <= coord.Y {
				n := make([]*Chunk, coord.Y+1)
				copy(n, col)
				col = n
			}
			col[coord.Y] = chunk
			cs.colIndex[key] = col
		}
	}
}
