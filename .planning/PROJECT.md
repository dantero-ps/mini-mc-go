# Minecraft Clone (Go)

## What This Is

A full-featured Minecraft clone written in Go from scratch. A voxel-based sandbox game with procedurally generated infinite worlds, survival and creative modes, multiplayer support, and modern rendering. Built as a long-term passion project to explore game development, procedural generation, and Go's concurrency for real-time applications.

## Core Value

**Infinite, beautiful worlds that are fun to explore.**

Everything else—combat, crafting, building—depends on having compelling terrain that players want to discover and inhabit. World generation quality drives the entire experience.

## Requirements

### Validated

<!-- Already implemented and working -->

- ✓ Voxel rendering system with greedy meshing — existing codebase
- ✓ Chunk-based world (16x256x16) with sparse storage — existing codebase
- ✓ Deterministic world generation from seed — existing codebase
- ✓ Player physics and collision detection — existing codebase
- ✓ Block placement and breaking system — existing codebase
- ✓ Basic 2D heightmap terrain generation — existing codebase

### Active

<!-- Current development focus: World Generation -->

**Phase Focus: World & Generation**
- [ ] 3D density-based terrain (enables caves, overhangs, floating islands)
- [ ] Multi-biome system with smooth transitions
- [ ] Natural cave generation (cheese caves + spaghetti tunnels)
- [ ] Ore distribution and resource placement
- [ ] Structure generation (trees, villages, dungeons)

**Future Core Systems:**
- [ ] Inventory and item system
- [ ] Crafting and recipes
- [ ] Survival mechanics (health, hunger, day/night)
- [ ] Mob AI and spawning
- [ ] Multiplayer networking (client-server)
- [ ] Save/load world persistence

### Out of Scope

- **Exact Minecraft data compatibility** — Legal and technical complexity; build inspired-by, not compatible
- **Redstone circuits** — Defer to late phases; complex simulation system
- **Modding API initially** — Focus on core game first; extensibility later
- **Mobile/web platforms** — Desktop-first; cross-platform is future consideration
- **Pixel-perfect Minecraft clone** — Inspired by, not replication; legal and creative freedom

## Context

### Current Codebase State

**Foundation is solid** (from `.planning/codebase/` analysis):
- Custom voxel engine with no external game framework dependencies
- Worker pool-based parallel chunk generation and meshing
- Memory-efficient sparse chunk storage
- Clean separation: world generation, rendering, physics
- Go-idiomatic patterns (channels, goroutines, sync primitives)

**Architecture:**
- Entry: `cmd/main.go` → game loop
- World: `internal/world/` → chunks, generation, storage
- Rendering: `internal/render/` → OpenGL, meshes, shaders
- Physics: `internal/physics/` → collision, player movement

**Tech Stack:**
- Go 1.21+
- OpenGL 3.3 via go-gl/gl
- GLFW for windowing
- Custom noise/generation (no external world gen libs)

### Known Issues (from `.planning/codebase/CONCERNS.md`)

**High Priority:**
- 2D heightmap limits terrain variety (blocks caves/overhangs)
- No biome system (all terrain looks same)
- Missing world persistence (can't save/load)
- Performance degrades at high render distances

**Medium Priority:**
- Some TODOs in physics collision handling
- Test coverage gaps in world generation
- Documentation needs expansion

### Research Insights (from `.planning/research/`)

**Critical Path:** 3D terrain → biomes → caves → structures
- Each phase builds on previous
- 3D density function is foundation for everything
- Determinism testing is crucial at each step

**Performance Strategy:**
- Profile before optimizing (current patterns are good)
- Chunk-level caching for expensive noise
- LOD system for distant chunks (future)

## Constraints

- **Tech Stack:** Go + OpenGL — Chosen for learning/performance; locked for project continuity
- **Timeline:** Long-term hobby project — No deadlines, prioritize learning and quality over speed
- **Scope:** Full Minecraft-like feature set — Large scope acknowledged; phased approach required
- **Platform:** Desktop (Windows/Mac/Linux) — Single-player focus initially, multiplayer later
- **Dependencies:** Minimal external libs — Maintain control over core systems (world gen, physics)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Custom noise implementation | Full determinism control; no dependency on external RNG behavior | ✓ Good — Working well |
| Greedy meshing | Reduce vertex count dramatically; industry standard for voxel games | ✓ Good — Performant |
| Worker pool pattern | Go's goroutines excel at parallel chunk generation | ✓ Good — Clean scaling |
| Chunk-based world | Standard for infinite worlds; memory-efficient streaming | ✓ Good — Proven pattern |
| 2D heightmap (current) | Quick start; simple terrain | ⚠️ Revisit — Limits features, replace with 3D |
| No game framework | Full control; learning focus | — Pending — More code but better understanding |

---
*Last updated: 2026-02-08 after project initialization*
