---
phase: quick
plan: 2
type: execute
wave: 1
depends_on: []
files_modified: [internal/physics/raycast.go]
autonomous: true
must_haves:
  truths:
    - "Raycast uses voxel traversal (DDA) instead of naive stepping"
    - "Performance overhead reduced by ~10x (O(N) blocks vs O(M) steps)"
  artifacts:
    - path: "internal/physics/raycast.go"
      contains: "tMaxX"
  key_links: []
---

<objective>
Fix physics raycast performance regression by replacing naive stepping with efficient voxel traversal.

Purpose: Eliminate redundant world lookups in the hot path (Raycast is called every frame).
Output: Optimized internal/physics/raycast.go
</objective>

<execution_context>
@/Users/furkandogan/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/furkandogan/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/physics/raycast.go
@internal/world/world.go
</context>

<tasks>

<task type="auto">
  <name>Optimize Raycast with DDA</name>
  <files>internal/physics/raycast.go</files>
  <action>
    Reimplement `Raycast` using the Amanatides & Woo voxel traversal algorithm.
    
    Implementation details:
    - Remove `stepSize` loop.
    - Setup DDA state: `stepX/Y/Z`, `tMaxX/Y/Z`, `tDeltaX/Y/Z`.
    - Handle `direction` components being 0 (avoid div by zero or infinite tDelta).
    - Loop until `totalDistance > maxDist`.
    - Inside loop:
      - Determine which axis to step (min tMax).
      - Update current block coord.
      - Check `world.IsAir`.
      - If hit:
        - Calculate `AdjacentPosition` based on the face entered (or use `lastBlockPos`).
        - Ensure `dist >= minDist` check is preserved.
        - Return `Hit: true`.
    - Maintain `lastBlockPos` for adjacency if preferred, or derive from face.
    - Preserve `defer profiling.Track`.
  </action>
  <verify>
    go build ./internal/physics/...
  </verify>
  <done>
    Raycast implementation uses DDA loop (no fixed step size).
  </done>
</task>

</tasks>

<verification>
Code compiles and algorithm is correct.
</verification>

<success_criteria>
Raycast function uses efficient voxel traversal.
</success_criteria>

<output>
After completion, create .planning/quick/2-fix-physics-raycast-performance-regressi/2-SUMMARY.md
</output>
