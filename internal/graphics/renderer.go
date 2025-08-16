package graphics

import (
	"math"
	"mini-mc/internal/meshing"
	"mini-mc/internal/player"
	"mini-mc/internal/world"
	"path/filepath"

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

	// Greedy meshing cache per chunk
	chunkMeshes map[world.ChunkCoord]*chunkMesh

	// Frustum culling margin in blocks (inflates AABBs before testing)
	frustumMargin float32
}

type chunkMesh struct {
	vao         uint32
	vbo         uint32
	vertexCount int32
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
	}

	// Initialize VAOs and VBOs
	renderer.setupBlockVAO()
	renderer.setupWireframeVAO()
	renderer.setupCrosshairVAO()
	renderer.setupDirectionVAO()
	renderer.setupLetterVAO()
	renderer.chunkMeshes = make(map[world.ChunkCoord]*chunkMesh)

	return renderer, nil
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

func (r *Renderer) Render(w *world.World, p *player.Player, dt float64) {
	// Clear the screen
	gl.ClearColor(0.53, 0.81, 0.92, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

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

	// Get view and projection matrices
	view := p.GetViewMatrix()
	projection := r.camera.GetProjectionMatrix()

	// Render blocks
	if player.WireframeMode {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	r.renderBlocks(w, view, projection)

	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

	// Render highlighted block
	if p.HasHoveredBlock {
		r.renderHighlightedBlock(p.HoveredBlock, view, projection)
	}

	// Render crosshair
	r.renderCrosshair()

	// Render direction indicator
	r.renderDirection(p)
}

func (r *Renderer) renderBlocks(w *world.World, view, projection mgl32.Mat4) {
	r.mainShader.Use()

	r.mainShader.SetMatrix4("proj", &projection[0])
	r.mainShader.SetMatrix4("view", &view[0])

	// Set light direction
	light := mgl32.Vec3{0.3, 1.0, 0.3}.Normalize()
	r.mainShader.SetVector3("lightDir", light.X(), light.Y(), light.Z())

	// Set face colors from Go code
	northColor := world.GetBlockColor(world.FaceNorth)
	southColor := world.GetBlockColor(world.FaceSouth)
	eastColor := world.GetBlockColor(world.FaceEast)
	westColor := world.GetBlockColor(world.FaceWest)
	topColor := world.GetBlockColor(world.FaceTop)
	bottomColor := world.GetBlockColor(world.FaceBottom)
	defaultColor := mgl32.Vec3{0.5, 0.5, 0.5} // Gray (fallback)

	r.mainShader.SetVector3("faceColorNorth", northColor.X(), northColor.Y(), northColor.Z())
	r.mainShader.SetVector3("faceColorSouth", southColor.X(), southColor.Y(), southColor.Z())
	r.mainShader.SetVector3("faceColorEast", eastColor.X(), eastColor.Y(), eastColor.Z())
	r.mainShader.SetVector3("faceColorWest", westColor.X(), westColor.Y(), westColor.Z())
	r.mainShader.SetVector3("faceColorTop", topColor.X(), topColor.Y(), topColor.Z())
	r.mainShader.SetVector3("faceColorBottom", bottomColor.X(), bottomColor.Y(), bottomColor.Z())
	r.mainShader.SetVector3("faceColorDefault", defaultColor.X(), defaultColor.Y(), defaultColor.Z())

	// Draw greedy-meshed chunks that intersect the camera frustum
	chunks := w.GetAllChunks()
	clip := projection.Mul4(view)
	for _, cc := range chunks {
		coord := cc.Coord
		ch := cc.Chunk
		if ch == nil {
			continue
		}
		min, max := r.computeChunkAABB(coord)
		if !r.aabbIntersectsFrustum(min, max, clip) {
			continue
		}
		mesh := r.ensureChunkMesh(w, coord, ch)
		if mesh == nil || mesh.vertexCount == 0 {
			continue
		}
		gl.BindVertexArray(mesh.vao)
		gl.DrawArrays(gl.TRIANGLES, 0, mesh.vertexCount)
	}
}

func (r *Renderer) ensureChunkMesh(w *world.World, coord world.ChunkCoord, ch *world.Chunk) *chunkMesh {
	if ch == nil {
		return nil
	}
	existing := r.chunkMeshes[coord]
	// Rebuild if missing or dirty
	if existing == nil || ch.IsDirty() {
		verts := meshing.BuildGreedyMeshForChunk(w, ch)
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
			gl.BufferData(gl.ARRAY_BUFFER, len(verts)*4, gl.Ptr(verts), gl.DYNAMIC_DRAW)
			existing.vertexCount = int32(len(verts) / 6)
		} else {
			existing.vertexCount = 0
			// Still upload zero to keep state valid
			gl.BufferData(gl.ARRAY_BUFFER, 0, nil, gl.DYNAMIC_DRAW)
		}
		r.chunkMeshes[coord] = existing
		ch.SetClean()
	}
	return existing
}

func (r *Renderer) renderHighlightedBlock(blockPos [3]int, view, projection mgl32.Mat4) {
	r.wireframeShader.Use()

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

	gl.BindVertexArray(r.wireframeVAO)
	gl.LineWidth(2.0)
	gl.DrawArrays(gl.LINES, 0, int32(len(world.CubeWireframeVertices)/3))
}

func (r *Renderer) renderCrosshair() {
	r.crosshairShader.Use()

	aspectRatio := r.camera.AspectRatio

	r.crosshairShader.SetFloat("aspectRatio", aspectRatio)

	gl.BindVertexArray(r.crosshairVAO)
	gl.LineWidth(2.0)
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
	gl.BindVertexArray(r.letterVAO)
	gl.LineWidth(2.0)
	gl.DrawArrays(gl.LINES, 0, int32(len(vertices)/2))
}

func (r *Renderer) renderDirection(p *player.Player) {
	r.directionShader.Use()

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
	gl.BindVertexArray(r.directionVAO)
	gl.LineWidth(2.0)
	gl.DrawArrays(gl.LINE_LOOP, 0, 4) // Draw arrow body (rectangle)
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

// aabbIntersectsFrustum tests AABB against the camera frustum using clip-space half-space tests.
// clip is projection * view.
func (r *Renderer) aabbIntersectsFrustum(min, max mgl32.Vec3, clip mgl32.Mat4) bool {
	// build 8 corners
	corners := [8]mgl32.Vec4{
		{min.X(), min.Y(), min.Z(), 1.0},
		{max.X(), min.Y(), min.Z(), 1.0},
		{min.X(), max.Y(), min.Z(), 1.0},
		{max.X(), max.Y(), min.Z(), 1.0},
		{min.X(), min.Y(), max.Z(), 1.0},
		{max.X(), min.Y(), max.Z(), 1.0},
		{min.X(), max.Y(), max.Z(), 1.0},
		{max.X(), max.Y(), max.Z(), 1.0},
	}

	// Transform corners to clip space
	var v [8]mgl32.Vec4
	for i := 0; i < 8; i++ {
		v[i] = clip.Mul4x1(corners[i])
	}

	// For each frustum plane, if all corners are outside, cull
	// Right plane:  x - w <= 0  -> outside if x - w > 0
	allOutside := true
	for i := 0; i < 8; i++ {
		if v[i].X()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}
	// Left plane: -x - w <= 0
	allOutside = true
	for i := 0; i < 8; i++ {
		if -v[i].X()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}
	// Top plane: y - w <= 0
	allOutside = true
	for i := 0; i < 8; i++ {
		if v[i].Y()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}
	// Bottom plane: -y - w <= 0
	allOutside = true
	for i := 0; i < 8; i++ {
		if -v[i].Y()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}
	// Far plane: z - w <= 0
	allOutside = true
	for i := 0; i < 8; i++ {
		if v[i].Z()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}
	// Near plane: -z - w <= 0
	allOutside = true
	for i := 0; i < 8; i++ {
		if -v[i].Z()-v[i].W() <= 0 {
			allOutside = false
			break
		}
	}
	if allOutside {
		return false
	}

	return !allOutside
}
