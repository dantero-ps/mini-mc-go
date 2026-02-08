# Codebase Concerns

**Analysis Date:** 2026-02-08

## Tech Debt

**Incomplete Feature Implementation:**
- Issue: Death handling is not implemented. Players take damage and health reaches zero but respawn/death logic is unimplemented
- Files: `internal/player/state.go` (line 180)
- Impact: Players cannot die properly in the game; game state becomes inconsistent after health reaches 0
- Fix approach: Implement respawn mechanics, death screen UI, and proper game state reset

**Missing Sound Effects:**
- Issue: Fall sound effect is not played when players take fall damage
- Files: `internal/player/movement.go` (line 436)
- Impact: Audio feedback for fall damage missing; reduced player awareness of damage events
- Fix approach: Integrate audio engine for fall sounds; implement consistent audio system for all damage types

**Placeholder Jump Boost Logic:**
- Issue: Jump boost effect amplifier is not retrieved from any effect system; hardcoded to 0.0
- Files: `internal/player/movement.go` (line 429)
- Impact: Jump boost potion effects have no practical effect on fall damage calculation
- Fix approach: Implement effect/potion system with amplifier tracking

**Atlas Region Management Hack:**
- Issue: Atlas compaction and region cleanup is incomplete; commented out code and unclear cleanup strategy
- Files: `internal/graphics/renderables/blocks/meshing.go` (line 168-172)
- Impact: Memory may not be properly reclaimed when chunks are pruned; atlas fragmentation could accumulate
- Fix approach: Implement proper atlas compaction with safe region reclamation; document the strategy clearly

## Race Conditions & Concurrency Issues

**Unsynchronized Global Maps (Critical):**
- Issue: `chunkMeshes` and `columnMeshes` maps are accessed from multiple goroutines without proper synchronization on all accesses
- Files: `internal/graphics/renderables/blocks/meshing.go` (lines 10, 13, 67, 85-88, 101, 151-176)
- Impact: Data race conditions when rendering threads access maps while worker threads modify them. Can cause crashes or corrupt mesh data
- Trigger: High frequency of chunk loading/meshing under player movement
- Safe modification: Wrap all map accesses with mutex. Currently only `pendingMeshJobs` has mutex protection (line 20)
- Recommended fix: Add mutex protection for `chunkMeshes` and `columnMeshes` similar to `pendingMeshJobs`

**Unsafe Pointer Arithmetic Without Validation:**
- Issue: Extensive use of `unsafe.Pointer` with arithmetic in chunk block access without bounds validation
- Files: `internal/world/chunk.go` (lines 64, 82, 110, 155)
- Impact: Out-of-bounds pointer arithmetic could corrupt adjacent memory; undefined behavior if `basePtr` becomes stale
- Cause: Using unsafe pointers to avoid slice bounds checking for performance
- Safe modification: Validate section existence and slice initialization before pointer arithmetic; consider if performance gain justifies safety risk
- Test coverage gap: No tests validating unsafe pointer operations under edge cases

**Chunk Stream Buffer Overflow Risk:**
- Issue: Job queue has fixed capacity (4096) but no backpressure mechanism beyond maxPending check
- Files: `internal/world/chunk_streamer.go` (lines 31, 34, 148-150)
- Impact: If chunk generation lags, incoming stream requests may fail silently; chunks fail to load if pending cap reached
- Trigger: Large render distance with slow terrain generation
- Workaround: maxPending prevents queue from growing unbounded but chunks are dropped
- Improvement path: Implement priority queue or adaptive streaming radius based on generation lag

**Memory Pool Without Lifecycle Management:**
- Issue: `maskPool` in greedy meshing reuses buffers without validation that previously allocated buffers are not still in use
- Files: `internal/meshing/greedy.go` (lines 31-37)
- Impact: Potential buffer corruption if a borrowed buffer is reused before being released from previous job
- Fix approach: Implement proper buffer lifecycle tracking; add assertions or reference counting

## Known Bugs

**Unsafe Slice Lifetime in Atlas Copy:**
- Symptoms: Memory corruption or access violations during atlas buffer resize operations
- Files: `internal/graphics/renderables/blocks/atlas.go` (lines 65-78)
- Trigger: Simultaneous reads and unsafe slice creation from mapped buffers during compaction
- Current mitigation: Nil checks before dereference (lines 64, 71-77)
- Recommendation: Ensure gl.UnmapBuffer is called even on error; document that srcPtr/dstPtr lifetime is bounded by unmap call

