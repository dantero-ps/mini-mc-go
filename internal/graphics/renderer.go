package graphics

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"mini-mc/internal/player"
	"mini-mc/internal/world"
	"path/filepath"
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
	simpleShader    *Shader
	crosshairShader *Shader
	directionShader *Shader
	camera          *Camera

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
}

func NewRenderer() (*Renderer, error) {
	if err := gl.Init(); err != nil {
		return nil, err
	}

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)

	// Create shaders from files
	mainVertPath := filepath.Join(ShadersDir, MainVertShader)
	mainFragPath := filepath.Join(ShadersDir, MainFragShader)
	mainShader, err := NewShader(mainVertPath, mainFragPath)
	if err != nil {
		return nil, err
	}

	wireframeVertPath := filepath.Join(ShadersDir, WireframeVertShader)
	wireframeFragPath := filepath.Join(ShadersDir, WireframeFragShader)
	simpleShader, err := NewShader(wireframeVertPath, wireframeFragPath)
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

	renderer := &Renderer{
		mainShader:      mainShader,
		simpleShader:    simpleShader,
		crosshairShader: crosshairShader,
		directionShader: directionShader,
		camera:          camera,
	}

	// Initialize VAOs and VBOs
	renderer.setupBlockVAO()
	renderer.setupWireframeVAO()
	renderer.setupCrosshairVAO()
	renderer.setupDirectionVAO()
	renderer.setupLetterVAO()

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
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(3*4))

	// Instance buffer
	gl.GenBuffers(1, &r.instanceVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.instanceVBO)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
	gl.VertexAttribDivisor(2, 1)
}

func (r *Renderer) setupWireframeVAO() {
	gl.GenVertexArrays(1, &r.wireframeVAO)
	gl.BindVertexArray(r.wireframeVAO)

	gl.GenBuffers(1, &r.wireframeVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.wireframeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(world.CubeWireframeVertices)*4, gl.Ptr(world.CubeWireframeVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 3*4, gl.PtrOffset(0))
}

func (r *Renderer) setupCrosshairVAO() {
	gl.GenVertexArrays(1, &r.crosshairVAO)
	gl.BindVertexArray(r.crosshairVAO)

	gl.GenBuffers(1, &r.crosshairVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.crosshairVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(CrosshairVertices)*4, gl.Ptr(CrosshairVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
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

func (r *Renderer) Render(w *world.World, p *player.Player) {
	// Clear the screen
	gl.ClearColor(0.53, 0.81, 0.92, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

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

	gl.BindVertexArray(r.blockVAO)
	positions := w.GetActiveBlocks()
	if len(positions) > 0 {
		gl.BindBuffer(gl.ARRAY_BUFFER, r.instanceVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(positions)*3*4, gl.Ptr(positions), gl.DYNAMIC_DRAW)
		gl.DrawArraysInstanced(gl.TRIANGLES, 0, int32(len(world.CubeVertices)/6), int32(len(positions)))
	}
}

func (r *Renderer) renderHighlightedBlock(blockPos [3]int, view, projection mgl32.Mat4) {
	r.simpleShader.Use()

	r.simpleShader.SetMatrix4("proj", &projection[0])
	r.simpleShader.SetMatrix4("view", &view[0])

	// Create model matrix for the highlighted block
	model := mgl32.Translate3D(
		float32(blockPos[0]),
		float32(blockPos[1]),
		float32(blockPos[2]),
	).Mul4(mgl32.Scale3D(1.01, 1.01, 1.01))

	r.simpleShader.SetMatrix4("model", &model[0])
	r.simpleShader.SetVector3("color", 0.0, 0.0, 0.0) // Black outline

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
