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
	jobQueue chan MeshJob
	workers  int
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewWorkerPool creates a new mesh worker pool
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		jobQueue: make(chan MeshJob, queueSize),
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
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
			vertices := BuildGreedyMeshForChunk(job.World, job.Chunk)

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
