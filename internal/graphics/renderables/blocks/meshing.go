package blocks

import (
	"mini-mc/internal/meshing"
	"mini-mc/internal/world"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Chunk meshes cache per chunk
var chunkMeshes map[world.ChunkCoord]*chunkMesh

// Per-column (XZ) combined meshes to reduce draw/cull granularity
var columnMeshes map[[2]int]*columnMesh

func ensureChunkMesh(w *world.World, coord world.ChunkCoord, ch *world.Chunk) *chunkMesh {
	if ch == nil {
		return nil
	}
	existing := chunkMeshes[coord]
	// Rebuild if missing or dirty
	if existing == nil || ch.IsDirty() {
		// Create or update GL resources
		if existing == nil {
			existing = &chunkMesh{}
			gl.GenVertexArrays(1, &existing.vao)
			gl.GenBuffers(1, &existing.vbo)
			// Setup VAO attribute layout (pos.xyz, normal.xyz)
			gl.BindVertexArray(existing.vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, existing.vbo)
			gl.EnableVertexAttribArray(0)
			gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
			gl.EnableVertexAttribArray(1)
			gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
		} else {
			gl.BindVertexArray(existing.vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, existing.vbo)
		}

		// Build greedy mesh
		verts := meshing.BuildGreedyMeshForChunk(w, ch)

		if len(verts) > 0 {
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)
			existing.vertexCount = int32(len(verts) / 6)
			// Keep CPU copy for atlas updates
			existing.cpuVerts = verts
			// Append/update in atlas
			appendOrUpdateAtlas(existing)
			// Invalidate combined column mesh
			key := [2]int{coord.X, coord.Z}
			if col := columnMeshes[key]; col != nil {
				col.dirty = true
			}
		} else {
			existing.vertexCount = 0
			existing.cpuVerts = nil
			// Still upload zero to keep state valid
			gl.BufferData(gl.ARRAY_BUFFER, 0, nil, gl.STATIC_DRAW)
		}
		chunkMeshes[coord] = existing
		ch.SetClean()
	}
	return existing
}

// PruneMeshesByWorld removes cached meshes that are not in the world anymore or beyond a radius from center.
// Returns number of meshes freed.
func PruneMeshesByWorld(w *world.World, centerX, centerZ float32, radiusChunks int) int {
	retain := make(map[world.ChunkCoord]struct{})
	all := w.GetAllChunks()
	for _, cc := range all {
		retain[cc.Coord] = struct{}{}
	}
	cx := int(centerX) / world.ChunkSizeX
	cz := int(centerZ) / world.ChunkSizeZ

	freed := 0
	for coord, m := range chunkMeshes {
		// Keep if present and within radius
		_, present := retain[coord]
		dx := coord.X - cx
		dz := coord.Z - cz
		if !present || dx*dx+dz*dz > radiusChunks*radiusChunks {
			if m != nil {
				if m.vbo != 0 {
					gl.DeleteBuffers(1, &m.vbo)
				}
				if m.vao != 0 {
					gl.DeleteVertexArrays(1, &m.vao)
				}
			}
			delete(chunkMeshes, coord)
			freed++
		}
	}
	return freed
}
