package meshing

import (
	"math"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// neighbors6 holds the 6 face-adjacent pre-fetched neighbor chunks.
// Index layout matches the face directions used throughout this file:
//
//	0: +X (east),  1: -X (west)
//	2: +Y (up),    3: -Y (down)
//	4: +Z (south), 5: -Z (north)
type neighbors6 = [6]*world.Chunk

// getBlockLocal returns the block type for a neighbor offset from a local chunk
// coordinate (lx, ly, lz).  When the offset stays inside the chunk bounds it
// calls c.GetBlock directly (no mutex).  When it crosses a chunk boundary it
// uses the appropriate pre-fetched neighbor chunk (also no mutex).  A nil
// neighbor is treated as air, which makes the face visible — the conservative
// choice.
func getBlockLocal(c *world.Chunk, nb neighbors6, lx, ly, lz int) world.BlockType {
	// Fast path: fully interior — most calls land here.
	if lx >= 0 && lx < world.ChunkSizeX &&
		ly >= 0 && ly < world.ChunkSizeY &&
		lz >= 0 && lz < world.ChunkSizeZ {
		return c.GetBlock(lx, ly, lz)
	}

	// Determine which neighbor chunk to use and translate local coordinates.
	var nc *world.Chunk
	nlx, nly, nlz := lx, ly, lz

	switch {
	case lx >= world.ChunkSizeX:
		nc = nb[0] // +X
		nlx = lx - world.ChunkSizeX
	case lx < 0:
		nc = nb[1] // -X
		nlx = lx + world.ChunkSizeX
	case ly >= world.ChunkSizeY:
		nc = nb[2] // +Y
		nly = ly - world.ChunkSizeY
	case ly < 0:
		nc = nb[3] // -Y
		nly = ly + world.ChunkSizeY
	case lz >= world.ChunkSizeZ:
		nc = nb[4] // +Z
		nlz = lz - world.ChunkSizeZ
	case lz < 0:
		nc = nb[5] // -Z
		nlz = lz + world.ChunkSizeZ
	}

	if nc == nil {
		return world.BlockTypeAir
	}
	return nc.GetBlock(nlx, nly, nlz)
}

// getMetaLocal is like getBlockLocal but returns the metadata byte.
func getMetaLocal(c *world.Chunk, nb neighbors6, lx, ly, lz int) uint8 {
	if lx >= 0 && lx < world.ChunkSizeX &&
		ly >= 0 && ly < world.ChunkSizeY &&
		lz >= 0 && lz < world.ChunkSizeZ {
		return c.GetMeta(lx, ly, lz)
	}

	var nc *world.Chunk
	nlx, nly, nlz := lx, ly, lz

	switch {
	case lx >= world.ChunkSizeX:
		nc = nb[0]
		nlx = lx - world.ChunkSizeX
	case lx < 0:
		nc = nb[1]
		nlx = lx + world.ChunkSizeX
	case ly >= world.ChunkSizeY:
		nc = nb[2]
		nly = ly - world.ChunkSizeY
	case ly < 0:
		nc = nb[3]
		nly = ly + world.ChunkSizeY
	case lz >= world.ChunkSizeZ:
		nc = nb[4]
		nlz = lz - world.ChunkSizeZ
	case lz < 0:
		nc = nb[5]
		nlz = lz + world.ChunkSizeZ
	}

	if nc == nil {
		return 0
	}
	return nc.GetMeta(nlx, nly, nlz)
}

// BuildFluidMesh generates vertices for fluid blocks (water/lava) in the chunk.
// Vertex format: Pos(3), UV(2), TexID(1), Tint(3), FlowAngle(1) = 10 floats.
func BuildFluidMesh(w *world.World, c *world.Chunk) []float32 {
	var vertices []float32

	// Pre-fetch all 6 face-adjacent neighbor chunks once.
	// This is the key optimisation: instead of going through ChunkStore.Get()
	// (with its RWMutex) for every single neighbor lookup, we pay the 6 mutex
	// acquisitions once here and then do all subsequent lookups mutex-free.
	nb := neighbors6{
		w.GetChunk(c.X+1, c.Y, c.Z, false), // 0: +X east
		w.GetChunk(c.X-1, c.Y, c.Z, false), // 1: -X west
		w.GetChunk(c.X, c.Y+1, c.Z, false), // 2: +Y up
		w.GetChunk(c.X, c.Y-1, c.Z, false), // 3: -Y down
		w.GetChunk(c.X, c.Y, c.Z+1, false), // 4: +Z south
		w.GetChunk(c.X, c.Y, c.Z-1, false), // 5: -Z north
	}

	// Base world coordinates for this chunk.
	baseX := c.X * world.ChunkSizeX
	baseY := c.Y * world.ChunkSizeY
	baseZ := c.Z * world.ChunkSizeZ

	// Iterate section-by-section so we can skip empty sections cheaply.
	for secIdx := 0; secIdx < world.NumSections; secIdx++ {
		if c.IsSectionEmpty(secIdx) {
			continue
		}

		sectionBaseY := secIdx * world.SectionHeight

		for lx := 0; lx < world.ChunkSizeX; lx++ {
			for lz := 0; lz < world.ChunkSizeZ; lz++ {
				for ly := 0; ly < world.SectionHeight; ly++ {
					y := sectionBaseY + ly
					blockType := c.GetBlock(lx, y, lz)

					if blockType != world.BlockTypeWater && blockType != world.BlockTypeLava {
						continue
					}

					renderFluidBlock(c, nb, lx, y, lz, baseX, baseY, baseZ, blockType, &vertices)
				}
			}
		}
	}

	return vertices
}

func renderFluidBlock(c *world.Chunk, nb neighbors6, lx, ly, lz int, baseX, baseY, baseZ int, blockType world.BlockType, vertices *[]float32) {
	// World-space position of this block.
	wx, wy, wz := baseX+lx, baseY+ly, baseZ+lz

	// Get Material/Textures
	isLava := blockType == world.BlockTypeLava

	var stillTex, flowTex int

	if isLava {
		stillTex = registry.TextureMap["lava_still.png"]
		flowTex = registry.TextureMap["lava_flow.png"]
	} else {
		stillTex = registry.TextureMap["water_still.png"]
		flowTex = registry.TextureMap["water_flow.png"]
	}

	// Neighbor visibility checks — all mutex-free via chunk-local lookups.
	shouldRenderFace := func(dlx, dly, dlz int) bool {
		nType := getBlockLocal(c, nb, lx+dlx, ly+dly, lz+dlz)
		if nType == blockType {
			return false
		}
		if world.BlockSolidTable[nType] {
			return false
		}
		return true
	}

	renderUp := shouldRenderFace(0, 1, 0)
	renderDown := shouldRenderFace(0, -1, 0)
	renderNorth := shouldRenderFace(0, 0, -1)
	renderSouth := shouldRenderFace(0, 0, 1)
	renderWest := shouldRenderFace(-1, 0, 0)
	renderEast := shouldRenderFace(1, 0, 0)

	if !renderUp && !renderDown && !renderNorth && !renderSouth && !renderWest && !renderEast {
		return
	}

	// Heights — use local variant that avoids ChunkStore lookups.
	f7 := getFluidHeightLocal(c, nb, lx, ly, lz, blockType)     // NW
	f8 := getFluidHeightLocal(c, nb, lx, ly, lz+1, blockType)   // SW
	f9 := getFluidHeightLocal(c, nb, lx+1, ly, lz+1, blockType) // SE
	f10 := getFluidHeightLocal(c, nb, lx+1, ly, lz, blockType)  // NE

	// Fluid margin
	f11 := float32(0.001)

	// Render Top
	if renderUp {
		f7 -= f11
		f8 -= f11
		f9 -= f11
		f10 -= f11

		meta := int(c.GetMeta(lx, ly, lz))
		var texID float32
		var flowAngle float32
		if meta == 0 {
			// Source block: still texture, slow isotropic animation
			texID = float32(stillTex)
			flowAngle = -1.0
		} else {
			// Flowing block: flow texture, directional animation
			texID = float32(flowTex)
			flowAngle = computeFlowAngleLocal(c, nb, lx, ly, lz, blockType)
		}

		u1, v1 := float32(0.0), float32(0.0)
		u2, v2 := float32(0.0), float32(1.0)
		u3, v3 := float32(1.0), float32(1.0)
		u4, v4 := float32(1.0), float32(0.0)

		// Tri 1: NW, SW, SE
		emitVertex(vertices, float32(wx), float32(wy)+f7, float32(wz), u1, v1, texID, flowAngle)
		emitVertex(vertices, float32(wx), float32(wy)+f8, float32(wz)+1.0, u2, v2, texID, flowAngle)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f9, float32(wz)+1.0, u3, v3, texID, flowAngle)

		// Tri 2: NW, SE, NE
		emitVertex(vertices, float32(wx), float32(wy)+f7, float32(wz), u1, v1, texID, flowAngle)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f9, float32(wz)+1.0, u3, v3, texID, flowAngle)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f10, float32(wz), u4, v4, texID, flowAngle)
	}

	// Render Bottom
	if renderDown {
		texID := float32(stillTex)
		yBottom := float32(wy)

		emitVertex(vertices, float32(wx), yBottom, float32(wz)+1.0, 0, 1, texID, -3.0)
		emitVertex(vertices, float32(wx), yBottom, float32(wz), 0, 0, texID, -3.0)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz), 1, 0, texID, -3.0)

		emitVertex(vertices, float32(wx), yBottom, float32(wz)+1.0, 0, 1, texID, -3.0)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz), 1, 0, texID, -3.0)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz)+1.0, 1, 1, texID, -3.0)
	}

	// Sides: flow texture, scroll downward (-2.0 sentinel)
	texID := float32(flowTex)

	renderSide := func(x1, z1, x2, z2 float32, h1, h2 float32, uStart float32) {
		v1 := 1.0 - h1
		v2 := 1.0 - h2

		// Tri 1
		emitVertex(vertices, float32(wx)+x1, float32(wy)+h1, float32(wz)+z1, uStart, v1, texID, -2.0)
		emitVertex(vertices, float32(wx)+x2, float32(wy)+h2, float32(wz)+z2, uStart+1.0, v2, texID, -2.0)
		emitVertex(vertices, float32(wx)+x2, float32(wy), float32(wz)+z2, uStart+1.0, 1.0, texID, -2.0)

		// Tri 2
		emitVertex(vertices, float32(wx)+x1, float32(wy)+h1, float32(wz)+z1, uStart, v1, texID, -2.0)
		emitVertex(vertices, float32(wx)+x2, float32(wy), float32(wz)+z2, uStart+1.0, 1.0, texID, -2.0)
		emitVertex(vertices, float32(wx)+x1, float32(wy), float32(wz)+z1, uStart, 1.0, texID, -2.0)
	}

	if renderNorth { // -Z
		renderSide(0, 0, 1, 0, f7, f10, 0)
	}
	if renderSouth { // +Z
		renderSide(1, 1, 0, 1, f9, f8, 0)
	}
	if renderWest { // -X
		renderSide(0, 1, 0, 0, f8, f7, 0)
	}
	if renderEast { // +X
		renderSide(1, 0, 1, 1, f10, f9, 0)
	}
}

