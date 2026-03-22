package world

import "container/heap"

// BlockPos represents a world-space block position.
type BlockPos struct {
	X, Y, Z int
}

// ScheduledTick represents a scheduled block update.
type ScheduledTick struct {
	Pos        BlockPos
	TargetTick int64
	Priority   int
	index      int // maintained by heap.Interface for O(log n) removal
}

// tickHeap implements heap.Interface over a slice of ScheduledTick.
// Ordering: ascending TargetTick, then ascending Priority (lower priority value = higher urgency).
type tickHeap []*ScheduledTick

func (h tickHeap) Len() int { return len(h) }

func (h tickHeap) Less(i, j int) bool {
	if h[i].TargetTick != h[j].TargetTick {
		return h[i].TargetTick < h[j].TargetTick
	}
	return h[i].Priority < h[j].Priority
}

func (h tickHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *tickHeap) Push(x interface{}) {
	n := len(*h)
	t := x.(*ScheduledTick)
	t.index = n
	*h = append(*h, t)
}

func (h *tickHeap) Pop() interface{} {
	old := *h
	n := len(old)
	t := old[n-1]
	old[n-1] = nil // avoid memory leak
	t.index = -1
	*h = old[:n-1]
	return t
}

// TickScheduler manages scheduled block updates using a min-heap keyed on target tick.
// A pending map prevents duplicate entries and enables O(1) lazy cancellation.
type TickScheduler struct {
	currentTick int64
	h           tickHeap
	pending     map[BlockPos]struct{} // deduplication guard
	resultBuf   []BlockPos            // pre-allocated; reused each Process call
}

// NewTickScheduler returns an initialised TickScheduler.
func NewTickScheduler() *TickScheduler {
	ts := &TickScheduler{
		pending:   make(map[BlockPos]struct{}),
		resultBuf: make([]BlockPos, 0, 64),
	}
	heap.Init(&ts.h)
	return ts
}

// Schedule enqueues a block update for pos to fire after delay ticks from now.
// If pos already has a pending tick it is silently ignored (prevents heap growth).
func (ts *TickScheduler) Schedule(pos BlockPos, delay int, priority int) {
	if _, exists := ts.pending[pos]; exists {
		return
	}
	ts.pending[pos] = struct{}{}
	t := &ScheduledTick{
		Pos:        pos,
		TargetTick: ts.currentTick + int64(delay),
		Priority:   priority,
	}
	heap.Push(&ts.h, t)
}

// Process advances the tick counter by one and pops all due ticks (TargetTick <= currentTick),
// up to maxUpdates. Lazily skips any entry whose position was removed from the pending map
// (i.e. cancelled). The returned slice is backed by an internal buffer and must not be retained
// across calls.
func (ts *TickScheduler) Process(maxUpdates int) []BlockPos {
	ts.currentTick++
	ts.resultBuf = ts.resultBuf[:0]

	for ts.h.Len() > 0 && len(ts.resultBuf) < maxUpdates {
		top := ts.h[0]
		if top.TargetTick > ts.currentTick {
			break
		}
		heap.Pop(&ts.h)

		// Lazy cancellation: skip if pos was removed from pending.
		if _, ok := ts.pending[top.Pos]; !ok {
			continue
		}
		delete(ts.pending, top.Pos)
		ts.resultBuf = append(ts.resultBuf, top.Pos)
	}

	return ts.resultBuf
}

// Cancel removes a pending tick for pos. The heap entry is not removed eagerly;
// it will be skipped during the next Process call (lazy cancel).
func (ts *TickScheduler) Cancel(pos BlockPos) {
	delete(ts.pending, pos)
}

// CancelInRange lazily cancels all pending ticks whose block position falls inside
// the chunk at chunk coordinates (chunkX, chunkZ).
func (ts *TickScheduler) CancelInRange(chunkX, chunkZ int) {
	for pos := range ts.pending {
		if floorDiv(pos.X, ChunkSizeX) == chunkX && floorDiv(pos.Z, ChunkSizeZ) == chunkZ {
			delete(ts.pending, pos)
		}
	}
}

// CancelOutsideRadius lazily cancels all pending ticks whose chunk coordinate is
// further than radius chunks (Chebyshev square) from (cx, cz).
func (ts *TickScheduler) CancelOutsideRadius(cx, cz, radius int) {
	for pos := range ts.pending {
		pcx := floorDiv(pos.X, ChunkSizeX)
		pcz := floorDiv(pos.Z, ChunkSizeZ)
		dx := pcx - cx
		dz := pcz - cz
		if dx*dx+dz*dz > radius*radius {
			delete(ts.pending, pos)
		}
	}
}

// CurrentTick returns the current tick count.
func (ts *TickScheduler) CurrentTick() int64 {
	return ts.currentTick
}
