# Roadmap: Mini-MC Benchmarks

## Overview

Add `go test -bench` coverage to Mini-MC's performance-critical hot paths — starting with pure-CPU noise and chunk benchmarks (no display server needed), then layering on meshing benchmarks that need registry initialization. By the end, every hot-path subsystem has a runnable benchmark that catches performance regressions.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Noise & Chunk Operations** — Pure CPU benchmarks for noise generation and chunk operations
- [ ] **Phase 2: Meshing** — Meshing benchmarks requiring registry init, building on Phase 1 patterns

## Phase Details

### Phase 1: Noise & Chunk Operations

**Goal**: Developers can benchmark all noise generation and chunk operation hot paths without any display server or external dependencies.

**Depends on**: Nothing (first phase)

**Requirements**: NOISE-01, NOISE-02, NOISE-03, CHNK-01, CHNK-02, CHNK-03

**Success Criteria** (what must be TRUE):
1. `go test -bench=BenchmarkValueNoise ./internal/world/` reports ns/op, allocs/op, and samples/op for 2D and 3D value noise
2. `go test -bench=BenchmarkOctaveNoise ./internal/world/` runs sub-benchmarks for varying octave counts (2, 4, 6) with allocation metrics
3. `go test -bench=BenchmarkNoiseArray ./internal/world/` reports timing for authentic MC 1.8.9 noise grid population per chunk
4. `go test -bench=BenchmarkChunk ./internal/world/` reports GetBlock/SetBlock throughput and section-aware iteration performance (including IsSectionEmpty fast-path validation)
5. `go test -bench=BenchmarkChunkNeighborMeshing ./internal/world/` reports cross-border overhead comparing 0 vs 6 neighbor chunks

**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md — Noise benchmarks (NOISE-01, NOISE-02, NOISE-03)
- [x] 01-02-PLAN.md — Chunk benchmarks (CHNK-01, CHNK-02, CHNK-03)

### Phase 2: Meshing

**Goal**: Developers can benchmark all meshing hot paths — greedy meshing, fluid meshing, custom model meshing, and fluid tick simulation — in isolation.

**Depends on**: Phase 1 (chunk setup patterns, b.Loop conventions established)

**Requirements**: MESH-01, MESH-02, MESH-03, MESH-04

**Success Criteria** (what must be TRUE):
1. `go test -bench=BenchmarkGreedyDirection ./internal/meshing/` isolates single-direction greedy meshing with timing independent of custom model pass
2. `go test -bench=BenchmarkFluidMesh ./internal/meshing/` runs sub-benchmarks for source-only, flowing, and mixed fluid densities with flow angle metrics
3. `go test -bench=BenchmarkCustomModel ./internal/meshing/` benchmarks transparent/complex block meshing through the non-greedy path
4. `go test -bench=BenchmarkFluidTick ./internal/world/` benchmarks cascading water simulation reporting flow directions evaluated per op

**Plans**: 2 plans

Plans:
- [ ] 02-01-PLAN.md — Greedy, fluid mesh, and custom model benchmarks (MESH-01, MESH-02, MESH-03)
- [x] 02-02-PLAN.md — FluidTick benchmark (MESH-04)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Noise & Chunk Operations | 0/2 | Planned | - |
| 2. Meshing | 0/2 | Not started | - |
