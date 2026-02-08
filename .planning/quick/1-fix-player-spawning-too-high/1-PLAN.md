---
phase: 01-3d-terrain-foundation
plan: 01-fix-player-spawning-too-high-1
type: execute
wave: 1
depends_on: []
files_modified: [internal/game/session.go]
autonomous: true
must_haves:
  truths:
    - "Player spawns on the actual ground surface, not high in the air"
  artifacts:
    - path: "internal/game/session.go"
      contains: "physics.FindGroundLevel"
  key_links:
    - from: "internal/game/session.go"
      to: "internal/physics/collision.go"
      via: "FindGroundLevel call"
---

<objective>
Fix the issue where the player spawns too high by implementing a downward scan for the actual ground level at the spawn position.

Purpose: Improve user experience by preventing the long fall at the start of the game.
Output: Updated session.go with precise spawn logic.
</objective>

<execution_context>
@/Users/furkandogan/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/furkandogan/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@internal/game/session.go
@internal/physics/collision.go
@internal/world/world.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Implement precise spawn height calculation</name>
  <files>internal/game/session.go</files>
  <action>
    Update `internal/game/session.go`:
    1. Import `mini-mc/internal/physics` and `github.com/go-gl/mathgl/mgl32`.
    2. In `NewSession`, before setting player position:
       - Call `gameWorld.StreamChunksAroundSync(0, 0, 2)` to ensure spawn chunks are generated.
       - Calculate `approxY` using `SurfaceHeightAt` (theoretical max).
       - Create a search start position `mgl32.Vec3{0, float32(approxY) + 5, 0}`.
       - Call `physics.FindGroundLevel` to find the exact surface height.
       - If `groundY > -1000` (valid ground found), use it. Otherwise fallback to `approxY`.
       - Set `gamePlayer.Position[1]` to `groundY` (ensure feet are ON ground).
  </action>
  <verify>
    Check that `session.go` calls `StreamChunksAroundSync` and `FindGroundLevel`.
    Ensure imports are correct.
  </verify>
  <done>
    Spawn logic scans for actual ground level instead of using theoretical max.
  </done>
</task>

</tasks>

<verification>
- [ ] Code compiles (no missing imports)
- [ ] Logic handles "no ground found" case (though unlikely at 0,0)
</verification>

<success_criteria>
- Player spawn Y position matches actual terrain height
</success_criteria>

<output>
After completion, create `.planning/phases/01-3d-terrain-foundation/01-fix-player-spawning-too-high-1-SUMMARY.md`
</output>
