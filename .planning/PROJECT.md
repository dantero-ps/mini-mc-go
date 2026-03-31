# Mini-MC Benchmarks

## What This Is

A Go + OpenGL Minecraft clone (Mini-MC) that needs comprehensive Go benchmark files covering all hot-path subsystems. The benchmarks will live as `_test.go` files runnable via `go test -bench` and serve as a performance regression safety net — catching slowdowns before they land in future changes.

## Core Value

Every hot-path subsystem has a benchmark you can run with `go test -bench` to immediately detect performance regressions after code changes.

## Requirements

### Validated

- ✓ Chunk generation pipeline benchmarks (gen → mesh → unpack → GPU upload, E2E, parallel) — `internal/benchmark/pipeline_test.go`
- ✓ Density generator populate chunk benchmark — `internal/world/generator_test.go`
- ✓ Greedy meshing with direction worker pools — existing
- ✓ MC 1.8.9 terrain generation (ChunkProvider189) — existing
- ✓ AABB collision detection system — existing
- ✓ Block raytracing — existing
- ✓ Fluid meshing with flow angle computation — existing
- ✓ Value noise / octave noise generation — existing
- ✓ Chunk storage with sectioned layout (16x16x16 sections) — existing

### Active

- [ ] Benchmark AABB collision detection (single entity vs chunk blocks)
- [ ] Benchmark block raytracing at various distances and angles
- [ ] Benchmark fluid mesh generation (flow angle, fluid height)
- [ ] Benchmark value noise and octave noise generation
- [ ] Benchmark ChunkProvider189 terrain generation in isolation
- [ ] Benchmark chunk block operations (GetBlock/SetBlock/section allocation)
- [ ] Benchmark chunk store cache operations (get/put/eviction)

### Out of Scope

- CI performance gates — manual `go test -bench` runs are sufficient
- Baseline snapshot comparison — just the benchmark files
- Rendering benchmarks (GPU-bound, already covered by GPU upload stage) — OpenGL calls require headless window context, already benchmarked in pipeline
- Inventory/UI/player model benchmarks — not hot paths
- Network benchmarks — no network layer exists

## Context

- **Language:** Go 1.24, OpenGL 4.1 Core, GLFW
- **Existing benchmarks:** `internal/benchmark/pipeline_test.go` covers the full chunk generation pipeline (gen, meshing, vertex unpack, GPU upload, E2E, parallel meshing). Well-structured with `TestMain` for GLFW/GL context setup, `b.Loop()` (Go 1.24 style), `b.ReportAllocs()`, and domain-specific metrics (`vertices/op`, `bytes/op`).
- **Existing test patterns:** Unit tests co-located with code, same package naming, `world.NewEmpty()` for minimal test worlds, `init()` for manual table population.
- **Key packages:** `internal/physics` (collision, raycast), `internal/meshing` (greedy, fluid), `internal/world` (noise, chunk, chunk_provider_189, chunk_store, generator).
- **Performance risks from CONCERNS.md:** `SetMeta`/`SetBlock` do O(4096) zero-scan on block removal, `colSet` map allocated every frame causes GC pressure, height-cache lookup race in chunk streamer.
- **Test convention:** `b.Loop()` for new benchmarks (not `for i := 0; i < b.N; i++`), `b.ReportAllocs()` on all benchmarks, `b.ReportMetric()` for domain-specific metrics.

## Constraints

- **Tech Stack:** Go 1.24 standard `testing` package only — no third-party benchmark libraries
- **Package Convention:** Benchmarks that test cross-package hot paths go in `internal/benchmark/` (uses GLFW `TestMain` for GL context). Package-internal benchmarks can stay in the same package.
- **No GL context needed for most:** Physics, noise, chunk ops don't need OpenGL — they can use simpler test setup without GLFW
- **Package naming:** `internal/benchmark/` uses `package benchmark`, physics benchmarks may use `package physics_test` (external) to match existing pattern

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Hot paths only (not full coverage) | Regression safety on performance-critical code, not exhaustive coverage | — Pending |
| `internal/benchmark/` for cross-package, co-located for package-internal | Follows existing convention from pipeline_test.go | — Pending |

---
*Last updated: 2026-03-31 after initialization*
