# Codebase Review: Single Responsibility Principle & Technical Debt Analysis

**Date**: 2026-02-07  
**Scope**: Full `internal/` package analysis  

---

## Executive Summary

This document provides a comprehensive review of the codebase focusing on:
1. **Single Responsibility Principle (SRP) Violations** - "God Files" that do too much
2. **Critical Bugs & Technical Debt**
3. **Refactoring Recommendations**

### Overall Health: ⚠️ MODERATE CONCERN

The codebase is functional but has several architectural issues that will compound over time. The most critical issues are:
- 1 **Critical Bug** (Race Condition in `world.go`)
- 3 **God Files** that violate SRP severely
- 4 **Medium-severity** files with mixed responsibilities

---

## Part 1: God Files & SRP Analysis

### Severity Legend
- 🔴 **SEVERE** - Multiple unrelated responsibilities, should be split immediately
- 🟠 **MODERATE** - Mixed concerns, should be split when opportunity arises  
- 🟢 **CLEAN** - Single, well-defined responsibility

---

### 🔴 SEVERE: `internal/player/player.go` (1253 lines)

**Current Responsibilities:**
1. **Player State**: Position, velocity, health, food level, game mode
2. **Movement Physics**: Gravity, friction, drag calculations, jump mechanics
3. **Collision Detection Integration**: Calls physics functions, resolves collisions
4. **Camera/View**: View matrix calculation, head bobbing, arm sway
5. **Input Handling**: Mouse movement, mining, block placement
6. **Inventory Management**: Item dropping, cursor interactions
7. **Mining State Machine**: Break progress, break block logic
8. **Animation State**: Hand swing, equip progress
9. **Entity Collision**: Item pickup logic with world entities

**Why This Is Bad:**
- Changing mining mechanics requires touching the same file as camera calculations
- The file imports 11 packages - a sign of too many concerns
- Testing physics requires mocking the entire player

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `player/state.go` | Core player state (position, health, game mode) |
| `player/movement.go` | Physics: gravity, friction, velocity updates |
| `player/camera.go` | View matrix, bobbing, arm sway, mouse look |
| `player/mining.go` | Break progress, block destruction logic |
| `player/interaction.go` | Block placement, item dropping, entity pickup |

---

### 🔴 SEVERE: `internal/graphics/renderables/hud/hud.go` (1028 lines)

**Current Responsibilities:**
1. **Hotbar Rendering**: Drawing slots, selector, item counts
2. **Inventory Screen Rendering**: Full inventory UI, slot grid, player model
3. **Health/Food Bars**: Heart and hunger icon rendering
4. **Profiling Display**: Frame time statistics, performance metrics
5. **Text Rendering**: Coordinates, FPS counter
6. **Inventory Click Handling**: Slot click detection, item swapping, double-click logic
7. **Slot Management**: `MoveHoveredItemToHotbar`, hover detection

**Why This Is Bad:**
- UI rendering mixed with click handling logic (MVC violation)
- Profiling display has nothing to do with HUD gameplay
- Inventory logic duplicates some logic from `inventory.go`

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `hud/hotbar.go` | Hotbar rendering only |
| `hud/inventory_screen.go` | Inventory UI rendering |
| `hud/status_bars.go` | Health, food, XP bars |
| `hud/profiling_overlay.go` | Debug/profiling display |
| `hud/inventory_controller.go` | Click handling and slot manipulation |

---

### 🔴 SEVERE: `internal/world/world.go` (601 lines)

**Current Responsibilities:**
1. **Chunk Storage**: Map management, column index, mod count
2. **Entity Lifecycle**: `AddEntity`, `UpdateEntities`, dead entity cleanup
3. **World Generation**: `populateChunk`, noise parameters, height calculation
4. **Async Job Scheduling**: Worker pool, pending jobs, job queue
5. **Block Access**: Get/Set block, neighbor dirty marking
6. **Chunk Streaming**: `StreamChunksAroundAsync`, `EvictFarChunks`

**Why This Is Bad:**
- Changing terrain generation requires touching the same file as entity management
- The `World` struct has 18 fields mixing storage, threading, and generation concerns
- Testing chunk streaming requires setting up noise parameters

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `world/chunk_store.go` | Chunk map, column index, Get/Set operations |
| `world/entity_manager.go` | Entity storage, ticking, cleanup |
| `world/generator.go` | Terrain generation, noise, height calculation |
| `world/chunk_streamer.go` | Async loading, eviction, job scheduling |

