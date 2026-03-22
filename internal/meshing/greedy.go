package meshing

import (
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
	"sync"
)

// VertexStride is number of uint32 per vertex (packed data)
const VertexStride = 2

// directionJob represents a single direction mesh job
type directionJob struct {
	world         *world.World
	chunk         *world.Chunk
	nx, ny, nz    int
	neighborChunk *world.Chunk
	resultChan    chan []uint32
}

// DirectionWorkerPool manages workers for processing face directions in parallel
type DirectionWorkerPool struct {
	jobQueue chan directionJob
	workers  int
	started  bool
	mu       sync.Mutex
}

var (
	// Buffer pool for greedy meshing masks to reduce GC pressure
	// Max size is usually ChunkSizeY * ChunkSizeX (256*16 = 4096 ints)
	maskPool = sync.Pool{
		New: func() interface{} {
			// Allocate for the largest possible face (Dimensions are usually 16, 256, 16)
			// Max face is 256x16 = 4096
			return make([]int, 0, 4096)
		},
	}

	// resultChanPool pools the per-direction result channels to avoid allocating
	// a new buffered channel (and its internal hchan struct) for every mesh job.
	resultChanPool = sync.Pool{
		New: func() any {
			return make(chan []uint32, 1)
		},
	}
)

// NewDirectionWorkerPool creates a new direction worker pool
func NewDirectionWorkerPool(workers int, queueSize int) *DirectionWorkerPool {
	return &DirectionWorkerPool{
		jobQueue: make(chan directionJob, queueSize),
		workers:  workers,
		started:  false,
	}
}

// Start starts the worker pool goroutines
func (p *DirectionWorkerPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	for i := 0; i < p.workers; i++ {
		go p.worker(i)
	}

	p.started = true
}

// worker is the worker goroutine that processes direction jobs
func (p *DirectionWorkerPool) worker(id int) {
	for job := range p.jobQueue {
		result := buildGreedyForDirection(job.world, job.chunk, job.nx, job.ny, job.nz, job.neighborChunk)
		job.resultChan <- result
	}
}

// SubmitJob submits a direction job to the pool and returns a result channel
func (p *DirectionWorkerPool) SubmitJob(w *world.World, c *world.Chunk, nx, ny, nz int, neighborChunk *world.Chunk) chan []uint32 {
	resultChan := resultChanPool.Get().(chan []uint32)
	job := directionJob{
		world:         w,
		chunk:         c,
		nx:            nx,
		ny:            ny,
		nz:            nz,
		neighborChunk: neighborChunk,
		resultChan:    resultChan,
	}
	p.jobQueue <- job
	return resultChan
}

// packVertex encodes local x,y,z, normal, brightness, textureID and tint into two uint32s.
// V1 Layout: X[4:0] Y[13:5] Z[18:14] N[21:19] B[29:22]
// V2 Layout: T[15:0] C[31:16]
func packVertex(x, y, z int, normal byte, texID int, brightness byte, tint uint16) (uint32, uint32) {
	v1 := uint32(x) | (uint32(y) << 5) | (uint32(z) << 14) | (uint32(normal) << 19) | (uint32(brightness) << 22)
	v2 := uint32(texID) | (uint32(tint) << 16)
	return v1, v2
}

// emitQuad appends two triangles (6 vertices, 12 uint32s) to the vertices slice.
// Triangle 1: v0,v1,v2  Triangle 2: v2,v3,v0
func emitQuad(vertices *[]uint32, x0, y0, z0, x1, y1, z1, x2, y2, z2, x3, y3, z3 int, encodedNormal byte, texID int, tint uint16) {
	// Calculate brightness based on normal (Top=255, Bottom=128, Sides=204)
	var brightness byte = 204 // Sides (0.8 * 255)
	if encodedNormal == 4 {   // Top
		brightness = 255 // 1.0 * 255
	} else if encodedNormal == 5 { // Bottom
		brightness = 128 // 0.5 * 255
	}

	v1a, v2a := packVertex(x0, y0, z0, encodedNormal, texID, brightness, tint)
	v1b, v2b := packVertex(x1, y1, z1, encodedNormal, texID, brightness, tint)
	v1c, v2c := packVertex(x2, y2, z2, encodedNormal, texID, brightness, tint)
	v1d, v2d := packVertex(x3, y3, z3, encodedNormal, texID, brightness, tint)

	*vertices = append(*vertices, v1a, v2a, v1b, v2b, v1c, v2c, v1c, v2c, v1d, v2d, v1a, v2a)
}

