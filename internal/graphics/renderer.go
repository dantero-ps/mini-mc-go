package graphics

import (
	"fmt"
	"math"
	"mini-mc/internal/meshing"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	WinWidth  = 900
	WinHeight = 600
)

// Shader file paths
const (
	ShadersDir = "assets/shaders"

	MainVertShader      = "main.vert"
	MainFragShader      = "main.frag"
	WireframeVertShader = "wireframe.vert"
	WireframeFragShader = "wireframe.frag"
	CrosshairVertShader = "crosshair.vert"
	CrosshairFragShader = "crosshair.frag"
	DirectionVertShader = "direction.vert"
	DirectionFragShader = "direction.frag"
)

var CrosshairVertices = []float32{
	-0.02, 0.0,
	0.02, 0.0,
	0.0, -0.02,
	0.0, 0.02,
}

// Direction indicator vertices (arrow shape pointing up)
var DirectionVertices = []float32{
	// Arrow body
	-0.01, -0.08,
	0.01, -0.08,
	0.01, -0.02,
	-0.01, -0.02,
	// Arrow head
	-0.03, -0.02,
	0.03, -0.02,
	0.0, 0.02,
}

// Letter vertices for N, E, S, W
var LetterN = []float32{
	-0.02, -0.02, // Left vertical line start
	-0.02, 0.02, // Left vertical line end
	-0.02, 0.02, // Diagonal line start
	0.02, -0.02, // Diagonal line end
	0.02, -0.02, // Right vertical line start
	0.02, 0.02, // Right vertical line end
}

var LetterE = []float32{
	-0.02, -0.02, // Left vertical line start
	-0.02, 0.02, // Left vertical line end
	-0.02, 0.02, // Top horizontal line start
	0.02, 0.02, // Top horizontal line end
	-0.02, 0.0, // Middle horizontal line start
	0.01, 0.0, // Middle horizontal line end
	-0.02, -0.02, // Bottom horizontal line start
	0.02, -0.02, // Bottom horizontal line end
}

var LetterS = []float32{
	0.02, 0.02, // Top horizontal line start
	-0.02, 0.02, // Top horizontal line end
	-0.02, 0.02, // Top left vertical line start
	-0.02, 0.0, // Top left vertical line end
	-0.02, 0.0, // Middle horizontal line start
	0.02, 0.0, // Middle horizontal line end
	0.02, 0.0, // Bottom right vertical line start
	0.02, -0.02, // Bottom right vertical line end
	0.02, -0.02, // Bottom horizontal line start
	-0.02, -0.02, // Bottom horizontal line end
}

var LetterW = []float32{
	-0.02, 0.02, // Left outer vertical line start
	-0.02, -0.02, // Left outer vertical line end
	-0.02, -0.02, // Left diagonal line start
	-0.01, 0.0, // Left diagonal line end
	-0.01, 0.0, // Middle diagonal line start
	0.01, -0.02, // Middle diagonal line end
	0.01, -0.02, // Right diagonal line start
	0.02, 0.0, // Right diagonal line end
	0.02, 0.0, // Right outer vertical line start
	0.02, 0.02, // Right outer vertical line end
}

type Renderer struct {
	mainShader      *Shader
	wireframeShader *Shader
	crosshairShader *Shader
	directionShader *Shader
	camera          *Camera

	// FOV transition
	targetFOV  float32
	currentFOV float32

	// OpenGL objects
	blockVAO     uint32
	blockVBO     uint32
	instanceVBO  uint32
	wireframeVAO uint32
	wireframeVBO uint32
	crosshairVAO uint32
	crosshairVBO uint32
	directionVAO uint32
	directionVBO uint32
	letterVAO    uint32
	letterVBO    uint32

	// Global atlas VBO/VAO for static chunk geometry (pos+normal interleaved)
	atlasVAO           uint32
	atlasVBO           uint32
	atlasCapacityBytes int
	atlasTotalFloats   int
	firstsScratch      []int32
	countsScratch      []int32

	// UI rendering (rectangles)
	uiShader *Shader
	uiVAO    uint32
	uiVBO    uint32

	// Font rendering
	fontAtlas    *FontAtlasInfo
	fontRenderer *FontRenderer

	// Greedy meshing cache per chunk
	chunkMeshes map[world.ChunkCoord]*chunkMesh

	// Frustum culling margin in blocks (inflates AABBs before testing)
	frustumMargin float32

	// Frustum plane caching to avoid recalculating when camera hasn't moved much
	cachedFrustumPlanes [6]plane
	cachedViewMatrix    mgl32.Mat4
	cachedProjMatrix    mgl32.Mat4
	frustumCacheValid   bool

	// FPS tracking
	frames       int
	lastFPSCheck time.Time
	currentFPS   int

	// Enhanced profiling metrics
	profilingStats struct {
		// Frame timing
		frameStartTime    time.Time
		frameEndTime      time.Time
		frameDuration     time.Duration
		lastFrameDuration time.Duration

		// Rendering statistics
		totalDrawCalls int
		totalVertices  int
		totalTriangles int
		visibleChunks  int
		culledChunks   int
		meshedChunks   int

		// GPU memory usage
		gpuMemoryUsage     int64
		textureMemoryUsage int64
		bufferMemoryUsage  int64

		// Performance counters
		frustumCullTime time.Duration
		meshingTime     time.Duration
		shaderBindTime  time.Duration
		vaoBindTime     time.Duration

		// Historical data for averaging
		frameTimeHistory []time.Duration
		maxFrameTime     time.Duration
		minFrameTime     time.Duration
		avgFrameTime     time.Duration

		// Total frame duration measured externally (prev frame)
		lastTotalFrameDuration time.Duration
		// Update phase duration measured in main (prev frame)
		lastUpdateDuration time.Duration

		// Non-render breakdown from previous frame
		lastPlayerDuration  time.Duration
		lastWorldDuration   time.Duration
		lastGlfwDuration    time.Duration
		lastHudDuration     time.Duration
		lastPruneDuration   time.Duration
		lastOtherDuration   time.Duration
		lastPhysicsDuration time.Duration

		// Phase durations (prev frame)
		lastPreRenderDuration  time.Duration // from frame start to Render() call
		lastSwapEventsDuration time.Duration // SwapBuffers + PollEvents actual time
	}

	// Scratch buffers to avoid per-frame allocs
	visibleScratch []world.ChunkWithCoord

	// Per-column (XZ) combined meshes to reduce draw/cull granularity
	columnMeshes map[[2]int]*columnMesh
}

// Internal atlas helpers
// appendOrUpdateAtlas appends a new chunk's vertices into the atlas or updates an existing region.