// flowAngle encoding:
//
//	-1.0 = still water top (slow isotropic scroll)
//	-2.0 = side face (scroll downward)
//	-3.0 = bottom face (no animation)
//	>=0  = directional flowing top (angle in radians, XZ plane)
func emitVertex(vertices *[]float32, x, y, z float32, u, v float32, texID float32, flowAngle float32) {
	r, g, b := float32(1.0), float32(1.0), float32(1.0)
	*vertices = append(*vertices, x, y, z, u, v, texID, r, g, b, flowAngle)
}

// computeFlowAngleLocal is the chunk-local variant of computeFlowAngle.
// It uses getBlockLocal / getMetaLocal instead of w.Get / w.GetMeta so that
// none of the neighbor lookups require a mutex acquisition.
func computeFlowAngleLocal(c *world.Chunk, nb neighbors6, lx, ly, lz int, blockType world.BlockType) float32 {
	myMeta := int(c.GetMeta(lx, ly, lz))
	if myMeta >= 8 {
		myMeta = 0 // falling blocks have source-level effective level
	}

	fdx, fdz := float32(0), float32(0)

	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range dirs {
		nlx, nlz := lx+d[0], lz+d[1]
		nb_ := getBlockLocal(c, nb, nlx, ly, nlz)

		if nb_ == blockType {
			nm := int(getMetaLocal(c, nb, nlx, ly, nlz))
			if nm >= 8 {
				nm = 0
			}
			// Water flows from lower level (source=0) toward higher level (7)
			diff := nm - myMeta
			fdx += float32(d[0]) * float32(diff)
			fdz += float32(d[1]) * float32(diff)
		} else if !world.BlockSolidTable[nb_] {
			// Air or replaceable neighbor — check for a drop below
			below := getBlockLocal(c, nb, nlx, ly-1, nlz)
			if !world.BlockSolidTable[below] && below != blockType {
				// Strong attraction toward drop-off edges
				fdx += float32(d[0]) * 8.0
				fdz += float32(d[1]) * 8.0
			}
		}
	}

	if fdx == 0 && fdz == 0 {
		return -1.0
	}

	// Normalize to [0, 2π) so negative angles don't collide with sentinel values.
	angle := math.Atan2(float64(fdz), float64(fdx))
	if angle < 0 {
		angle += 2 * math.Pi
	}
	return float32(angle)
}

