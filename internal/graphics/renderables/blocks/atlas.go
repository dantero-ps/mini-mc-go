package blocks

import (
	"log"
	"mini-mc/internal/world"
	"sort"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	initialAtlasBytes = 100 * 1024 * 1024
	maxAtlasBytes     = 300 * 1024 * 1024
	compactInterval   = 2000 // frames
)

// Atlas VBO/VAO management
var (
	atlasRegions        map[[2]int]*atlasRegion
	firstsScratch       []int32
	countsScratch       []int32
	currentFrame        uint64
	totalAllocatedBytes int
)

func regionKeyForXZ(x, z int) [2]int {
	// Tek bölge: 256MB atlas, region anahtarı sabit
	return [2]int{0, 0}
}

func glCheckError(label string) {
	if err := gl.GetError(); err != gl.NO_ERROR {
		log.Printf("gl error %s: 0x%x", label, err)
	}
}

func copyAtlasBuffer(oldVBO, newVBO uint32, bytes int) {
	if oldVBO == 0 || newVBO == 0 || bytes <= 0 {
		return
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, oldVBO)
	srcPtr := gl.MapBufferRange(gl.ARRAY_BUFFER, 0, bytes, gl.MAP_READ_BIT)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVBO)
	dstPtr := gl.MapBufferRange(gl.ARRAY_BUFFER, 0, bytes, gl.MAP_WRITE_BIT|gl.MAP_INVALIDATE_BUFFER_BIT)

	if srcPtr != nil && dstPtr != nil {
		src := unsafe.Slice((*int16)(srcPtr), bytes/2)
		dst := unsafe.Slice((*int16)(dstPtr), bytes/2)
		copy(dst, src)
		gl.UnmapBuffer(gl.ARRAY_BUFFER) // unmap newVBO
		gl.BindBuffer(gl.ARRAY_BUFFER, oldVBO)
		gl.UnmapBuffer(gl.ARRAY_BUFFER) // unmap oldVBO
	} else {
		// clean up any mapped buffers
		if dstPtr != nil {
			gl.UnmapBuffer(gl.ARRAY_BUFFER)
		}
		gl.BindBuffer(gl.ARRAY_BUFFER, oldVBO)
		if srcPtr != nil {
			gl.UnmapBuffer(gl.ARRAY_BUFFER)
		}
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func setupRegionVAO(region *atlasRegion) {
	gl.BindVertexArray(region.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, region.vbo)
	gl.EnableVertexAttribArray(0)
	// Position: 3 shorts, NOT normalized (converted to float directly), stride 4*2=8 bytes, offset 0
	gl.VertexAttribIPointer(0, 3, gl.SHORT, 4*2, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	// Normal: 1 short, NOT normalized (0..5), stride 8 bytes, offset 3*2=6
	gl.VertexAttribPointer(1, 1, gl.SHORT, false, 4*2, gl.PtrOffset(3*2))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func getOrCreateRegion(key [2]int) *atlasRegion {
	if atlasRegions == nil {
		atlasRegions = make(map[[2]int]*atlasRegion)
	}
	if r := atlasRegions[key]; r != nil {
		return r
	}
	if totalAllocatedBytes+initialAtlasBytes > maxAtlasBytes {
		log.Printf("atlas region %v not allocated: budget exceeded (%d/%d)", key, totalAllocatedBytes, maxAtlasBytes)
		return nil
	}
	r := &atlasRegion{key: key, capacityBytes: initialAtlasBytes}
	gl.GenVertexArrays(1, &r.vao)
	gl.GenBuffers(1, &r.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, r.capacityBytes, nil, gl.DYNAMIC_DRAW)
	setupRegionVAO(r)
	atlasRegions[key] = r
	totalAllocatedBytes += r.capacityBytes
	return r
}

func ensureRegionCapacity(r *atlasRegion, requiredBytes int) bool {
	if requiredBytes <= r.capacityBytes {
		return true
	}

	// If requiredBytes exceeds maxAtlasBytes, we can't grow.
	if requiredBytes > maxAtlasBytes {
		log.Printf("atlas region %v out of capacity (need %d, max %d)", r.key, requiredBytes, maxAtlasBytes)
		return false
	}

	// Grow strategy: double until max
	newCap := r.capacityBytes * 2
	if newCap < requiredBytes {
		newCap = requiredBytes
	}
	if newCap > maxAtlasBytes {
		newCap = maxAtlasBytes
	}

	// Create new buffer
	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVBO)
	gl.BufferData(gl.ARRAY_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)

	// Copy old data
	copyAtlasBuffer(r.vbo, newVBO, r.capacityBytes)

	// Delete old buffer
	gl.DeleteBuffers(1, &r.vbo)
	r.vbo = newVBO

	// Update capacity
	totalAllocatedBytes -= r.capacityBytes
	r.capacityBytes = newCap
	totalAllocatedBytes += r.capacityBytes

	// Re-bind VAO to new VBO
	setupRegionVAO(r)

	log.Printf("atlas region %v grew to %d bytes", r.key, r.capacityBytes)
	return true
}

func queueRegionWrite(r *atlasRegion, offsetBytes int, data []int16) {
	if r == nil || len(data) == 0 {
		return
	}
	r.pendingWrites = append(r.pendingWrites, atlasWrite{offsetBytes: offsetBytes, data: data})
}

func flushRegionWrites(r *atlasRegion) {
	if r == nil || len(r.pendingWrites) == 0 {
		return
	}
	minOffset := r.pendingWrites[0].offsetBytes
	maxEnd := r.pendingWrites[0].offsetBytes + len(r.pendingWrites[0].data)*2
	for _, w := range r.pendingWrites[1:] {
		if w.offsetBytes < minOffset {
			minOffset = w.offsetBytes
		}
		end := w.offsetBytes + len(w.data)*2
		if end > maxEnd {
			maxEnd = end
		}
	}
	size := maxEnd - minOffset
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	flags := uint32(gl.MAP_WRITE_BIT | gl.MAP_UNSYNCHRONIZED_BIT | gl.MAP_INVALIDATE_RANGE_BIT)
	ptr := gl.MapBufferRange(gl.ARRAY_BUFFER, minOffset, size, flags)
	if ptr != nil {
		base := unsafe.Slice((*byte)(ptr), size)
		for _, w := range r.pendingWrites {
			start := w.offsetBytes - minOffset
			dst := unsafe.Slice((*int16)(unsafe.Pointer(&base[start])), len(w.data))
			copy(dst, w.data)
		}
		gl.UnmapBuffer(gl.ARRAY_BUFFER)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	r.pendingWrites = r.pendingWrites[:0]
}

func flushAllRegionWrites() {
	for _, r := range atlasRegions {
		flushRegionWrites(r)
	}
}

// Helper to gather vertices for a column (from X,Z) by checking relevant Y chunks
func collectColumnVerts(x, z int) []int16 {
	var buf []int16
	// We assume a reasonable max height.
	// world.ChunkSizeY is 256, so usually only Y=0 exists.
	// But let's check a few slots just in case.
	// We check Y from 0 to 15 (covering 4096 blocks height which is enough).
	for y := 0; y < 16; y++ {
		coord := world.ChunkCoord{X: x, Y: y, Z: z}
		if cm := chunkMeshes[coord]; cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			buf = append(buf, cm.cpuVerts...)
		}
	}
	return buf
}

// Helper to calculate total floats for a column without allocating
func calculateColumnFloats(x, z int) int {
	total := 0
	for y := 0; y < 16; y++ {
		coord := world.ChunkCoord{X: x, Y: y, Z: z}
		if cm := chunkMeshes[coord]; cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			total += len(cm.cpuVerts)
		}
	}
	return total
}

func compactRegion(r *atlasRegion) {
	if r == nil {
		return
	}

	// Reconstruct total floats from all columns that belong to this region
	totalFloats := 0

	// We only track columns now
	for _, c := range columnMeshes {
		if c == nil || c.regionKey != r.key {
			continue
		}
		// Calculate exact size from current chunks because c.vertexCount might be stale
		size := calculateColumnFloats(c.x, c.z)
		totalFloats += size
	}

	requiredBytes := totalFloats * 2 // 2 bytes per short
	if requiredBytes > r.capacityBytes {
		if !ensureRegionCapacity(r, requiredBytes) {
			return
		}
	}
	if totalFloats == 0 {
		r.totalFloats = 0
		r.orderedColumns = r.orderedColumns[:0]
		return
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, r.capacityBytes, nil, gl.DYNAMIC_DRAW)
	flags := uint32(gl.MAP_WRITE_BIT | gl.MAP_INVALIDATE_BUFFER_BIT)
	ptr := gl.MapBufferRange(gl.ARRAY_BUFFER, 0, requiredBytes, flags)
	if ptr == nil {
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		return
	}
	dst := unsafe.Slice((*int16)(ptr), requiredBytes/2)
	offset := 0

	// NOTE: We do NOT iterate chunkMeshes here anymore because we don't support standalone chunks in atlas.

	cols := make([]*columnMesh, 0, len(columnMeshes))
	for _, c := range columnMeshes {
		if c == nil || c.regionKey != r.key {
			continue
		}

		// Re-gather data from chunks
		verts := collectColumnVerts(c.x, c.z)
		if len(verts) == 0 {
			// Column became empty?
			c.vertexCount = 0
			c.firstFloat = -1
			c.firstVertex = -1
			// Don't append to cols, effectively removing from atlas render list
			continue
		}

		copy(dst[offset:], verts)
		c.firstFloat = offset
		c.firstVertex = int32(offset / 4)     // Stride is 4 shorts
		c.vertexCount = int32(len(verts) / 4) // Update just in case
		offset += len(verts)
		cols = append(cols, c)
	}
	gl.UnmapBuffer(gl.ARRAY_BUFFER)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	r.totalFloats = offset
	sort.Slice(cols, func(i, j int) bool { return cols[i].firstVertex < cols[j].firstVertex })
	r.orderedColumns = cols
	r.lastCompact = currentFrame
}

func maybeCompactRegions() {
	for _, r := range atlasRegions {
		if len(r.pendingWrites) > 0 {
			continue // önce pending flush
		}
		usedBytes := r.totalFloats * 2
		freeBytes := r.capacityBytes - usedBytes
		if freeBytes > (r.capacityBytes*3)/4 {
			continue // çok boşluk var, kompaktlama gereksiz
		}
		if (currentFrame - r.lastCompact) > compactInterval {
			compactRegion(r)
		}
	}
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
	rkey := regionKeyForXZ(x, z)
	r := getOrCreateRegion(rkey)
	if r == nil {
		return col
	}
	col.regionKey = rkey

	// Efficiently gather vertices from chunks in this column
	buf := collectColumnVerts(x, z)

	if len(buf) == 0 {
		col.vertexCount = 0
		col.firstFloat = -1
		col.firstVertex = -1
		col.dirty = false
		return col
	}

	if int32(len(buf)/4) == col.vertexCount && col.firstFloat >= 0 {
		// Size same, try to update in place
		queueRegionWrite(r, col.firstFloat*2, buf)
		col.dirty = false
		col.firstVertex = int32(col.firstFloat / 4)
		return col
	}

	requiredBytes := (r.totalFloats + len(buf)) * 2
	if requiredBytes > r.capacityBytes {
		if !ensureRegionCapacity(r, requiredBytes) {
			return col
		}
	}
	offsetBytes := r.totalFloats * 2
	queueRegionWrite(r, offsetBytes, buf)

	// Do NOT store cpuVerts in col
	// col.cpuVerts = buf

	col.vertexCount = int32(len(buf) / 4)
	col.firstFloat = r.totalFloats
	col.firstVertex = int32(r.totalFloats / 4)
	r.totalFloats += len(buf)
	col.dirty = false

	if col.firstVertex >= 0 {
		inserted := false
		if r.orderedColumns == nil {
			r.orderedColumns = make([]*columnMesh, 0, 8)
		}
		for i, c := range r.orderedColumns {
			if c == nil || c.firstVertex < 0 {
				continue
			}
			if col.firstVertex < c.firstVertex {
				r.orderedColumns = append(r.orderedColumns, nil)
				copy(r.orderedColumns[i+1:], r.orderedColumns[i:])
				r.orderedColumns[i] = col
				inserted = true
				break
			}
		}
		if !inserted {
			r.orderedColumns = append(r.orderedColumns, col)
		}
	}
	return col
}

func setupAtlas() {
	if atlasRegions == nil {
		atlasRegions = make(map[[2]int]*atlasRegion)
	}
}
