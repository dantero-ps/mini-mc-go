# Codebase Structure

**Analysis Date:** 2026-02-08

## Directory Layout

```
gogl/
├── cmd/                                  # Executable entry points
│   ├── mini-mc/                         # Main game binary
│   │   └── main.go
│   └── triangle/                        # Test/example binary
│       └── main.go
├── internal/                            # Private packages (not importable externally)
│   ├── config/                          # Game configuration and flags
│   │   └── config.go
│   ├── entity/                          # Entity types (items, NPCs)
│   │   ├── entity.go
│   │   └── item_entity.go
│   ├── game/                            # Game loop and session management
│   │   ├── app.go
│   │   ├── session.go
│   │   ├── setup.go
│   │   ├── input_handlers.go
│   │   ├── fps_limiter.go
│   │   └── item_stacking_init.go
│   ├── graphics/                        # Rendering systems
│   │   ├── renderer/                    # Main render orchestrator
│   │   │   ├── renderer.go
│   │   │   └── api.go
│   │   ├── renderables/                 # Pluggable renderable features
│   │   │   ├── blocks/                  # Block rendering with frustum culling
│   │   │   │   ├── blocks.go
│   │   │   │   ├── meshing.go
│   │   │   │   ├── texture.go
│   │   │   │   ├── atlas.go
│   │   │   │   ├── frustum.go
│   │   │   │   └── types.go
│   │   │   ├── items/                   # Dropped item rendering
│   │   │   │   ├── items.go
│   │   │   │   └── mesh.go
│   │   │   ├── breaking/                # Block breaking animation
│   │   │   │   └── breaking.go
│   │   │   ├── hand/                    # First-person hand/arm
│   │   │   │   └── hand.go
│   │   │   ├── crosshair/               # Crosshair reticle
│   │   │   │   └── crosshair.go
│   │   │   ├── wireframe/               # Debug wireframe overlay
│   │   │   │   └── wireframe.go
│   │   │   ├── playermodel/             # Player model rendering
│   │   │   │   └── player_model.go
│   │   │   ├── font/                    # Font rendering system
│   │   │   │   └── font.go
│   │   │   ├── ui/                      # UI overlay rendering
│   │   │   │   └── ui.go
│   │   │   └── hud/                     # HUD components (hotbar, status, profiling)
│   │   │       ├── hud.go
│   │   │       ├── hotbar.go
│   │   │       ├── inventory_screen.go
│   │   │       ├── container_screen.go
│   │   │       ├── status_bars.go
│   │   │       ├── profiling_overlay.go
│   │   │       ├── screen.go
│   │   │       └── null_screen.go
│   │   ├── camera.go                    # Camera system and matrices
│   │   ├── shader.go                    # Shader compilation and management
│   │   └── texture_manager.go           # Texture loading and caching
│   ├── input/                           # Input handling and action mapping
│   │   └── input.go
│   ├── inventory/                       # Player inventory system
│   │   ├── inventory.go
│   │   ├── container.go
│   │   ├── player_container.go
│   │   └── slot.go
│   ├── item/                            # Item definitions
│   │   └── item.go
│   ├── meshing/                         # Voxel mesh generation
│   │   ├── greedy.go                    # Greedy meshing algorithm
│   │   ├── pool.go                      # Worker pool management
│   │   └── custom_model.go              # Custom model meshing
│   ├── physics/                         # Collision and raycast
│   │   ├── collision.go
│   │   └── raycast.go
│   ├── player/                          # Player state and behavior
│   │   ├── player.go                    # Player struct and main update
│   │   ├── state.go                     # Game mode and player state
│   │   ├── movement.go                  # Position/velocity updates
│   │   ├── camera.go                    # Camera matrix and FOV
│   │   ├── mining.go                    # Block breaking logic
│   │   ├── interaction.go               # Block placing and entity pickup
│   │   └── movement.go
│   ├── profiling/                       # Performance profiling system
│   │   └── profiling.go
│   ├── registry/                        # Block and item definitions registry
│   │   └── blocks.go
│   ├── ui/                              # Menu UI system
│   │   ├── menu/                        # Menu screens
│   │   │   ├── main_menu.go
│   │   │   ├── pause_menu.go
│   │   │   └── types.go
│   │   └── widget/                      # UI widget components
│   │       ├── button.go
│   │       ├── toggle.go
│   │       ├── slider.go
│   │       └── component.go
│   └── world/                           # World and chunk management
│       ├── world.go                     # World composition
│       ├── chunk.go                     # Chunk structure (16x256x16)
│       ├── chunk_store.go               # Chunk storage and lookup
│       ├── chunk_streamer.go            # Async chunk generation and loading
│       ├── entity_manager.go            # Entity storage and queries
│       ├── generator.go                 # Terrain generation
│       ├── noise.go                     # Simplex noise
│       ├── block.go                     # Block type definitions
│       └── generator_test.go
├── pkg/                                 # Public packages (external import OK)
│   └── blockmodel/                      # Block model JSON/texture loading
│       └── (various model loading files)
├── assets/                              # Game assets
│   ├── models/                          # Block and item models
│   │   ├── block/
│   │   └── item/
│   ├── textures/                        # Block, entity, GUI textures
│   │   ├── blocks/
│   │   ├── entity/
│   │   └── gui/
│   ├── shaders/                         # OpenGL shaders
│   │   ├── blocks/
│   │   ├── breaking/
│   │   ├── crosshair/
│   │   ├── hand/
│   │   ├── hud/
│   │   ├── item/
│   │   ├── ui/
│   │   └── wireframe/
│   ├── blockstates/                     # Block state JSON definitions
│   └── fonts/                           # Font files
├── assets-test/                         # Test assets
│   └── models/
│       └── block/
├── .planning/                           # GSD planning documents
│   └── codebase/
├── go.mod                               # Go module definition
├── go.sum                               # Go dependency checksums
├── README.md                            # Project overview
├── BCE.md                               # Block coordinate explanation
└── mini-mc                              # Compiled binary
```