---

### 🟠 MODERATE: `internal/meshing/greedy.go` (644 lines)

**Current Responsibilities:**
1. **Greedy Meshing Algorithm**: Core face merging logic
2. **Worker Pool Management**: `DirectionWorkerPool`, job submission
3. **Vertex Packing**: Bit-packing coordinates, normals, textures
4. **Custom Block Pass**: Non-solid block rendering fallback

**Issues:**
- Worker pool is tightly coupled to meshing algorithm
- Vertex packing format is hardcoded (not reusable by other systems)
- Custom block logic is a second meshing system within the same file

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `meshing/worker_pool.go` | Direction worker pool (already exists as `pool.go`, should absorb direction pool) |
| `meshing/vertex_format.go` | Vertex packing/unpacking utilities |
| `meshing/greedy.go` | Only the greedy algorithm |

---

### 🟠 MODERATE: `internal/graphics/renderables/blocks/atlas.go` (468 lines)

**Current Responsibilities:**
1. **GPU Memory Management**: VBO/VAO creation, buffer growing
2. **Vertex Data Compaction**: Defragmentation, hole management
3. **Column Mesh Assembly**: Collecting vertices from chunks
4. **Write Queue**: Batched buffer updates

**Issues:**
- Low-level GL operations mixed with high-level mesh management
- Compaction logic duplicates some column mesh concepts

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `blocks/gpu_buffer.go` | VBO/VAO management, memory allocation |
| `blocks/atlas.go` | Column mesh assembly, write batching |

---

### 🟠 MODERATE: `internal/graphics/renderables/ui/ui.go` (480 lines)

**Current Responsibilities:**
1. **Draw Command Batching**: FIFO command list, flush logic
2. **Filled Rectangle Rendering**: Solid color quads
3. **Textured Rectangle Rendering**: UV-mapped quads
4. **Slider Widget**: Interactive slider with drag state
5. **Text Integration**: Font renderer delegation

**Issues:**
- Slider widget logic belongs in `ui/widget/` not in the core renderer
- Mix of immediate-mode (DrawSlider) and retained-mode (Flush) patterns

**Recommended Split:**

| New File | Responsibility |
|----------|----------------|
| `ui/ui.go` | Core batching, DrawFilledRect, DrawTexturedRect, Flush |
| `ui/widget/slider.go` | Already exists, but UI.DrawSlider should delegate to it |

---

### 🟠 MODERATE: `internal/game/session.go` (298 lines)

**Current Responsibilities:**
1. **Session Lifecycle**: Creation, cleanup
2. **Update Loop**: Orchestrating player/world updates
3. **Input Dispatch**: Hotbar, inventory, pause handling
4. **Chunk Streaming Orchestration**: Calling world methods

**Issues:**
- Input handling is verbose (handleInputActions is 70+ lines of if statements)
- Mixes orchestration with detailed input logic

**Recommended Improvement:**
- Extract `handleInputActions` to `input_handlers.go` (already partially exists)
- Keep session.go as pure orchestration

---

### 🟢 CLEAN: Well-Structured Files

These files demonstrate good SRP:

| File | Responsibility | Lines |
|------|----------------|-------|
| `internal/graphics/renderer/renderer.go` | Orchestrates renderables | 128 |
| `internal/inventory/inventory.go` | Inventory storage and manipulation | 134 |
| `internal/entity/entity.go` | Entity interface definition | 22 |
| `internal/entity/item_entity.go` | Item entity behavior | ~200 |
| `internal/physics/collision.go` | Collision detection functions | 158 |
| `internal/physics/raycast.go` | Raycasting utility | ~50 |
| `internal/config/config.go` | Configuration storage | ~100 |
| `internal/profiling/profiling.go` | Performance tracking | ~100 |
| `internal/meshing/pool.go` | Generic worker pool | 113 |

---

## Part 2: Critical Bugs & Technical Debt

### 🔴 CRITICAL: Race Condition in `GetChunk`

**File**: `internal/world/world.go:129-154`

**Problem**: Double-checked locking pattern is incomplete. After releasing RLock and acquiring Lock, the code does not re-check if another goroutine already created the chunk.

