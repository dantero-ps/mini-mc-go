# Coding Conventions

**Analysis Date:** 2026-02-08

## Naming Patterns

**Files:**
- Package files use lowercase with underscores: `entity_manager.go`, `chunk_streamer.go`
- Test files use suffix pattern: `generator_test.go`, `loader_test.go`
- Logical grouping by domain: entity, world, input, game, graphics, player, etc.

**Functions:**
- Exported (public) functions start with uppercase: `NewGenerator()`, `GetBlock()`, `Update()`, `IsAir()`
- Unexported (private) functions start with lowercase: `indexInSection()`, `hash2()`, `fade()`
- Constructor functions use `New` prefix: `NewChunk()`, `NewWorld()`, `NewItemEntity()`
- Getter/setter functions use `Get`/`Set` prefix: `GetRenderDistance()`, `SetFPSLimit()`, `GetBlock()`
- Boolean check functions use `Is` prefix: `IsAir()`, `IsDead()`, `IsActive()`
- Update/tick functions use `Update()` name: `Update(dt float64)`, `UpdatePosition()`

**Variables:**
- CamelCase for all variable names: `chunkX`, `playerPos`, `maxY`, `renderDistance`
- Constants use UPPER_SNAKE_CASE: `ChunkSizeX`, `InfinitePickupDelay`, `NoDespawnAge`, `StackSearchInterval`
- Package-level variables prefix with lowercase: `globalRenderSettings`, `ItemEntityConfigurator`
- Receiver variables typically single or two letters: `c *Chunk`, `p *Player`, `im *InputManager`, `w *World`

**Types:**
- Exported struct/interface types start with uppercase: `Chunk`, `Entity`, `World`, `ItemEntity`, `InputManager`
- Interface types typically use verb-noun pattern: `TerrainGenerator`, `EntityManager`, `Ticker` (singular form)
- Exported interfaces define contracts: `Entity`, `WorldSource`, `Ticker`, `TerrainGenerator`

## Code Style

**Formatting:**
- Standard Go formatting with `gofmt` (implicit, not explicitly configured in codebase)
- Brace placement: Opening brace on same line as declaration (Go standard)
- Indentation: Tabs (Go standard)
- Line length: Generally under 120 characters, no hard limit enforced

**Linting:**
- Not detected (no `.golangci.yml`, `revive.toml`, or linter config files present)
- Code appears to follow Go conventions organically

**Comments:**
- Inline comments for non-obvious logic: `// Double-check locking: another goroutine might have...`
- Block comments above exported functions/types: `// NewGenerator creates a new generator...`
- Comments explain "why", not "what": Implementation is self-documenting

**Documentation:**
- Exported functions have comment blocks starting with function name
- Example from `generator.go`: `// NewGenerator creates a new generator with default settings.`
- Comments on constants explain their purpose: `// InfinitePickupDelay = -1.0  // Equivalent to MC's 32767 (never pick up)`

## Import Organization

**Order:**
1. Standard library imports first: `import "log"`, `import "math"`, `import "sync"`
2. Third-party/external imports next: `github.com/go-gl/...`, `github.com/go-gl/mathgl/mgl32`
3. Local project imports last: `mini-mc/internal/...`, `mini-mc/pkg/...`

**Path Aliases:**
- No aliases used in standard practice
- When conflicts exist (e.g., `input` package collides with `standardInput` import), use alias:
  - Example from `app.go`: `standardInput "mini-mc/internal/input"` to disambiguate local package

**Package imports typically grouped:**
```go
import (
	"package"
	"other/package"

	"github.com/external"

	"mini-mc/internal/package"
)
```

## Error Handling

**Patterns:**
- Explicit error checking: Always capture `err` and check immediately
- `panic()` for initialization failures (non-recoverable): `panic(err)` in constructors
- Return errors for recoverable situations: Functions return `error` as last return value
- Error propagation: Log errors before returning them where context matters
- Named return values rarely used; errors always last return parameter

