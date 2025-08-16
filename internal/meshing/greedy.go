package meshing

import (
	"mini-mc/internal/world"
)

// VertexStride is number of float32 per vertex (pos.xyz + normal.xyz)
const VertexStride = 6

// BuildGreedyMeshForChunk builds a greedy-meshed triangle list (pos+normal interleaved)
// for the given chunk using world coordinates to decide face visibility across chunk borders.
func BuildGreedyMeshForChunk(w *world.World, c *world.Chunk) []float32 {
	if c == nil {
		return nil
	}

	vertices := make([]float32, 0, 1024)

	// World-space offset of this chunk
	baseX := c.X * world.ChunkSizeX
	baseY := c.Y * world.ChunkSizeY
	baseZ := c.Z * world.ChunkSizeZ

	// +X faces (east)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, +1, 0, 0)...)
	// -X faces (west)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, -1, 0, 0)...)
	// +Y faces (top)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, 0, +1, 0)...)
	// -Y faces (bottom)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, 0, -1, 0)...)
	// +Z faces (north)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, 0, 0, +1)...)
	// -Z faces (south)
	vertices = append(vertices, buildGreedyForDirection(w, c, baseX, baseY, baseZ, 0, 0, -1)...)

	return vertices
}

// buildGreedyForDirection performs 2D greedy meshing for one face direction.
// The direction is specified by a normal (nx,ny,nz) where exactly one component is -1 or +1 and the others are 0.
// It returns interleaved vertices (pos+normal) forming triangles.
func buildGreedyForDirection(w *world.World, c *world.Chunk, baseX, baseY, baseZ int, nx, ny, nz int) []float32 {
	// Determine the axis fixed by the face normal and the two in-plane axes (u,v)
	// We will iterate layers along the normal axis, and build a UxV mask for each layer.
	var (
		// size along x,y,z
		sx, sy, sz = world.ChunkSizeX, world.ChunkSizeY, world.ChunkSizeZ
		vertices   []float32
	)

	// Helper lambda to push a quad made of two triangles with given 4 corners and normal
	emitQuad := func(x0, y0, z0, x1, y1, z1, x2, y2, z2, x3, y3, z3 float32, fnx, fny, fnz float32) {
		// shift coordinates by -0.5 to align block centers to integer positions
		ox := float32(0.5)
		// Triangle 1: v0,v1,v2
		vertices = append(vertices,
			x0-ox, y0-ox, z0-ox, fnx, fny, fnz,
			x1-ox, y1-ox, z1-ox, fnx, fny, fnz,
			x2-ox, y2-ox, z2-ox, fnx, fny, fnz,
		)
		// Triangle 2: v2,v3,v0
		vertices = append(vertices,
			x2-ox, y2-ox, z2-ox, fnx, fny, fnz,
			x3-ox, y3-ox, z3-ox, fnx, fny, fnz,
			x0-ox, y0-ox, z0-ox, fnx, fny, fnz,
		)
	}

	// Build per-layer masks and greedy-merge
	// Cases per normal axis
	if nx != 0 { // Faces perpendicular to X axis, plane is Y-Z
		// Layers along X
		for x := 0; x < sx; x++ {
			// Mask size sy x sz, store 0 or 1 (face visible)
			mask := make([]int, sy*sz)
			for y := 0; y < sy; y++ {
				for z := 0; z < sz; z++ {
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}
					// Prefer local neighbor when within this chunk; only query world across chunk borders
					localNX := x + nx
					if localNX >= 0 && localNX < sx {
						if c.IsAir(localNX, y, z) {
							mask[y*sz+z] = 1
						}
					} else {
						wx := baseX + x
						wy := baseY + y
						wz := baseZ + z
						nxw := wx + nx
						nyw := wy
						nzw := wz
						// If neighbor chunk missing, treat as air so outward faces render until neighbor arrives
						if w.GetChunkFromBlockCoords(nxw, nyw, nzw, false) == nil || w.IsAir(nxw, nyw, nzw) {
							mask[y*sz+z] = 1
						}
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
				// compute width
				z0 := i % sz
				y0 := i / sz
				wWidth := 1
				for z1 := z0 + 1; z1 < sz && mask[y0*sz+z1] == 1; z1++ {
					wWidth++
				}
				// compute height
				hHeight := 1
			outerYZ:
				for y1 := y0 + 1; y1 < sy; y1++ {
					for z1 := z0; z1 < z0+wWidth; z1++ {
						if mask[y1*sz+z1] != 1 {
							break outerYZ
						}
					}
					hHeight++
				}
				// emit quad at plane x or x+1 depending on nx sign
				fx := float32(baseX + x)
				if nx > 0 {
					fx = float32(baseX + x + 1)
				}
				fy0 := float32(baseY + y0)
				fz0 := float32(baseZ + z0)
				fy1 := float32(baseY + y0 + hHeight)
				fz1 := float32(baseZ + z0 + wWidth)
				fnx := float32(nx)
				fny := float32(0)
				fnz := float32(0)
				// CCW winding for outward normal
				if nx > 0 { // +X
					emitQuad(
						fx, fy0, fz0,
						fx, fy1, fz0,
						fx, fy1, fz1,
						fx, fy0, fz1,
						fnx, fny, fnz,
					)
				} else { // -X
					emitQuad(
						fx, fy0, fz0,
						fx, fy0, fz1,
						fx, fy1, fz1,
						fx, fy1, fz0,
						fnx, fny, fnz,
					)
				}
				// zero-out mask region
				for yy := y0; yy < y0+hHeight; yy++ {
					for zz := z0; zz < z0+wWidth; zz++ {
						mask[yy*sz+zz] = 0
					}
				}
			}
		}
		return vertices
	}

	if ny != 0 { // Faces perpendicular to Y axis, plane is X-Z
		for y := 0; y < sy; y++ {
			mask := make([]int, sx*sz)
			for x := 0; x < sx; x++ {
				for z := 0; z < sz; z++ {
					bt := c.GetBlock(x, y, z)
					if bt == world.BlockTypeAir {
						continue
					}
					// Prefer local neighbor when within this chunk; only query world across chunk borders
					localNY := y + ny
					if localNY >= 0 && localNY < sy {
						if c.IsAir(x, localNY, z) {
							mask[x*sz+z] = 1
						}
					} else {
						// Crossing into neighbor chunk along Y. Treat missing neighbor as AIR so top/bottom
						// surfaces at the world boundary are rendered.
						wx := baseX + x
						wy := baseY + y
						wz := baseZ + z
						nxw := wx
						nyw := wy + ny
						nzw := wz
						if w.GetChunkFromBlockCoords(nxw, nyw, nzw, false) == nil || w.IsAir(nxw, nyw, nzw) {
							mask[x*sz+z] = 1
						}
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
				x0 := i / sz
				z0 := i % sz
				wWidth := 1
				for z1 := z0 + 1; z1 < sz && mask[x0*sz+z1] == 1; z1++ {
					wWidth++
				}
				hHeight := 1
			outerXZ:
				for x1 := x0 + 1; x1 < sx; x1++ {
					for z1 := z0; z1 < z0+wWidth; z1++ {
						if mask[x1*sz+z1] != 1 {
							break outerXZ
						}
					}
					hHeight++
				}
				fy := float32(baseY + y)
				if ny > 0 {
					fy = float32(baseY + y + 1)
				}
				fx0 := float32(baseX + x0)
				fz0 := float32(baseZ + z0)
				fx1 := float32(baseX + x0 + hHeight)
				fz1 := float32(baseZ + z0 + wWidth)
				fnx := float32(0)
				fny := float32(ny)
				fnz := float32(0)
				if ny > 0 { // +Y (top) — CCW
					emitQuad(
						fx0, fy, fz0,
						fx0, fy, fz1,
						fx1, fy, fz1,
						fx1, fy, fz0,
						fnx, fny, fnz,
					)
				} else { // -Y (bottom) — CCW
					emitQuad(
						fx0, fy, fz0,
						fx1, fy, fz0,
						fx1, fy, fz1,
						fx0, fy, fz1,
						fnx, fny, fnz,
					)
				}
				for xx := x0; xx < x0+hHeight; xx++ {
					for zz := z0; zz < z0+wWidth; zz++ {
						mask[xx*sz+zz] = 0
					}
				}
			}
		}
		return vertices
	}

	// nz != 0 // Faces perpendicular to Z axis, plane is X-Y
	for z := 0; z < sz; z++ {
		mask := make([]int, sx*sy)
		for x := 0; x < sx; x++ {
			for y := 0; y < sy; y++ {
				bt := c.GetBlock(x, y, z)
				if bt == world.BlockTypeAir {
					continue
				}
				// Prefer local neighbor when within this chunk; only query world across chunk borders
				localNZ := z + nz
				if localNZ >= 0 && localNZ < sz {
					if c.IsAir(x, y, localNZ) {
						mask[x*sy+y] = 1
					}
				} else {
					wx := baseX + x
					wy := baseY + y
					wz := baseZ + z
					nxw := wx
					nyw := wy
					nzw := wz + nz
					// If neighbor chunk missing, treat as air so outward faces render until neighbor arrives
					if w.GetChunkFromBlockCoords(nxw, nyw, nzw, false) == nil || w.IsAir(nxw, nyw, nzw) {
						mask[x*sy+y] = 1
					}
				}
			}
		}
		// Greedy (u=x, v=y)
		i := 0
		for i < sx*sy {
			if mask[i] == 0 {
				i++
				continue
			}
			x0 := i / sy
			y0 := i % sy
			wWidth := 1
			for y1 := y0 + 1; y1 < sy && mask[x0*sy+y1] == 1; y1++ {
				wWidth++
			}
			hHeight := 1
		outerXY:
			for x1 := x0 + 1; x1 < sx; x1++ {
				for y1 := y0; y1 < y0+wWidth; y1++ {
					if mask[x1*sy+y1] != 1 {
						break outerXY
					}
				}
				hHeight++
			}
			fz := float32(baseZ + z)
			if nz > 0 {
				fz = float32(baseZ + z + 1)
			}
			fx0 := float32(baseX + x0)
			fy0 := float32(baseY + y0)
			fx1 := float32(baseX + x0 + hHeight)
			fy1 := float32(baseY + y0 + wWidth)
			fnx := float32(0)
			fny := float32(0)
			fnz := float32(nz)
			if nz > 0 { // +Z (north) — CCW outward (matches gl.FrontFace(CCW))
				emitQuad(
					fx0, fy0, fz,
					fx1, fy0, fz,
					fx1, fy1, fz,
					fx0, fy1, fz,
					fnx, fny, fnz,
				)
			} else { // -Z (south) — CCW outward
				emitQuad(
					fx0, fy0, fz,
					fx0, fy1, fz,
					fx1, fy1, fz,
					fx1, fy0, fz,
					fnx, fny, fnz,
				)
			}
			for xx := x0; xx < x0+hHeight; xx++ {
				for yy := y0; yy < y0+wWidth; yy++ {
					mask[xx*sy+yy] = 0
				}
			}
		}
	}
	return vertices
}