**Nil Pointer Dereference in Chunk Mesh Apply:**
- Symptoms: Crash when applying mesh results if columnMeshes map doesn't contain expected key
- Files: `internal/graphics/renderables/blocks/meshing.go` (lines 85-88)
- Trigger: Race condition between mesh result processing and column mesh cleanup
- Current mitigation: Nil check (`if col != nil`)
- Recommendation: Add debug logging when unexpected nil encountered; consider more defensive design

## Security Considerations

**Panic on Asset Load Failure (Non-hardened):**
- Risk: Application crashes if font or texture assets are missing or corrupted; no graceful degradation
- Files: `internal/game/app.go` (lines 47-58); `internal/graphics/renderables/hud/inventory_screen.go` (line 26)
- Current mitigation: None - direct panic
- Recommendations: Implement fallback assets; validate asset integrity before load; provide user-friendly error messages

**No Input Validation on Block Coordinates:**
- Risk: Negative or extremely large coordinates could cause undefined behavior in pointer arithmetic
- Files: `internal/world/chunk.go` (lines 50-52, 69-71) - has bounds check but downstream unsafe arithmetic has no additional checks
- Current mitigation: Bounds check in GetBlock/SetBlock but not in unsafe pointer operations
- Recommendations: Add assertions; validate that computed offsets are within section bounds

## Performance Bottlenecks

**Greedy Mesh Worker Thread Pool Underutilization:**
- Problem: Direction pool uses fixed 6 workers per 1 mesh worker, leading to potential stalls if one direction is much slower
- Files: `internal/meshing/pool.go` (lines 40-45); `internal/meshing/greedy.go` (lines 41-46)
- Cause: Static worker count; no dynamic load balancing across directions
- Improvement path: Make direction worker count configurable; implement work-stealing or adaptive pooling

**Atlas Buffer Compaction Performance:**
- Problem: Compaction involves full buffer copy for entire atlas (up to 512MB)
- Files: `internal/graphics/renderables/blocks/atlas.go` (lines 353-370)
- Cause: No incremental compaction strategy; triggered every 2000 frames unconditionally if regions have holes
- Current capacity: 256MB initial, 512MB max - limited scalability
- Improvement path: Implement incremental/streaming compaction; region-based compaction instead of full buffer

**Physics Collision Detection Linear Scan:**
- Problem: Collides() checks 27 blocks in cube around player (3³) with full iteration every frame
- Files: `internal/physics/collision.go` (lines 23-49)
- Cause: Brute force AABB collision check with no spatial optimization
- Scaling limit: Performance degrades with chunk density; fine for current scope but would limit world complexity
- Improvement path: Implement spatial indexing (grid, BVH); profile actual performance impact under load

**Mesh Result Processing Not Batched:**
- Problem: ProcessMeshResults processes one result per frame (for loop with default), not batching
- Files: `internal/graphics/renderables/blocks/meshing.go` (lines 43-51)
- Impact: If mesh results accumulate faster than processing, latency increases linearly
- Improvement path: Process batch of results per frame; adaptive batching based on queue depth

## Fragile Areas

**Chunk Dirty Flag & Meshing State Machine:**
- Files: `internal/graphics/renderables/blocks/meshing.go` (lines 104-133)
- Why fragile: Complex interplay between `ch.IsDirty()`, pending job tracking, and `ch.SetClean()`. Race condition window: chunk marked clean at line 131 after job submitted but before result processed
- Safe modification: Never call SetClean() outside of ensureChunkMesh; consider embedding dirty flag in pending job map key instead
- Test coverage: No tests for concurrent modification scenarios

**Unsafe Pointer Lifecycle in Chunk Sections:**
- Files: `internal/world/chunk.go` (lines 57, 64, 82, 107, 110)
- Why fragile: basePtr becomes invalid if blocks slice is reallocated; no protection against concurrent access to blocks[]
- Safe modification: Treat sections as immutable after initialization; validate basePtr before each access; add reference counting
- Test coverage: No tests for concurrent SetBlock/GetBlock access

