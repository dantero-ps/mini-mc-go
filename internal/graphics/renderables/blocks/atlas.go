package blocks

import (
	"log"
	"mini-mc/internal/world"
	"sort"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	initialRegionBytes = 512 * 1024         // 512 KB per region initial allocation
	maxRegionBytes     = 128 * 1024 * 1024  // 64 MB per region max
	globalMaxBytes     = 1024 * 1024 * 1024 // total GPU budget across all regions
)

// Atlas VBO/VAO management
var (
	atlasRegions        map[[2]int]*atlasRegion
	firstsScratch       []int32
	countsScratch       []int32
	currentFrame        uint64
	totalAllocatedBytes int
)

// ---------- Helper functions ----------
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
	return [2]int{x >> 4, z >> 4}
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

	stride := int32(6 * 2)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.SHORT, false, stride, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(1)
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
	for totalAllocatedBytes+initialRegionBytes > globalMaxBytes {
		before := totalAllocatedBytes
		evictColdRegionsGlobal(initialRegionBytes + initialRegionBytes/2)
		if totalAllocatedBytes >= before {
			log.Printf("CRITICAL: no evictable memory, atlas full")
			return nil
		}
	}

	r := &atlasRegion{key: key, capacityBytes: initialRegionBytes}
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
	if requiredBytes > maxRegionBytes {
		log.Printf("atlas region %v out of capacity (need %d, max %d)", r.key, requiredBytes, maxRegionBytes)
		return false
	}

	newCap := min(max(r.capacityBytes*2, requiredBytes), maxRegionBytes)
	if totalAllocatedBytes-r.capacityBytes+newCap > globalMaxBytes {
		needed := (totalAllocatedBytes - r.capacityBytes + newCap) - globalMaxBytes
		evictColdRegionsGlobal(needed)
		if totalAllocatedBytes-r.capacityBytes+newCap > globalMaxBytes {
			log.Printf("atlas region %v growth blocked: global budget would exceed %d", r.key, globalMaxBytes)
			return false
		}
	}

	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVBO)
	gl.BufferData(gl.ARRAY_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)

	copyAtlasBuffer(r.vbo, newVBO, r.capacityBytes)

	gl.DeleteBuffers(1, &r.vbo)
	r.vbo = newVBO

	totalAllocatedBytes -= r.capacityBytes
	r.capacityBytes = newCap
	totalAllocatedBytes += r.capacityBytes

	setupRegionVAO(r)

	r.growthCount++
	log.Printf("atlas region %v grew to %d bytes", r.key, r.capacityBytes)
	return true
}

// ---------- Free-list allocation ----------
func regionFragmentedBytes(r *atlasRegion) int {
	total := 0
	for _, s := range r.freeList {
		total += s.sizeShorts * 2
	}
	return total
}

func allocInRegion(r *atlasRegion, sizeShorts int) (int, bool) {
	// Best-fit search
	bestIdx := -1
	for i, span := range r.freeList {
		if span.sizeShorts >= sizeShorts {
			if bestIdx < 0 || span.sizeShorts < r.freeList[bestIdx].sizeShorts {
				bestIdx = i
			}
		}
	}
	if bestIdx >= 0 {
		span := r.freeList[bestIdx]
		remaining := span.sizeShorts - sizeShorts
		if remaining > 0 {
			r.freeList[bestIdx] = freeSpan{span.offsetShorts + sizeShorts, remaining}
		} else {
			r.freeList = append(r.freeList[:bestIdx], r.freeList[bestIdx+1:]...)
		}
		return span.offsetShorts, true
	}
	// Append
	requiredBytes := (r.totalFloats + sizeShorts) * 2
	if !ensureRegionCapacity(r, requiredBytes) {
		return 0, false
	}
	offset := r.totalFloats
	r.totalFloats += sizeShorts
	return offset, true
}

