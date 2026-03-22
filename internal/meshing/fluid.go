package meshing

import (
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// BuildFluidMesh generates vertices for fluid blocks (water/lava) in the chunk.
// It uses a custom vertex format: Pos(3), UV(2), TexID(1), Tint(3) -> 9 floats.
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

				// Check if we should render this block
				// Optimization: If surrounded by same fluid, usually we skip faces,
				// but since we do volume rendering or custom height, we need to be careful.
				// The Java code renders "faces" individually.

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
	// We use the world to check neighbors to handle chunk boundaries correct
	shouldRenderFace := func(dx, dy, dz int, face world.BlockFace) bool {
		// Java: shouldSideBeRendered
		// If neighbor is same material (same fluid), we generally don't render side,
		// UNLESS it's the top and we have height difference?
		// For simplicity/parity: don't render if neighbor is same fluid.

		nType := w.Get(wx+dx, wy+dy, wz+dz)
		if nType == blockType {
			return false
		}
		// If neighbor is solid, don't render
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

		// Use still texture for source blocks (level 0), flow texture for flowing
		meta := int(w.GetMeta(wx, wy, wz))
		texID := float32(stillTex)
		if meta != 0 {
			texID = float32(flowTex)
		}

		// UVs
		// Java code interpolates UVs. We just use 0..1
		u1, v1 := float32(0.0), float32(0.0)
		u2, v2 := float32(0.0), float32(1.0)
		u3, v3 := float32(1.0), float32(1.0)
		u4, v4 := float32(1.0), float32(0.0)

		// 2 Triangles for Quad
		// (0, f7, 0) -> (0, f8, 1) -> (1, f9, 1) -> (1, f10, 0)

		// Tri 1: NW, SW, SE
		emitVertex(vertices, float32(wx), float32(wy)+f7, float32(wz), u1, v1, texID)
		emitVertex(vertices, float32(wx), float32(wy)+f8, float32(wz)+1.0, u2, v2, texID)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f9, float32(wz)+1.0, u3, v3, texID)

		// Tri 2: NW, SE, NE
		emitVertex(vertices, float32(wx), float32(wy)+f7, float32(wz), u1, v1, texID)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f9, float32(wz)+1.0, u3, v3, texID)
		emitVertex(vertices, float32(wx)+1.0, float32(wy)+f10, float32(wz), u4, v4, texID)
	}

	// Render Bottom
	if renderDown {
		texID := float32(stillTex)
		// Flat bottom
		yBottom := float32(wy)

		emitVertex(vertices, float32(wx), yBottom, float32(wz)+1.0, 0, 1, texID)
		emitVertex(vertices, float32(wx), yBottom, float32(wz), 0, 0, texID)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz), 1, 0, texID)

		emitVertex(vertices, float32(wx), yBottom, float32(wz)+1.0, 0, 1, texID)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz), 1, 0, texID)
		emitVertex(vertices, float32(wx)+1.0, yBottom, float32(wz)+1.0, 1, 1, texID)
	}

	// Sides
	// Using Flow Texture for sides
	texID := float32(flowTex)

	// Helper for sides
	renderSide := func(x1, z1, x2, z2 float32, h1, h2 float32, uStart float32) {
		// Quad from (x1,y,z1) to (x2,y,z2) with heights h1, h2

		// Simplified UVs for now
		v1 := 1.0 - h1
		v2 := 1.0 - h2

		// Tri 1
		emitVertex(vertices, float32(wx)+x1, float32(wy)+h1, float32(wz)+z1, uStart, v1, texID)
		emitVertex(vertices, float32(wx)+x2, float32(wy)+h2, float32(wz)+z2, uStart+1.0, v2, texID)
		emitVertex(vertices, float32(wx)+x2, float32(wy), float32(wz)+z2, uStart+1.0, 1.0, texID)

		// Tri 2
		emitVertex(vertices, float32(wx)+x1, float32(wy)+h1, float32(wz)+z1, uStart, v1, texID)
		emitVertex(vertices, float32(wx)+x2, float32(wy), float32(wz)+z2, uStart+1.0, 1.0, texID)
		emitVertex(vertices, float32(wx)+x1, float32(wy), float32(wz)+z1, uStart, 1.0, texID)
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

func emitVertex(vertices *[]float32, x, y, z float32, u, v float32, texID float32) {
	// Tint: White (1,1,1) for now
	r, g, b := float32(1.0), float32(1.0), float32(1.0)

	*vertices = append(*vertices, x, y, z, u, v, texID, r, g, b)
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
