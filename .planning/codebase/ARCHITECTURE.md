# Architecture

**Analysis Date:** 2026-02-08

## Pattern Overview

**Overall:** Layered Model-View-Controller (MVC) pattern with a clear separation between game logic, rendering, and input handling. The codebase follows Go idioms with package-based organization and composition over inheritance.

**Key Characteristics:**
- Stateful game loop driven by a fixed-timestep update/render cycle
- Composable renderer using a strategy pattern with pluggable "renderable" components
- World-centric data model where chunks manage block data and entities
- Async mesh generation using worker pools for performance
- Entity-component style interaction between systems

## Layers

**Presentation/Rendering Layer:**
- Purpose: Handles all OpenGL rendering, shader management, and visual output
- Location: `internal/graphics/` (with `renderables/` subdirectory for feature renderers)
- Contains: Shader wrapper, texture management, camera system, and pluggable renderable strategies
- Depends on: World, Player, Config, Profiling
- Used by: Session, App (via Renderer)

**Game Logic Layer:**
- Purpose: Controls game state, lifecycle, input handling, and session management
- Location: `internal/game/`
- Contains: App (lifecycle coordinator), Session (game session manager), input handlers, FPS limiting
- Depends on: Player, World, Input, Graphics, UI
- Used by: Main entry point

**Entity/World Layer:**
- Purpose: Manages world state, chunks, entity storage, terrain generation, and world streaming
- Location: `internal/world/`
- Contains: Chunk storage, entity manager, chunk streamer, noise-based terrain generator, block definitions
- Depends on: Physics (for collision checks), Registry (for block data)
- Used by: Session, Graphics, Player, Physics

**Physics Layer:**
- Purpose: Provides collision detection, AABB intersection tests, and ground level discovery
- Location: `internal/physics/`
- Contains: Collision queries, raycast algorithms, block intersection logic
- Depends on: World (for block queries)
- Used by: Player, Entity system

**Player System:**
- Purpose: Manages player state, movement, mining, inventory, and camera
- Location: `internal/player/`
- Contains: Player struct with position/velocity, movement physics, mining state, inventory management, camera matrix calculations, head bobbing
- Depends on: World, Physics, Input, Inventory, Profiling
- Used by: Session, Graphics renderables, HUD

**Input System:**
- Purpose: Abstracts physical input (keyboard/mouse) into logical game actions
- Location: `internal/input/`
- Contains: InputManager with key/mouse button to action mapping, frame state tracking, edge detection (just pressed)
- Depends on: GLFW
- Used by: Session, Player, Menu systems

**Inventory/Item System:**
- Purpose: Manages player inventory slots, item stacks, and item metadata
- Location: `internal/inventory/`, `internal/item/`
- Contains: Item definitions, container abstractions, slot management, inventory UI state
- Depends on: Registry (for item definitions)
- Used by: Player, HUD renderer, Item entities

**Meshing Layer:**
- Purpose: Converts voxel chunk data into GPU-compatible mesh data using greedy meshing
- Location: `internal/meshing/`
- Contains: Greedy meshing algorithm, direction worker pool, buffer pooling for GC efficiency
- Depends on: World, Registry
- Used by: Graphics renderables (blocks)

**UI Layer:**
- Purpose: Manages UI rendering, menus, and HUD display
- Location: `internal/ui/`, `internal/graphics/renderables/ui/`, `internal/graphics/renderables/hud/`
- Contains: Menu widgets (buttons, toggles, sliders), pause menu, main menu, HUD components (hotbar, inventory screen, status bars)
- Depends on: Graphics, Font rendering, Input
- Used by: App, Session

**Configuration/Registry Layer:**
- Purpose: Centralized block and item definitions, configurable parameters, rendering options
- Location: `internal/config/`, `internal/registry/`, `pkg/blockmodel/`
- Contains: Block type definitions, texture mappings, game configuration flags (wireframe mode, chunk load radius, etc.)
- Depends on: None (leaf dependencies)
- Used by: All systems that need block/item metadata or configuration

## Data Flow

**Main Game Loop:**

1. `App.Run()` executes main tick loop
2. `App.tick()` handles state management:
   - If `StateMainMenu`: update/render main menu
   - If `StatePlaying`: delegate to `Session.Update()` and `Session.Render()`
3. `glfw.PollEvents()` processes OS-level input
4. `InputManager.PostUpdate()` clears edge-detection flags
5. `App.fpsLimiter.Wait()` throttles to target FPS

**Session Game Update (StatePlaying):**

1. Player input processed via `InputManager` (action state retrieved)
2. `Player.Update(dt, inputManager)`:
   - Update camera from mouse movement
   - Update hovered block via raycast
   - Check entity collisions
   - Update position based on movement input and physics
   - Update mining state if applicable
   - Update head bobbing and animation
3. `World.UpdateEntities(dt)`:
   - Updates all dropped item entities
   - Removes dead entities
4. Input action handlers:
   - Hotbar selection, inventory toggle, block breaking/placing
   - Pause menu, wireframe toggle, profiling toggle