func freeInRegion(r *atlasRegion, offsetShorts, sizeShorts int) {
	if sizeShorts <= 0 {
		return
	}
	span := freeSpan{offsetShorts, sizeShorts}
	i := sort.Search(len(r.freeList), func(j int) bool {
		return r.freeList[j].offsetShorts >= offsetShorts
	})
	r.freeList = append(r.freeList, freeSpan{})
	copy(r.freeList[i+1:], r.freeList[i:])
	r.freeList[i] = span

	coalesceFreelist(r)
}

func coalesceFreelist(r *atlasRegion) {
	i := 0
	for i < len(r.freeList)-1 {
		a := &r.freeList[i]
		b := r.freeList[i+1]
		if a.offsetShorts+a.sizeShorts == b.offsetShorts {
			a.sizeShorts += b.sizeShorts
			r.freeList = append(r.freeList[:i+1], r.freeList[i+2:]...)
		} else {
			i++
		}
	}
	// Trim trailing free space
	for len(r.freeList) > 0 {
		last := r.freeList[len(r.freeList)-1]
		if last.offsetShorts+last.sizeShorts == r.totalFloats {
			r.totalFloats = last.offsetShorts
			r.freeList = r.freeList[:len(r.freeList)-1]
		} else {
			break
		}
	}
}

// ---------- Write queuing and flushing ----------
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

	// Sort writes by offset so we can merge contiguous ones into blocks.
	// Using insertion sort — write counts are typically small.
	writes := r.pendingWrites
	for i := 1; i < len(writes); i++ {
		for j := i; j > 0 && writes[j].offsetBytes < writes[j-1].offsetBytes; j-- {
			writes[j], writes[j-1] = writes[j-1], writes[j]
		}
	}

	// Flush each contiguous block separately so MAP_INVALIDATE_RANGE_BIT never
	// touches gaps that contain valid, unrelated vertex data.
	const mergeGapBytes = 256 // merge writes within this gap to amortize map overhead
	gl.BindBuffer(gl.ARRAY_BUFFER, r.vbo)
	flags := uint32(gl.MAP_WRITE_BIT | gl.MAP_UNSYNCHRONIZED_BIT | gl.MAP_INVALIDATE_RANGE_BIT)

	blockStart := 0
	for i := 1; i <= len(writes); i++ {
		isLast := i == len(writes)
		merge := !isLast
		if merge {
			prevEnd := writes[i-1].offsetBytes + len(writes[i-1].data)*2
			merge = writes[i].offsetBytes-prevEnd <= mergeGapBytes
		}
		if !merge {
			// Flush block [blockStart, i)
			minOff := writes[blockStart].offsetBytes
			maxEnd := writes[i-1].offsetBytes + len(writes[i-1].data)*2
			size := maxEnd - minOff
			ptr := gl.MapBufferRange(gl.ARRAY_BUFFER, minOff, size, flags)
			if ptr != nil {
				base := unsafe.Slice((*byte)(ptr), size)
				for _, w := range writes[blockStart:i] {
					start := w.offsetBytes - minOff
					dst := unsafe.Slice((*int16)(unsafe.Pointer(&base[start])), len(w.data))
					copy(dst, w.data)
				}
				gl.UnmapBuffer(gl.ARRAY_BUFFER)
			}
			blockStart = i
		}
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	r.pendingWrites = r.pendingWrites[:0]
}

func flushAllRegionWrites() {
	for _, r := range atlasRegions {
		flushRegionWrites(r)
	}
}

// ---------- Vertex data collection ----------
func collectColumnVerts(x, z int) []int16 {
	var buf []int16
	for y := range world.NumSections {
		coord := world.ChunkCoord{X: x, Y: y, Z: z}
		if cm := chunkMeshes[coord]; cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			baseX := x * world.ChunkSizeX
			baseY := y * world.ChunkSizeY
			baseZ := z * world.ChunkSizeZ

			count := len(cm.cpuVerts) / 2
			for i := range count {
				v1 := cm.cpuVerts[i*2]
				v2 := cm.cpuVerts[i*2+1]

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

				info := int16(norm | (brightness << 8))
				texInfo := int16(texID)
				extra := int16(tint)

				buf = append(buf, wx, wy, wz, info, texInfo, extra)
			}
		}
	}
	return buf
}

