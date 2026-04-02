# Requirements: Mini-MC Benchmarks

**Defined:** 2026-03-31
**Core Value:** Every hot-path subsystem has a benchmark you can run with `go test -bench` to immediately detect performance regressions after code changes.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Noise Generation

- [x] **NOISE-01**: Benchmark value noise generation (2D and 3D) with `b.ReportAllocs()` and domain-specific metrics (samples/op)
- [x] **NOISE-02**: Benchmark octave noise generation (2D and 3D) with varying octave counts via `b.Run` sub-benchmarks
- [x] **NOISE-03**: Benchmark authentic MC 1.8.9 noise array population (PopulateNoiseArray) for a single chunk's noise grid

### Chunk Operations

- [x] **CHNK-01**: Benchmark GetBlock/SetBlock/SetBlockFast throughput on a pre-allocated chunk, comparing section-backed vs empty paths
- [x] **CHNK-02**: Benchmark section-aware chunk iteration (all sections populated vs mostly-empty) to validate IsSectionEmpty fast-path optimization
- [x] **CHNK-03**: Benchmark chunk meshing with 0 and 6 neighbor chunks present to measure cross-border overhead independently

### Meshing

- [ ] **MESH-01**: Benchmark greedy meshing single-direction isolation (buildGreedyForDirection) independent of custom model pass
- [ ] **MESH-02**: Benchmark fluid mesh generation (BuildFluidMesh) with varying fluid densities (source-only, flowing, mixed)
- [ ] **MESH-03**: Benchmark custom model meshing (meshCustomBlock) for transparent/complex blocks (non-greedy path)
- [x] **MESH-04**: Benchmark fluid tick simulation (FluidTick) with cascading water, reporting flow directions evaluated

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Physics

- **PHYS-01**: Benchmark raycast (DDA) with hit/miss/diagonal scenarios
- **PHYS-02**: Benchmark AABB collision (Collides) with ground/air/surrounded scenarios
- **PHYS-03**: Benchmark FindGroundLevel and FindCeilingLevel, reporting blocks_scanned/op
- **PHYS-04**: Benchmark parallel collision with b.RunParallel for RWMutex contention testing

### Advanced

- **ADVN-01**: Noise throughput head-to-head comparison (value noise vs authentic MC noise)
- **ADVN-02**: Benchmark comparison baseline files with benchstat integration

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full frame render benchmarks | GPU pipeline stalls, vsync, OS scheduler make results non-deterministic; use runtime profiling instead |
| Individual vertex emit function benchmarks | Compiler inlining makes isolated benchmarks misleading; measure at mesh level |
| Memory layout / cache-line benchmarks | Go testing.B cannot measure cache behavior; use pprof with hardware counters instead |
| Fuzz-style benchmarks with random inputs | Makes results non-reproducible; use fixed-seed parameterized b.Run instead |
| GPU upload benchmarks without headless GLFW | Requires visible window/display server; keep existing headless pattern |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| NOISE-01 | Phase 1 | Complete |
| NOISE-02 | Phase 1 | Complete |
| NOISE-03 | Phase 1 | Complete |
| CHNK-01 | Phase 1 | Complete |
| CHNK-02 | Phase 1 | Complete |
| CHNK-03 | Phase 1 | Complete |
| MESH-01 | Phase 2 | Pending |
| MESH-02 | Phase 2 | Pending |
| MESH-03 | Phase 2 | Pending |
| MESH-04 | Phase 2 | Complete |

**Coverage:**
- v1 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-31*
*Last updated: 2026-03-31 after roadmap creation*