## Directory Purposes

**cmd/mini-mc:**
- Purpose: Main game executable entry point
- Contains: `main.go` with GLFW initialization, input setup, and app lifecycle
- Key files: `cmd/mini-mc/main.go`

**cmd/triangle:**
- Purpose: Example/test application (simple triangle rendering)
- Contains: Minimal OpenGL example
- Key files: `cmd/triangle/main.go`

**internal/config:**
- Purpose: Centralized configuration and game flags
- Contains: Wireframe mode, chunk load/evict radius, FPS limits, cursor settings
- Key files: `internal/config/config.go`

**internal/game:**
- Purpose: Game lifecycle and main loop orchestration
- Contains: App state machine, Session initialization, input handler routing, FPS limiting
- Key files: `internal/game/app.go`, `internal/game/session.go`

**internal/graphics:**
- Purpose: All rendering and visual systems
- Contains: Renderer orchestrator, shader management, texture handling, camera system
- Key files: `internal/graphics/renderer/renderer.go`

**internal/graphics/renderables:**
- Purpose: Pluggable visual feature modules
- Contains: Blocks, items, hand, HUD, menus, debug overlays
- Key files: One per renderable type (e.g., `blocks.go`, `hud.go`)

**internal/input:**
- Purpose: Input abstraction and action mapping
- Contains: Keyboard/mouse to logical action mapping, frame state tracking
- Key files: `internal/input/input.go`

**internal/inventory:**
- Purpose: Inventory management and item containers
- Contains: Slot-based inventory, stack management, container abstractions
- Key files: `internal/inventory/inventory.go`

**internal/item:**
- Purpose: Item definitions and metadata
- Contains: Item type definitions, stack limits, durability
- Key files: `internal/item/item.go`

**internal/meshing:**
- Purpose: Voxel-to-mesh conversion
- Contains: Greedy meshing algorithm, async worker pool for CPU-intensive meshing
- Key files: `internal/meshing/greedy.go`, `internal/meshing/pool.go`

**internal/physics:**
- Purpose: Collision detection and raycasting
- Contains: AABB collision queries, block intersection tests, ground level detection, block selection raycasts
- Key files: `internal/physics/collision.go`, `internal/physics/raycast.go`

**internal/player:**
- Purpose: Player state and behavior
- Contains: Position/velocity, movement physics, mining state, inventory, camera matrices, animations
- Key files: `internal/player/player.go` (main state struct and update)

**internal/profiling:**
- Purpose: Performance monitoring and frame profiling
- Contains: Named task tracking, frame snapshot collection, HUD profiling display
- Key files: `internal/profiling/profiling.go`

**internal/registry:**
- Purpose: Centralized definitions for blocks and items
- Contains: Block hardness, textures, solid/transparent flags, tint colors, model references
- Key files: `internal/registry/blocks.go`

**internal/ui:**
- Purpose: Menu system and HUD widgets
- Contains: Main menu, pause menu, button/toggle/slider widgets, container screen
- Key files: `internal/ui/menu/main_menu.go`, `internal/ui/menu/pause_menu.go`

**internal/world:**
- Purpose: World state, chunks, and terrain
- Contains: Chunk storage (16x256x16), entity manager, terrain generator, chunk streaming
- Key files: `internal/world/world.go`, `internal/world/chunk.go`

**pkg/blockmodel:**
- Purpose: Block model and asset loading
- Contains: JSON parsing for block states, texture atlas management, model element definitions
- Key files: Various in `pkg/blockmodel/`

**assets:**
- Purpose: Game content (textures, models, shaders)
- Contains: Block/item models, textures, OpenGL shaders, font files, block state definitions
- Generated: No (hand-crafted or extracted from Minecraft assets)
- Committed: Yes (required for gameplay)

## Key File Locations

**Entry Points:**
- `cmd/mini-mc/main.go` - Program entry, GLFW init, window setup, main loop invocation

**Configuration:**
- `internal/config/config.go` - Game flags and tunable parameters

**Core Logic:**
- `internal/game/app.go` - Application lifecycle and state machine
- `internal/game/session.go` - Game session (when playing)
- `internal/player/player.go` - Player state struct and main update loop
- `internal/world/world.go` - World composition and public API

