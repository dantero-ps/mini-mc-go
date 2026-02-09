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
	world      *world.World
	chunk      *world.Chunk
	nx, ny, nz int
	resultChan chan []uint32
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
		// Process the direction
		result := buildGreedyForDirection(job.world, job.chunk, job.nx, job.ny, job.nz)

		// Send result back
		job.resultChan <- result
	}
}

// SubmitJob submits a direction job to the pool and returns a result channel
func (p *DirectionWorkerPool) SubmitJob(w *world.World, c *world.Chunk, nx, ny, nz int) chan []uint32 {
	resultChan := make(chan []uint32, 1)
	job := directionJob{
		world:      w,
		chunk:      c,
		nx:         nx,
		ny:         ny,
		nz:         nz,
		resultChan: resultChan,
	}
	p.jobQueue <- job
	return resultChan
}

// BuildGreedyMeshForChunk builds a greedy-meshed triangle list (packed uint32)
// for the given chunk using world coordinates to decide face visibility across chunk borders.
// Uses the provided worker pool to process all 6 directions in parallel.
// Returns []uint32 where each vertex is 2 packed uint32s containing:
// V1: X (5), Y (9), Z (5), Normal (3), Brightness (8)
// V2: TextureID (16), Tint (1 bit - unused in pack but available)
func BuildGreedyMeshForChunk(w *world.World, c *world.Chunk, pool *DirectionWorkerPool) []uint32 {
	if c == nil {
		return nil
	}

	// World-space offset of this chunk is NOT baked into vertices anymore to save bits.
	// We use chunk-local coordinates (0-15 for X/Z, 0-255 for Y).

	// Submit all 6 direction jobs to the worker pool
	directions := []struct {
		nx, ny, nz int
		resultChan chan []uint32
	}{
		{nx: +1, ny: 0, nz: 0, resultChan: pool.SubmitJob(w, c, +1, 0, 0)}, // +X (east)
		{nx: -1, ny: 0, nz: 0, resultChan: pool.SubmitJob(w, c, -1, 0, 0)}, // -X (west)
		{nx: 0, ny: +1, nz: 0, resultChan: pool.SubmitJob(w, c, 0, +1, 0)}, // +Y (top)
		{nx: 0, ny: -1, nz: 0, resultChan: pool.SubmitJob(w, c, 0, -1, 0)}, // -Y (bottom)
		{nx: 0, ny: 0, nz: +1, resultChan: pool.SubmitJob(w, c, 0, 0, +1)}, // +Z (north)
		{nx: 0, ny: 0, nz: -1, resultChan: pool.SubmitJob(w, c, 0, 0, -1)}, // -Z (south)
	}

	// Collect results from all directions
	results := make([][]uint32, 6)
	totalSize := 0
	for i, dir := range directions {
		results[i] = <-dir.resultChan
		totalSize += len(results[i])
		close(dir.resultChan)
	}

	// Combine all results into a single slice
	vertices := make([]uint32, 0, totalSize)
	for _, result := range results {
		vertices = append(vertices, result...)
	}

	// ---------------------------------------------------------
	// Custom/Complex Block Pass
	// Iterate chunk to find blocks that were skipped by greedy mesher
	// ---------------------------------------------------------
	for x := 0; x < world.ChunkSizeX; x++ {
		for z := 0; z < world.ChunkSizeZ; z++ {
			for y := 0; y < world.ChunkSizeY; y++ {
				// Optimization: check section first?
				// For now simple loop.
				bt := c.GetBlock(x, y, z)
				if bt == world.BlockTypeAir {
					continue
				}

				def, ok := registry.Blocks[bt]
				if !ok {
					continue
				}

				// If it wasn't solid OR it was complex, we skipped it in greedy pass. Render it now.
				if !def.IsSolid || len(def.Elements) > 1 {
					// This function emits raw quads for each element face defined in JSON
					customVerts := meshCustomBlock(w, c, x, y, z, def)
					vertices = append(vertices, customVerts...)
				}
			}
		}
	}

	return vertices
}