func (r *Renderer) ensureAtlasCapacity(requiredBytes int) {
	if requiredBytes <= r.atlasCapacityBytes {
		return
	}
	capBytes := r.atlasCapacityBytes
	if capBytes == 0 {
		capBytes = 1 << 22 // 4MB
	}
	for capBytes < requiredBytes {
		capBytes *= 2
	}
	// Allocate new buffer; we'll rebuild contents from CPU copies (portable)
	var newVBO uint32
	gl.GenBuffers(1, &newVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, newVBO)
	gl.BufferData(gl.ARRAY_BUFFER, capBytes, nil, gl.DYNAMIC_DRAW)

	// Swap buffers: rebind VAO attributes to the new VBO
	oldVBO := r.atlasVBO
	r.atlasVBO = newVBO
	r.atlasCapacityBytes = capBytes

	gl.BindVertexArray(r.atlasVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	// Rebuild atlas from CPU-side chunk meshes (avoids CopyBufferSubData)
	r.rebuildAtlasFromCPU()

	// All column meshes' atlas offsets are now invalid; mark for rebuild
	for _, col := range r.columnMeshes {
		if col == nil {
			continue
		}
		col.firstFloat = -1
		col.dirty = true
	}

	if oldVBO != 0 {
		gl.DeleteBuffers(1, &oldVBO)
	}
}

// rebuildAtlasFromCPU compacts and re-uploads all available chunk meshes into the atlas VBO.
func (r *Renderer) rebuildAtlasFromCPU() {
	if r.atlasVBO == 0 {
		return
	}
	// Reset offset
	r.atlasTotalFloats = 0
	gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
	for coord, m := range r.chunkMeshes {
		_ = coord
		if m == nil || m.vertexCount == 0 || len(m.cpuVerts) == 0 {
			m.firstFloat = -1
			continue
		}
		bytes := len(m.cpuVerts) * 4
		offsetBytes := r.atlasTotalFloats * 4
		gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(m.cpuVerts))
		m.firstFloat = r.atlasTotalFloats
		r.atlasTotalFloats += len(m.cpuVerts)
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func (r *Renderer) appendOrUpdateAtlas(coord world.ChunkCoord, m *chunkMesh) {
	if m == nil {
		return
	}
	verts := m.cpuVerts
	if len(verts) == 0 {
		m.firstFloat = -1
		return
	}
	bytes := len(verts) * 4
	if m.firstFloat < 0 && m.vertexCount == int32(len(verts)/6) {
		// Append new
		requiredBytes := (r.atlasTotalFloats + len(verts)) * 4
		r.ensureAtlasCapacity(requiredBytes)
		offsetBytes := r.atlasTotalFloats * 4
		gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(verts))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		m.firstFloat = r.atlasTotalFloats
		r.atlasTotalFloats += len(verts)
		return
	}
	// Update existing region (size may change; simple strategy: if different, re-append)
	oldCountFloats := int(m.vertexCount) * 6
	if m.firstFloat >= 0 && oldCountFloats == len(verts) {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, m.firstFloat*4, bytes, gl.Ptr(verts))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		return
	}
	// Size changed: append new region and invalidate old by leaving a hole (simple, avoids compaction for now)
	requiredBytes := (r.atlasTotalFloats + len(verts)) * 4
	r.ensureAtlasCapacity(requiredBytes)
	offsetBytes := r.atlasTotalFloats * 4
	gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, bytes, gl.Ptr(verts))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	m.firstFloat = r.atlasTotalFloats
	r.atlasTotalFloats += len(verts)
}

type chunkMesh struct {
	vao         uint32 // legacy per-chunk VAO (kept for now)
	vbo         uint32 // legacy per-chunk VBO (kept for cleanup)
	vertexCount int32
	cpuVerts    []float32 // kept for atlas updates
	firstFloat  int       // offset into atlas in floats (pos+normal interleaved)
}

// columnMesh represents a combined mesh for an XZ column across Y layers
type columnMesh struct {
	cpuVerts    []float32
	vertexCount int32
	firstFloat  int
	dirty       bool
}

// ensureColumnMeshForXZ builds or updates a combined mesh for a given XZ column
func (r *Renderer) ensureColumnMeshForXZ(x, z int) *columnMesh {
	key := [2]int{x, z}
	col := r.columnMeshes[key]
	if col == nil {
		col = &columnMesh{firstFloat: -1, dirty: true}
		r.columnMeshes[key] = col
	}
	if !col.dirty {
		return col
	}
	// Count total floats across Y-chunk meshes in this column
	total := 0
	for coord, cm := range r.chunkMeshes {
		if coord.X == x && coord.Z == z && cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			total += len(cm.cpuVerts)
		}
	}
	// If currently nothing ready to build, keep previous geometry to avoid flicker
	if total == 0 {
		// Keep column marked dirty so renderer uses per-chunk fallback until ready
		return col
	}
	buf := make([]float32, 0, total)
	for coord, cm := range r.chunkMeshes {
		if coord.X == x && coord.Z == z && cm != nil && cm.vertexCount > 0 && len(cm.cpuVerts) > 0 {
			buf = append(buf, cm.cpuVerts...)
		}
	}
	// If size unchanged and region valid, update in place
	if int32(len(buf)/6) == col.vertexCount && col.firstFloat >= 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
		gl.BufferSubData(gl.ARRAY_BUFFER, col.firstFloat*4, len(buf)*4, gl.Ptr(buf))
		gl.BindBuffer(gl.ARRAY_BUFFER, 0)
		col.cpuVerts = buf
		col.dirty = false
		return col
	}
	// Otherwise append new region
	requiredBytes := (r.atlasTotalFloats + len(buf)) * 4
	r.ensureAtlasCapacity(requiredBytes)
	offsetBytes := r.atlasTotalFloats * 4
	gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, offsetBytes, len(buf)*4, gl.Ptr(buf))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	col.cpuVerts = buf
	col.vertexCount = int32(len(buf) / 6)
	col.firstFloat = r.atlasTotalFloats
	r.atlasTotalFloats += len(buf)
	col.dirty = false
	return col
}

func NewRenderer() (*Renderer, error) {
	if err := gl.Init(); err != nil {
		return nil, err
	}

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	// Enable back-face culling (meshing emits CCW front faces)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)

	// Create shaders from files
	mainVertPath := filepath.Join(ShadersDir, MainVertShader)
	mainFragPath := filepath.Join(ShadersDir, MainFragShader)
	mainShader, err := NewShader(mainVertPath, mainFragPath)
	if err != nil {
		return nil, err
	}

	// Set static face colors once after linking the main shader
	mainShader.Use()
	northColor := world.GetBlockColor(world.FaceNorth)
	southColor := world.GetBlockColor(world.FaceSouth)
	eastColor := world.GetBlockColor(world.FaceEast)
	westColor := world.GetBlockColor(world.FaceWest)
	topColor := world.GetBlockColor(world.FaceTop)
	bottomColor := world.GetBlockColor(world.FaceBottom)
	defaultColor := mgl32.Vec3{0.5, 0.5, 0.5}
	mainShader.SetVector3("faceColorNorth", northColor.X(), northColor.Y(), northColor.Z())
	mainShader.SetVector3("faceColorSouth", southColor.X(), southColor.Y(), southColor.Z())
	mainShader.SetVector3("faceColorEast", eastColor.X(), eastColor.Y(), eastColor.Z())
	mainShader.SetVector3("faceColorWest", westColor.X(), westColor.Y(), westColor.Z())
	mainShader.SetVector3("faceColorTop", topColor.X(), topColor.Y(), topColor.Z())
	mainShader.SetVector3("faceColorBottom", bottomColor.X(), bottomColor.Y(), bottomColor.Z())
	mainShader.SetVector3("faceColorDefault", defaultColor.X(), defaultColor.Y(), defaultColor.Z())

	wireframeVertPath := filepath.Join(ShadersDir, WireframeVertShader)
	wireframeFragPath := filepath.Join(ShadersDir, WireframeFragShader)
	wireframeShader, err := NewShader(wireframeVertPath, wireframeFragPath)
	if err != nil {
		return nil, err
	}

	crosshairVertPath := filepath.Join(ShadersDir, CrosshairVertShader)
	crosshairFragPath := filepath.Join(ShadersDir, CrosshairFragShader)
	crosshairShader, err := NewShader(crosshairVertPath, crosshairFragPath)
	if err != nil {
		return nil, err
	}

	directionVertPath := filepath.Join(ShadersDir, DirectionVertShader)
	directionFragPath := filepath.Join(ShadersDir, DirectionFragShader)
	directionShader, err := NewShader(directionVertPath, directionFragPath)
	if err != nil {
		return nil, err
	}

	// Create camera
	camera := NewCamera(WinWidth, WinHeight)

	// Default FOV value
	defaultFOV := float32(60.0)

	renderer := &Renderer{
		mainShader:      mainShader,
		wireframeShader: wireframeShader,
		crosshairShader: crosshairShader,
		directionShader: directionShader,
		camera:          camera,
		targetFOV:       defaultFOV,
		currentFOV:      defaultFOV,
		frustumMargin:   1.0, // one block margin
		frames:          0,
		lastFPSCheck:    time.Now(),
		currentFPS:      0,
	}

	// Initialize VAOs and VBOs
	renderer.setupBlockVAO()
	renderer.setupWireframeVAO()
	renderer.setupCrosshairVAO()
	renderer.setupDirectionVAO()
	renderer.setupLetterVAO()
	renderer.setupAtlas()
	renderer.chunkMeshes = make(map[world.ChunkCoord]*chunkMesh)
	renderer.columnMeshes = make(map[[2]int]*columnMesh)

	// Load font atlas and renderer for HUD text
	fontPath := filepath.Join("assets", "fonts", "OpenSans-LightItalic.ttf")
	atlas, err := BuildFontAtlas(fontPath, 24)
	if err != nil {
		return nil, err
	}
	fontRenderer, err := NewFontRenderer(atlas)
	if err != nil {
		return nil, err
	}
	renderer.fontAtlas = atlas
	renderer.fontRenderer = fontRenderer

	// Setup simple UI pipeline
	if err := renderer.setupUI(); err != nil {
		return nil, err
	}

	return renderer, nil
}