**Rendering:**
- `internal/graphics/renderer/renderer.go` - Main render orchestrator
- `internal/graphics/renderables/blocks/blocks.go` - Block rendering and frustum culling
- `internal/graphics/renderables/hud/hud.go` - HUD rendering and inventory UI

**Physics:**
- `internal/physics/collision.go` - AABB collision and ground detection
- `internal/physics/raycast.go` - Block selection raycast

**Meshing:**
- `internal/meshing/greedy.go` - Greedy meshing algorithm
- `internal/meshing/pool.go` - Async worker pool management

**Data:**
- `internal/world/chunk.go` - Chunk structure (stores block data)
- `internal/world/chunk_store.go` - Chunk lookup and caching
- `internal/inventory/inventory.go` - Inventory state

## Naming Conventions

**Files:**
- Lowercase with underscores: `chunk_store.go`, `entity_manager.go`
- Descriptive names matching primary type or function: `player.go` (Player struct), `renderer.go` (Renderer struct)
- Test files use `_test.go` suffix: `generator_test.go`

**Directories:**
- Lowercase, plural when containing multiple implementations: `renderables/`, `shaders/`, `textures/`
- Singular for functional packages: `player/`, `world/`, `input/`
- Hierarchical by feature: `graphics/renderables/blocks/` for block-specific rendering

**Packages:**
- Match directory name (Go convention)
- Descriptive nouns: `player`, `world`, `physics`, `meshing`
- Avoid abbreviations except for common ones: `hud` (heads-up display)

**Functions/Methods:**
- CamelCase: `GetBlock()`, `UpdateEntities()`, `IsAir()`
- Receiver type not repeated in name: `(c *Chunk) GetBlock()` not `(c *Chunk) GetChunkBlock()`
- Boolean queries start with "Is" or "Has": `IsAir()`, `HasHoveredBlock`

**Constants/Types:**
- All-caps for constants: `ChunkSizeX`, `ChunkSizeY`, `SectionHeight`
- CamelCase for types: `Chunk`, `BlockType`, `Action`
- Enum constants use type-prefix convention: `BlockTypeAir`, `BlockTypeGrass` (in registry)

**Variables:**
- Lowercase camelCase: `playerPos`, `renderContext`, `meshResults`
- Short names acceptable in tight scopes: `x`, `y`, `z` for coordinates; `dt` for delta-time
- Receiver variable single letter: `(p *Player)`, `(w *World)`, `(c *Chunk)`

## Where to Add New Code

**New Feature (e.g., new block, item):**
- Primary code: `internal/registry/blocks.go` (block definition) or `internal/item/item.go` (item definition)
- Tests: `internal/world/generator_test.go` if testing terrain generation
- Assets: `assets/textures/blocks/` (textures), `assets/models/block/` (models)

**New Renderable (e.g., new visual effect):**
- Implementation: `internal/graphics/renderables/{feature_name}/{feature}.go`
- Register with: `internal/game/session.go:NewSession()` when creating Renderer
- Shader: `assets/shaders/{feature_name}/vertex.glsl`, `assets/shaders/{feature_name}/fragment.glsl`
- Conform to: `Renderable` interface (`Init() error`, `Render(RenderContext)`)

**New Physics System (e.g., gravity overhaul):**
- Implementation: `internal/physics/{system}.go` for public functions; call from `internal/player/movement.go`
- Tests: Standalone `*_test.go` in `internal/physics/`

**New Player Mechanic (e.g., enchantments, status effects):**
- State: Add field to `Player` struct in `internal/player/state.go`
- Update logic: `internal/player/player.go` or dedicated file `internal/player/{mechanic}.go`
- Integration: Wire into `Session.Update()` or `Player.Update()` if affecting main loop

**New Menu Screen:**
- Widget structure: `internal/ui/widget/` for reusable components
- Screen: `internal/ui/menu/{screen_name}.go` implementing menu interface
- Register with: `internal/game/app.go` (App state machine)

**Utilities/Helpers:**
- Shared helpers: Create in relevant domain package (e.g., `internal/graphics/util.go`) or new `internal/common/` if cross-cutting

**Tests:**
- Unit tests: Co-located with source in same package, e.g., `internal/world/generator_test.go`
- No separate test directory; `*_test.go` files are excluded from distribution builds

## Special Directories

**assets:**
- Purpose: Game content (textures, models, shaders, fonts)
- Generated: No
- Committed: Yes (required for runtime)
- Notes: Structured by asset type (textures/blocks, models/item, shaders/hud)

**assets-test:**
- Purpose: Test fixtures (minimal block models for unit tests)
- Generated: No
- Committed: Yes
- Notes: Used by meshing tests to verify algorithm without full asset pipeline

**.planning/codebase:**
- Purpose: GSD codebase analysis documents
- Generated: Yes (by GSD mapper)
- Committed: Yes
- Notes: ARCHITECTURE.md, STRUCTURE.md, CONVENTIONS.md, TESTING.md, CONCERNS.md

**internal:**
- Purpose: Private packages (import restricted to this module)
- Generated: No
- Committed: Yes
- Notes: Go convention; prevents external consumers from depending on internal APIs

---

*Structure analysis: 2026-02-08*
