package ui

import (
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/ui"
	WinWidth   = 900
	WinHeight  = 600
)

var (
	UIVertShader = filepath.Join(ShadersDir, "ui.vert")
	UIFragShader = filepath.Join(ShadersDir, "ui.frag")
)

// UI implements UI rendering for rectangles and text
type UI struct {
	shader *graphics.Shader
	vao    uint32
	vbo    uint32
}

// NewUI creates a new UI renderable
func NewUI() *UI {
	return &UI{}
}

// Init initializes the UI rendering system
func (u *UI) Init() error {
	// Create shader from external files
	var err error
	u.shader, err = graphics.NewShader(UIVertShader, UIFragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO
	gl.GenVertexArrays(1, &u.vao)
	gl.GenBuffers(1, &u.vbo)
	gl.BindVertexArray(u.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 6*2*4, nil, gl.DYNAMIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	return nil
}

// Render handles UI rendering - this is mainly called for text and HUD rendering
func (u *UI) Render(ctx renderer.RenderContext) {
	// UI rendering is mainly handled by HUD feature now
	// This could be extended for other UI elements
}

// Dispose cleans up OpenGL resources
func (u *UI) Dispose() {
	if u.vao != 0 {
		gl.DeleteVertexArrays(1, &u.vao)
	}
	if u.vbo != 0 {
		gl.DeleteBuffers(1, &u.vbo)
	}
}

// DrawFilledRect draws a screen-space rectangle (pixels, top-left origin) with RGBA color.
func (u *UI) DrawFilledRect(x, y, w, h float32, color mgl32.Vec3, alpha float32) {
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

	u.shader.Use()
	loc := gl.GetUniformLocation(u.shader.ID, gl.Str("uColor\x00"))
	gl.Uniform4f(loc, color.X(), color.Y(), color.Z(), alpha)

	gl.BindVertexArray(u.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.vbo)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(verts)*4, gl.Ptr(verts))
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
}