// BuildGreedyMeshForChunk builds a greedy-meshed triangle list (packed uint32)
// for the given chunk using world coordinates to decide face visibility across chunk borders.
// Uses the provided worker pool to process all 6 directions in parallel.
// Returns []uint32 where each vertex is 2 packed uint32s containing:
// V1: X (5), Y (9), Z (5), Normal (3), Brightness (8)
// V2: TextureID (16), Tint (16 bits RGB565)
func BuildGreedyMeshForChunk(w *world.World, c *world.Chunk, pool *DirectionWorkerPool) []uint32 {
	if c == nil {
		return nil
	}

	// Pre-fetch neighbor chunks once to avoid repeated RWMutex acquisitions during meshing.
	neighbors := [6]*world.Chunk{
		w.GetChunk(c.X+1, c.Y, c.Z, false), // +X (east)
		w.GetChunk(c.X-1, c.Y, c.Z, false), // -X (west)
		w.GetChunk(c.X, c.Y+1, c.Z, false), // +Y (up)
		w.GetChunk(c.X, c.Y-1, c.Z, false), // -Y (down)
		w.GetChunk(c.X, c.Y, c.Z+1, false), // +Z (north)
		w.GetChunk(c.X, c.Y, c.Z-1, false), // -Z (south)
	}

	// Submit all 6 direction jobs to the worker pool.
	// Use a fixed-size array to avoid a heap allocation for the slice header.
	var directions [6]struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}
	directions[0] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{nx: +1, neighborChunk: neighbors[0]}
	directions[1] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{nx: -1, neighborChunk: neighbors[1]}
	directions[2] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{ny: +1, neighborChunk: neighbors[2]}
	directions[3] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{ny: -1, neighborChunk: neighbors[3]}
	directions[4] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{nz: +1, neighborChunk: neighbors[4]}
	directions[5] = struct {
		nx, ny, nz    int
		neighborChunk *world.Chunk
		resultChan    chan []uint32
	}{nz: -1, neighborChunk: neighbors[5]}

	for i := range directions {
		directions[i].resultChan = pool.SubmitJob(w, c, directions[i].nx, directions[i].ny, directions[i].nz, directions[i].neighborChunk)
	}

	// Collect results from all directions
	var results [6][]uint32
	totalSize := 0
	for i := range directions {
		results[i] = <-directions[i].resultChan
		totalSize += len(results[i])
		resultChanPool.Put(directions[i].resultChan)
	}

	// Combine all results into a single slice
	vertices := make([]uint32, 0, totalSize)
	for _, result := range results {
		vertices = append(vertices, result...)
	}

	// ---------------------------------------------------------
	// Custom/Complex Block Pass
	// Iterate chunk to find blocks that were skipped by greedy mesher.
	// Use section-aware loop to skip nil sections entirely.
	// ---------------------------------------------------------
	for x := 0; x < world.ChunkSizeX; x++ {
		for z := 0; z < world.ChunkSizeZ; z++ {
			for secIdx := 0; secIdx < world.NumSections; secIdx++ {
				if c.IsSectionEmpty(secIdx) {
					continue // skip entire 16-block Y section
				}
				baseY := secIdx * world.SectionHeight
				for ly := 0; ly < world.SectionHeight; ly++ {
					y := baseY + ly
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}

					def := registry.BlockDefs[bt]
					if def == nil {
						continue
					}

					// If it wasn't solid OR it was complex, we skipped it in greedy pass. Render it now.
					if !def.IsSolid || len(def.Elements) > 1 {
						// Appends directly into vertices to avoid an intermediate allocation.
						meshCustomBlock(&vertices, w, c, x, y, z, def)
					}
				}
			}
		}
	}

	return vertices
}