// ---------- Compaction (with flush and empty handling) ----------
func compactRegion(r *atlasRegion) {
	if r == nil {
		return
	}
	// Flush any pending writes first
	flushRegionWrites(r)

	// Collect active columns
	activeCols := make([]*columnMesh, 0, len(columnMeshes))
	for _, c := range columnMeshes {
		if c == nil || c.regionKey != r.key {
			continue
		}
		if c.vertexCount <= 0 || c.firstFloat < 0 {
			continue
		}
		activeCols = append(activeCols, c)
	}

	// If no active columns, delete the entire region
	if len(activeCols) == 0 {
		if r.vbo != 0 {
			gl.DeleteBuffers(1, &r.vbo)
		}
		if r.vao != 0 {
			gl.DeleteVertexArrays(1, &r.vao)
		}
		totalAllocatedBytes -= r.capacityBytes
		delete(atlasRegions, r.key)
		log.Printf("atlas region %v deleted (empty)", r.key)
		return
	}

	sort.Slice(activeCols, func(i, j int) bool {
		return activeCols[i].firstFloat < activeCols[j].firstFloat
	})

	totalShorts := 0
	for _, c := range activeCols {
		totalShorts += int(c.vertexCount) * 6
	}
	requiredBytes := totalShorts * 2
	newCap := max(requiredBytes+requiredBytes/5, initialRegionBytes) // 1.2x headroom
	newCap = min(newCap, maxRegionBytes)

	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, newVBO)
	gl.BufferData(gl.COPY_WRITE_BUFFER, newCap, nil, gl.DYNAMIC_DRAW)

	gl.BindBuffer(gl.COPY_READ_BUFFER, r.vbo)
	srcPtr := gl.MapBufferRange(gl.COPY_READ_BUFFER, 0, r.capacityBytes, gl.MAP_READ_BIT)
	dstPtr := gl.MapBufferRange(gl.COPY_WRITE_BUFFER, 0, newCap, gl.MAP_WRITE_BIT|gl.MAP_INVALIDATE_BUFFER_BIT)

	if srcPtr == nil || dstPtr == nil {
		// Map failed — abort without mutating column offsets or deleting the old VBO.
		// Columns retain their current firstFloat so they can still be drawn correctly.
		if dstPtr != nil {
			gl.UnmapBuffer(gl.COPY_WRITE_BUFFER)
		}
		if srcPtr != nil {
			gl.UnmapBuffer(gl.COPY_READ_BUFFER)
		}
		gl.BindBuffer(gl.COPY_READ_BUFFER, 0)
		gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)
		gl.DeleteBuffers(1, &newVBO)
		return
	}

	srcData := unsafe.Slice((*byte)(srcPtr), r.capacityBytes)
	dstData := unsafe.Slice((*byte)(dstPtr), newCap)

	currentOffsetShorts := 0
	for _, c := range activeCols {
		sizeShorts := int(c.vertexCount) * 6
		sizeBytes := sizeShorts * 2
		srcOffsetBytes := c.firstFloat * 2
		dstOffsetBytes := currentOffsetShorts * 2

		if srcOffsetBytes+sizeBytes > len(srcData) {
			c.vertexCount = 0
			c.firstFloat = -1
			c.firstVertex = -1
			c.dirty = true
			continue
		}
		copy(dstData[dstOffsetBytes:], srcData[srcOffsetBytes:srcOffsetBytes+sizeBytes])
		c.firstFloat = currentOffsetShorts
		c.firstVertex = int32(currentOffsetShorts / 6)
		currentOffsetShorts += sizeShorts
	}
	gl.UnmapBuffer(gl.COPY_WRITE_BUFFER)
	gl.UnmapBuffer(gl.COPY_READ_BUFFER)

	r.totalFloats = currentOffsetShorts
	r.freeList = r.freeList[:0]

	gl.BindBuffer(gl.COPY_READ_BUFFER, 0)
	gl.BindBuffer(gl.COPY_WRITE_BUFFER, 0)

	gl.DeleteBuffers(1, &r.vbo)
	r.vbo = newVBO

	if newCap != r.capacityBytes {
		totalAllocatedBytes -= r.capacityBytes
		totalAllocatedBytes += newCap
	}
	r.capacityBytes = newCap

	setupRegionVAO(r)
	r.orderedColumns = activeCols
	r.activeColumns = len(activeCols)
	r.lastCompact = currentFrame

	log.Printf("atlas region %v compacted: %d bytes used, %d columns", r.key, r.totalFloats*2, len(activeCols))
}

