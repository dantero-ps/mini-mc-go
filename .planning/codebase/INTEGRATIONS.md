# External Integrations

**Analysis Date:** 2026-02-08

## APIs & External Services

**None detected** - This is a standalone offline game with no external API dependencies.

## Data Storage

**Databases:**
- Not applicable - No persistent database layer detected
- World data is generated procedurally at runtime using Perlin noise
- No save/load functionality for world state

**File Storage:**
- **Local filesystem only** - Asset files loaded from disk:
  - Textures from `assets/textures/blocks/`
  - Models from `assets/models/block/` and `assets/models/item/`
  - Block states from `assets/blockstates/`
  - Shaders from `assets/shaders/`
  - Fonts from `assets/fonts/`

**Caching:**
- In-memory texture caching via OpenGL texture handles (stored as `uint32` IDs)
- Chunk mesh caching in GPU buffers
- Font atlas cached in `internal/graphics/renderables/font/font.go`

## Authentication & Identity

**Auth Provider:**
- Not applicable - Standalone single-player game with no authentication

## Monitoring & Observability

**Error Tracking:**
- Not detected - No error tracking service integration

**Logs:**
- Go standard `log` package used for error reporting
- Example: `log.Fatal()` and `panic()` for critical failures
- No structured logging framework detected
- Profiling support via `internal/profiling/profiling.go` package for frame-by-frame performance analysis

**Performance Metrics:**
- Built-in FPS limiter and profiling infrastructure at `internal/profiling/profiling.go`
- Runtime profiling metadata: frame time tracking via `profiling.ResetFrame()` and `profiling.Mark()` calls

## CI/CD & Deployment

**Hosting:**
- None - Standalone desktop application executable only

**Deployment:**
- Direct binary execution from compiled Go code
- No containerization or deployment pipeline detected

**Build Process:**
- Native Go build: `go build ./cmd/mini-mc` or `go run ./cmd/mini-mc`
- Cross-compilation capable but requires CGO setup for target platform
- Binary output: `mini-mc` (macOS/Linux) or `mini-mc.exe` (Windows)

## Environment Configuration

**Required env vars:**
- None detected - All configuration hardcoded in `internal/config/config.go`

**Runtime Configuration:**
- Render distance: 25 chunks (configurable via `config.SetRenderDistance()`, clamped 5-50)
- FPS limit: 180 FPS (configurable via `config.SetFPSLimit()`, range 0-240)
- Wireframe mode: toggleable via `config.ToggleWireframeMode()`
- View bobbing: toggleable via `config.ToggleViewBobbing()`

**Secrets location:**
- Not applicable - No secrets or credentials required

## Operating System Integration

**Window Management:**
- GLFW 3.3 provides cross-platform windowing (X11/Wayland on Linux, Cocoa on macOS, Win32 on Windows)
- Window callbacks handled through `internal/input/input.go` for keyboard and mouse events
- OS-specific resource requirements: see `internal/game/setup.go` for platform initialization

**Input Handling:**
- Keyboard input via GLFW callbacks bound in `internal/game/input_handlers.go`
- Mouse input via GLFW pointer position and button callbacks
- Raw input state managed by `internal/input/input.go`

## Asset Pipeline

**Asset Loading:**
- PNG textures loaded via `golang.org/x/image` and GPU uploaded in `internal/graphics/texture_util.go`
- JSON model files parsed in `pkg/blockmodel/loader.go`
- Shader compilation happens at `internal/graphics/shader.go` startup
- Font atlas built from OTF files in `internal/graphics/renderables/font/font.go`

**Asset Encoding:**
- PNG format for textures (8-bit RGBA)
- JSON format for models and block states
- OTF format for fonts (Minecraft font)
- GLSL for shader source code

## Webhooks & Callbacks

**Incoming:**
- None - Standalone application

**Outgoing:**
- None - No external communication layer

---

*Integration audit: 2026-02-08*