5. World update processing:
   - Async chunk streaming around player (background)
   - Mesh result processing from worker threads
   - Periodic chunk eviction (every 1 second)

**Rendering Pipeline:**

1. `Session.Render(dt)` calls `Renderer.Render(world, player, dt)`
2. `Renderer.Render()`:
   - Clears framebuffer
   - Updates FOV based on sprint state
   - Builds view/projection matrices from player camera
   - Creates `RenderContext` with camera, world, player, matrices
   - Calls `Render(ctx)` on each renderable in sequence:
     - Blocks (world geometry)
     - Items (dropped items in world)
     - Breaking blocks (mining animation)
     - Wireframe (debug visualization)
     - Crosshair (center reticle)
     - Hand (first-person arm)
     - UI (inventory screen overlay)
     - HUD (hotbar, status bars, profiling overlay)
3. Window buffer swap

**State Management:**

- `App.state` tracks StateMainMenu vs StatePlaying
- `Session.Paused` controls pause menu vs gameplay
- `Player.IsInventoryOpen` controls inventory screen vs HUD hotbar
- Inventory state changes trigger `Player.OnInventoryStateChange` callback to update HUD

## Key Abstractions

**Renderable Interface:**
- Purpose: Pluggable rendering strategy allowing composition of visual features
- Examples: `internal/graphics/renderables/blocks/blocks.go`, `internal/graphics/renderables/hand/hand.go`, `internal/graphics/renderables/hud/hud.go`
- Pattern: Each renderable implements `Init()` and `Render(RenderContext)`, registered with `Renderer` on creation

**Ticker Interface (world.go):**
- Purpose: Abstracts entity update behavior to avoid circular imports between world and entity packages
- Examples: Used by `internal/entity/item_entity.go` and entity manager
- Pattern: Entities implement `Update(dt)`, `IsDead()`, `SetDead()`, `Position()` methods

**Action Enum (input.go):**
- Purpose: Maps physical input (keys, mouse) to logical game actions
- Examples: `ActionMoveForward`, `ActionMouseLeft`, `ActionInventory`
- Pattern: `InputManager.BindKey()` maps GLFW keys to actions; actions queried with `JustPressed()` and `IsActive()`

**ChunkCoord (world.go):**
- Purpose: Uniquely identifies chunks by 3D coordinates (X, Y, Z)
- Examples: Used in chunk storage, meshing pool, nearby entity queries
- Pattern: Chunk coordinates map world positions to chunk indices via integer division

**BlockType (world/block.go):**
- Purpose: Enumerated block types with associated metadata from registry
- Examples: Air, Grass, Stone, etc.
- Pattern: Resolved to `BlockDefinition` from registry for rendering and interaction

## Entry Points

**Main Entry Point:**
- Location: `cmd/mini-mc/main.go`
- Triggers: Program execution
- Responsibilities:
  - Lock OS thread (required for GL)
  - Initialize GLFW
  - Create window and input manager
  - Instantiate `App` and run main loop

**Session Entry Point:**
- Location: `internal/game/session.go:NewSession()`
- Triggers: Menu action to start survival/creative mode
- Responsibilities:
  - Initialize all renderable components
  - Create world with terrain generation
  - Spawn player at world surface
  - Configure HUD and input handling
  - Initialize mesh system with worker pool

**Update Entry Points:**
- `Session.Update()` - Per-frame game logic
- `Player.Update()` - Player state and input processing
- `World.UpdateEntities()` - Entity updates
- `blocks.ProcessMeshResults()` - Consume async mesh jobs

**Render Entry Points:**
- `Session.Render()` - Orchestrates frame render
- `Renderer.Render()` - Renders all features
- Each `Renderable.Render(ctx)` - Feature-specific rendering

## Error Handling

**Strategy:** Panic on initialization errors (setup phase), graceful degradation for runtime errors.

**Patterns:**
- Setup errors propagate as panics: `if err != nil { panic(err) }` in `main()`, `NewApp()`, `NewSession()`
- Profiling tracking uses deferred cleanup: `defer profiling.Track("task.name")()`
- Async operations (chunk generation, meshing) use channels; errors logged, not propagated

## Cross-Cutting Concerns

**Logging:** No structured logging framework; uses standard `log` package for slow frames only. Frame processing time > 16ms triggers log output with profiling snapshot.

**Validation:** Minimal runtime validation; relies on Go type system. Block/chunk coordinates validated on access (range checks in `Chunk.GetBlock()`). Physics collision checks validate AABB bounds before block queries.

**Authentication:** Not applicable (single-player game).

**Configuration:** Centralized in `internal/config/config.go` with flags for wireframe mode, chunk load radius, chunk evict radius, FPS limits.

**Profiling:** Custom profiling system in `internal/profiling/profiling.go` tracks named tasks per frame, displayed in HUD overlay with top 5 slowest tasks logged for slow frames.

**Asset Loading:** Texture atlasing done at startup; meshes generated on-demand; font atlas built once per menu/session. Block models loaded from asset files (JSON/PNG).

---

*Architecture analysis: 2026-02-08*
