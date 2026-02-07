package blocks

import (
	"log"
	"mini-mc/internal/world"
	"sort"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	initialAtlasBytes = 256 * 1024 * 1024
	maxAtlasBytes     = 512 * 1024 * 1024
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

func CleanupAtlas() {
	if atlasRegions != nil {
		for _, r := range atlasRegions {
			if r.vbo != 0 {
				gl.DeleteBuffers(1, &r.vbo)
			}
			if r.vao != 0 {
				gl.DeleteVertexArrays(1, &r.vao)
			}
		}
		atlasRegions = nil
	}
	totalAllocatedBytes = 0
	currentFrame = 0
}

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
	gl.BindBuffer(gl.COPY_READ_BUFFER, oldVBO)
	srcPtr := gl.MapBufferRange(gl.COPY_READ_BUFFER, 0, bytes, gl.MAP_READ_BIT)

	gl.BindBuffer(gl.COPY_WRITE_BUFFER, newVBO)
	dstPtr := gl.MapBufferRange(gl.COPY_WRITE_BUFFER, 0, bytes, gl.MAP_WRITE_BIT|gl.MAP_INVALIDATE_BUFFER_BIT)

	if srcPtr != nil && dstPtr != nil {
		src := unsafe.Slice((*byte)(srcPtr), bytes)
		dst := unsafe.Slice((*byte)(dstPtr), bytes)
		copy(dst, src)

		gl.UnmapBuffer(gl.COPY_WRITE_BUFFER)
		gl.UnmapBuffer(gl.COPY_READ_BUFFER)
	} else {
		if dstPtr != nil {
			gl.UnmapBuffer(gl.COPY_WRITE_BUFFER)
		}
		if srcPtr != nil {
			gl.UnmapBuffer(gl.COPY_READ_BUFFER)
		}
	}
	gl.BindBuffer(gl.COPY_READ_BUFFER, 0)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)
}

func setupRegionVAO(region *atlasRegion) {
	gl.BindVertexArray(region.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, region.vbo)

	// Stride: 6 shorts = 12 bytes
	stride := int32(6 * 2)

	gl.EnableVertexAttribArray(0)
	// Position: 3 shorts (X,Y,Z), offset 0
	gl.VertexAttribPointer(0, 3, gl.SHORT, false, stride, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(1)
	// Data: 3 shorts (Info, TexID, Tint), offset 6
	// Info: Normal + Brightness
	// TexID: Texture Layer
	// Tint: Reserved
	gl.VertexAttribPointer(1, 3, gl.UNSIGNED_SHORT, false, stride, gl.PtrOffset(3*2))

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
	newCap := min(max(r.capacityBytes*2, requiredBytes), maxAtlasBytes)

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
// It unpacks the compressed chunk-local uint32 vertices into world-space int16 atlas format.
func collectColumnVerts(x, z int) []int16 {
	var buf []int16
	// We assume a reasonable max height.
	// world.ChunkSizeY is 256, so usually only Y=0 exists.
	// But let's check a few slots just in case.
	// We check Y from 0 to 15 (covering 4096 blocks height which is enough).
	for y := range 16 {
		coord := world.ChunkCoord{X: x, Y: y, Z: z}
		if cm := chunkMeshes[coord]; cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			// Unpack vertices
			baseX := x * world.ChunkSizeX
			baseY := y * world.ChunkSizeY
			baseZ := z * world.ChunkSizeZ

			// VertexStride is 2 uint32s
			// Output vertex size is 6 int16s
			count := len(cm.cpuVerts) / 2

			for i := range count {
				v1 := cm.cpuVerts[i*2]
				v2 := cm.cpuVerts[i*2+1]

				// V1 Layout:
				// X: 5 bits (0-4)
				// Y: 9 bits (5-13)
				// Z: 5 bits (14-18)
				// N: 3 bits (19-21)
				// B: 8 bits (22-29) - Brightness

				// V2 Layout:
				// T: 16 bits (0-15) - TextureID

				lx := int(v1 & 0x1F)
				ly := int((v1 >> 5) & 0x1FF)
				lz := int((v1 >> 14) & 0x1F)
				norm := int((v1 >> 19) & 0x7)
				brightness := int((v1 >> 22) & 0xFF)

				texID := int(v2 & 0xFFFF)
				tint := int((v2 >> 16) & 0xFFFF)

				wx := int16(baseX + lx)
				wy := int16(baseY + ly)
				wz := int16(baseZ + lz)

				// Packed Info:
				// Lower byte: Normal
				// Upper byte: Brightness
				info := int16(norm | (brightness << 8))

				// TextureID as int16
				texInfo := int16(texID)

				// Extra/Tint
				extra := int16(tint)

				buf = append(buf, wx, wy, wz, info, texInfo, extra)
			}
		}
	}
	return buf
}

// Helper to calculate total floats for a column without allocating
func calculateColumnFloats(x, z int) int {
	total := 0
	for y := range 16 {
		coord := world.ChunkCoord{X: x, Y: y, Z: z}
		if cm := chunkMeshes[coord]; cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			// cpuVerts has 2 ints per vertex.
			// Each vertex becomes 6 shorts.
			// So total shorts = (len(cpuVerts) / 2) * 6 = len(cpuVerts) * 3
			total += len(cm.cpuVerts) * 3
		}
	}
	return total
}