```go
// BUGGY CODE
func (w *World) GetChunk(chunkX, chunkY, chunkZ int, create bool) *Chunk {
    coord := ChunkCoord{X: chunkX, Y: chunkY, Z: chunkZ}
    w.mu.RLock()
    chunk, exists := w.chunks[coord]
    w.mu.RUnlock()  // <-- RELEASES LOCK
    if !exists && create {
        chunk = NewChunk(chunkX, chunkY, chunkZ)
        w.mu.Lock()  // <-- ANOTHER THREAD COULD HAVE CREATED IT HERE
        w.chunks[coord] = chunk  // <-- OVERWRITES EXISTING CHUNK!
        // ...
    }
}
```

**Impact**: Two threads requesting the same new chunk will both create chunks. The second overwrites the first, losing any data.

**Fix**: Add existence check after acquiring write lock.

---

### 🟠 HIGH: Memory Allocation Churn in Meshing

**File**: `internal/meshing/greedy.go:325, 436, 539`

**Problem**: `buildGreedyForDirection` allocates a new `mask` slice for every layer of every chunk.

**Impact**: ~300 allocations per chunk rebuild causing GC pressure and frame stutters.

**Fix**: Use a pooled scratch buffer per worker.

---

### 🟠 HIGH: Non-Standard Physics Coordinate System

**File**: `internal/physics/collision.go:29-31`

**Problem**: Uses "top-at-integer" mapping where block Y=64 occupies range `[63.0, 64.0)`. This conflicts with standard conventions where Y=64 occupies `[64.0, 65.0)`.

**Impact**: Frequent off-by-one errors, confusing debug sessions, incompatibility with standard math.

**Fix**: Requires coordinated refactor of physics and rendering to use standard conventions.

---

### 🟡 MEDIUM: Hardcoded Block Types in Generation

**File**: `internal/world/world.go:510-521`

**Problem**: `populateChunk` explicitly references `BlockTypeBedrock`, `BlockTypeDirt`, `BlockTypeGrass`.

**Impact**: Adding biomes or new terrain features requires modifying core engine code.

**Fix**: Inject a `TerrainGenerator` interface that returns block types.

---

### 🟡 MEDIUM: Singleton Worker Pool

**File**: `internal/meshing/greedy.go:36-43`

**Problem**: `GetDirectionPool()` uses a singleton pattern.

**Impact**: 
- Cannot run multiple worlds in parallel (e.g., client + server)
- Unit tests share state
- Hard to configure pool size per-world

**Fix**: Worker pool should be created per-World or passed via context.

---

### 🟡 MEDIUM: Hardcoded Player Dimensions

**File**: `internal/physics/collision.go:15-16, internal/player/player.go:22-23`

**Problem**: Player width `0.3` and height `1.8` are hardcoded constants.

**Impact**: Cannot have different entity sizes (spiders, zombies, fences).

**Fix**: Pass entity dimensions as parameters or use an `Entity` interface with `GetBounds()`.

---

## Part 3: Refactoring Roadmap

### Phase 1: Critical Fixes (Immediate)
1. ✅ Fix `GetChunk` race condition (add double-check after Lock)
2. ✅ Add buffer pooling to greedy mesher

### Phase 2: God File Splits (Next Sprint)
1. ✅ Split `player/player.go` into 5 files
2. ✅ Split `hud/hud.go` into 5 files
3. ✅ Split `world/world.go` into 4 files

### Phase 3: Architecture Improvements (Backlog)
1. ⬜ Standardize physics coordinate system
2. ✅ Extract `TerrainGenerator` interface
3. ⬜ Remove singleton pattern from meshing pool
4. ⬜ Create `Entity` interface with `GetBounds()`

---

## Appendix: File Size Summary

| File | Lines | Verdict |
|------|-------|---------|
| `player/player.go` | 1253 | 🔴 Split immediately |
| `hud/hud.go` | 1028 | 🔴 Split immediately |
| `greedy.go` | 644 | 🟠 Split when possible |
| `world.go` | 601 | 🔴 Split immediately |
| `ui.go` | 480 | 🟠 Minor cleanup |
| `atlas.go` | 468 | 🟠 Split when possible |
| `playermodel/player_model.go` | 429 | 🟢 Acceptable (single model) |
| `blocks.go` | 325 | 🟢 Acceptable |
| `session.go` | 298 | 🟠 Extract input handlers |
| `registry/blocks.go` | 295 | 🟢 Acceptable |
| `meshing.go` | 197 | 🟢 Clean |
| `collision.go` | 158 | 🟢 Clean |
| `inventory.go` | 134 | 🟢 Clean |
| `renderer.go` | 128 | 🟢 Very clean |
| `pool.go` | 113 | 🟢 Very clean |
