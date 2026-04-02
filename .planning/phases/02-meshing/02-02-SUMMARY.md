---
phase: 02-meshing
plan: 02
subsystem: testing
tags: [benchmark, fluid, water-simulation, performance]

requires:
  - phase: 02-meshing
    provides: FluidTick implementation in internal/world/fluid.go

provides:
  - BenchmarkFluidTick benchmark measuring cascading water simulation with flow_dirs/op metric

affects: [performance-regression-detection, fluid-simulation]

tech-stack:
  added: []
  patterns: [per-iteration world reset using b.StopTimer/StartTimer, flow_dirs/op custom metric]

key-files:
  created: [internal/world/fluid_bench_test.go]
  modified: []

key-decisions:
  - "Used b.StopTimer/StartTimer inside b.Loop() to isolate world construction cost from FluidTick measurement"
  - "flow_dirs/op = (colDepth+1)*4 = 36, counting 4 horizontal neighbor checks per block in 8-block column"
  - "No TestMain added — reuses existing one from chunk_neighbor_bench_test.go per D-03"

patterns-established:
  - "Per-iteration stateful reset: b.StopTimer() → setup → b.StartTimer() → measure for stateful benchmarks"

requirements-completed: [MESH-04]

duration: 5min
completed: 2026-04-02
---

# Phase 02 Plan 02: FluidTick Benchmark Summary

**FluidTick benchmark added to internal/world/ measuring cascading 8-block water column simulation with 36 flow_dirs/op custom metric.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-04-02T00:00:00Z
- **Completed:** 2026-04-02T00:05:00Z
- **Tasks:** 1 completed
- **Files modified:** 1

## Accomplishments

- Created `internal/world/fluid_bench_test.go` with `BenchmarkFluidTick/cascading_column`
- Benchmark uses `world.NewEmpty()` + per-iteration world reset via `b.StopTimer`/`b.StartTimer`
- Reports `flow_dirs/op` (36 for 8-block column) and `allocs/op` for regression detection
- Uses `b.Loop()` idiom consistent with existing benchmarks in the package
- All existing benchmarks in `./internal/world/` continue to pass

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- `internal/world/fluid_bench_test.go` exists
- Commit 64bde98 confirmed in git log
- `go test -bench=BenchmarkFluidTick -benchtime=1x ./internal/world/` exits 0 with flow_dirs/op output
- No TestMain duplication