// buildGreedyForDirection performs 2D greedy meshing for one face direction.
// The direction is specified by a normal (nx,ny,nz) where exactly one component is -1 or +1 and the others are 0.
// It returns packed vertices forming triangles.
func buildGreedyForDirection(w *world.World, c *world.Chunk, nx, ny, nz int) []uint32 {
	// Determine the axis fixed by the face normal and the two in-plane axes (u,v)
	// We will iterate layers along the normal axis, and build a UxV mask for each layer.
	var (
		// size along x,y,z
		sx, sy, sz = world.ChunkSizeX, world.ChunkSizeY, world.ChunkSizeZ
		vertices   []uint32
	)

	// Base coordinates for world lookups (still needed for neighbor checks)
	baseX := c.X * world.ChunkSizeX
	baseY := c.Y * world.ChunkSizeY
	baseZ := c.Z * world.ChunkSizeZ

	// packVertex encodes local x,y,z, normal, brightness, textureID and tint into two uint32s
	// V1 Layout:
	// X: bits 0-4   (5 bits)
	// Y: bits 5-13  (9 bits)
	// Z: bits 14-18 (5 bits)
	// N: bits 19-21 (3 bits)
	// B: bits 22-29 (8 bits) - Brightness
	// V2 Layout:
	// T: bits 0-15  (16 bits) - Texture Layer ID
	// C: bits 16-31 (16 bits) - Tint Color (RGB565)
	packVertex := func(x, y, z int, normal byte, texID int, brightness byte, tint uint16) (uint32, uint32) {
		v1 := uint32(x) | (uint32(y) << 5) | (uint32(z) << 14) | (uint32(normal) << 19) | (uint32(brightness) << 22)
		v2 := uint32(texID) | (uint32(tint) << 16)
		return v1, v2
	}

	// encodeNormal converts normal vector to a single byte encoding (0-5 for 6 face directions)
	encodeNormal := func(nx, ny, nz float32) byte {
		if nz > 0.5 {
			return 0 // North (+Z)
		} else if nz < -0.5 {
			return 1 // South (-Z)
		} else if nx > 0.5 {
			return 2 // East (+X)
		} else if nx < -0.5 {
			return 3 // West (-X)
		} else if ny > 0.5 {
			return 4 // Top (+Y)
		} else if ny < -0.5 {
			return 5 // Bottom (-Y)
		}
		return 6 // Default/unknown
	}

	// Helper to pack RGB to RGB565
	packColor := func(c uint32) uint16 {
		if c == 0 {
			return 0xFFFF // Use White (0xFFFF) for no tint? Or 0?
			// If we multiply in shader: Color * Tint.
			// If Tint is 0 (Black), block becomes black.
			// So default should be White (0xFFFF).
		}
		r := (c >> 16) & 0xFF
		g := (c >> 8) & 0xFF
		b := c & 0xFF

		r5 := (r >> 3) & 0x1F
		g6 := (g >> 2) & 0x3F
		b5 := (b >> 3) & 0x1F

		return uint16((r5 << 11) | (g6 << 5) | b5)
	}

	defaultTint := uint16(0xFFFF) // White

	// Helper lambda to push a quad made of two triangles with given 4 corners and normal
	emitQuad := func(x0, y0, z0, x1, y1, z1, x2, y2, z2, x3, y3, z3 int, fnx, fny, fnz float32, texID int, tint uint16) {
		// Vertices are passed as Chunk-Local Integers.
		encodedNormal := encodeNormal(fnx, fny, fnz)

		// Calculate brightness based on normal (Top=255, Bottom=128, Sides=204)
		var brightness byte = 204 // Sides (0.8 * 255)
		if encodedNormal == 4 {   // Top
			brightness = 255 // 1.0 * 255
		} else if encodedNormal == 5 { // Bottom
			brightness = 128 // 0.5 * 255
		}

		// Triangle 1: v0,v1,v2
		v1a, v2a := packVertex(x0, y0, z0, encodedNormal, texID, brightness, tint)
		v1b, v2b := packVertex(x1, y1, z1, encodedNormal, texID, brightness, tint)
		v1c, v2c := packVertex(x2, y2, z2, encodedNormal, texID, brightness, tint)

		vertices = append(vertices, v1a, v2a, v1b, v2b, v1c, v2c)

		// Triangle 2: v2,v3,v0
		v1d, v2d := packVertex(x3, y3, z3, encodedNormal, texID, brightness, tint)

		vertices = append(vertices, v1c, v2c, v1d, v2d, v1a, v2a)
	}

	// Helper to get texture layer
	getTex := func(blockType world.BlockType, normalIdx int) int {
		// Map normal index back to face
		var face world.BlockFace
		switch normalIdx {
		case 0:
			face = world.FaceNorth // +Z
		case 1:
			face = world.FaceSouth // -Z
		case 2:
			face = world.FaceEast // +X
		case 3:
			face = world.FaceWest // -X
		case 4:
			face = world.FaceTop // +Y
		case 5:
			face = world.FaceBottom // -Y
		}

		return registry.GetTextureLayer(blockType, face)
	}

	// Helper to get tint
	getTint := func(blockType world.BlockType, normalIdx int) uint16 {
		if def, ok := registry.Blocks[blockType]; ok && def.TintColor != 0 {
			var face world.BlockFace
			switch normalIdx {
			case 0:
				face = world.FaceNorth
			case 1:
				face = world.FaceSouth
			case 2:
				face = world.FaceEast
			case 3:
				face = world.FaceWest
			case 4:
				face = world.FaceTop
			case 5:
				face = world.FaceBottom
			}
			if def.TintFaces[face] {
				return packColor(def.TintColor)
			}
		}
		return defaultTint
	}

	// Build per-layer masks and greedy-merge
	// Cases per normal axis
	if nx != 0 { // Faces perpendicular to X axis, plane is Y-Z
		// Layers along X
		for x := range sx {
			// Mask size sy x sz
			// Get buffer from pool
			maskPtr := maskPool.Get().([]int)
			// Resize cap if needed (shouldn't be if pool is sized right, but safe is safe)
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
				for z := range sz {
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}
					// Skip custom/complex/transparent blocks for greedy meshing
					// X-Axis (Sides): Skip multi-element blocks (Grass) so Custom Mesher handles the detailed sides
					if def, ok := registry.Blocks[bt]; ok && (!def.IsSolid || len(def.Elements) > 1) {
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
						} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
							// If neighbor is transparent (like Water/Lava), we MUST render this face.
							// Exception: if neighbor is same material and it merges (like glass-to-glass),
							// but here we are rendering OPAQUE block against TRANSPARENT block.
							// So if neighbor is not solid (transparent), we see through it -> visible = true.
							visible = true
						}
					} else {
						wx, wy, wz := baseX+x, baseY+y, baseZ+z
						nxw, nyw, nzw := wx+nx, wy, wz

						// Check global neighbor
						nChunk := w.GetChunkFromBlockCoords(nxw, nyw, nzw, false)
						if nChunk == nil {
							visible = true // Render boundary if chunk not loaded
						} else {
							// Get block type from world
							nbt := w.Get(nxw, nyw, nzw)
							if nbt == world.BlockTypeAir {
								visible = true
							} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
								visible = true
							}
						}
					}

					if visible {
						// Determine face
						faceIdx := 2 // East
						if nx < 0 {
							faceIdx = 3
						} // West
						texID := getTex(bt, faceIdx)
						tint := getTint(bt, faceIdx)
						// Pack tint and texID: (tint << 16) | texID
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

				// compute width
				z0 := i % sz
				y0 := i / sz
				wWidth := 1
				for z1 := z0 + 1; z1 < sz && mask[y0*sz+z1] == mask[i]; z1++ {
					wWidth++
				}
				// compute height
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
				// emit quad
				fx := x
				if nx > 0 {
					fx = x + 1
				}
				// Local coordinates
				fnx, fny, fnz := float32(nx), float32(0), float32(0)
				// CCW winding
				if nx > 0 { // +X
					emitQuad(
						fx, y0, z0,
						fx, y0+hHeight, z0,
						fx, y0+hHeight, z0+wWidth,
						fx, y0, z0+wWidth,
						fnx, fny, fnz,
						texID,
						tint,
					)
				} else { // -X
					emitQuad(
						fx, y0, z0,
						fx, y0, z0+wWidth,
						fx, y0+hHeight, z0+wWidth,
						fx, y0+hHeight, z0,
						fnx, fny, fnz,
						texID,
						tint,
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
			// Get buffer from pool
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
					// Also skip multi-element blocks (like Grass with overlay) even if solid
					if def, ok := registry.Blocks[bt]; ok && !def.IsSolid {
						continue
					}
					visible := false
					localNY := y + ny
					if localNY >= 0 && localNY < sy {
						nbt := c.GetBlock(x, localNY, z)
						if nbt == world.BlockTypeAir {
							visible = true
						} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
							visible = true
						}
					} else {
						wx, wy, wz := baseX+x, baseY+y, baseZ+z
						nxw, nyw, nzw := wx, wy+ny, wz

						nChunk := w.GetChunkFromBlockCoords(nxw, nyw, nzw, false)
						if nChunk == nil {
							visible = true
						} else {
							nbt := w.Get(nxw, nyw, nzw)
							if nbt == world.BlockTypeAir {
								visible = true
							} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
								visible = true
							}
						}
					}

					if visible {
						faceIdx := 4 // Top
						if ny < 0 {
							faceIdx = 5
						} // Bottom
						texID := getTex(bt, faceIdx)
						tint := getTint(bt, faceIdx)
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
				fnx, fny, fnz := float32(0), float32(ny), float32(0)
				if ny > 0 { // +Y
					emitQuad(
						x0, fy, z0,
						x0, fy, z0+wWidth,
						x0+hHeight, fy, z0+wWidth,
						x0+hHeight, fy, z0,
						fnx, fny, fnz,
						texID,
						tint,
					)
				} else { // -Y
					emitQuad(
						x0, fy, z0,
						x0+hHeight, fy, z0,
						x0+hHeight, fy, z0+wWidth,
						x0, fy, z0+wWidth,
						fnx, fny, fnz,
						texID,
						tint,
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
		// Get buffer from pool
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
				bt := c.GetBlock(x, y, z)
				if bt == world.BlockTypeAir {
					continue
				}
				// Skip custom/complex/transparent blocks for greedy meshing
				// Z-Axis (Sides): Skip multi-element blocks (Grass) so Custom Mesher handles the detailed sides
				if def, ok := registry.Blocks[bt]; ok && (!def.IsSolid || len(def.Elements) > 1) {
					continue
				}
				visible := false
				localNZ := z + nz
				if localNZ >= 0 && localNZ < sz {
					nbt := c.GetBlock(x, y, localNZ)
					if nbt == world.BlockTypeAir {
						visible = true
					} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
						visible = true
					}
				} else {
					wx, wy, wz := baseX+x, baseY+y, baseZ+z
					nxw, nyw, nzw := wx, wy, wz+nz

					nChunk := w.GetChunkFromBlockCoords(nxw, nyw, nzw, false)
					if nChunk == nil {
						visible = true
					} else {
						nbt := w.Get(nxw, nyw, nzw)
						if nbt == world.BlockTypeAir {
							visible = true
						} else if def, ok := registry.Blocks[nbt]; ok && !def.IsSolid {
							visible = true
						}
					}
				}

				if visible {
					faceIdx := 0 // North
					if nz < 0 {
						faceIdx = 1
					} // South
					texID := getTex(bt, faceIdx)
					tint := getTint(bt, faceIdx)
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
			fnx, fny, fnz := float32(0), float32(0), float32(nz)
			if nz > 0 { // +Z
				emitQuad(
					x0, y0, fz,
					x0+hHeight, y0, fz,
					x0+hHeight, y0+wWidth, fz,
					x0, y0+wWidth, fz,
					fnx, fny, fnz,
					texID,
					tint,
				)
			} else { // -Z
				emitQuad(
					x0, y0, fz,
					x0, y0+wWidth, fz,
					x0+hHeight, y0+wWidth, fz,
					x0+hHeight, y0, fz,
					fnx, fny, fnz,
					texID,
					tint,
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
