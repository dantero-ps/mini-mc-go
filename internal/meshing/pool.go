package meshing

import (
	"context"
	"mini-mc/internal/world"
	"sync"
)

// MeshJob represents a meshing job request
type MeshJob struct {
	World *world.World
	Chunk *world.Chunk
	Coord world.ChunkCoord
	// Result channel - will be sent the result when done
	ResultChan chan MeshResult
}

// MeshResult contains the result of a meshing operation
type MeshResult struct {
	Coord    world.ChunkCoord
	Vertices []uint32 // Packed vertices
	Error    error
}

// WorkerPool manages goroutines for mesh generation
type WorkerPool struct {
	jobQueue      chan MeshJob
	workers       int
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	directionPool *DirectionWorkerPool
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
		jobQueue:      make(chan MeshJob, queueSize),
		workers:       workers,
		ctx:           ctx,
		cancel:        cancel,
		directionPool: directionPool,
	}

	// Start worker goroutines
	for i := range workers {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

// SubmitJob submits a mesh generation job to the pool
// Returns true if job was submitted successfully, false if queue is full
func (p *WorkerPool) SubmitJob(job MeshJob) bool {
	select {
	case p.jobQueue <- job:
		return true
	default:
		return false // Queue is full
	}
}

// SubmitJobBlocking submits a job and blocks until it's queued
func (p *WorkerPool) SubmitJobBlocking(job MeshJob) {
	select {
	case p.jobQueue <- job:
	case <-p.ctx.Done():
	}
}

// worker is the worker goroutine that processes mesh jobs
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case job := <-p.jobQueue:
			// Process the mesh job
			vertices := BuildGreedyMeshForChunk(job.World, job.Chunk, p.directionPool)

			result := MeshResult{
				Coord:    job.Coord,
				Vertices: vertices,
				Error:    nil,
			}

			// Send result back
			select {
			case job.ResultChan <- result:
			case <-p.ctx.Done():
				return
			}

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