// ---------- Eviction (with flush) ----------
func evictLRUColumns(r *atlasRegion, targetFreeBytes int) int {
	if r == nil || targetFreeBytes <= 0 {
		return 0
	}
	flushRegionWrites(r) // ensure all pending writes are applied before eviction

	type candidate struct {
		key [2]int
		col *columnMesh
	}
	candidates := make([]candidate, 0, len(columnMeshes))
	for k, c := range columnMeshes {
		if c == nil || c.regionKey != r.key {
			continue
		}
		if c.firstFloat < 0 || c.vertexCount <= 0 {
			continue
		}
		candidates = append(candidates, candidate{k, c})
	}
	if len(candidates) == 0 {
		return 0
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].col.drawnFrame < candidates[j].col.drawnFrame
	})

	freedBytes := 0
	for _, cand := range candidates {
		if freedBytes >= targetFreeBytes {
			break
		}
		col := cand.col
		colBytes := int(col.vertexCount) * 12
		freeInRegion(r, col.firstFloat, int(col.vertexCount)*6)
		r.activeColumns--
		col.vertexCount = 0
		col.firstFloat = -1
		col.firstVertex = -1
		col.dirty = true
		freedBytes += colBytes
	}
	if freedBytes > 0 {
		compactRegion(r)
		log.Printf("atlas region %v: evicted LRU columns, freed ~%d bytes", r.key, freedBytes)
	}
	return freedBytes
}

func evictColdRegionsGlobal(neededBytes int) int {
	flushAllRegionWrites() // flush everything before global eviction

	type candidate struct {
		r   *atlasRegion
		col *columnMesh
	}
	candidates := make([]candidate, 0, len(columnMeshes))
	for _, c := range columnMeshes {
		if c == nil || c.firstFloat < 0 || c.vertexCount <= 0 {
			continue
		}
		r := atlasRegions[c.regionKey]
		if r == nil {
			continue
		}
		candidates = append(candidates, candidate{r, c})
	}
	if len(candidates) == 0 {
		return 0
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].col.drawnFrame < candidates[j].col.drawnFrame
	})

	dirtyRegions := map[[2]int]*atlasRegion{}
	logicalFreed := 0
	for _, cand := range candidates {
		// Stop evicting once logical freed bytes exceed target; compaction will
		// then shrink capacityBytes and reduce totalAllocatedBytes for real.
		if logicalFreed >= neededBytes {
			break
		}
		col := cand.col
		logicalFreed += int(col.vertexCount) * 12
		freeInRegion(cand.r, col.firstFloat, int(col.vertexCount)*6)
		cand.r.activeColumns--
		col.vertexCount = 0
		col.firstFloat = -1
		col.firstVertex = -1
		col.dirty = true
		dirtyRegions[cand.r.key] = cand.r
	}
	before := totalAllocatedBytes
	for _, r := range dirtyRegions {
		compactRegion(r)
	}
	gpuFreed := before - totalAllocatedBytes
	if gpuFreed > 0 {
		log.Printf("atlas global eviction: freed ~%d GPU bytes across %d regions", gpuFreed, len(dirtyRegions))
	}
	return gpuFreed
}

