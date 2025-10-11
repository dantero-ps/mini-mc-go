package blocks

import (
	"math"
	"mini-mc/internal/config"
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// Blocks implements block rendering feature
type Blocks struct {
	mainShader     *graphics.Shader
	visibleScratch []world.ChunkWithCoord
	// throttling & state
	ensureEvery time.Duration
	lastEnsure  time.Time
	lastChunkX  int
	lastChunkZ  int
	// cache of nearby chunks to avoid per-frame radius scans
	cachedPCX    int
	cachedPCZ    int
	cachedRadius int
	cachedNearby []world.ChunkWithCoord
}

// NewBlocks creates a new blocks renderable
func NewBlocks() *Blocks {
	return &Blocks{
		visibleScratch: make([]world.ChunkWithCoord, 0, 1024),
		ensureEvery:    200 * time.Millisecond,
		lastChunkX:     1<<31 - 1, // sentinel so first run triggers
		lastChunkZ:     1<<31 - 1,
		cachedPCX:      1<<31 - 1,
		cachedPCZ:      1<<31 - 1,
		cachedRadius:   -1,
		cachedNearby:   make([]world.ChunkWithCoord, 0, 1024),
	}
}

// Init initializes the blocks rendering system
func (b *Blocks) Init() error {
	// Initialize shader
	var err error
	b.mainShader, err = graphics.NewShader(MainVertShader, MainFragShader)
	if err != nil {
		return err
	}

	// Set static face colors once after linking the main shader
	b.mainShader.Use()
	northColor := world.GetBlockColor(world.FaceNorth)
	southColor := world.GetBlockColor(world.FaceSouth)
	eastColor := world.GetBlockColor(world.FaceEast)
	westColor := world.GetBlockColor(world.FaceWest)
	topColor := world.GetBlockColor(world.FaceTop)
	bottomColor := world.GetBlockColor(world.FaceBottom)
	defaultColor := mgl32.Vec3{0.5, 0.5, 0.5}
	b.mainShader.SetVector3("faceColorNorth", northColor.X(), northColor.Y(), northColor.Z())
	b.mainShader.SetVector3("faceColorSouth", southColor.X(), southColor.Y(), southColor.Z())
	b.mainShader.SetVector3("faceColorEast", eastColor.X(), eastColor.Y(), eastColor.Z())
	b.mainShader.SetVector3("faceColorWest", westColor.X(), westColor.Y(), westColor.Z())
	b.mainShader.SetVector3("faceColorTop", topColor.X(), topColor.Y(), topColor.Z())
	b.mainShader.SetVector3("faceColorBottom", bottomColor.X(), bottomColor.Y(), bottomColor.Z())
	b.mainShader.SetVector3("faceColorDefault", defaultColor.X(), defaultColor.Y(), defaultColor.Z())

	// Initialize global data structures
	chunkMeshes = make(map[world.ChunkCoord]*chunkMesh)
	columnMeshes = make(map[[2]int]*columnMesh)

	// Setup atlas
	setupAtlas()

	return nil
}

// Render renders all visible blocks
func (b *Blocks) Render(ctx renderer.RenderContext) {
	// Apply wireframe polygon mode if toggled, then always reset to FILL after drawing blocks
	if player.WireframeMode {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
		defer gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}

	func() {
		defer profiling.Track("renderer.renderBlocks")()
		b.renderBlocksInternal(ctx)
	}()
}

// Dispose cleans up OpenGL resources
func (b *Blocks) Dispose() {
	// Clean up shaders
	if b.mainShader != nil {
		// Note: Shader cleanup would need to be implemented in the Shader type
	}

	// Clean up atlas resources
	if atlasVAO != 0 {
		gl.DeleteVertexArrays(1, &atlasVAO)
	}
	if atlasVBO != 0 {
		gl.DeleteBuffers(1, &atlasVBO)
	}

	// Clean up chunk meshes
	for _, m := range chunkMeshes {
		if m != nil {
			if m.vao != 0 {
				gl.DeleteVertexArrays(1, &m.vao)
			}
			if m.vbo != 0 {
				gl.DeleteBuffers(1, &m.vbo)
			}
		}
	}
}

func (b *Blocks) renderBlocksInternal(ctx renderer.RenderContext) {
	func() {
		defer profiling.Track("renderer.renderBlocks.shaderSetup")()
		// Track shader binding time (only the Use() call)
		b.mainShader.Use()

		b.mainShader.SetMatrix4("proj", &ctx.Proj[0])
		b.mainShader.SetMatrix4("view", &ctx.View[0])

		// Set light direction
		light := mgl32.Vec3{0.3, 1.0, 0.3}.Normalize()
		b.mainShader.SetVector3("lightDir", light.X(), light.Y(), light.Z())
	}()

	// Draw greedy-meshed chunks that intersect the camera frustum
	planes := func() [6]plane {
		defer profiling.Track("renderer.renderBlocks.frustumSetup")()
		clip := ctx.Proj.Mul4(ctx.View)
		return extractFrustumPlanes(clip)
	}()

	// Hard cap for render radius to shrink candidate set pre-cull/sort
	maxRenderRadiusChunks := config.GetMaxRenderRadius()

	px := int(math.Floor(float64(ctx.Player.Position[0])))
	pz := int(math.Floor(float64(ctx.Player.Position[2])))
	pcx := px / world.ChunkSizeX
	pcz := pz / world.ChunkSizeZ

	// Phase 1: collect nearby chunks only when player enters a new chunk or radius changes
	var nearbyChunks []world.ChunkWithCoord
	sameCell := pcx == b.cachedPCX && pcz == b.cachedPCZ && maxRenderRadiusChunks == b.cachedRadius
	if !sameCell {
		// Query only chunks in radius using world's column index and cache the result
		b.visibleScratch = b.visibleScratch[:0]
		tmp := ctx.World.AppendChunksInRadiusXZ(pcx, pcz, maxRenderRadiusChunks, b.visibleScratch)
		b.cachedNearby = append(b.cachedNearby[:0], tmp...)
		b.cachedPCX, b.cachedPCZ = pcx, pcz
		b.cachedRadius = maxRenderRadiusChunks
	}
	nearbyChunks = b.cachedNearby

	shouldEnsure := false
	// If any nearby chunk is dirty, rebuild immediately (reflect edits without delay)
	hasDirty := false
	for _, cc := range nearbyChunks {
		if cc.Chunk != nil && cc.Chunk.IsDirty() {
			hasDirty = true
			break
		}
	}
	if b.lastChunkX != pcx || b.lastChunkZ != pcz {
		shouldEnsure = true
		b.lastChunkX, b.lastChunkZ = pcx, pcz
	} else if time.Since(b.lastEnsure) >= b.ensureEvery {
		shouldEnsure = true
	} else if hasDirty {
		shouldEnsure = true
	}
	if shouldEnsure {
		stop := profiling.Track("renderer.renderBlocks.ensureMeshes")
		for _, cc := range nearbyChunks {
			coord := cc.Coord
			ch := cc.Chunk
			if ch == nil {
				continue
			}
			existing := chunkMeshes[coord]
			needsBuild := existing == nil || ch.IsDirty()
			if needsBuild {
				_ = ensureChunkMesh(ctx.World, coord, ch)
			}
		}
		b.lastEnsure = time.Now()
		stop()
	}

	// Collect visible chunks with frustum culling (for rendering only)
	var visible []world.ChunkWithCoord
	{
		stop := profiling.Track("renderer.renderBlocks.collectVisible")
		visible = make([]world.ChunkWithCoord, 0, len(nearbyChunks))

		// Pre-calculate common values to avoid repeated calculations
		chunkSizeXf := float32(world.ChunkSizeX)
		chunkSizeYf := float32(world.ChunkSizeY)
		chunkSizeZf := float32(world.ChunkSizeZ)
		margin := frustumMargin

		for _, cc := range nearbyChunks {
			// Calculate chunk bounds with pre-computed constants
			cx := float32(cc.Coord.X) * chunkSizeXf
			cy := float32(cc.Coord.Y) * chunkSizeYf
			cz := float32(cc.Coord.Z) * chunkSizeZf

			// Apply margin directly to avoid intermediate variables
			minx := cx - margin
			miny := cy - margin
			minz := cz - margin
			maxx := cx + chunkSizeXf + margin
			maxy := cy + chunkSizeYf + margin
			maxz := cz + chunkSizeZf + margin

			if aabbIntersectsFrustumPlanesF(minx, miny, minz, maxx, maxy, maxz, planes) {
				visible = append(visible, cc)
			}
		}
		stop()
	}

	// Phase 2: single multi-draw over atlas
	func() {
		defer profiling.Track("renderer.renderBlocks.drawAtlas")()
		// Aggregate visible chunks into unique XZ columns
		type xz struct{ x, z int }
		colSet := make(map[xz]struct{}, len(visible))
		for _, vc := range visible {
			colSet[xz{vc.Coord.X, vc.Coord.Z}] = struct{}{}
		}
		// First ensure/update columns
		// Increment frame and mark visible columns for this frame to avoid per-frame maps
		forMarked := false
		for k := range colSet {
			col := ensureColumnMeshForXZ(k.x, k.z)
			if !forMarked {
				currentFrame++
				forMarked = true
			}
			if col != nil {
				col.visibleFrame = currentFrame
			}
		}
		// Draw ready columns using a single pass over orderedColumns (already ordered by firstVertex)
		if len(colSet) > 0 {
			if cap(firstsScratch) < len(colSet) {
				firstsScratch = make([]int32, len(colSet))
				countsScratch = make([]int32, len(colSet))
			}
			firsts := firstsScratch[:0]
			counts := countsScratch[:0]
			var lastFirst int32
			var lastCount int32
			hasRun := false
			for _, c := range orderedColumns {
				if c == nil {
					continue
				}
				// Skip if not visible this frame or already drawn via a duplicate entry
				if c.visibleFrame != currentFrame || c.drawnFrame == currentFrame {
					continue
				}
				if c.dirty || c.vertexCount <= 0 || c.firstFloat < 0 {
					continue
				}
				if c.firstVertex < 0 {
					c.firstVertex = int32(c.firstFloat / 6)
				}
				cf := c.firstVertex
				cc := c.vertexCount
				if hasRun && cf == lastFirst+lastCount {
					lastCount += cc
					counts[len(counts)-1] = lastCount
				} else {
					firsts = append(firsts, cf)
					counts = append(counts, cc)
					lastFirst = cf
					lastCount = cc
					hasRun = true
				}
				c.drawnFrame = currentFrame
			}
			if len(counts) > 0 {
				gl.BindVertexArray(atlasVAO)
				gl.MultiDrawArrays(gl.TRIANGLES, &firsts[0], &counts[0], int32(len(counts)))
			}
		}
		// Collect and draw fallback chunks (columns not drawn this frame)
		if cap(fallbackScratch) < len(visible) {
			fallbackScratch = make([]*chunkMesh, 0, len(visible))
		}
		fallbackChunks := fallbackScratch[:0]
		for _, vc := range visible {
			key := [2]int{vc.Coord.X, vc.Coord.Z}
			col := columnMeshes[key]
			if col != nil && col.drawnFrame == currentFrame {
				continue
			}
			if m := chunkMeshes[vc.Coord]; m != nil && m.vertexCount > 0 && m.firstFloat >= 0 {
				if m.firstVertex < 0 {
					m.firstVertex = int32(m.firstFloat / 6)
				}
				fallbackChunks = append(fallbackChunks, m)
			}
		}
		if len(fallbackChunks) > 0 {
			if cap(firstsScratch) < len(fallbackChunks) {
				firstsScratch = make([]int32, len(fallbackChunks))
				countsScratch = make([]int32, len(fallbackChunks))
			}
			firsts := firstsScratch[:0]
			counts := countsScratch[:0]
			for _, m := range fallbackChunks {
				if m.firstVertex < 0 || m.vertexCount <= 0 {
					continue
				}
				firsts = append(firsts, m.firstVertex)
				counts = append(counts, m.vertexCount)
			}
			if len(counts) > 0 {
				gl.BindVertexArray(atlasVAO)
				gl.MultiDrawArrays(gl.TRIANGLES, &firsts[0], &counts[0], int32(len(counts)))
			}
		}
	}()
}
