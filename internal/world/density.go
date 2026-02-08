package world

// DensityGenerator generates 3D terrain using density fields instead of heightmaps.
// This enables overhangs, floating formations, and underground voids.
type DensityGenerator struct {
	seed             int64
	scale            float64 // noise frequency (default: 1/64)
	baseHeight       int     // target surface level (default: 64)
	gradientStrength float64 // altitude density gradient (default: 32)
	octaves          int
	persistence      float64
	lacunarity       float64
}

// NewDensityGenerator creates a 3D density-based terrain generator.
func NewDensityGenerator(seed int64) TerrainGenerator {
	return &DensityGenerator{
		seed:             seed,
		scale:            1.0 / 64.0,
		baseHeight:       64,
		gradientStrength: 32.0,
		octaves:          4,
		persistence:      0.5,
		lacunarity:       2.0,
	}
}

// computeDensity calculates the density value at a world coordinate.
// Positive density = solid block, negative/zero = air.
func (g *DensityGenerator) computeDensity(worldX, worldY, worldZ int) float64 {
	// Convert to noise space
	nx := float64(worldX) * g.scale
	ny := float64(worldY) * g.scale
	nz := float64(worldZ) * g.scale

	// Sample 3D octave noise [0,1]
	noiseValue := octaveNoise3D(nx, ny, nz, g.seed, g.octaves, g.persistence, g.lacunarity)

	// Normalize to [-1,1]
	noiseValue = noiseValue*2.0 - 1.0

	// Compute height gradient (higher altitude = more negative)
	heightGradient := (float64(g.baseHeight) - float64(worldY)) / g.gradientStrength

	// Combine noise with height gradient
	return noiseValue + heightGradient
}

// HeightAt returns the maximum terrain height for chunk generation purposes.
// For 3D density terrain, this returns a constant upper bound since terrain
// can exist anywhere the density is positive.
func (g *DensityGenerator) HeightAt(worldX, worldZ int) int {
	// Return theoretical maximum height where terrain could exist
	// Above baseHeight + gradientStrength, density is always negative
	// (because noise max is 1.0 and gradient = (baseHeight-y)/gradientStrength)
	return g.baseHeight + int(g.gradientStrength)
}

// PopulateChunk fills a chunk using 3D density evaluation with trilinear interpolation.
func (g *DensityGenerator) PopulateChunk(c *Chunk) {
	chunkBaseY := c.Y * ChunkSizeY

	// Optimization: Determine the maximum height where terrain can exist.
	maxGenHeight := g.baseHeight + int(g.gradientStrength) + 1
	localMaxY := maxGenHeight - chunkBaseY
	if localMaxY < 0 {
		c.dirty = true
		return
	}
	if localMaxY > ChunkSizeY {
		localMaxY = ChunkSizeY
	}

	// Interpolation settings
	// We sample noise every 4 blocks on X/Z and every 8 blocks on Y
	const (
		xScale = 4
		yScale = 8
		zScale = 4
	)

	// Grid dimensions:
	// X: 0, 4, 8, 12, 16 -> 5 points
	// Z: 0, 4, 8, 12, 16 -> 5 points
	// Y: steps of 8 covering at least localMaxY
	numX := (ChunkSizeX / xScale) + 1 // 5
	numZ := (ChunkSizeZ / zScale) + 1 // 5

	// Determine necessary Y samples
	// If localMaxY is 10, we need to cover up to 16 (0, 8, 16) -> 3 points
	numY := (localMaxY+yScale-1)/yScale + 1

	// Sample buffer: flattened 3D array [x][z][y] for cache locality if traversing x,z then y?
	// Actually typical traversal is x,z,y, but let's just use flattened index or 1D array.
	densities := make([]float64, numX*numY*numZ)

	idx := func(x, y, z int) int {
		return (x*numY+y)*numZ + z
	}

	// 1. Generate sparse samples
	for dx := 0; dx < numX; dx++ {
		lx := dx * xScale
		worldX := c.X*ChunkSizeX + lx

		for dz := 0; dz < numZ; dz++ {
			lz := dz * zScale
			worldZ := c.Z*ChunkSizeZ + lz

			for dy := 0; dy < numY; dy++ {
				ly := dy * yScale
				worldY := chunkBaseY + ly

				densities[idx(dx, dy, dz)] = g.computeDensity(worldX, worldY, worldZ)
			}
		}
	}

	// 2. Interpolate and fill
	// Iterate cells
	for cx := 0; cx < numX-1; cx++ {
		for cz := 0; cz < numZ-1; cz++ {
			for cy := 0; cy < numY-1; cy++ {
				// Base indices for this cell
				d000 := densities[idx(cx, cy, cz)]
				d100 := densities[idx(cx+1, cy, cz)]
				d010 := densities[idx(cx, cy+1, cz)]
				d110 := densities[idx(cx+1, cy+1, cz)]
				d001 := densities[idx(cx, cy, cz+1)]
				d101 := densities[idx(cx+1, cy, cz+1)]
				d011 := densities[idx(cx, cy+1, cz+1)]
				d111 := densities[idx(cx+1, cy+1, cz+1)]

				// Interpolate within the cell
				// Cell size is xScale * yScale * zScale (4x8x4)

				// Calculate start positions
				startX := cx * xScale
				startY := cy * yScale
				startZ := cz * zScale

				// Limit Y loop if it exceeds localMaxY or ChunkSizeY
				limitY := startY + yScale
				if limitY > localMaxY {
					limitY = localMaxY
				}

				for lx := 0; lx < xScale; lx++ {
					// Interpolate along X
					tx := float64(lx) / float64(xScale)
					d00 := lerp(d000, d100, tx)
					d01 := lerp(d001, d101, tx)
					d10 := lerp(d010, d110, tx)
					d11 := lerp(d011, d111, tx)

					for lz := 0; lz < zScale; lz++ {
						// Interpolate along Z
						tz := float64(lz) / float64(zScale)
						d0 := lerp(d00, d01, tz) // Bottom face density at this x,z
						d1 := lerp(d10, d11, tz) // Top face density at this x,z

						for ly := 0; ly < (limitY - startY); ly++ {
							// Interpolate along Y
							// Note: ly here is relative to cell startY
							ty := float64(ly) / float64(yScale)
							density := lerp(d0, d1, ty)

							if density > 0 {
								targetY := startY + ly
								// Double check we are within bounds (should be safe due to limitY)
								if targetY < ChunkSizeY {
									var blockType BlockType
									if (chunkBaseY + targetY) == 0 {
										blockType = BlockTypeBedrock
									} else {
										blockType = BlockTypeStone
									}
									c.SetBlock(startX+lx, targetY, startZ+lz, blockType)
								}
							}
						}
					}
				}
			}
		}
	}

	c.dirty = true
}

// lerp is defined in noise.go
