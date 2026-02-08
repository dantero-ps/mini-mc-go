# Technology Stack

**Analysis Date:** 2026-02-08

## Languages

**Primary:**
- Go 1.24 - Core application language for graphics engine, game logic, and rendering pipeline

**Secondary:**
- GLSL - Fragment and vertex shaders for rendering (`.frag` and `.vert` files)
- JSON - Asset configuration for block models, block states, and item models

## Runtime

**Environment:**
- Go 1.24+ runtime with CGO enabled (required for OpenGL and GLFW bindings)

**Package Manager:**
- Go modules (go mod)
- Lockfile: `go.sum` (present)

## Frameworks

**Core Graphics:**
- github.com/go-gl/gl v0.0.0-20231021071112-07e5d0ea2e71 - OpenGL 4.1 Core bindings
- github.com/go-gl/glfw/v3.3/glfw v0.0.0-20250301202403-da16c1255728 - Window management and input handling

**Math & Graphics:**
- github.com/go-gl/mathgl v1.2.0 - 3D math library for matrix operations and vector calculations

**Image Processing:**
- golang.org/x/image v0.19.0 - Image decoding and font rendering
- golang.org/x/image/font/opentype - OpenType font support for UI text rendering
- golang.org/x/text v0.17.0 (indirect) - Text handling utilities

## Key Dependencies

**Critical:**
- `github.com/go-gl/glfw/v3.3/glfw` - Window lifecycle, event loop, input polling, and screen management
- `github.com/go-gl/gl/v4.1-core/gl` - GPU programming interface, shader compilation, buffer management
- `github.com/go-gl/mathgl/mgl32` - Matrix multiplication for camera transforms, projection calculations

**Graphics:**
- `golang.org/x/image/font` - Font parsing and rasterization for HUD text rendering at `assets/fonts/Minecraft.otf` and `assets/fonts/Minecraft-Bold.otf`
- `golang.org/x/image/font/opentype` - OpenType font format support

## Configuration

**Environment:**
- Configuration is hardcoded in `internal/config/config.go` using thread-safe global settings
- Key settings: render distance (default 25 chunks), FPS limit (default 180), wireframe mode, view bobbing

**Build:**
- No build configuration files (Makefile, gradle, cargo) - uses `go run ./cmd/mini-mc` and `go build`
- Requires C compiler for CGO compilation (GCC, Clang, or Mingw-w64)

## Assets

**Textures:**
- PNG block textures located at `assets/textures/blocks/` (example: `dirt.png`, `grass_top.png`, `destroy_stage_*.png`)

**Models:**
- JSON block model definitions at `assets/models/block/` (example: `cube.json`, `cube_all.json`)
- JSON item model definitions at `assets/models/item/`
- JSON block state definitions at `assets/blockstates/`

**Shaders:**
- Fragment and vertex shaders located at `assets/shaders/` in subdirectories:
  - `blocks/` - Main terrain rendering
  - `breaking/` - Block break animation
  - `crosshair/` - Crosshair rendering
  - `direction/` - Direction visualization
  - `hand/` - Player hand model
  - `hud/` - HUD and font rendering
  - `item/` - Item rendering
  - `ui/` - UI elements
  - `wireframe/` - Wireframe debug mode

**Fonts:**
- `assets/fonts/Minecraft.otf` - Primary Minecraft font
- `assets/fonts/Minecraft-Bold.otf` - Bold variant

## Platform Requirements

**Development:**
- Go 1.24+ installed and in PATH
- C compiler required for CGO:
  - macOS: Xcode Command Line Tools (`xcode-select --install`)
  - Windows: Mingw-w64 or MSVC
  - Linux (Ubuntu/Debian): `libgl1-mesa-dev xorg-dev`
- OpenGL drivers with support for OpenGL 4.1 Core Profile

**Production:**
- Executable compiled from Go source: `mini-mc` (macOS/Linux) or `mini-mc.exe` (Windows)
- OpenGL 4.1 capable GPU with up-to-date drivers
- GLFW 3.3+ compatible windowing system (X11 on Linux, Cocoa on macOS, Win32 on Windows)

## Rendering Architecture

**Graphics Context:**
- OpenGL 4.1 Core Profile
- Depth testing enabled by default
- Back-face culling enabled (CCW front face)
- Dynamic viewport management for window resizing

**Shader Pipeline:**
- Vertex and fragment shader pairs for each rendering pass (blocks, UI, HUD, etc.)
- Compiled at runtime via `internal/graphics/shader.go`
- No shader caching detected - shaders compiled on each application startup

**Texture Management:**
- PNG textures loaded via `golang.org/x/image` and converted to OpenGL textures
- Centralized texture manager at `internal/graphics/texture_manager.go`
- Texture caching and binding handled per render pass

---

*Stack analysis: 2026-02-08*
