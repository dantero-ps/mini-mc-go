package blocks

import (
	"mini-mc/internal/meshing"
	"mini-mc/internal/world"
	"sync"
)

// Chunk meshes cache per chunk
var chunkMeshes map[world.ChunkCoord]*chunkMesh

// Per-column (XZ) combined meshes to reduce draw/cull granularity
var columnMeshes map[[2]int]*columnMesh

// Global mesh worker pool
var meshPool *meshing.WorkerPool

// Pending mesh jobs - tracks which chunks have jobs in progress
var pendingMeshJobs map[world.ChunkCoord]chan meshing.MeshResult
var pendingMeshMutex sync.RWMutex

// Results channel for completed mesh jobs
var meshResultsChannel = make(chan meshing.MeshResult, 100)

// InitMeshSystem initializes the mesh worker pool and data structures
func InitMeshSystem(workers int) {
	meshPool = meshing.NewWorkerPool(workers, 200) // 200 job queue size
	chunkMeshes = make(map[world.ChunkCoord]*chunkMesh)
	columnMeshes = make(map[[2]int]*columnMesh)
	pendingMeshJobs = make(map[world.ChunkCoord]chan meshing.MeshResult)
}

// ShutdownMeshSystem gracefully shuts down the mesh worker pool
func ShutdownMeshSystem() {
	if meshPool != nil {
		meshPool.Shutdown()
	}
	CleanupAtlas()
}

// ProcessMeshResults processes completed mesh results from worker pool
// Should be called regularly from the main render thread
func ProcessMeshResults() {
	for {
		select {
		case result := <-meshResultsChannel:
			applyMeshResult(result)
		default:
			return // No more results to process this frame
		}
	}
}

// applyMeshResult applies a completed mesh result to OpenGL buffers
func applyMeshResult(result meshing.MeshResult) {
	coord := result.Coord

	// Remove from pending jobs
	pendingMeshMutex.Lock()
	delete(pendingMeshJobs, coord)
	pendingMeshMutex.Unlock()

	if result.Error != nil {
		return // Skip on error
	}

	// Only mark the chunk clean if its generation hasn't advanced since the job
	// was submitted. If the generation differs, the chunk was modified while the
	// job was in-flight (e.g. the player broke a block), so the result is stale.
	// Leaving dirty=true ensures ensureChunkMesh will queue a fresh job next frame.
	if result.Chunk != nil && result.Chunk.Generation() == result.ChunkGeneration {
		result.Chunk.SetClean()
	}

	existing := chunkMeshes[coord]
	if existing == nil {
		existing = &chunkMesh{
			firstFloat:  -1,
			firstVertex: -1,
		}
	}

	verts := result.Vertices
	fluidVerts := result.FluidVertices
	if len(verts) > 0 || len(fluidVerts) > 0 {
		// Vertex count is just length of packed array (one uint32 per vertex)
		existing.vertexCount = int32(len(verts))
		// Keep CPU copy for column meshing
		existing.cpuVerts = verts
		existing.fluidVerts = fluidVerts
	} else {
		existing.vertexCount = 0
		existing.cpuVerts = nil
		existing.fluidVerts = nil
	}
	// Mark the column as dirty in all cases: even when transitioning from a full chunk to an empty one
	// ensureColumnMeshForXZ should free the atlas slot and shrink the column.
	if col := columnMeshes[[2]int{coord.X, coord.Z}]; col != nil {
		col.dirty = true
	}
	chunkMeshes[coord] = existing
}

func ensureChunkMesh(w *world.World, coord world.ChunkCoord, ch *world.Chunk) *chunkMesh {
	if ch == nil {
		return nil
	}

	existing := chunkMeshes[coord]

	// Return existing mesh if present and chunk is clean
	if existing != nil && !ch.IsDirty() {
		return existing
	}

	// Check if we already have a pending job for this chunk
	pendingMeshMutex.RLock()
	_, hasPendingJob := pendingMeshJobs[coord]
	pendingMeshMutex.RUnlock()

	// If chunk is dirty or has no mesh and no job is pending, submit a new mesh job
	if (ch.IsDirty() || existing == nil) && !hasPendingJob && meshPool != nil {
		job := meshing.MeshJob{
			World:           w,
			Chunk:           ch,
			Coord:           coord,
			ResultChan:      meshResultsChannel,
			ChunkGeneration: ch.Generation(),
		}

		// Chunks that already have a mesh are being updated (e.g. player broke a
		// block). Submit them to the priority queue so they aren't delayed behind
		// the initial-load backlog in the normal queue.
		submitted := false
		if existing != nil {
			submitted = meshPool.SubmitPriorityJob(job)
		}
		if !submitted {
			submitted = meshPool.SubmitJob(job)
		}

		if submitted {
			pendingMeshMutex.Lock()
			pendingMeshJobs[coord] = meshResultsChannel
			pendingMeshMutex.Unlock()
		}
	}

	// Return existing mesh if available, even if it's being updated
	return existing
}

// PruneMeshesByWorld removes cached meshes that are not in the world anymore or beyond a radius from center.
// Returns number of meshes freed.
func PruneMeshesByWorld(w *world.World, centerX, centerZ float32, radiusChunks int) int {
	retain := make(map[world.ChunkCoord]struct{})
	all := w.GetAllChunks()
	for _, cc := range all {
		retain[cc.Coord] = struct{}{}
	}
	cx := int(centerX) / world.ChunkSizeX
	cz := int(centerZ) / world.ChunkSizeZ

	freed := 0
	for coord, m := range chunkMeshes {
		// Keep if present and within radius
		_, present := retain[coord]
		dx := coord.X - cx
		dz := coord.Z - cz
		if !present || dx*dx+dz*dz > radiusChunks*radiusChunks {
			if m != nil {
				m.cpuVerts = nil
				m.fluidVerts = nil
			}
			delete(chunkMeshes, coord)
			colKey := [2]int{coord.X, coord.Z}
			if col := columnMeshes[colKey]; col != nil {
				// Y-chunk içeriği değişti; rebuild gerekiyor.
				// firstFloat/vertexCount'a DOKUNMA — eski slot referansı kaybolursa
				// freeInRegion çağrılamaz ve atlas region'ında orphan veri kalır.
				// ensureColumnMeshForXZ next frame collectColumnVerts ile yeni boyutu
				// görür ve ya same-size in-place yazar ya da alloc-new + free-old yapar.
				col.dirty = true
			}
			freed++
		}
	}

	// Also prune column meshes that are completely out of range
	for key, col := range columnMeshes {
		dx := key[0] - cx
		dz := key[1] - cz
		if dx*dx+dz*dz > radiusChunks*radiusChunks {
			// Mark as empty and reclaim space tracking
			if col.firstFloat >= 0 && col.vertexCount > 0 {
				if r := atlasRegions[col.regionKey]; r != nil {
					freeInRegion(r, col.firstFloat, int(col.vertexCount)*6)
					r.activeColumns--
				}
			}

			col.vertexCount = 0
			col.firstFloat = -1
			col.firstVertex = -1
			col.dirty = true
			// We remove it from the map so it can be GC'd.
			// The reference in atlasRegion.orderedColumns will be dropped during the next compaction
			delete(columnMeshes, key)
		}
	}

	return freed
}
