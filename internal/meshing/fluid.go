package meshing

import (
	"math"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// BuildFluidMesh generates vertices for fluid blocks (water/lava) in the chunk.
// Vertex format: Pos(3), UV(2), TexID(1), Tint(3), FlowAngle(1) = 10 floats.
func BuildFluidMesh(w *world.World, c *world.Chunk) []float32 {
	var vertices []float32

	// Base coordinates
	baseX := c.X * world.ChunkSizeX
	baseY := c.Y * world.ChunkSizeY
	baseZ := c.Z * world.ChunkSizeZ

	for x := 0; x < world.ChunkSizeX; x++ {
		for z := 0; z < world.ChunkSizeZ; z++ {
			for y := 0; y < world.ChunkSizeY; y++ {
				blockType := c.GetBlock(x, y, z)

				// Handle Water and Lava
				if blockType != world.BlockTypeWater && blockType != world.BlockTypeLava {
					continue
				}

				renderFluidBlock(w, c, x, y, z, baseX, baseY, baseZ, blockType, &vertices)

			}
		}
	}

	return vertices
}

func renderFluidBlock(w *world.World, c *world.Chunk, x, y, z int, baseX, baseY, baseZ int, blockType world.BlockType, vertices *[]float32) {
	// BlockPos in world coords
	wx, wy, wz := baseX+x, baseY+y, baseZ+z

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

	// Neighbor checks for visibility
	shouldRenderFace := func(dx, dy, dz int, face world.BlockFace) bool {
		nType := w.Get(wx+dx, wy+dy, wz+dz)
		if nType == blockType {
			return false
		}
		if world.BlockSolidTable[nType] {
			return false
		}
		return true
	}

	renderUp := shouldRenderFace(0, 1, 0, world.FaceTop)
	renderDown := shouldRenderFace(0, -1, 0, world.FaceBottom)
	renderNorth := shouldRenderFace(0, 0, -1, world.FaceNorth)
	renderSouth := shouldRenderFace(0, 0, 1, world.FaceSouth)
	renderWest := shouldRenderFace(-1, 0, 0, world.FaceWest)
	renderEast := shouldRenderFace(1, 0, 0, world.FaceEast)

	if !renderUp && !renderDown && !renderNorth && !renderSouth && !renderWest && !renderEast {
		return
	}

	// Heights
	f7 := getFluidHeight(w, wx, wy, wz, blockType)     // NW
	f8 := getFluidHeight(w, wx, wy, wz+1, blockType)   // SW
	f9 := getFluidHeight(w, wx+1, wy, wz+1, blockType) // SE
	f10 := getFluidHeight(w, wx+1, wy, wz, blockType)  // NE

	// Fluid margin
	f11 := float32(0.001)

	// Render Top
	if renderUp {
		f7 -= f11
		f8 -= f11
		f9 -= f11
		f10 -= f11

		meta := int(w.GetMeta(wx, wy, wz))
		var texID float32
		var flowAngle float32
		if meta == 0 {
			// Source block: still texture, slow isotropic animation
			texID = float32(stillTex)
			flowAngle = -1.0
		} else {
			// Flowing block: flow texture, directional animation
			texID = float32(flowTex)
			flowAngle = computeFlowAngle(w, wx, wy, wz, blockType)
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

// computeFlowAngle computes the MC-style flow direction for a flowing fluid block.
// Returns the angle (radians) in the XZ plane, or -1.0 if no directional flow.
func computeFlowAngle(w *world.World, wx, wy, wz int, blockType world.BlockType) float32 {
	myMeta := int(w.GetMeta(wx, wy, wz))
	if myMeta >= 8 {
		myMeta = 0 // falling blocks have source-level effective level
	}

	fdx, fdz := float32(0), float32(0)

	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range dirs {
		nx, nz := wx+d[0], wz+d[1]
		nb := w.Get(nx, wy, nz)

		if nb == blockType {
			nm := int(w.GetMeta(nx, wy, nz))
			if nm >= 8 {
				nm = 0
			}
			// Water flows from lower level (source=0) toward higher level (7)
			diff := nm - myMeta
			fdx += float32(d[0]) * float32(diff)
			fdz += float32(d[1]) * float32(diff)
		} else if !world.BlockSolidTable[nb] {
			// Air or replaceable neighbor - check for a drop below
			below := w.Get(nx, wy-1, nz)
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

func getFluidHeight(w *world.World, x, y, z int, blockType world.BlockType) float32 {
	// MC 1.8.9 BlockLiquid.getFluidHeight() - computes the average fluid height
	// at corner (x, z) by sampling the 4 blocks sharing that corner.
	count := 0
	sum := float32(0.0)

	for j := 0; j < 4; j++ {
		dx := -(j & 1)
		dz := -(j >> 1 & 1)

		bx, by, bz := x+dx, y, z+dz

		// If same fluid is directly above this corner block, height is 1.0
		if w.Get(bx, by+1, bz) == blockType {
			return 1.0
		}

		b := w.Get(bx, by, bz)

		if b != blockType {
			// Not fluid - if not solid, contributes height 0 (1 unit of drop)
			if !world.BlockSolidTable[b] {
				sum += 1.0
				count++
			}
			// Solid blocks are ignored (don't contribute to average)
		} else {
			// Same fluid block - read metadata for level
			meta := int(w.GetMeta(bx, by, bz))
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