func compactRegion(r *atlasRegion) {
	if r == nil {
		return
	}

	// Calculate total size needed for all active columns
	totalFloats := 0
	activeCols := make([]*columnMesh, 0, len(columnMeshes))

	// Find all columns belonging to this region
	for _, c := range columnMeshes {
		if c == nil || c.regionKey != r.key {
			continue
		}
		if c.vertexCount <= 0 || c.firstVertex < 0 {
			continue
		}
		// Stride is 6 shorts (12 bytes). VertexCount is number of vertices.
		// Size in floats? The codebase tracks 'firstFloat' which is offset in shorts/ints?
		// Wait, 'totalFloats' in struct is actually 'totalShorts' or 'totalAllocatedItems'.
		// In queueRegionWrite: offsetBytes.
		// c.firstFloat is actually index in the int16 buffer.
		// c.vertexCount is number of vertices. 1 vertex = 6 int16s.
		// So size in int16s = c.vertexCount * 6.

		size := int(c.vertexCount) * 6
		totalFloats += size
		activeCols = append(activeCols, c)
	}

	// Sort active columns by their current position in VBO to hopefully make reads sequential
	sort.Slice(activeCols, func(i, j int) bool {
		return activeCols[i].firstVertex < activeCols[j].firstVertex
	})

	requiredBytes := totalFloats * 2 // 2 bytes per short

	// Create new VBO
	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, newVBO)

	// Ensure capacity
	newCap := r.capacityBytes
	if requiredBytes > newCap {
		newCap = requiredBytes * 3 / 2 // Grow if needed, though compact usually shrinks
		if newCap > maxAtlasBytes {
			newCap = maxAtlasBytes
		}
	}
	// If drastically smaller, maybe shrink capability? For now keep capacity stable to avoid thrashing.

	gl.BufferData(gl.COPY_WRITE_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)

	// Map old VBO as read source (CopyBufferSubData is unsupported on this driver)
	gl.BindBuffer(gl.COPY_READ_BUFFER, r.vbo)
	srcPtr := gl.MapBufferRange(gl.COPY_READ_BUFFER, 0, r.capacityBytes, gl.MAP_READ_BIT)

	// Map new VBO as write target
	dstPtr := gl.MapBufferRange(gl.COPY_WRITE_BUFFER, 0, newCap, gl.MAP_WRITE_BIT|gl.MAP_INVALIDATE_BUFFER_BIT)

	if srcPtr != nil && dstPtr != nil {
		srcData := unsafe.Slice((*byte)(srcPtr), r.capacityBytes)
		dstData := unsafe.Slice((*byte)(dstPtr), newCap)

		currentOffsetShorts := 0

		for _, c := range activeCols {
			sizeShorts := int(c.vertexCount) * 6
			sizeBytes := sizeShorts * 2

			// Previous offset in bytes
			srcOffsetBytes := int(c.firstVertex) * 6 * 2
			dstOffsetBytes := currentOffsetShorts * 2

			// Copy this column's data
			if srcOffsetBytes+sizeBytes <= len(srcData) {
				copy(dstData[dstOffsetBytes:], srcData[srcOffsetBytes:srcOffsetBytes+sizeBytes])
			}

			// Update column metadata
			c.firstFloat = currentOffsetShorts
			c.firstVertex = int32(currentOffsetShorts / 6)
			// vertexCount remains same

			currentOffsetShorts += sizeShorts
		}

		gl.UnmapBuffer(gl.COPY_WRITE_BUFFER)
		gl.UnmapBuffer(gl.COPY_READ_BUFFER)

		// Update region metadata
		r.totalFloats = currentOffsetShorts
	}

	gl.BindBuffer(gl.COPY_READ_BUFFER, 0)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)

	// Delete old VBO
	gl.DeleteBuffers(1, &r.vbo)
	r.vbo = newVBO
	r.capacityBytes = newCap

	// Re-bind VAO to new VBO
	setupRegionVAO(r)

	// Update ordered columns for rendering
	r.orderedColumns = activeCols
	r.lastCompact = currentFrame
	r.fragmentedBytes = 0

	log.Printf("atlas region %v compacted (CPU-MemCpy): %d bytes used, %d columns moved", r.key, r.totalFloats*2, len(activeCols))
}

func maybeCompactRegions() {
	for _, r := range atlasRegions {
		if len(r.pendingWrites) > 0 {
			continue // önce pending flush
		}

		// Only compact if fragmentation is significant (>25% of capacity OR >10MB wasted)
		isHighFragmentation := r.fragmentedBytes > (r.capacityBytes/4) || r.fragmentedBytes > 10*1024*1024
		if !isHighFragmentation {
			continue
		}

		// Ensure interval passed
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

	// 6 shorts per vertex
	vertexCount := int32(len(buf) / 6)

	if vertexCount == col.vertexCount && col.firstFloat >= 0 {
		// Size same, try to update in place
		queueRegionWrite(r, col.firstFloat*2, buf)
		col.dirty = false
		col.firstVertex = int32(col.firstFloat / 6)
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

	// Mark old location as fragmented if valid (before overwriting firstFloat)
	if col.firstFloat >= 0 && col.vertexCount > 0 {
		r.fragmentedBytes += int(col.vertexCount) * 12
	}

	col.vertexCount = vertexCount
	col.firstFloat = r.totalFloats
	col.firstVertex = int32(r.totalFloats / 6)
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
