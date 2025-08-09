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
)

var CrosshairVertices = []float32{
	-0.02, 0.0,
	0.02, 0.0,
	0.0, -0.02,
	0.0, 0.02,
}

type Renderer struct {
	mainShader      *Shader
	simpleShader    *Shader
	crosshairShader *Shader
	camera          *Camera

	// OpenGL objects
	blockVAO     uint32
	blockVBO     uint32
	instanceVBO  uint32
	wireframeVAO uint32
	wireframeVBO uint32
	crosshairVAO uint32
	crosshairVBO uint32
}

func NewRenderer() (*Renderer, error) {
	if err := gl.Init(); err != nil {
		return nil, err
	}

	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

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

	// Create camera
	camera := NewCamera(WinWidth, WinHeight)

	renderer := &Renderer{
		mainShader:      mainShader,
		simpleShader:    simpleShader,
		crosshairShader: crosshairShader,
		camera:          camera,
	}

	// Initialize VAOs and VBOs
	renderer.setupBlockVAO()
	renderer.setupWireframeVAO()
	renderer.setupCrosshairVAO()

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

func (r *Renderer) Render(w *world.World, p *player.Player) {
	// Clear the screen
	gl.ClearColor(0.53, 0.81, 0.92, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// Get view and projection matrices
	view := p.GetViewMatrix()
	projection := r.camera.GetProjectionMatrix()

	// Render blocks
	r.renderBlocks(w, view, projection)

	// Render highlighted block
	if p.HasHoveredBlock {
		r.renderHighlightedBlock(p.HoveredBlock, view, projection)
	}

	// Render crosshair
	r.renderCrosshair()
}

func (r *Renderer) renderBlocks(w *world.World, view, projection mgl32.Mat4) {
	r.mainShader.Use()

	r.mainShader.SetMatrix4("proj", &projection[0])
	r.mainShader.SetMatrix4("view", &view[0])

	// Set light direction
	light := mgl32.Vec3{0.5, 1.0, 0.3}.Normalize()
	r.mainShader.SetVector3("lightDir", light.X(), light.Y(), light.Z())

	// Set block color
	color := world.GetBlockColor()
	r.mainShader.SetVector3("color", color.X(), color.Y(), color.Z())

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
