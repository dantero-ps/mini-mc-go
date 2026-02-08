# Domain Research: Voxel Games & World Generation

**Project:** Minecraft Clone in Go
**Research Date:** 2026-02-08
**Confidence:** HIGH (codebase analysis)

## Current Implementation Status

Your codebase has a solid foundation for a voxel game:

### Chunk Architecture
- **Size:** 16x256x16 blocks (Minecraft-standard dimensions)
- **Sections:** 16x16x16 sub-chunks for vertical slicing
- **Sparse Storage:** Sections only allocate when non-empty (memory efficient)
- **Coordinate System:** World coordinates for deterministic generation

### World Generation
- **Noise System:** Custom value noise with integer hashing (deterministic)
- **Current Approach:** 2D heightmap with basic surface decoration
- **Determinism:** SplitMix64-style hashing ensures reproducibility

### Rendering
- **Greedy Meshing:** Parallel processing with worker pools
- **Vertex Format:** Packed 2x uint32 (GPU efficient)
- **Culling:** Face culling and dirty flagging implemented

### Physics
- **AABB Collision:** Player bounding box with world interaction
- **Movement:** Basic physics with gravity and collision response

## Minecraft World Generation Pipeline

Modern Minecraft (1.18+) uses a layered approach:

1. **2D Biome Selection**
   - Continental noise → climate parameters (temperature, humidity)
   - Multi-octave noise for variety
   - Biome assignment based on climate

2. **3D Terrain Shaping**
   - Density functions: `density = noise3D(x,y,z) + heightGradient(y)`
   - If density > threshold → solid block
   - Enables overhangs, caves, floating islands

3. **Surface Decoration**
   - Replace top blocks with biome-appropriate materials
   - Grass, sand, snow based on biome

4. **Feature Placement** (order matters!)
   - Caves (carve through terrain)
   - Ores (height-based distribution)
   - Structures (villages, dungeons)
   - Trees and vegetation

## Critical Insights

### 1. Determinism is Everything
- Same seed + coordinates **must** produce identical terrain
- Required for multiplayer, infinite worlds, chunk regeneration
- Your current hash function achieves this
- **Avoid:** Floating-point in critical paths, random map iteration

### 2. 3D Density vs 2D Heightmap
**Current:** 2D heightmap
- ✓ Fast, simple
- ✓ Works for basic terrain
- ✗ No caves, overhangs, or vertical features

**Upgrade:** 3D density function
- ✓ Natural caves and overhangs
- ✓ Foundation for all advanced features
- ✗ More computation (mitigate with caching)

### 3. Performance Bottlenecks
**Noise Computation:**
- 16×256×16 = 65,536 blocks per chunk
- Multiple octaves = 3-5x samples
- Solution: Octave early-out, chunk-level caching, worker pools ✓

**Mesh Generation:**
- Your greedy meshing is already well-optimized
- Parallel processing ✓
- Buffer pooling ✓

## Recommended Enhancement Path

### Phase 1: 3D Terrain System
**Goal:** Replace 2D heightmap with 3D density function

**Implementation:**
```go
density := noise3D(x, y, z) + heightGradient(y)
if density > 0 {
    // Solid block
}
```

**Benefits:**
- Enables caves, overhangs, arches
- Foundation for all other features
- Still deterministic

**Challenges:**
- More computation per chunk
- Need density caching strategy
- Tune parameters for good terrain shape

### Phase 2: Biome System
**Goal:** Multiple biomes with smooth transitions

**Components:**
- Climate noise (temperature, humidity, continentalness)
- Biome registry (plains, forest, desert, mountains, etc.)
- Interpolation between biomes (avoid sharp boundaries)
- Surface block mapping (grass, sand, snow)

**Benefits:**
- Visual variety
- Gameplay diversity
- Realistic world feel

### Phase 3: Cave Generation
**Goal:** Natural underground systems