func (r *Renderer) setupUI() error {
	// Minimal shader that draws solid-colored triangles in NDC space
	vertSrc := `#version 410 core
layout (location = 0) in vec2 aPos;
void main() {
    gl_Position = vec4(aPos, 0.0, 1.0);
}`
	fragSrc := `#version 410 core
out vec4 FragColor;
uniform vec4 uColor;
void main() {
    FragColor = uColor;
}`
	sh, err := NewShaderFromSource(vertSrc, fragSrc)
	if err != nil {
		return err
	}
	r.uiShader = sh
	gl.GenVertexArrays(1, &r.uiVAO)
	gl.GenBuffers(1, &r.uiVBO)
	gl.BindVertexArray(r.uiVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.uiVBO)
	gl.BufferData(gl.ARRAY_BUFFER, 6*2*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	return nil
}

// DrawFilledRect draws a screen-space rectangle (pixels, top-left origin) with RGBA color.
func (r *Renderer) DrawFilledRect(x, y, w, h float32, color mgl32.Vec3, alpha float32) {
	// Convert to NDC [-1,1]
	x0 := (x/float32(WinWidth))*2 - 1
	y0 := 1 - (y/float32(WinHeight))*2
	x1 := ((x+w)/float32(WinWidth))*2 - 1
	y1 := 1 - ((y+h)/float32(WinHeight))*2
	verts := []float32{
		x0, y0,
		x1, y0,
		x1, y1,
		x0, y0,
		x1, y1,
		x0, y1,
	}

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	r.uiShader.Use()
	r.uiShader.SetVector3("uColor", color.X(), color.Y(), color.Z())
	// Set alpha via separate uniform by packing into vec4: re-use SetFloat? create specific SetVector4? We'll pack via gl.Uniform4f
	loc := gl.GetUniformLocation(r.uiShader.ID, gl.Str("uColor\x00"))
	gl.Uniform4f(loc, color.X(), color.Y(), color.Z(), alpha)

	gl.BindVertexArray(r.uiVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.uiVBO)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(verts)*4, gl.Ptr(verts))
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
}

func (r *Renderer) setupBlockVAO() {
	gl.GenVertexArrays(1, &r.blockVAO)
	gl.BindVertexArray(r.blockVAO)

	gl.GenBuffers(1, &r.blockVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.blockVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(world.CubeVertices)*4, gl.Ptr(world.CubeVertices), gl.STATIC_DRAW)

	stride := int32(6 * 4)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, 3*4)

	// Instance buffer
	gl.GenBuffers(1, &r.instanceVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.instanceVBO)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 3, gl.FLOAT, false, 3*4, 0)
	gl.VertexAttribDivisor(2, 1)
}

