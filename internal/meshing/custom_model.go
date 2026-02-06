package meshing

import (
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// meshCustomBlock generates vertices for a block with custom model elements
// matching the compressed vertex format used by the greedy mesher.
// meshCustomBlock generates vertices for a block with custom model elements.
// Note: Current vertex format enforces integer coordinates, so sub-voxel precision is lost/rounded.
func meshCustomBlock(w *world.World, c *world.Chunk, x, y, z int, def *registry.BlockDefinition) []uint32 {
	var vertices []uint32

	// Re-implementation of packVertex from neighbor logic (duplicated for now, or should be shared)
	// V1: X(5) Y(9) Z(5) N(3) B(8)
	// V2: T(16) C(16)
	packVertex := func(lx, ly, lz int, normal byte, texID int, brightness byte, tint uint16) (uint32, uint32) {
		v1 := uint32(lx) | (uint32(ly) << 5) | (uint32(lz) << 14) | (uint32(normal) << 19) | (uint32(brightness) << 22)
		v2 := uint32(texID) | (uint32(tint) << 16)
		return v1, v2
	}

	// Helper to resolve texture from the block definition for a specific face
	// In the registry, we only stored Top/Bot/Side.
	// We really should look at the ELEMENT's texture logic.
	// But `def.Elements[i].Faces[dir].Texture` is a string reference.
	// `registry.Blocks` doesn't expose the texture map anymore.
	// Fallback strategy: Use Top/Bot/Side based on direction.
	getTexID := func(faceDir string) int {
		var name string
		switch faceDir {
		case "up":
			name = def.TextureTop
		case "down":
			name = def.TextureBot
		default:
			name = def.TextureSide
		}
		if idx, ok := registry.TextureMap[name]; ok {
			return idx
		}
		return 0
	}

	for _, elem := range def.Elements {
		// Optimization for Hybrid Rendering:
		// If the block is Solid (like Grass), the Greedy Mesher handles the Top/Bottom faces (for optimization).
		// We must SKIP "up" and "down" faces here to prevent Z-fighting and double rendering.
		// However, we MUST render the side faces (North/South/East/West) because Greedy Mesher SKIPS them for complex blocks.
		// This ensures detailed sides (Base+Overlay) but optimized tops.

		// Convert 0-16 -> Integer Coords
		// minX := int(elem.From[0] / 16.0) // 0
		// maxX := int(elem.To[0] / 16.0)   // 1
		// This forces slabs to be flat or full.
		// For compliance, we emit the faces that exist, clamped to integer grid.

		for dir, face := range elem.Faces {
			// Skip Top/Bottom for solid blocks (Greedy handles them)
			if def.IsSolid && (dir == "up" || dir == "down") {
				continue
			}

			// determine normal
			var nm byte
			var nx, ny, nz float32
			switch dir {
			case "up":
				nm, nx, ny, nz = 4, 0, 1, 0
			case "down":
				nm, nx, ny, nz = 5, 0, -1, 0
			case "north":
				nm, nx, ny, nz = 1, 0, 0, -1
			case "south":
				nm, nx, ny, nz = 0, 0, 0, 1
			case "west":
				nm, nx, ny, nz = 3, -1, 0, 0
			case "east":
				nm, nx, ny, nz = 2, 1, 0, 0
			}
			_ = nx
			_ = ny
			_ = nz // unused logic vars

			texID := getTexID(dir)

			// Tint?
			tint := uint16(0xFFFF)
			if face.TintIndex != nil && *face.TintIndex > -1 && def.TintColor != 0 {
				// Simplified tint logic
				r := (def.TintColor >> 16) & 0xFF
				g := (def.TintColor >> 8) & 0xFF
				b := def.TintColor & 0xFF
				r5, g6, b5 := (r>>3)&0x1F, (g>>2)&0x3F, (b>>3)&0x1F
				tint = uint16((r5 << 11) | (g6 << 5) | b5)
			}
			brightness := byte(204)
			if nm == 4 {
				brightness = 255
			}
			if nm == 5 {
				brightness = 128
			}

			// Coordinates
			// Convert Element From/To (0-16) to Local Integer +0 or +1
			// x0 = From, x1 = To
			x0, y0, z0 := x+int(elem.From[0]/16.0), y+int(elem.From[1]/16.0), z+int(elem.From[2]/16.0)
			x1, y1, z1 := x+int(elem.To[0]/16.0), y+int(elem.To[1]/16.0), z+int(elem.To[2]/16.0)

			// Determine which 4 points form the quad based on face direction
			// We define 4 vertices: qa, qb, qc, qd in CCW order
			var qa, qb, qc, qd [3]int

			switch dir {
			case "up": // +Y, constant y1. Plane X-Z.
				// v0(0,1,0), v1(0,1,1), v2(1,1,1), v3(1,1,0) - following greedy convention likely
				qa = [3]int{x0, y1, z0}
				qb = [3]int{x0, y1, z1}
				qc = [3]int{x1, y1, z1}
				qd = [3]int{x1, y1, z0}
			case "down": // -Y, constant y0. Plane X-Z.
				// Winding usually inverted or same? Greedy -Y emits: x0, z0 -> x1, z0 -> x1, z1 -> x0, z1 (Clockwise/Different?)
				// Let's assume standard CCW looking from outside.
				// Looking from bottom: 0,0 -> 1,0 -> 1,1 -> 0,1
				qa = [3]int{x0, y0, z0}
				qb = [3]int{x1, y0, z0}
				qc = [3]int{x1, y0, z1}
				qd = [3]int{x0, y0, z1}
			case "north": // -Z, constant z0. Plane X-Y. (Back face)
				// Looking at -Z face: x1,0 -> x0,0 -> x0,1 -> x1,1 (Mirror X?)
				// Greedy -X faces (Wait, North is Z in greedy? No, North is +Z?
				// I said: JSON North = -Z.
				// Looking from -Z towards +Z: x1 is left, x0 is right?
				// Let's try standard CCW Quad:
				// (x1, y0, z0) -> (x0, y0, z0) -> (x0, y1, z0) -> (x1, y1, z0)
				qa = [3]int{x1, y0, z0}
				qb = [3]int{x0, y0, z0}
				qc = [3]int{x0, y1, z0}
				qd = [3]int{x1, y1, z0}
			case "south": // +Z, constant z1. Plane X-Y. (Front face)
				// (x0, y0, z1) -> (x1, y0, z1) -> (x1, y1, z1) -> (x0, y1, z1)
				qa = [3]int{x0, y0, z1}
				qb = [3]int{x1, y0, z1}
				qc = [3]int{x1, y1, z1}
				qd = [3]int{x0, y1, z1}
			case "west": // -X, constant x0. Plane Z-Y. (Left face)
				// (x0, y0, z0) -> (x0, y0, z1) -> (x0, y1, z1) -> (x0, y1, z0)
				qa = [3]int{x0, y0, z0}
				qb = [3]int{x0, y0, z1}
				qc = [3]int{x0, y1, z1}
				qd = [3]int{x0, y1, z0}
			case "east": // +X, constant x1. Plane Z-Y. (Right face)
				// (x1, y0, z1) -> (x1, y0, z0) -> (x1, y1, z0) -> (x1, y1, z1)
				qa = [3]int{x1, y0, z1}
				qb = [3]int{x1, y0, z0}
				qc = [3]int{x1, y1, z0}
				qd = [3]int{x1, y1, z1}
			default:
				continue
			}

			// Occlusion Culling Check
			// Only draw face if the neighbor in that direction is NOT a solid full block.
			// This is critical for performance.

			emit := true

			// Check if face is on the boundary
			// (Assuming 0-16 range. Local From/To are 0..16)
			// Face Up (Y=16) -> Check neighbor above
			// Face Down (Y=0) -> Check neighbor below
			// etc.

			// Determine neighbor coordinate
			var bx, by, bz int // neighbor relative to current block
			onBoundary := false

			switch dir {
			case "up":
				if int(elem.To[1]) == 16 {
					onBoundary = true
					bx = 0
					by = 1
					bz = 0
				}
			case "down":
				if int(elem.From[1]) == 0 {
					onBoundary = true
					bx = 0
					by = -1
					bz = 0
				}
			case "north": // -Z
				if int(elem.From[2]) == 0 {
					onBoundary = true
					bx = 0
					by = 0
					bz = -1
				}
			case "south": // +Z
				if int(elem.To[2]) == 16 {
					onBoundary = true
					bx = 0
					by = 0
					bz = 1
				}
			case "west": // -X
				if int(elem.From[0]) == 0 {
					onBoundary = true
					bx = -1
					by = 0
					bz = 0
				}
			case "east": // +X
				if int(elem.To[0]) == 16 {
					onBoundary = true
					bx = 1
					by = 0
					bz = 0
				}
			}

			if onBoundary {
				// Check neighbor info
				// Need world/chunk access to neighbor
				// Simple check: inside chunk?
				nx, ny, nz := x+bx, y+by, z+bz
				var neighborDef *registry.BlockDefinition

				// We'll use the logic from greedy.go:151+ for neighbors
				// But to keep it simple and safe for this file:
				// 1. Check local chunk bounds
				if nx >= 0 && nx < world.ChunkSizeX && ny >= 0 && ny < world.ChunkSizeY && nz >= 0 && nz < world.ChunkSizeZ {
					nbt := c.GetBlock(nx, ny, nz)
					if nbt != world.BlockTypeAir {
						neighborDef = registry.Blocks[nbt]
					}
				} else {
					// Global lookup for cross-chunk culling
					// Must convert local chunk coord to world coord
					wx := c.X*world.ChunkSizeX + nx
					wy := c.Y*world.ChunkSizeY + ny
					wz := c.Z*world.ChunkSizeZ + nz

					nbt := w.Get(wx, wy, wz)
					if nbt != world.BlockTypeAir {
						neighborDef = registry.Blocks[nbt]
					}
				}

				if neighborDef != nil {
					// If neighbor is a solid full block, we don't need to draw this face
					if neighborDef.IsSolid && !neighborDef.IsTransparent {
						emit = false
					}
				}
			}

			if !emit {
				continue
			}

			// Emit Quad (2 Triangles)
			// Tri 1: qa, qb, qc
			v1, v2 := packVertex(qa[0], qa[1], qa[2], nm, texID, brightness, tint)
			vertices = append(vertices, v1, v2)
			v1, v2 = packVertex(qb[0], qb[1], qb[2], nm, texID, brightness, tint)
			vertices = append(vertices, v1, v2)
			v1, v2 = packVertex(qc[0], qc[1], qc[2], nm, texID, brightness, tint)
			vertices = append(vertices, v1, v2)

			// Tri 2: qc, qd, qa
			vertices = append(vertices, v1, v2) // reuse qc
			v1, v2 = packVertex(qd[0], qd[1], qd[2], nm, texID, brightness, tint)
			vertices = append(vertices, v1, v2)
			v1, v2 = packVertex(qa[0], qa[1], qa[2], nm, texID, brightness, tint) // reuse qa
			vertices = append(vertices, v1, v2)
		}
	}
	return vertices
}