// getFluidHeightLocal is the chunk-local variant of getFluidHeight.
// (lx, ly, lz) are the local coordinates of the fluid block whose corner height
// we are computing.  The four corner-sharing blocks are sampled via
// getBlockLocal / getMetaLocal, so no ChunkStore mutex is involved.
func getFluidHeightLocal(c *world.Chunk, nb neighbors6, lx, ly, lz int, blockType world.BlockType) float32 {
	// MC 1.8.9 BlockLiquid.getFluidHeight() - computes the average fluid height
	// at corner (lx, lz) by sampling the 4 blocks sharing that corner.
	count := 0
	sum := float32(0.0)

	for j := 0; j < 4; j++ {
		dx := -(j & 1)
		dz := -(j >> 1 & 1)

		bx, by, bz := lx+dx, ly, lz+dz

		// If same fluid is directly above this corner block, height is 1.0
		if getBlockLocal(c, nb, bx, by+1, bz) == blockType {
			return 1.0
		}

		b := getBlockLocal(c, nb, bx, by, bz)

		if b != blockType {
			// Not fluid - if not solid, contributes height 0 (1 unit of drop)
			if !world.BlockSolidTable[b] {
				sum += 1.0
				count++
			}
			// Solid blocks are ignored (don't contribute to average)
		} else {
			// Same fluid block - read metadata for level
			meta := int(getMetaLocal(c, nb, bx, by, bz))
			if meta >= 8 || meta == 0 {
				// Source (0) or falling (>=8) - weighted ×10
				sum += world.GetLiquidHeightPercent(meta) * 10.0
				count += 10
			}
			sum += world.GetLiquidHeightPercent(meta)
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return 1.0 - sum/float32(count)
}

// computeFlowAngle is the World-based wrapper kept for tests and callers that
// don't have pre-fetched neighbor chunks.  Hot path code should use
// computeFlowAngleLocal instead.
func computeFlowAngle(w *world.World, wx, wy, wz int, blockType world.BlockType) float32 {
	c, nb, lx, ly, lz := worldToLocal(w, wx, wy, wz)
	if c == nil {
		return -1.0
	}
	return computeFlowAngleLocal(c, nb, lx, ly, lz, blockType)
}

// getFluidHeight is the World-based wrapper kept for tests and callers that
// don't have pre-fetched neighbor chunks.
func getFluidHeight(w *world.World, wx, wy, wz int, blockType world.BlockType) float32 {
	c, nb, lx, ly, lz := worldToLocal(w, wx, wy, wz)
	if c == nil {
		return 0.0
	}
	return getFluidHeightLocal(c, nb, lx, ly, lz, blockType)
}

// worldToLocal resolves world coordinates to a chunk, its 6 neighbors, and
// local coordinates.  Returns nil chunk when the position is unloaded.
func worldToLocal(w *world.World, wx, wy, wz int) (*world.Chunk, neighbors6, int, int, int) {
	cx := floorDivM(wx, world.ChunkSizeX)
	cy := floorDivM(wy, world.ChunkSizeY)
	cz := floorDivM(wz, world.ChunkSizeZ)
	c := w.GetChunk(cx, cy, cz, false)
	if c == nil {
		return nil, neighbors6{}, 0, 0, 0
	}
	nb := neighbors6{
		w.GetChunk(cx+1, cy, cz, false),
		w.GetChunk(cx-1, cy, cz, false),
		w.GetChunk(cx, cy+1, cz, false),
		w.GetChunk(cx, cy-1, cz, false),
		w.GetChunk(cx, cy, cz+1, false),
		w.GetChunk(cx, cy, cz-1, false),
	}
	lx := modM(wx, world.ChunkSizeX)
	ly := modM(wy, world.ChunkSizeY)
	lz := modM(wz, world.ChunkSizeZ)
	return c, nb, lx, ly, lz
}

func floorDivM(a, b int) int {
	if a >= 0 {
		return a / b
	}
	return (a - b + 1) / b
}

func modM(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}