**Examples from codebase:**
- `loader.go`: Returns `(*Model, error)` from `LoadModel()`
- `app.go`: Panics on critical setup failures like font loading
- `chunk_store.go`: Uses double-check locking pattern to prevent race conditions

## Logging

**Framework:**
- Standard `log` package (Go stdlib)
- Some places use `fmt` for formatting

**Patterns:**
- Frame performance warnings: `log.Printf("Slow frame: %v. Top tasks: %s", processingDuration, ...)`
- Debugging: Console output or log output as needed
- No structured logging framework (Zap, Slog) detected

## Concurrency Patterns

**Synchronization:**
- `sync.RWMutex` for read-heavy operations: `ChunkStore`, `RenderSettings`, `InputManager`
- Double-check locking in `ChunkStore.GetChunk()` to minimize lock contention
- Goroutines spawned for background chunk generation in `ChunkStreamer`

**Example from `chunk_store.go`:**
```go
cs.mu.RLock()
chunk, exists := cs.chunks[coord]
cs.mu.RUnlock()
if !exists && create {
	cs.mu.Lock()
	// Double-check locking
	if existing, ok := cs.chunks[coord]; ok {
		cs.mu.Unlock()
		return existing
	}
	// ... create chunk
	cs.mu.Unlock()
}
```

## Function Design

**Size:**
- Functions typically 20-50 lines
- Larger functions (100+ lines) only when combining multiple concerns (e.g., `Update()` in `ItemEntity`)
- No visible attempts to break down overly long functions

**Parameters:**
- Functions prefer simple parameter lists (1-4 params typical)
- Multiple related parameters passed as struct when > 3: `ItemEntity.Pos`, `ItemEntity.Vel`, `ItemEntity.World`
- Context rarely used (game is single-threaded gameplay loop)

**Return Values:**
- Single return value typical for getters: `GetBlock() BlockType`
- Error-returning functions use `(value, error)` pattern
- Multiple return values (2-3) for complex operations: `Position() mgl32.Vec3`, `GetBounds() (width, height float32)`

## Module Design

**Exports:**
- Minimal public API philosophy: Only exported symbols needed externally
- Unexported helper functions within packages: `fade()`, `lerp()` in `noise.go`

**Package Organization:**
- Domain-driven: `world/`, `entity/`, `player/`, `game/`, `graphics/`, `input/`
- Clear separation of concerns
- `internal/` enforces encapsulation (prevents external imports)
- `pkg/` for shareable libraries (e.g., `blockmodel`)

**Barrel Files:**
- Not used in this codebase (no `index.go` or re-export patterns)

## Interface Design

**Pattern:**
- Interfaces defined at point of use, not centralized
- Small, focused interfaces: `Entity`, `TerrainGenerator`, `Ticker`
- Example from `entity.go`: `Entity` interface defines minimal contract (Update, Position, IsDead, SetDead, GetBounds)

**Type Assertions:**
- Used to avoid circular imports: `GetNearbyEntities()` returns `[]interface{}` cast to specific types by caller
- Example from `world.go`: `ItemEntityConfigurator func(item Ticker, world interface{})` uses function pointers to break cycles

## Constants vs Magic Numbers

**Constants defined for:**
- Chunk dimensions: `ChunkSizeX`, `ChunkSizeY`, `ChunkSizeZ` (16, 256, 16)
- Physics values: `ItemEntityWidth`, `ItemEntityHeight`, `TickDuration`
- Timing intervals: `StackSearchInterval`, `InfinitePickupDelay`
- Block types: `BlockTypeBedrock`, `BlockTypeDirt`, `BlockTypeGrass`, `BlockTypeAir`

**Magic numbers in code:**
- Grid coordinates and offsets often use raw numbers in loops
- Interpolation values (0.0-1.0) appear inline
- Small hardcoded values for UI positioning

---

*Convention analysis: 2026-02-08*