func (r *Renderer) setupWireframeVAO() {
	gl.GenVertexArrays(1, &r.wireframeVAO)
	gl.BindVertexArray(r.wireframeVAO)

	gl.GenBuffers(1, &r.wireframeVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.wireframeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(world.CubeWireframeVertices)*4, gl.Ptr(world.CubeWireframeVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*4, 0)
}

func (r *Renderer) setupCrosshairVAO() {
	gl.GenVertexArrays(1, &r.crosshairVAO)
	gl.BindVertexArray(r.crosshairVAO)

	gl.GenBuffers(1, &r.crosshairVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.crosshairVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(CrosshairVertices)*4, gl.Ptr(CrosshairVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 2*4, 0)
}

func (r *Renderer) setupDirectionVAO() {
	gl.GenVertexArrays(1, &r.directionVAO)
	gl.BindVertexArray(r.directionVAO)

	gl.GenBuffers(1, &r.directionVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.directionVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(DirectionVertices)*4, gl.Ptr(DirectionVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
}

func (r *Renderer) setupLetterVAO() {
	gl.GenVertexArrays(1, &r.letterVAO)
	gl.BindVertexArray(r.letterVAO)

	gl.GenBuffers(1, &r.letterVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.letterVBO)
	// We'll update the buffer data when drawing specific letters

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
}

func (r *Renderer) setupAtlas() {
	gl.GenVertexArrays(1, &r.atlasVAO)
	gl.BindVertexArray(r.atlasVAO)

	gl.GenBuffers(1, &r.atlasVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.atlasVBO)
	initial := 1 << 22 // 4MB
	gl.BufferData(gl.ARRAY_BUFFER, initial, nil, gl.DYNAMIC_DRAW)
	r.atlasCapacityBytes = initial
	r.atlasTotalFloats = 0

	// Attributes: pos.xyz + normal.xyz
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))

	gl.BindVertexArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

func (r *Renderer) Render(w *world.World, p *player.Player, dt float64) {
	// Start frame profiling
	r.startFrameProfiling()

	// Update FPS tracking
	r.frames++
	if time.Since(r.lastFPSCheck) >= time.Second {
		r.currentFPS = r.frames
		r.lastFPSCheck = time.Now()
		r.frames = 0
	}

	// Clear the screen
	func() {
		defer profiling.Track("renderer.clear")()
		gl.ClearColor(0.53, 0.81, 0.92, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	}()

	func() {
		defer profiling.Track("renderer.updateFOV")()
		// Adjust FOV based on player sprint state
		normalFOV := float32(60.0)
		sprintFOV := float32(70.0)

		// Set target FOV based on sprint state and actual horizontal movement
		// Only increase FOV if sprinting AND moving horizontally above a small threshold
		horizontalSpeed := float32(math.Hypot(float64(p.Velocity[0]), float64(p.Velocity[2])))
		isActivelySprinting := p.IsSprinting && horizontalSpeed > 0.01
		if isActivelySprinting {
			r.targetFOV = sprintFOV
		} else {
			r.targetFOV = normalFOV
		}

		// Smooth interpolation between current and target FOV
		// Adjust this value to control the speed of the transition
		transitionSpeed := float32(100.0)

		// Calculate the interpolation step based on delta time
		step := float32(dt) * transitionSpeed

		// Smoothly interpolate towards the target FOV
		if r.currentFOV < r.targetFOV {
			// Increase current FOV but don't exceed target
			r.currentFOV += step
			if r.currentFOV > r.targetFOV {
				r.currentFOV = r.targetFOV
			}
		} else if r.currentFOV > r.targetFOV {
			// Decrease current FOV but don't go below target
			r.currentFOV -= step
			if r.currentFOV < r.targetFOV {
				r.currentFOV = r.targetFOV
			}
		}

		// Apply the interpolated FOV
		r.camera.FOV = r.currentFOV
	}()

	// Get view and projection matrices
	var view, projection mgl32.Mat4
	func() {
		defer profiling.Track("renderer.computeMatrices")()
		view = p.GetViewMatrix()
		projection = r.camera.GetProjectionMatrix()
	}()

	// Reset profiling counters for this frame
	r.resetFrameCounters()

	// Render blocks
	if player.WireframeMode {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	func() { defer profiling.Track("renderer.renderBlocks")(); r.renderBlocks(w, p, view, projection) }()

	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Render highlighted block
	if p.HasHoveredBlock {
		func() {
			defer profiling.Track("renderer.renderHighlightedBlock")()
			r.renderHighlightedBlock(p.HoveredBlock, view, projection)
		}()
	}

	// Render crosshair
	func() { defer profiling.Track("renderer.renderCrosshair")(); r.renderCrosshair() }()

	// Render direction indicator
	func() { defer profiling.Track("renderer.renderDirection")(); r.renderDirection(p) }()

	// Render player position (HUD)
	func() { defer profiling.Track("renderer.renderPlayerPos")(); r.renderPlayerPosition(p) }()

	// HUD is rendered from main after frame profiling is finalized

	// End frame profiling
	r.endFrameProfiling()
}

// startFrameProfiling initializes profiling for a new frame
func (r *Renderer) startFrameProfiling() {
	r.profilingStats.frameStartTime = time.Now()
}

// endFrameProfiling finalizes profiling for the current frame
func (r *Renderer) endFrameProfiling() {
	r.profilingStats.frameEndTime = time.Now()
	r.profilingStats.frameDuration = r.profilingStats.frameEndTime.Sub(r.profilingStats.frameStartTime)

	// Update frame time history (keep last 60 frames for averaging)
	if len(r.profilingStats.frameTimeHistory) >= 60 {
		r.profilingStats.frameTimeHistory = r.profilingStats.frameTimeHistory[1:]
	}
	r.profilingStats.frameTimeHistory = append(r.profilingStats.frameTimeHistory, r.profilingStats.frameDuration)

	// Calculate statistics
	r.calculateFrameTimeStats()

	// Update GPU memory usage
	r.updateGPUMemoryUsage()

	// Store last frame duration for next frame
	r.profilingStats.lastFrameDuration = r.profilingStats.frameDuration

	// Unaccounted is computed at HUD time from renderer-local tracked buckets
}

// resetFrameCounters resets per-frame counters
func (r *Renderer) resetFrameCounters() {
	r.profilingStats.totalDrawCalls = 0
	r.profilingStats.totalVertices = 0
	r.profilingStats.totalTriangles = 0
	r.profilingStats.visibleChunks = 0
	r.profilingStats.culledChunks = 0
	r.profilingStats.meshedChunks = 0
	r.profilingStats.frustumCullTime = 0
	r.profilingStats.meshingTime = 0
	r.profilingStats.shaderBindTime = 0
	r.profilingStats.vaoBindTime = 0
}

// calculateFrameTimeStats calculates frame time statistics
func (r *Renderer) calculateFrameTimeStats() {
	if len(r.profilingStats.frameTimeHistory) == 0 {
		return
	}

	var total time.Duration
	r.profilingStats.maxFrameTime = 0
	r.profilingStats.minFrameTime = r.profilingStats.frameTimeHistory[0]

	for _, duration := range r.profilingStats.frameTimeHistory {
		total += duration
		if duration > r.profilingStats.maxFrameTime {
			r.profilingStats.maxFrameTime = duration
		}
		if duration < r.profilingStats.minFrameTime {
			r.profilingStats.minFrameTime = duration
		}
	}

	r.profilingStats.avgFrameTime = total / time.Duration(len(r.profilingStats.frameTimeHistory))
}

// updateGPUMemoryUsage queries OpenGL for current memory usage
func (r *Renderer) updateGPUMemoryUsage() {
	// Note: These extensions may not be available on all systems
	// We'll use safe fallbacks if they're not supported

	// Try to get GPU memory info (NVidia)
	if gl.GetString(gl.EXTENSIONS) != nil {
		extensions := gl.GoStr(gl.GetString(gl.EXTENSIONS))
		if strings.Contains(extensions, "GL_NVX_gpu_memory_info") {
			var totalMemory int32
			gl.GetIntegerv(0x9048, &totalMemory)                        // GL_GPU_MEMORY_INFO_TOTAL_AVAILABLE_MEMORY_NVX
			r.profilingStats.gpuMemoryUsage = int64(totalMemory) * 1024 // Convert KB to bytes
		}
	}

	// For now, we'll estimate based on our buffers and textures
	r.profilingStats.bufferMemoryUsage = r.estimateBufferMemoryUsage()
	r.profilingStats.textureMemoryUsage = r.estimateTextureMemoryUsage()
}

// estimateBufferMemoryUsage estimates memory used by OpenGL buffers
func (r *Renderer) estimateBufferMemoryUsage() int64 {
	var total int64

	// Estimate chunk mesh buffers
	for _, mesh := range r.chunkMeshes {
		if mesh != nil && mesh.vertexCount > 0 {
			// Count only once via atlas below
		}
	}
	// Atlas buffer
	if r.atlasCapacityBytes > 0 {
		total += int64(r.atlasCapacityBytes)
	}

	// Add static buffers
	total += int64(len(world.CubeVertices)) * 4
	total += int64(len(world.CubeWireframeVertices)) * 4
	total += int64(len(CrosshairVertices)) * 4
	total += int64(len(DirectionVertices)) * 4

	return total
}

// estimateTextureMemoryUsage estimates memory used by textures
func (r *Renderer) estimateTextureMemoryUsage() int64 {
	var total int64

	// Font atlas texture
	if r.fontAtlas != nil {
		// Estimate based on atlas dimensions
		total += int64(r.fontAtlas.AtlasW * r.fontAtlas.AtlasH * 4) // RGBA
	}

	return total
}

// trackDrawCall increments draw call counter and tracks vertex count
func (r *Renderer) trackDrawCall(vertexCount int) {
	r.profilingStats.totalDrawCalls++
	r.profilingStats.totalVertices += vertexCount
	r.profilingStats.totalTriangles += vertexCount / 3
}

func (r *Renderer) renderBlocks(w *world.World, p *player.Player, view, projection mgl32.Mat4) {
	func() {
		defer profiling.Track("renderer.renderBlocks.shaderSetup")()
		// Track shader binding time (only the Use() call)
		sbStart := time.Now()
		r.mainShader.Use()
		r.profilingStats.shaderBindTime += time.Since(sbStart)

		r.mainShader.SetMatrix4("proj", &projection[0])
		r.mainShader.SetMatrix4("view", &view[0])

		// Set light direction
		light := mgl32.Vec3{0.3, 1.0, 0.3}.Normalize()
		r.mainShader.SetVector3("lightDir", light.X(), light.Y(), light.Z())
	}()

	// Draw greedy-meshed chunks that intersect the camera frustum
	planes := func() [6]plane {
		defer profiling.Track("renderer.renderBlocks.frustumSetup")()
		// Check if we can reuse cached frustum planes
		const matrixEpsilon = 0.001
		canReuseFrustum := r.frustumCacheValid &&
			matrixNearEqual(r.cachedViewMatrix, view, matrixEpsilon) &&
			matrixNearEqual(r.cachedProjMatrix, projection, matrixEpsilon)

		var planes [6]plane
		if canReuseFrustum {
			planes = r.cachedFrustumPlanes
		} else {
			clip := projection.Mul4(view)
			planes = extractFrustumPlanes(clip)

			// Cache the planes and matrices
			r.cachedFrustumPlanes = planes
			r.cachedViewMatrix = view
			r.cachedProjMatrix = projection
			r.frustumCacheValid = true
		}

		return planes
	}()

	nearRadiusChunks, nearBuildBudget, farBuildBudget, maxRenderRadiusChunks, pcx, pcz := func() (int, int, int, int, int, int) {
		defer profiling.Track("renderer.renderBlocks.initParams")()
		nearRadiusChunks := 12
		// Per-frame build budgets to avoid long stalls on first frames
		nearBuildBudget := 8
		farBuildBudget := 12

		// Hard cap for render radius to shrink candidate set pre-cull/sort
		maxRenderRadiusChunks := 24

		px := int(math.Floor(float64(p.Position[0])))
		pz := int(math.Floor(float64(p.Position[2])))
		pcx := px / world.ChunkSizeX
		pcz := pz / world.ChunkSizeZ
		return nearRadiusChunks, nearBuildBudget, farBuildBudget, maxRenderRadiusChunks, pcx, pcz
	}()

	// Track frustum culling time
	frustumStart := time.Now()

	// Collect visible chunks with optimized frustum culling
	var visible []world.ChunkWithCoord
	{
		stop := profiling.Track("renderer.renderBlocks.collectVisible")
		all := w.GetAllChunks()
		if cap(r.visibleScratch) < len(all) {
			r.visibleScratch = make([]world.ChunkWithCoord, 0, len(all))
		} else {
			r.visibleScratch = r.visibleScratch[:0]
		}
		visible = r.visibleScratch

		// Pre-calculate common values to avoid repeated calculations
		maxRadiusSq := maxRenderRadiusChunks * maxRenderRadiusChunks
		chunkSizeXf := float32(world.ChunkSizeX)
		chunkSizeYf := float32(world.ChunkSizeY)
		chunkSizeZf := float32(world.ChunkSizeZ)
		margin := r.frustumMargin

		for _, cc := range all {
			// Early reject by chunk-distance radius to avoid plane tests for far chunks
			dxr := cc.Coord.X - pcx
			dzr := cc.Coord.Z - pcz
			if dxr*dxr+dzr*dzr > maxRadiusSq {
				continue
			}

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

	// Update frustum culling statistics
	func() {
		defer profiling.Track("renderer.renderBlocks.updateStats")()
		r.profilingStats.frustumCullTime = time.Since(frustumStart)
		r.profilingStats.visibleChunks = len(visible)
		r.profilingStats.culledChunks = len(w.GetAllChunks()) - len(visible)
		// Removed sort step to save CPU; near/far budgets still prioritize nearby chunks
	}()

	// Phase 1: ensure meshes per budgets (prepare data for columns)
	func() {
		defer profiling.Track("renderer.renderBlocks.ensureMeshes")()
		nearRadiusSq := nearRadiusChunks * nearRadiusChunks
		for _, cc := range visible {
			coord := cc.Coord
			ch := cc.Chunk
			if ch == nil {
				continue
			}
			dx := coord.X - pcx
			dz := coord.Z - pcz
			isNear := dx*dx+dz*dz <= nearRadiusSq
			existing := r.chunkMeshes[coord]
			needsBuild := existing == nil || ch.IsDirty()
			if needsBuild {
				if isNear && nearBuildBudget > 0 {
					_ = r.ensureChunkMesh(w, coord, ch)
					nearBuildBudget--
				} else if !isNear && farBuildBudget > 0 {
					_ = r.ensureChunkMesh(w, coord, ch)
					farBuildBudget--
				}
			}
		}
	}()

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
		for k := range colSet {
			_ = r.ensureColumnMeshForXZ(k.x, k.z)
		}
		// Build draw lists after any atlas growth to avoid stale offsets
		cols := make([]*columnMesh, 0, len(colSet))
		drawnCols := make(map[[2]int]struct{}, len(colSet))
		for k := range colSet {
			key := [2]int{k.x, k.z}
			col := r.columnMeshes[key]
			if col != nil && !col.dirty && col.vertexCount > 0 && col.firstFloat >= 0 {
				cols = append(cols, col)
				drawnCols[key] = struct{}{}
			}
		}
		// Fallback per-chunk for columns not drawn
		fallbackChunks := make([]*chunkMesh, 0, len(visible))
		for _, vc := range visible {
			key := [2]int{vc.Coord.X, vc.Coord.Z}
			if _, ok := drawnCols[key]; ok {
				continue
			}
			if m := r.chunkMeshes[vc.Coord]; m != nil && m.vertexCount > 0 && m.firstFloat >= 0 {
				fallbackChunks = append(fallbackChunks, m)
			}
		}
		// Draw ready columns
		if len(cols) > 0 {
			if cap(r.firstsScratch) < len(cols) {
				r.firstsScratch = make([]int32, len(cols))
				r.countsScratch = make([]int32, len(cols))
			}
			firsts := r.firstsScratch[:0]
			counts := r.countsScratch[:0]
			for _, c := range cols {
				if c.firstFloat < 0 || c.vertexCount <= 0 || c.dirty {
					continue
				}
				firsts = append(firsts, int32(c.firstFloat/6))
				counts = append(counts, c.vertexCount)
			}
			if len(counts) > 0 {
				gl.BindVertexArray(r.atlasVAO)
				sum := 0
				for i := 0; i < len(counts); i++ {
					sum += int(counts[i])
				}
				r.trackDrawCall(sum)
				gl.MultiDrawArrays(gl.TRIANGLES, &firsts[0], &counts[0], int32(len(counts)))
			}
		}
		// Draw fallback chunks (if any) to avoid pop-in on first sight
		if len(fallbackChunks) > 0 {
			if cap(r.firstsScratch) < len(fallbackChunks) {
				r.firstsScratch = make([]int32, len(fallbackChunks))
				r.countsScratch = make([]int32, len(fallbackChunks))
			}
			firsts := r.firstsScratch[:0]
			counts := r.countsScratch[:0]
			for _, m := range fallbackChunks {
				if m.firstFloat < 0 || m.vertexCount <= 0 {
					continue
				}
				firsts = append(firsts, int32(m.firstFloat/6))
				counts = append(counts, m.vertexCount)
			}
			if len(counts) > 0 {
				gl.BindVertexArray(r.atlasVAO)
				sum := 0
				for i := 0; i < len(counts); i++ {
					sum += int(counts[i])
				}
				r.trackDrawCall(sum)
				gl.MultiDrawArrays(gl.TRIANGLES, &firsts[0], &counts[0], int32(len(counts)))
			}
		}
	}()
}

// plane represents a normalized frustum plane ax + by + cz + d = 0
type plane struct {
	a, b, c, d float32
}

// extractFrustumPlanes builds six planes from the combined projection*view matrix.
// Planes are returned in order: left, right, bottom, top, near, far.
func extractFrustumPlanes(clip mgl32.Mat4) [6]plane {
	// Matrix is in column-major order in mgl32
	m00, m01, m02, m03 := clip[0], clip[4], clip[8], clip[12]
	m10, m11, m12, m13 := clip[1], clip[5], clip[9], clip[13]
	m20, m21, m22, m23 := clip[2], clip[6], clip[10], clip[14]
	m30, m31, m32, m33 := clip[3], clip[7], clip[11], clip[15]

	pl := [6]plane{}
	// Left  = m3 + m0
	pl[0] = normalizePlane(plane{m30 + m00, m31 + m01, m32 + m02, m33 + m03})
	// Right = m3 - m0
	pl[1] = normalizePlane(plane{m30 - m00, m31 - m01, m32 - m02, m33 - m03})
	// Bottom = m3 + m1
	pl[2] = normalizePlane(plane{m30 + m10, m31 + m11, m32 + m12, m33 + m13})
	// Top = m3 - m1
	pl[3] = normalizePlane(plane{m30 - m10, m31 - m11, m32 - m12, m33 - m13})
	// Near = m3 + m2
	pl[4] = normalizePlane(plane{m30 + m20, m31 + m21, m32 + m22, m33 + m23})
	// Far = m3 - m2
	pl[5] = normalizePlane(plane{m30 - m20, m31 - m21, m32 - m22, m33 - m23})
	return pl
}

func normalizePlane(p plane) plane {
	len := float32(math.Sqrt(float64(p.a*p.a + p.b*p.b + p.c*p.c)))
	if len == 0 {
		return p
	}
	return plane{p.a / len, p.b / len, p.c / len, p.d / len}
}

// aabbIntersectsFrustumPlanes tests AABB against precomputed planes.
func aabbIntersectsFrustumPlanes(min, max mgl32.Vec3, planes [6]plane) bool {
	for i := 0; i < 6; i++ {
		p := planes[i]
		// Select the positive vertex for this plane normal
		px := max.X()
		if p.a < 0 {
			px = min.X()
		}
		py := max.Y()
		if p.b < 0 {
			py = min.Y()
		}
		pz := max.Z()
		if p.c < 0 {
			pz = min.Z()
		}
		// If positive vertex is outside, AABB is outside
		if p.a*px+p.b*py+p.c*pz+p.d < 0 {
			return false
		}
	}
	return true
}

// aabbIntersectsFrustumPlanesF is a float-optimized variant to avoid Vec3 calls in hot paths
func aabbIntersectsFrustumPlanesF(minx, miny, minz, maxx, maxy, maxz float32, planes [6]plane) bool {
	// Unrolled loop for better performance - frustum culling is called very frequently
	p := planes[0] // left
	px := maxx
	if p.a < 0 {
		px = minx
	}
	py := maxy
	if p.b < 0 {
		py = miny
	}
	pz := maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[1] // right
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[2] // bottom
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[3] // top
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[4] // near
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	p = planes[5] // far
	px = maxx
	if p.a < 0 {
		px = minx
	}
	py = maxy
	if p.b < 0 {
		py = miny
	}
	pz = maxz
	if p.c < 0 {
		pz = minz
	}
	if p.a*px+p.b*py+p.c*pz+p.d < 0 {
		return false
	}

	return true
}

// matrixNearEqual compares two matrices for approximate equality within epsilon
func matrixNearEqual(a, b mgl32.Mat4, epsilon float32) bool {
	for i := 0; i < 16; i++ {
		if float32(math.Abs(float64(a[i]-b[i]))) > epsilon {
			return false
		}
	}
	return true
}

// PruneMeshesByWorld removes cached meshes that are not in the world anymore or beyond a radius from center.
// Returns number of meshes freed.
func (r *Renderer) PruneMeshesByWorld(w *world.World, center mgl32.Vec3, radiusChunks int) int {
	retain := make(map[world.ChunkCoord]struct{})
	all := w.GetAllChunks()
	for _, cc := range all {
		retain[cc.Coord] = struct{}{}
	}
	cx := int(math.Floor(float64(center[0]))) / world.ChunkSizeX
	cz := int(math.Floor(float64(center[2]))) / world.ChunkSizeZ

	freed := 0
	for coord, m := range r.chunkMeshes {
		// Keep if present and within radius
		_, present := retain[coord]
		dx := coord.X - cx
		dz := coord.Z - cz
		if !present || dx*dx+dz*dz > radiusChunks*radiusChunks {
			if m != nil {
				if m.vbo != 0 {
					gl.DeleteBuffers(1, &m.vbo)
				}
				if m.vao != 0 {
					gl.DeleteVertexArrays(1, &m.vao)
				}
			}
			delete(r.chunkMeshes, coord)
			freed++
		}
	}
	return freed
}

func (r *Renderer) ensureChunkMesh(w *world.World, coord world.ChunkCoord, ch *world.Chunk) *chunkMesh {
	defer profiling.Track("renderer.renderBlocks.ensureMeshes.build")()
	if ch == nil {
		return nil
	}
	existing := r.chunkMeshes[coord]
	// Rebuild if missing or dirty
	if existing == nil || ch.IsDirty() {
		// Track meshing time
		meshingStart := time.Now()
		verts := meshing.BuildGreedyMeshForChunk(w, ch)
		r.profilingStats.meshingTime += time.Since(meshingStart)
		r.profilingStats.meshedChunks++
		// Create or update GL resources
		if existing == nil {
			existing = &chunkMesh{}
			gl.GenVertexArrays(1, &existing.vao)
			gl.GenBuffers(1, &existing.vbo)
			// Setup VAO attribute layout (pos.xyz, normal.xyz)
			gl.BindVertexArray(existing.vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, existing.vbo)
			gl.EnableVertexAttribArray(0)
			gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
			gl.EnableVertexAttribArray(1)
			gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
		} else {
			gl.BindVertexArray(existing.vao)
			gl.BindBuffer(gl.ARRAY_BUFFER, existing.vbo)
		}
		if len(verts) > 0 {
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.STATIC_DRAW)
			existing.vertexCount = int32(len(verts) / 6)
			// Keep CPU copy for atlas updates
			existing.cpuVerts = verts
			// Append/update in atlas
			r.appendOrUpdateAtlas(coord, existing)
			// Invalidate combined column mesh
			key := [2]int{coord.X, coord.Z}
			if col := r.columnMeshes[key]; col != nil {
				col.dirty = true
			}
		} else {
			existing.vertexCount = 0
			existing.cpuVerts = nil
			// Still upload zero to keep state valid
			gl.BufferData(gl.ARRAY_BUFFER, 0, nil, gl.STATIC_DRAW)
		}
		r.chunkMeshes[coord] = existing
		ch.SetClean()
	}
	return existing
}

// growOrAllocBatch removed: batching path eliminated

// buildBatchFromMeshes removed: batching path eliminated

func (r *Renderer) renderHighlightedBlock(blockPos [3]int, view, projection mgl32.Mat4) {
	// Track shader binding time (only the Use() call)
	sbStart := time.Now()
	r.wireframeShader.Use()
	r.profilingStats.shaderBindTime += time.Since(sbStart)

	r.wireframeShader.SetMatrix4("proj", &projection[0])
	r.wireframeShader.SetMatrix4("view", &view[0])

	// Create model matrix for the highlighted block
	model := mgl32.Translate3D(
		float32(blockPos[0]),
		float32(blockPos[1]),
		float32(blockPos[2]),
	).Mul4(mgl32.Scale3D(1.01, 1.01, 1.01))

	r.wireframeShader.SetMatrix4("model", &model[0])
	r.wireframeShader.SetVector3("color", 0.0, 0.0, 0.0) // Black outline

	// Track VAO binding time (only the BindVertexArray call)
	vbStart := time.Now()
	gl.BindVertexArray(r.wireframeVAO)
	r.profilingStats.vaoBindTime += time.Since(vbStart)
	gl.LineWidth(2.0)

	// Track draw call
	r.trackDrawCall(len(world.CubeWireframeVertices))
	gl.DrawArrays(gl.LINES, 0, int32(len(world.CubeWireframeVertices)/3))
}

func (r *Renderer) renderCrosshair() {
	// Track shader binding time (only the Use() call)
	sbStart := time.Now()
	r.crosshairShader.Use()
	r.profilingStats.shaderBindTime += time.Since(sbStart)

	aspectRatio := r.camera.AspectRatio

	r.crosshairShader.SetFloat("aspectRatio", aspectRatio)

	// Track VAO binding time (only the BindVertexArray call)
	vbStart := time.Now()
	gl.BindVertexArray(r.crosshairVAO)
	r.profilingStats.vaoBindTime += time.Since(vbStart)
	gl.LineWidth(2.0)

	// Track draw call
	r.trackDrawCall(4)
	gl.DrawArrays(gl.LINES, 0, 4)
}

// drawDirectionLetter draws the specified direction letter
func (r *Renderer) drawDirectionLetter(letter string) {
	// Select the appropriate letter vertices
	var vertices []float32
	switch letter {
	case "N":
		vertices = LetterN
	case "E":
		vertices = LetterE
	case "S":
		vertices = LetterS
	case "W":
		vertices = LetterW
	default:
		return // Unknown letter
	}

	// Update the VBO with the letter vertices
	gl.BindBuffer(gl.ARRAY_BUFFER, r.letterVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	// Draw the letter
	// Track VAO binding time (only the BindVertexArray call)
	vbStart := time.Now()
	gl.BindVertexArray(r.letterVAO)
	r.profilingStats.vaoBindTime += time.Since(vbStart)
	gl.LineWidth(2.0)

	// Track draw call
	r.trackDrawCall(len(vertices))
	gl.DrawArrays(gl.LINES, 0, int32(len(vertices)/2))
}

func (r *Renderer) renderDirection(p *player.Player) {
	// Track shader binding time (only the Use() call)
	sbStart := time.Now()
	r.directionShader.Use()
	r.profilingStats.shaderBindTime += time.Since(sbStart)

	aspectRatio := r.camera.AspectRatio
	r.directionShader.SetFloat("aspectRatio", aspectRatio)

	// Set direction color (red)
	r.directionShader.SetVector3("directionColor", 1.0, 0.0, 0.0)

	// Position at the bottom of the screen
	r.directionShader.SetFloat("positionX", 0.0)
	r.directionShader.SetFloat("positionY", -0.85)

	// Rotate based on player's yaw (convert to radians)
	// Subtract 90 degrees because the arrow points up by default, but yaw 0 is north
	yawRadians := float32(mgl32.DegToRad(float32(p.CamYaw + 90.0)))
	r.directionShader.SetFloat("rotation", yawRadians)

	// Draw the direction indicator
	// Track VAO binding time (only the BindVertexArray call)
	vbStart := time.Now()
	gl.BindVertexArray(r.directionVAO)
	r.profilingStats.vaoBindTime += time.Since(vbStart)
	gl.LineWidth(2.0)

	// Track draw calls
	r.trackDrawCall(4)                // Arrow body
	gl.DrawArrays(gl.LINE_LOOP, 0, 4) // Draw arrow body (rectangle)
	r.trackDrawCall(3)                // Arrow head
	gl.DrawArrays(gl.LINE_LOOP, 4, 3) // Draw arrow head (triangle)

	// Determine cardinal direction based on player's yaw
	// Normalize yaw to 0-360 range
	normalizedYaw := float64(int(p.CamYaw+360) % 360)

	// Get cardinal direction letter
	var directionLetter string
	if normalizedYaw >= 315 || normalizedYaw < 45 {
		directionLetter = "E" // East
	} else if normalizedYaw >= 45 && normalizedYaw < 135 {
		directionLetter = "N" // North
	} else if normalizedYaw >= 135 && normalizedYaw < 225 {
		directionLetter = "W" // West
	} else {
		directionLetter = "S" // South
	}

	// Position for the text (above the arrow)
	r.directionShader.SetFloat("positionX", 0.0)
	r.directionShader.SetFloat("positionY", -0.75)
	r.directionShader.SetFloat("rotation", 0.0) // No rotation for text

	// Draw the direction letter
	r.drawDirectionLetter(directionLetter)
}

func (r *Renderer) renderPlayerPosition(p *player.Player) {
	if r.fontRenderer == nil {
		return
	}
	// Build text and draw at top-left
	text := fmt.Sprintf("Pos: %.2f, %.2f, %.2f", p.Position[0], p.Position[1], p.Position[2])
	x := float32(10)
	y := float32(30)
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	r.fontRenderer.Render(text, x, y, 0.7, color)

}

// computeChunkAABB returns the axis-aligned bounding box in world space for a chunk
func (r *Renderer) computeChunkAABB(coord world.ChunkCoord) (mgl32.Vec3, mgl32.Vec3) {
	min := mgl32.Vec3{
		float32(coord.X * world.ChunkSizeX),
		float32(coord.Y * world.ChunkSizeY),
		float32(coord.Z * world.ChunkSizeZ),
	}
	max := mgl32.Vec3{
		min.X() + float32(world.ChunkSizeX),
		min.Y() + float32(world.ChunkSizeY),
		min.Z() + float32(world.ChunkSizeZ),
	}
	// Inflate by frustumMargin (in blocks)
	m := r.frustumMargin
	min = mgl32.Vec3{min.X() - m, min.Y() - m, min.Z() - m}
	max = mgl32.Vec3{max.X() + m, max.Y() + m, max.Z() + m}
	return min, max
}

func (r *Renderer) RenderText(text string, x, y float32, size float32, color mgl32.Vec3) {
	r.fontRenderer.Render(text, x, y, size, color)
}

// MeasureText returns width and height in pixels for the given text at scale.
func (r *Renderer) MeasureText(text string, scale float32) (float32, float32) {
	return r.fontRenderer.Measure(text, scale)
}

// ProfilingSetLastTotalFrameDuration updates the previous frame's total CPU frame duration
// including render, SwapBuffers, and event polling. Read and written only on the main thread.
func (r *Renderer) ProfilingSetLastTotalFrameDuration(d time.Duration) {
	r.profilingStats.lastTotalFrameDuration = d
}

// ProfilingSetLastUpdateDuration sets the previous frame's update duration
func (r *Renderer) ProfilingSetLastUpdateDuration(d time.Duration) {
	r.profilingStats.lastUpdateDuration = d
}

// ProfilingSetBreakdown stores last-frame non-render breakdown values
func (r *Renderer) ProfilingSetBreakdown(player, world, glfw, hud, prune time.Duration) {
	r.profilingStats.lastPlayerDuration = player
	r.profilingStats.lastWorldDuration = world
	r.profilingStats.lastGlfwDuration = glfw
	r.profilingStats.lastHudDuration = hud
	r.profilingStats.lastPruneDuration = prune
}

// ProfilingSetPhases stores phase durations from the main loop
func (r *Renderer) ProfilingSetPhases(preRender, swapEvents time.Duration) {
	r.profilingStats.lastPreRenderDuration = preRender
	r.profilingStats.lastSwapEventsDuration = swapEvents
}

// ProfilingSetPhysics stores last-frame physics duration
func (r *Renderer) ProfilingSetPhysics(physics time.Duration) {
	r.profilingStats.lastPhysicsDuration = physics
}

// renderProfilingInfo renders the current profiling information on screen
func (r *Renderer) RenderProfilingInfo() {
	defer profiling.Track("renderer.hud")()
	lines := make([]string, 0, 32)

	// FPS
	lines = append(lines, fmt.Sprintf("FPS: %d", r.currentFPS))

	// Frame timing (renderer-local)
	tracked := profiling.SumWithPrefix("renderer.")
	frameMs := float64(r.profilingStats.frameDuration.Microseconds()) / 1000.0
	trackedMs := float64(tracked.Microseconds()) / 1000.0
	lines = append(lines, fmt.Sprintf("Frame(render): %.2fms (%.2fms avg) | Tracked(render): %.2fms",
		frameMs,
		float64(r.profilingStats.avgFrameTime.Microseconds())/1000.0,
		trackedMs))

	// Frame(update) from previous frame's main-loop update duration
	if r.profilingStats.lastUpdateDuration > 0 {
		updateMs := float64(r.profilingStats.lastUpdateDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Frame(update): %.2fms", updateMs))
	}

	// Previous frame total and overhead
	if r.profilingStats.lastTotalFrameDuration > 0 {
		totalMs := float64(r.profilingStats.lastTotalFrameDuration.Microseconds()) / 1000.0
		overheadMs := totalMs - frameMs
		if overheadMs < 0 {
			overheadMs = 0
		}
		lines = append(lines, fmt.Sprintf("Frame(total): %.2fms | Overhead(non-render): %.2fms", totalMs, overheadMs))

		// Minimal overhead breakdown
		playerMs := float64(r.profilingStats.lastPlayerDuration.Microseconds()) / 1000.0
		worldMs := float64(r.profilingStats.lastWorldDuration.Microseconds()) / 1000.0
		glfwMs := float64(r.profilingStats.lastGlfwDuration.Microseconds()) / 1000.0
		hudMs := float64(r.profilingStats.lastHudDuration.Microseconds()) / 1000.0
		pruneMs := float64(r.profilingStats.lastPruneDuration.Microseconds()) / 1000.0
		physicsMs := float64(r.profilingStats.lastPhysicsDuration.Microseconds()) / 1000.0
		otherMs := overheadMs - (playerMs + worldMs + glfwMs + hudMs + pruneMs + physicsMs)
		if otherMs < 0 {
			otherMs = 0
		}
		lines = append(lines, fmt.Sprintf("Overhead breakdown -> player: %.2fms, world: %.2fms, glfw: %.2fms, hud: %.2fms, prune: %.2fms, physics: %.2fms, other: %.2fms", playerMs, worldMs, glfwMs, hudMs, pruneMs, physicsMs, otherMs))

		// Phase durations (prev)
		preRenderMs := float64(r.profilingStats.lastPreRenderDuration.Microseconds()) / 1000.0
		swapEventsMs := float64(r.profilingStats.lastSwapEventsDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Phases -> preRender: %.2fms, swap+events: %.2fms", preRenderMs, swapEventsMs))
	}

	// Rendering statistics
	lines = append(lines, fmt.Sprintf("Draw Calls: %d", r.profilingStats.totalDrawCalls))
	lines = append(lines, fmt.Sprintf("Vertices: %d (%.1fK)", r.profilingStats.totalVertices, float64(r.profilingStats.totalVertices)/1000.0))
	lines = append(lines, fmt.Sprintf("Triangles: %d (%.1fK)", r.profilingStats.totalTriangles, float64(r.profilingStats.totalTriangles)/1000.0))

	// Chunk statistics
	lines = append(lines, fmt.Sprintf("Chunks: %d visible, %d culled", r.profilingStats.visibleChunks, r.profilingStats.culledChunks))

	// Meshing stats
	lines = append(lines, fmt.Sprintf("Meshed (this frame): %d", r.profilingStats.meshedChunks))
	// Cached mesh count with geometry
	cached := 0
	for _, m := range r.chunkMeshes {
		if m != nil && m.vertexCount > 0 {
			cached++
		}
	}
	lines = append(lines, fmt.Sprintf("Meshes cached: %d", cached))

	// Detailed renderer breakdown
	lines = append(lines, fmt.Sprintf("Frustum: %.2fms", float64(r.profilingStats.frustumCullTime.Microseconds())/1000.0))
	lines = append(lines, fmt.Sprintf("Meshing: %.2fms", float64(r.profilingStats.meshingTime.Microseconds())/1000.0))
	lines = append(lines, fmt.Sprintf("Shader Bind: %.2fms", float64(r.profilingStats.shaderBindTime.Microseconds())/1000.0))
	lines = append(lines, fmt.Sprintf("VAO Bind: %.2fms", float64(r.profilingStats.vaoBindTime.Microseconds())/1000.0))

	// Top profiling lines
	if top := profiling.TopN(10); top != "" {
		for _, line := range strings.Split(top, ", ") {
			if line != "" && !strings.Contains(line, ":0ms") && strings.HasPrefix(line, "renderer.renderBlocks") {
				lines = append(lines, line)
			}
		}
	}

	textColor := mgl32.Vec3{1.0, 1.0, 1.0}
	startY := float32(60)
	lineStep := float32(14)
	r.fontRenderer.RenderLines(lines, 10, startY, lineStep, 0.6, textColor)
}

// GetProfilingStats returns a copy of the current profiling statistics
func (r *Renderer) GetProfilingStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Frame timing
	stats["fps"] = r.currentFPS
	stats["frameDuration"] = r.profilingStats.frameDuration
	stats["avgFrameTime"] = r.profilingStats.avgFrameTime
	stats["maxFrameTime"] = r.profilingStats.maxFrameTime
	stats["minFrameTime"] = r.profilingStats.minFrameTime

	// Rendering statistics
	stats["drawCalls"] = r.profilingStats.totalDrawCalls
	stats["vertices"] = r.profilingStats.totalVertices
	stats["triangles"] = r.profilingStats.totalTriangles
	stats["visibleChunks"] = r.profilingStats.visibleChunks
	stats["culledChunks"] = r.profilingStats.culledChunks
	stats["meshedChunks"] = r.profilingStats.meshedChunks

	// Performance breakdown
	stats["frustumCullTime"] = r.profilingStats.frustumCullTime
	stats["meshingTime"] = r.profilingStats.meshingTime
	stats["shaderBindTime"] = r.profilingStats.shaderBindTime
	stats["vaoBindTime"] = r.profilingStats.vaoBindTime

	// Memory usage
	stats["bufferMemory"] = r.profilingStats.bufferMemoryUsage
	stats["textureMemory"] = r.profilingStats.textureMemoryUsage
	stats["totalGPUMemory"] = r.profilingStats.gpuMemoryUsage

	return stats
}

// ResetProfilingStats resets all profiling statistics to zero
func (r *Renderer) ResetProfilingStats() {
	r.profilingStats.frameTimeHistory = r.profilingStats.frameTimeHistory[:0]
	r.profilingStats.maxFrameTime = 0
	r.profilingStats.minFrameTime = 0
	r.profilingStats.avgFrameTime = 0
	r.profilingStats.lastFrameDuration = 0
}
