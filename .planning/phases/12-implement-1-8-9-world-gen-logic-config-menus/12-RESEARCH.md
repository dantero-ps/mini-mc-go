# Phase 12: Implement 1.8.9 World Gen Logic & Config Menus - Research

**Researched:** 2026-02-08
**Domain:** Game Logic (World Generation) & UI
**Confidence:** HIGH

## Summary

This phase aims to implement the authentic Minecraft 1.8.9 world generation pipeline and expose it via configuration menus. The core noise algorithms are present in `internal/world/noise_authentic.go`, but **the file contains syntax errors (invalid ternary operators) and must be fixed** before use. The detailed generation logic is documented in `research/mc-1.8.9/chunk_provider_logic.md`. The existing `BioGenerator` serves as a reference but uses a slower per-block calculation method.

The UI system is ready for the new "World Gen Settings" menu. We will implement `WorldGenMenu` using existing widgets and integrate it into the `App` loop. Configuration will be stored in `internal/config/world_gen.go`.

**Primary recommendation:** Fix `internal/world/noise_authentic.go`, then implement `ChunkProvider189` using the 5x5x33 noise interpolation method. Create a `WorldGenMenu` to toggle authentic generation and other features.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `mini-mc/internal/world` | local | World Generation | Project's own world logic package |
| `mini-mc/internal/ui` | local | User Interface | Project's own UI framework |
| `mini-mc/internal/config` | local | Configuration | Project's global config store |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `ChunkProvider189` | New | 1.8.9 Gen Logic | For authentic world generation |
| `WorldGenMenu` | New | UI Screen | For configuring generation settings |

## Architecture Patterns

### World Generation Pipeline
1.  **Noise Generation:** Generate 5x5x33 density points using 4 noise generators (`depth`, `main`, `lower`, `upper`) and biome blending.
2.  **Interpolation:** Tri-linearly interpolate the 5x5x33 array to fill the 16x16x256 chunk.
3.  **Surface Replacement:** Replace top blocks with Grass/Dirt/Sand based on biome and noise.
4.  **Structure Generation:** (Hooks only for this phase).

### UI State Machine
1.  **Main Menu:** Add "World Gen Settings" button.
2.  **World Gen Menu:** New screen with toggles/sliders.
3.  **Config Store:** Global `WorldGenSettings` struct.

## Code Examples

### 1.8.9 Density Calculation (Concept)
```go
// Based on research/mc-1.8.9/chunk_provider_logic.md
func (g *ChunkProvider189) generateDensity(x, z int) []float64 {
    // 1. Initialize 5x5x33 noise array
    // 2. Run depth, main, lower, upper noise generators
    // 3. Blend with biome weights (parabolic field)
    // 4. Return density array for interpolation
}
```

### Config Integration
```go
// internal/config/world_gen.go
type WorldGenSettings struct {
    UseAuthenticGen bool
    GenerateCaves   bool
    SeaLevel        int
}
var WorldGen = &WorldGenSettings{ ... }
```

## Common Pitfalls

### Pitfall 1: Broken Noise Implementation
**What goes wrong:** `internal/world/noise_authentic.go` contains Java-style ternary operators (`? :`) which are invalid in Go.
**Why it happens:** Likely a direct copy-paste from Java source without conversion.
**How to avoid:** Fix the syntax errors immediately. Replace `cond ? a : b` with `if cond { return a } return b`.

### Pitfall 2: Per-Block Noise Calculation
**What goes wrong:** Calculating complex noise for every single block (65k calls).
**Why it happens:** Easier to implement than interpolation.
**How to avoid:** STRICTLY follow the 5x5x33 generation + tri-linear interpolation pattern (~825 calls).

### Pitfall 3: Coordinate Scaling
**What goes wrong:** Terrain looks wrong.
**Why it happens:** Using default Perlin scales instead of MC 1.8.9 constants.
**How to avoid:** Use the exact constants from `research/mc-1.8.9/constants_reference.md`.

## Sources

### Primary (HIGH confidence)
- `research/mc-1.8.9/chunk_provider_logic.md` - Detailed logic for 1.8.9 chunk generation.
- `internal/world/noise_authentic.go` - Existing (but broken) implementation of required noise algorithms.
- `internal/ui/menu/main_menu.go` - Reference for UI implementation.
- `internal/config/config.go` - Reference for configuration management.

### Secondary (MEDIUM confidence)
- `internal/world/bio_generator.go` - Current "inspired" implementation (useful for comparison).
