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
		if def, ok := registry.Blocks[nType]; ok && def.IsSolid && !def.IsTransparent {
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

		// Calculate Flow Direction
		// Java: getFlowDirection
		// Since we don't have metadata (level), flow direction is static or 0.
		// We'll skip complex flow direction calculation for now and use "Still" texture.
		// Or implement a simple version if possible.

		// For now: Use Still texture on top if level is source (which it is)
		// But usually top texture is animated.

		texID := float32(stillTex)

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
	// Java Logic adapted
	// Checks 4 neighbors around the corner (x,z)
	// (x-1, z-1), (x, z-1), (x-1, z), (x, z)
	// But `getFluidHeight` in Java takes a BlockPos and Material.
	// It checks the block itself and neighbors to compute the average height at that corner.

	// Implementation:
	// To get height at corner (x, z) (relative to block), we need to check the 4 blocks sharing that corner.
	// But the loop in renderFluid calls getFluidHeight with (x,z), (x, z+1)...
	// which implies getFluidHeight calculates height for the block column?
	// No, Java calls it with BlockPos.

	// Java Logic:
	/*
	   int i = 0;
	   float f = 0.0F;
	   for (int j = 0; j < 4; ++j) {
	       BlockPos blockpos = blockPosIn.add(-(j & 1), 0, -(j >> 1 & 1));
	       // Checks block ABOVE
	       if (blockAccess.getBlockState(blockpos.up()).getBlock().getMaterial() == blockMaterial) {
	           return 1.0F;
	       }
	       // Checks block itself
	       IBlockState iblockstate = blockAccess.getBlockState(blockpos);
	       Material material = iblockstate.getBlock().getMaterial();
	       if (material != blockMaterial) {
	           if (!material.isSolid()) {
	               ++f;
	               ++i;
	           }
	       } else {
	            int k = ((Integer)iblockstate.getValue(BlockLiquid.LEVEL)).intValue();
	            if (k >= 8 || k == 0) {
	                f += BlockLiquid.getLiquidHeightPercent(k) * 10.0F;
	                i += 10;
	            }
	            f += BlockLiquid.getLiquidHeightPercent(k);
	            ++i;
	       }
	   }
	   return 1.0F - f / (float)i;
	*/

	// We emulate this.
	// blockPosIn is the block coordinate being rendered? No, Java calls it with corner offsets?
	// Java calls: getFluidHeight(pos), getFluidHeight(pos.south()), etc.
	// This implies getFluidHeight computes height at the top-left corner of the block 'pos'.

	count := 0
	sum := float32(0.0)

	for j := 0; j < 4; j++ {
		dx := -(j & 1)
		dz := -(j >> 1 & 1)

		bx, by, bz := x+dx, y, z+dz

		// Check block above
		if w.Get(bx, by+1, bz) == blockType {
			return 1.0
		}

		b := w.Get(bx, by, bz)

		if b != blockType {
			// If not fluid, check if solid
			def, ok := registry.Blocks[b]
			if !ok || !def.IsSolid {
				// Treated as empty/air (height 0, effectively)
				// Java: ++f (adds 1 to sum? No, f is "fluid drop". Height = 1 - f/i)
				// Java says: if !solid, ++f, ++i.
				// This means it contributes "1 unit of drop" -> height 0.
				sum += 1.0
				count++
			}
			// If solid, it's ignored (doesn't contribute to average)
		} else {
			// Is fluid.
			// We assume level 0 (full source).
			// k=0.
			// Java: if (k>=8 || k==0) { f += 0 * 10; i += 10; }
			// f += 0; i++;
			// So f adds 0. i adds 11.
			// Height = 1 - 0/11 = 1.0.

			// If we want simply 1.0 for source blocks:
			// sum += 0.0
			count += 11 // Weight source blocks heavily?
		}
	}

	if count == 0 {
		return 0.0 // Or 1.0? Java returns 1.0 - f/i. If i=0, NaN.
		// But i only stays 0 if all 4 neighbors are SOLID.
		// If surrounded by stone, water height is?
		// Java: return 1.0F - f / (float)i;
		// If i=0, we should probably return 0.
		// But in MC, water against glass is 1.0?
		// Let's return 0.0.
		return 0.0
	}

	return 1.0 - sum/float32(count)
}