// buildGreedyForDirection performs 2D greedy meshing for one face direction.
// The direction is specified by a normal (nx,ny,nz) where exactly one component is -1 or +1 and the others are 0.
// neighborChunk is the pre-fetched chunk adjacent in the (nx,ny,nz) direction; may be nil if not loaded.
// It returns packed vertices forming triangles.
func buildGreedyForDirection(w *world.World, c *world.Chunk, nx, ny, nz int, neighborChunk *world.Chunk) []uint32 {
	// Determine the axis fixed by the face normal and the two in-plane axes (u,v)
	// We will iterate layers along the normal axis, and build a UxV mask for each layer.
	var (
		// size along x,y,z
		sx, sy, sz = world.ChunkSizeX, world.ChunkSizeY, world.ChunkSizeZ
	)
	// Pre-allocate to reduce grow-copy allocations from repeated appends.
	vertices := make([]uint32, 0, 512)

	// Build per-layer masks and greedy-merge
	if nx != 0 { // Faces perpendicular to X axis, plane is Y-Z
		// Layers along X
		for x := range sx {
			// Mask size sy x sz
			maskPtr := maskPool.Get().([]int)
			reqSize := sy * sz
			if cap(maskPtr) < reqSize {
				maskPtr = make([]int, reqSize)
			}
			mask := maskPtr[:reqSize]
			// Zero out the slice (since we reuse it)
			for k := range mask {
				mask[k] = 0
			}

			for y := range sy {
				// Skip entire 16-block sections that are empty
				if c.IsSectionEmpty(y >> 4) {
					continue
				}
				for z := range sz {
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}
					// Skip custom/complex/transparent blocks for greedy meshing
					def := registry.BlockDefs[bt]
					if def == nil || !def.IsSolid || len(def.Elements) > 1 {
						continue
					}
					// Check visibility
					visible := false
					localNX := x + nx
					if localNX >= 0 && localNX < sx {
						// Check local neighbor
						nbt := c.GetBlock(localNX, y, z)
						if nbt == world.BlockTypeAir {
							visible = true
						} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
							visible = true
						}
					} else {
						// Cross-chunk neighbor: use pre-fetched neighborChunk directly.
						// For nx=+1: localNX==sx, so neighbor local x is 0.
						// For nx=-1: localNX==-1, so neighbor local x is sx-1.
						if neighborChunk == nil {
							visible = true
						} else {
							var nlx int
							if nx > 0 {
								nlx = 0
							} else {
								nlx = sx - 1
							}
							nbt := neighborChunk.GetBlock(nlx, y, z)
							if nbt == world.BlockTypeAir {
								visible = true
							} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
								visible = true
							}
						}
					}

					if visible {
						// faceIdx: East=2, West=3
						faceIdx := 2
						if nx < 0 {
							faceIdx = 3
						}
						texID := registry.GetTexLayerFast(bt, faceIdx)
						tint := registry.GetTintFast(bt, faceIdx)
						mask[y*sz+z] = (int(tint)<<16 | texID) + 1
					}
				}
			}
			// Greedy merge over mask (u=y, v=z)
			i := 0
			for i < sy*sz {
				if mask[i] == 0 {
					i++
					continue
				}
				val := mask[i] - 1
				texID := val & 0xFFFF
				tint := uint16(val >> 16)

				z0 := i % sz
				y0 := i / sz
				wWidth := 1
				for z1 := z0 + 1; z1 < sz && mask[y0*sz+z1] == mask[i]; z1++ {
					wWidth++
				}
				hHeight := 1
			outerYZ:
				for y1 := y0 + 1; y1 < sy; y1++ {
					for z1 := z0; z1 < z0+wWidth; z1++ {
						if mask[y1*sz+z1] != mask[i] {
							break outerYZ
						}
					}
					hHeight++
				}

				fx := x
				if nx > 0 {
					fx = x + 1
				}
				// encodedNormal: East(+X)=2, West(-X)=3
				var encodedNormal byte = 2
				if nx < 0 {
					encodedNormal = 3
				}
				// CCW winding
				if nx > 0 { // +X
					emitQuad(
						&vertices,
						fx, y0, z0,
						fx, y0+hHeight, z0,
						fx, y0+hHeight, z0+wWidth,
						fx, y0, z0+wWidth,
						encodedNormal, texID, tint,
					)
				} else { // -X
					emitQuad(
						&vertices,
						fx, y0, z0,
						fx, y0, z0+wWidth,
						fx, y0+hHeight, z0+wWidth,
						fx, y0+hHeight, z0,
						encodedNormal, texID, tint,
					)
				}
				// zero-out mask
				for yy := y0; yy < y0+hHeight; yy++ {
					for zz := z0; zz < z0+wWidth; zz++ {
						mask[yy*sz+zz] = 0
					}
				}
			}
			// Return buffer to pool
			maskPool.Put(maskPtr)
		}
		return vertices
	}

	if ny != 0 { // Faces perpendicular to Y axis, plane is X-Z
		for y := range sy {
			// Skip layers whose section is empty
			if c.IsSectionEmpty(y >> 4) {
				continue
			}
			maskPtr := maskPool.Get().([]int)
			reqSize := sx * sz
			if cap(maskPtr) < reqSize {
				maskPtr = make([]int, reqSize)
			}
			mask := maskPtr[:reqSize]
			for k := range mask {
				mask[k] = 0
			}

			for x := range sx {
				for z := range sz {
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}
					// Skip custom/complex/transparent blocks for greedy meshing
					def := registry.BlockDefs[bt]
					if def == nil || !def.IsSolid {
						continue
					}
					visible := false
					localNY := y + ny
					if localNY >= 0 && localNY < sy {
						nbt := c.GetBlock(x, localNY, z)
						if nbt == world.BlockTypeAir {
							visible = true
						} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
							visible = true
						}
					} else {
						// Cross-chunk neighbor: use pre-fetched neighborChunk directly.
						// For ny=+1: localNY==sy, neighbor local y is 0.
						// For ny=-1: localNY==-1, neighbor local y is sy-1.
						if neighborChunk == nil {
							visible = true
						} else {
							var nly int
							if ny > 0 {
								nly = 0
							} else {
								nly = sy - 1
							}
							nbt := neighborChunk.GetBlock(x, nly, z)
							if nbt == world.BlockTypeAir {
								visible = true
							} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
								visible = true
							}
						}
					}

					if visible {
						// faceIdx: Top=4, Bottom=5
						faceIdx := 4
						if ny < 0 {
							faceIdx = 5
						}
						texID := registry.GetTexLayerFast(bt, faceIdx)
						tint := registry.GetTintFast(bt, faceIdx)
						mask[x*sz+z] = (int(tint)<<16 | texID) + 1
					}
				}
			}
			// Greedy (u=x, v=z)
			i := 0
			for i < sx*sz {
				if mask[i] == 0 {
					i++
					continue
				}
				val := mask[i] - 1
				texID := val & 0xFFFF
				tint := uint16(val >> 16)

				x0 := i / sz
				z0 := i % sz
				wWidth := 1
				for z1 := z0 + 1; z1 < sz && mask[x0*sz+z1] == mask[i]; z1++ {
					wWidth++
				}
				hHeight := 1
			outerXZ:
				for x1 := x0 + 1; x1 < sx; x1++ {
					for z1 := z0; z1 < z0+wWidth; z1++ {
						if mask[x1*sz+z1] != mask[i] {
							break outerXZ
						}
					}
					hHeight++
				}

				fy := y
				if ny > 0 {
					fy = y + 1
				}
				// encodedNormal: Top(+Y)=4, Bottom(-Y)=5
				var encodedNormal byte = 4
				if ny < 0 {
					encodedNormal = 5
				}
				if ny > 0 { // +Y
					emitQuad(
						&vertices,
						x0, fy, z0,
						x0, fy, z0+wWidth,
						x0+hHeight, fy, z0+wWidth,
						x0+hHeight, fy, z0,
						encodedNormal, texID, tint,
					)
				} else { // -Y
					emitQuad(
						&vertices,
						x0, fy, z0,
						x0+hHeight, fy, z0,
						x0+hHeight, fy, z0+wWidth,
						x0, fy, z0+wWidth,
						encodedNormal, texID, tint,
					)
				}
				for xx := x0; xx < x0+hHeight; xx++ {
					for zz := z0; zz < z0+wWidth; zz++ {
						mask[xx*sz+zz] = 0
					}
				}
			}
			// Return buffer to pool
			maskPool.Put(maskPtr)
		}
		return vertices
	}

	// nz != 0 // Faces perpendicular to Z axis, plane is X-Y
	for z := range sz {
		maskPtr := maskPool.Get().([]int)
		reqSize := sx * sy
		if cap(maskPtr) < reqSize {
			maskPtr = make([]int, reqSize)
		}
		mask := maskPtr[:reqSize]
		for k := range mask {
			mask[k] = 0
		}

		for x := range sx {
			for y := range sy {
				// Skip Y values whose section is empty — mask entry stays zero.
				if c.IsSectionEmpty(y >> 4) {
					continue
				}
				bt := c.GetBlock(x, y, z)
				if bt == world.BlockTypeAir {
					continue
				}
				// Skip custom/complex/transparent blocks for greedy meshing
				def := registry.BlockDefs[bt]
				if def == nil || !def.IsSolid || len(def.Elements) > 1 {
					continue
				}
				visible := false
				localNZ := z + nz
				if localNZ >= 0 && localNZ < sz {
					nbt := c.GetBlock(x, y, localNZ)
					if nbt == world.BlockTypeAir {
						visible = true
					} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
						visible = true
					}
				} else {
					// Cross-chunk neighbor: use pre-fetched neighborChunk directly.
					// For nz=+1: localNZ==sz, neighbor local z is 0.
					// For nz=-1: localNZ==-1, neighbor local z is sz-1.
					if neighborChunk == nil {
						visible = true
					} else {
						var nlz int
						if nz > 0 {
							nlz = 0
						} else {
							nlz = sz - 1
						}
						nbt := neighborChunk.GetBlock(x, y, nlz)
						if nbt == world.BlockTypeAir {
							visible = true
						} else if nDef := registry.BlockDefs[nbt]; nDef != nil && !nDef.IsSolid {
							visible = true
						}
					}
				}

				if visible {
					// faceIdx: North(+Z)=0, South(-Z)=1
					faceIdx := 0
					if nz < 0 {
						faceIdx = 1
					}
					texID := registry.GetTexLayerFast(bt, faceIdx)
					tint := registry.GetTintFast(bt, faceIdx)
					mask[x*sy+y] = (int(tint)<<16 | texID) + 1
				}
			}
		}
		// Greedy (u=x, v=y)
		// BCE hint: access last element to eliminate bounds checks
		if len(mask) > 0 {
			_ = mask[len(mask)-1]
		}
		i := 0
		for i < sx*sy {
			if mask[i] == 0 {
				i++
				continue
			}
			val := mask[i] - 1
			texID := val & 0xFFFF
			tint := uint16(val >> 16)

			x0 := i / sy
			y0 := i % sy
			wWidth := 1
			for y1 := y0 + 1; y1 < sy && mask[x0*sy+y1] == mask[i]; y1++ {
				wWidth++
			}
			hHeight := 1
		outerXY:
			for x1 := x0 + 1; x1 < sx; x1++ {
				for y1 := y0; y1 < y0+wWidth; y1++ {
					if mask[x1*sy+y1] != mask[i] {
						break outerXY
					}
				}
				hHeight++
			}

			fz := z
			if nz > 0 {
				fz = z + 1
			}
			// encodedNormal: North(+Z)=0, South(-Z)=1
			var encodedNormal byte = 0
			if nz < 0 {
				encodedNormal = 1
			}
			if nz > 0 { // +Z
				emitQuad(
					&vertices,
					x0, y0, fz,
					x0+hHeight, y0, fz,
					x0+hHeight, y0+wWidth, fz,
					x0, y0+wWidth, fz,
					encodedNormal, texID, tint,
				)
			} else { // -Z
				emitQuad(
					&vertices,
					x0, y0, fz,
					x0, y0+wWidth, fz,
					x0+hHeight, y0+wWidth, fz,
					x0+hHeight, y0, fz,
					encodedNormal, texID, tint,
				)
			}
			for xx := x0; xx < x0+hHeight; xx++ {
				for yy := y0; yy < y0+wWidth; yy++ {
					mask[xx*sy+yy] = 0
				}
			}
		}
		// Return buffer to pool
		maskPool.Put(maskPtr)
	}
	return vertices
}