// ---------- Periodic maintenance ----------
func maybeCompactRegions() {
	for key, r := range atlasRegions {
		// Skip regions with pending writes – they will be compacted after flush
		if len(r.pendingWrites) > 0 {
			continue
		}
		// Delete empty regions
		if r.totalFloats == 0 && len(r.freeList) == 0 {
			if r.vbo != 0 {
				gl.DeleteBuffers(1, &r.vbo)
			}
			if r.vao != 0 {
				gl.DeleteVertexArrays(1, &r.vao)
			}
			totalAllocatedBytes -= r.capacityBytes
			delete(atlasRegions, key)
			continue
		}
		// No active columns: force compaction so the region gets deleted immediately.
		if r.activeColumns <= 0 {
			compactRegion(r)
			continue
		}

		// Compact if high hole-fragmentation OR if the VBO has too much trailing waste.
		fragBytes := regionFragmentedBytes(r)
		usedBytes := r.totalFloats * 2
		if usedBytes == 0 {
			continue
		}
		isHighFragmentation := fragBytes > usedBytes/4 || fragBytes > 4*1024*1024
		hasTrailingWaste := r.capacityBytes > usedBytes*2+initialRegionBytes
		if (isHighFragmentation || hasTrailingWaste) && (currentFrame-r.lastCompact) > 60 {
			compactRegion(r)
		}
	}
}

// ---------- Column mesh update (main entry point) ----------
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
	// If a previous alloc failed, don't retry until retryFrame to avoid thrashing.
	if col.retryFrame > 0 && currentFrame < col.retryFrame {
		return col
	}
	col.retryFrame = 0

	rkey := regionKeyForXZ(x, z)
	r := getOrCreateRegion(rkey)
	if r == nil {
		return col
	}
	col.regionKey = rkey

	// Flush any pending writes for this region before modifying layout
	flushRegionWrites(r)

	buf := collectColumnVerts(x, z)

	if len(buf) == 0 {
		if col.firstFloat >= 0 && col.vertexCount > 0 {
			freeInRegion(r, col.firstFloat, int(col.vertexCount)*6)
			r.activeColumns--
		}
		col.vertexCount = 0
		col.firstFloat = -1
		col.firstVertex = -1
		col.dirty = false
		return col
	}

	vertexCount := int32(len(buf) / 6)

	// Same size: overwrite in-place
	if vertexCount == col.vertexCount && col.firstFloat >= 0 {
		queueRegionWrite(r, col.firstFloat*2, buf)
		col.dirty = false
		col.firstVertex = int32(col.firstFloat / 6)
		return col
	}

	// Different size: allocate new slot, then free old
	isNewColumn := col.firstFloat < 0
	oldOffset := col.firstFloat
	oldSize := int(col.vertexCount) * 6

	offsetShorts, ok := allocInRegion(r, len(buf))
	if !ok {
		compactRegion(r)
		offsetShorts, ok = allocInRegion(r, len(buf))
	}
	if !ok {
		evictLRUColumns(r, len(buf)*2)
		offsetShorts, ok = allocInRegion(r, len(buf))
	}
	if !ok {
		// Allocation failed; keep old data (if any).
		// Schedule a retry after 30 frames so the column eventually picks up new
		// vertex data once memory pressure eases, without thrashing every frame.
		col.dirty = false
		col.retryFrame = currentFrame + 30
		return col
	}

	// Success – now free old slot
	if oldOffset >= 0 && oldSize > 0 {
		freeInRegion(r, oldOffset, oldSize)
	}

	queueRegionWrite(r, offsetShorts*2, buf)

	col.vertexCount = vertexCount
	col.firstFloat = offsetShorts
	col.firstVertex = int32(offsetShorts / 6)
	col.dirty = false

	if isNewColumn {
		if r.orderedColumns == nil {
			r.orderedColumns = make([]*columnMesh, 0, 8)
		}
		r.orderedColumns = append(r.orderedColumns, col)
		col.drawnFrame = currentFrame
		r.activeColumns++
	}
	return col
}

func setupAtlas() {
	if atlasRegions == nil {
		atlasRegions = make(map[[2]int]*atlasRegion)
	}
}