**Atlas Region Memory Tracking:**
- Files: `internal/graphics/renderables/blocks/atlas.go` (lines 13-25, 192-217)
- Why fragile: `totalAllocatedBytes` and `currentFrame` are globals with no synchronization; atlas can exceed maxBytes if compaction timing is off
- Safe modification: Make atlasing thread-safe or ensure it's only called from main render thread; add hard limit enforcement
- Test coverage: No tests validating memory limits under load

## Scaling Limits

**Chunk Loading Synchronous Fallback:**
- Current capacity: StreamChunksAroundAsync queues up to 2048 chunks per call
- Limit: If terrain generation is very slow, synchronous StreamChunksAroundSync may block main thread, causing frame drops
- Files: `internal/world/chunk_streamer.go` (lines 75-92)
- Scaling path: Implement priority loading; separate critical chunks (player chunk) from distant chunks; add stream compression

**Atlas Texture Memory:**
- Current capacity: 256MB initial, 512MB max for single atlas region
- Limit: Large worlds with many unique texture patterns will exceed atlas before chunk count limit; no texture streaming
- Files: `internal/graphics/renderables/blocks/atlas.go` (lines 13-14)
- Scaling path: Implement texture atlas streaming with paging; tile-based texture addressing; implement texture compression

**Mesh Worker Queue Depth:**
- Current capacity: 200 jobs in queue
- Limit: If chunk generation is very slow, queue fills and mesh requests are dropped (returns false from SubmitJob)
- Files: `internal/meshing/pool.go` (line 27)
- Scaling path: Dynamic queue resizing; adaptive worker count based on queue depth; implement priority queue

## Dependencies at Risk

**No Formal Version Pinning in go.mod:**
- Risk: Indirect dependency updates could introduce breaking changes; no vendor directory
- Impact: go get -u could update to incompatible versions of go-gl or mathgl
- Migration plan: Add explicit version constraints in go.mod; test transitive dependency changes; consider vendoring critical deps

**go-gl/gl Compatibility:**
- Risk: GL driver API surface varies by platform; some GL operations may not be available on older GPUs
- Current usage: gl.MapBufferRange, gl.UnmapBuffer, unsafe slice operations
- Mitigation: Require minimum GL 4.1 core (already in use); test on target hardware
- Recommendation: Add feature detection for advanced features; provide fallbacks for basic rendering

## Missing Critical Features

**No Chunk Persistence:**
- Problem: Generated world is temporary; no save/load mechanism
- Blocks: Prevents progression; player worlds lost on exit
- Implementation approach: Serialize chunks to disk; implement chunk storage backend; add save point mechanics

**No Error Recovery:**
- Problem: Panics on asset load failure; no graceful degradation for resource issues
- Blocks: Cannot run on systems with incomplete assets or memory pressure
- Implementation approach: Implement fallback rendering; lazy asset loading; resource budget manager

**No Performance Monitoring in Production:**
- Problem: profiling.Track() is used but profiling output not exposed to users
- Blocks: Cannot diagnose performance issues without code inspection
- Implementation approach: Implement in-game profiler UI; export metrics; add frame time tracking

## Test Coverage Gaps

**Chunk Block Access with Concurrent Modifications:**
- What's not tested: Simultaneous GetBlock/SetBlock calls on same chunk from different goroutines
- Files: `internal/world/chunk.go`
- Risk: Race condition could cause memory corruption or panics
- Priority: High - safety critical

**Unsafe Pointer Arithmetic Edge Cases:**
- What's not tested: Block access at section boundaries; SetBlock on invalid coordinates; basePtr invalidation
- Files: `internal/world/chunk.go` (lines 64, 82, 110)
- Risk: Out-of-bounds access undetected
- Priority: High - safety critical

**Mesh Pool Resource Exhaustion:**
- What's not tested: Queue full condition; job timeout; worker goroutine panic
- Files: `internal/meshing/pool.go`
- Risk: Silent failures or deadlock under stress
- Priority: Medium - affects stability

**Physics Collision Extreme Coordinates:**
- What's not tested: Very large or negative block coordinates; coordinate overflow
- Files: `internal/physics/collision.go`
- Risk: Unexpected behavior or panic
- Priority: Medium - affects gameplay robustness

**Inventory Stack Operations:**
- What's not tested: Concurrent inventory modifications; negative counts; stack size overflow
- Files: `internal/inventory/inventory.go` (lines 60-105)
- Risk: Inventory corruption; duplicate or lost items
- Priority: High - gameplay critical

---

*Concerns audit: 2026-02-08*