**Types:**
- **Cheese caves:** Large caverns (3D noise > threshold)
- **Spaghetti caves:** Winding tunnels (worm carving)
- **Aquifer system:** Underground water sources

**Implementation:**
- Post-process carving after terrain generation
- Multiple cave layers (y-levels)
- Validate structural integrity

### Phase 4: Structures & Features
**Goal:** Ores, trees, villages

**Order:**
1. Ores (height-based distribution)
2. Trees (biome-specific, surface placement)
3. Structures (grid-based placement, template system)

**Challenges:**
- Chunk boundary overlap
- Generation order dependencies
- Template system for structures

## Major Pitfalls to Avoid

### 1. Chunk Boundary Artifacts
**Problem:** Features cut off at chunk edges

**Solution:**
- Always use world coordinates for noise
- Validate neighbors loaded before edge features
- Test with offset chunks, not just origin

### 2. Non-Determinism
**Problem:** Same seed produces different results

**Causes:**
- Go map iteration is random order
- Floating-point precision varies
- Race conditions in parallel generation

**Solution:**
- Sort before iteration
- Use integer math where possible
- Proper synchronization
- **Test:** Generate same chunk 1000x, verify byte-identical

### 3. Feature Overlap
**Problem:** Tree grows inside structure, ore replaces important block

**Solution:**
- Strict generation order: terrain → caves → ores → structures → trees
- Later features validate against earlier ones
- Priority system for conflicts

### 4. Performance Degradation
**Problem:** Game becomes unplayable at distance

**Solution:**
- Profile before optimizing
- Lazy mesh generation (you have dirty flagging ✓)
- Consider LOD for distant chunks
- Chunk loading/unloading strategies

## Technology Stack Recommendations

### Keep Current Approach ✓
- Custom noise implementation (full determinism control)
- No external world gen dependencies
- Worker pool pattern for parallelization
- Buffer pooling for memory efficiency

### Consider Adding
- **OpenSimplex noise:** Better gradient noise (evaluate determinism)
- **NBT library:** If loading Minecraft structures (otherwise JSON fine)
- **Serialization:** encoding/gob (native) or protobuf (cross-language for multiplayer)

### Avoid ✗
- Minecraft data extractors (legal/license issues)
- External world generators (breaks determinism control)
- Heavy framework dependencies (you're doing well without them)

## Testing Strategy

### Unit Tests
- Determinism: Same seed → same output (1000 iterations)
- Noise functions: Range validation, smoothness
- Chunk boundaries: No artifacts at edges

### Integration Tests
- Generate 10x10 chunk grid, verify seamless
- Memory usage under chunk load/unload
- Performance: Generation time per chunk

### Visual Tests
- Screenshot comparison for regression
- Manual exploration for artifacts
- Biome transition smoothness

## Success Metrics

### Performance Targets
- **Chunk generation:** <50ms per chunk
- **Mesh generation:** <20ms per chunk
- **Frame rate:** 60 FPS with 16-chunk render distance
- **Memory:** <2GB for 1000 loaded chunks

### Quality Targets
- **Determinism:** 100% reproducible from seed
- **Visual:** No chunk boundary artifacts
- **Gameplay:** Interesting terrain variety
- **Stability:** No crashes from terrain generation

## Research Gaps

Due to web search limitations, these areas need validation:

1. **Latest Minecraft specifics (1.19+):** Mojang doesn't publish algorithms
2. **Go voxel engine libraries (2026):** Ecosystem may have evolved
3. **Performance benchmarks:** Hardware-specific, need profiling
4. **Multiplayer patterns:** Server architecture best practices

## Next Steps

1. **Plan Phase 1:** 3D terrain system design
2. **Validate approach:** Prototype density function
3. **Performance baseline:** Profile current generation
4. **Biome research:** Study climate noise patterns

---

**Key Takeaway:** Your foundation is solid. The path forward is incremental enhancement, not replacement. Focus on 3D density → biomes → caves → features, in that order.
