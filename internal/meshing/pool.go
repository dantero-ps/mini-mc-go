package meshing

import (
	"context"
	"mini-mc/internal/world"
	"sync"
)

// MeshJob represents a meshing job request
type MeshJob struct {
	World           *world.World
	Chunk           *world.Chunk
	Coord           world.ChunkCoord
	ResultChan      chan MeshResult
	ChunkGeneration uint64 // snapshot of chunk.Generation() at submission time
}

// MeshResult contains the result of a meshing operation
type MeshResult struct {
	Coord           world.ChunkCoord
	Chunk           *world.Chunk // The chunk that was meshed; used to call SetClean after applying
	Vertices        []uint32     // Packed vertices
	FluidVertices   []float32    // Fluid vertices (custom format)
	Error           error
	ChunkGeneration uint64 // echoed from the job; compared against chunk.Generation() in applyMeshResult
}

// WorkerPool manages goroutines for mesh generation
type WorkerPool struct {
	jobQueue         chan MeshJob
	priorityJobQueue chan MeshJob // checked before jobQueue; for player-interaction updates
	workers          int
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	directionPool    *DirectionWorkerPool
}

// NewWorkerPool creates a new mesh worker pool
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize inner direction pool
	// 6 workers for 6 faces seems reasonable default, or maybe scaling with workers?
	// The direction pool processes the 6 faces OF A SINGLE CHUNK in parallel.
	// Since we have 'workers' chunks being processed in parallel, we might want to be careful.
	// If we have 4 mesh workers, and each spawns 6 direction jobs, that's 24 go routines.
	// That's fine.
	directionPool := NewDirectionWorkerPool(6, 32)
	directionPool.Start()

	pool := &WorkerPool{
		jobQueue:         make(chan MeshJob, queueSize),
		priorityJobQueue: make(chan MeshJob, 64),
		workers:          workers,
		ctx:              ctx,
		cancel:           cancel,
		directionPool:    directionPool,
	}

	// Start worker goroutines
	for i := range workers {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

// SubmitJob submits a mesh generation job to the normal (low-priority) queue.
// Returns true if the job was accepted, false if the queue is full.
func (p *WorkerPool) SubmitJob(job MeshJob) bool {
	select {
	case p.jobQueue <- job:
		return true
	default:
		return false
	}
}

// SubmitPriorityJob submits a job to the high-priority queue (checked before normal jobs).
// Use this for player-interaction updates so they are not delayed by initial-load backlog.
// Returns true if accepted, false if the priority queue is full.
func (p *WorkerPool) SubmitPriorityJob(job MeshJob) bool {
	select {
	case p.priorityJobQueue <- job:
		return true
	default:
		return false
	}
}

// SubmitJobBlocking submits a job and blocks until it's queued
func (p *WorkerPool) SubmitJobBlocking(job MeshJob) {
	select {
	case p.jobQueue <- job:
	case <-p.ctx.Done():
	}
}

// processJob executes a single mesh job and sends the result.
func (p *WorkerPool) processJob(job MeshJob) {
	vertices := BuildGreedyMeshForChunk(job.World, job.Chunk, p.directionPool)
	fluidVertices := BuildFluidMesh(job.World, job.Chunk)

	result := MeshResult{
		Coord:           job.Coord,
		Chunk:           job.Chunk,
		Vertices:        vertices,
		FluidVertices:   fluidVertices,
		ChunkGeneration: job.ChunkGeneration,
	}

	select {
	case job.ResultChan <- result:
	case <-p.ctx.Done():
	}
}

// worker is the worker goroutine that processes mesh jobs.
// Priority queue is always drained first so player-interaction jobs
// are not delayed by the initial-load backlog in the normal queue.
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		// Non-blocking drain of priority queue first
		select {
		case job := <-p.priorityJobQueue:
			p.processJob(job)
			continue
		default:
		}

		// Block waiting on either queue; priority still wins
		select {
		case job := <-p.priorityJobQueue:
			p.processJob(job)
		case job := <-p.jobQueue:
			p.processJob(job)
		case <-p.ctx.Done():
			return
		}
	}
}

// Shutdown gracefully shuts down the worker pool
func (p *WorkerPool) Shutdown() {
	p.cancel()
	close(p.jobQueue)
	p.wg.Wait()
}

// GetQueueLength returns the current number of jobs in the queue
func (p *WorkerPool) GetQueueLength() int {
	return len(p.jobQueue)
}
