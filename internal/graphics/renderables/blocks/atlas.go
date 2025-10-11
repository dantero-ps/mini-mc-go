package blocks

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Atlas VBO/VAO management
var (
	atlasVAO           uint32
	atlasVBO           uint32
	atlasCapacityBytes int
	atlasTotalFloats   int
	firstsScratch      []int32
	countsScratch      []int32
	fallbackScratch    []*chunkMesh
	currentFrame       uint64
	// ordered list of columns sorted by firstVertex; maintained incrementally
	orderedColumns []*columnMesh
)

func ensureAtlasCapacity(requiredBytes int) {
	if requiredBytes <= atlasCapacityBytes {
		return
	}
	capBytes := atlasCapacityBytes
	if capBytes == 0 {
		capBytes = 1 << 22 // 4MB
	}
	for capBytes < requiredBytes {
		capBytes *= 2
	}
	// Allocate new buffer; we'll rebuild contents from CPU copies (portable)
	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVBO)
	gl.BufferData(gl.ARRAY_BUFFER, capBytes, nil, gl.DYNAMIC_DRAW)

	// Swap buffers: rebind VAO attributes to the new VBO
	oldVBO := atlasVBO
	atlasVBO = newVBO
	atlasCapacityBytes = capBytes

	gl.BindVertexArray(atlasVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 1, gl.FLOAT, false, 4*4, gl.PtrOffset(3*4))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	// Rebuild atlas from CPU-side chunk meshes (avoids CopyBufferSubData)
	rebuildAtlasFromCPU()

	// All column meshes' atlas offsets are now invalid; mark for rebuild
	for _, col := range columnMeshes {
		if col == nil {
			continue
		}
		col.firstFloat = -1
		col.dirty = true
	}

	if oldVBO != 0 {
		gl.DeleteBuffers(1, &oldVBO)
	}
}

// rebuildAtlasFromCPU compacts and re-uploads all available chunk meshes into the atlas VBO.
func rebuildAtlasFromCPU() {
	if atlasVBO == 0 {
		return
	}
	// Reset offset
	atlasTotalFloats = 0
	gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
	for coord, m := range chunkMeshes {
		_ = coord
		if m == nil || m.vertexCount == 0 || len(m.cpuVerts) == 0 {
			m.firstFloat = -1
			m.firstVertex = -1
			continue
		}
		bytes := len(m.cpuVerts) * 4
		offsetBytes := atlasTotalFloats * 4
		gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(m.cpuVerts))
		m.firstFloat = atlasTotalFloats
		m.firstVertex = int32(atlasTotalFloats / 4)
		atlasTotalFloats += len(m.cpuVerts)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func appendOrUpdateAtlas(m *chunkMesh) {
	if m == nil {
		return
	}
	verts := m.cpuVerts
	if len(verts) == 0 {
		m.firstFloat = -1
		m.firstVertex = -1
		return
	}
	bytes := len(verts) * 4
	if m.firstFloat < 0 && m.vertexCount == int32(len(verts)/4) {
		// Append new
		requiredBytes := (atlasTotalFloats + len(verts)) * 4
		ensureAtlasCapacity(requiredBytes)
		offsetBytes := atlasTotalFloats * 4
		gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(verts))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		m.firstFloat = atlasTotalFloats
		m.firstVertex = int32(atlasTotalFloats / 4)
		atlasTotalFloats += len(verts)
		return
	}
	// Update existing region (size may change; simple strategy: if different, re-append)
	oldCountFloats := int(m.vertexCount) * 4
	if m.firstFloat >= 0 && oldCountFloats == len(verts) {
		gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, m.firstFloat*4, bytes, gl.Ptr(verts))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		return
	}
	// Size changed: append new region and invalidate old by leaving a hole (simple, avoids compaction for now)
	requiredBytes := (atlasTotalFloats + len(verts)) * 4
	ensureAtlasCapacity(requiredBytes)
	offsetBytes := atlasTotalFloats * 4
	gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(verts))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	m.firstFloat = atlasTotalFloats
	m.firstVertex = int32(atlasTotalFloats / 6)
	atlasTotalFloats += len(verts)
}

func ensureColumnMeshForXZ(x, z int) *columnMesh {
	key := [2]int{x, z}
	col := columnMeshes[key]
	if col == nil {
		col = &columnMesh{x: x, z: z, firstFloat: -1, firstVertex: -1, dirty: true}
		columnMeshes[key] = col
	}
	if !col.dirty {
		return col
	}
	// Count total floats across Y-chunk meshes in this column
	total := 0
	for coord, cm := range chunkMeshes {
		if coord.X == x && coord.Z == z && cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			total += len(cm.cpuVerts)
		}
	}
	// If currently nothing ready to build, keep previous geometry to avoid flicker
	if total == 0 {
		// Keep column marked dirty so renderer uses per-chunk fallback until ready
		return col
	}
	buf := make([]float32, 0, total)
	for coord, cm := range chunkMeshes {
		if coord.X == x && coord.Z == z && cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			buf = append(buf, cm.cpuVerts...)
		}
	}
	// If size unchanged and region valid, update in place
	if int32(len(buf)/4) == col.vertexCount && col.firstFloat >= 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, col.firstFloat*4, len(buf)*4, gl.Ptr(buf))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		col.cpuVerts = buf
		col.dirty = false
		col.firstVertex = int32(col.firstFloat / 4)
		// keep orderedColumns position
		return col
	}
	// Otherwise append new region
	requiredBytes := (atlasTotalFloats + len(buf)) * 4
	ensureAtlasCapacity(requiredBytes)
	offsetBytes := atlasTotalFloats * 4
	gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, len(buf)*4, gl.Ptr(buf))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	col.cpuVerts = buf
	col.vertexCount = int32(len(buf) / 4)
	col.firstFloat = atlasTotalFloats
	col.firstVertex = int32(atlasTotalFloats / 4)
	atlasTotalFloats += len(buf)
	col.dirty = false
	// insert into orderedColumns keeping order by firstVertex
	if col.firstVertex >= 0 {
		inserted := false
		for i, c := range orderedColumns {
			if c == nil || c.firstVertex < 0 {
				continue
			}
			if col.firstVertex < c.firstVertex {
				// insert at i
				orderedColumns = append(orderedColumns, nil)
				copy(orderedColumns[i+1:], orderedColumns[i:])
				orderedColumns[i] = col
				inserted = true
				break
			}
		}
		if !inserted {
			orderedColumns = append(orderedColumns, col)
		}
	}
	return col
}

func setupAtlas() {
	gl.GenVertexArrays(1, &atlasVAO)
	gl.BindVertexArray(atlasVAO)

	gl.GenBuffers(1, &atlasVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, atlasVBO)
	initial := 1 << 22 // 4MB
	gl.BufferData(gl.ARRAY_BUFFER, initial, nil, gl.DYNAMIC_DRAW)
	atlasCapacityBytes = initial
	atlasTotalFloats = 0

	// Attributes: pos.xyz + encodedNormal (float)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 1, gl.FLOAT, false, 4*4, gl.PtrOffset(3*4))

	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}
