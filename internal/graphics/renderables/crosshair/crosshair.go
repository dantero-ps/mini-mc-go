package crosshair

import (
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/profiling"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	ShadersDir = "assets/shaders/crosshair"
)

var (
	VertShader = filepath.Join(ShadersDir, "crosshair.vert")
	FragShader = filepath.Join(ShadersDir, "crosshair.frag")
)

var Vertices = []float32{
	-0.02, 0.0,
	0.02, 0.0,
	0.0, -0.02,
	0.0, 0.02,
}

// Crosshair implements crosshair rendering
type Crosshair struct {
	shader *graphics.Shader
	vao    uint32
	vbo    uint32
}

// NewCrosshair creates a new crosshair renderable
func NewCrosshair() *Crosshair {
	return &Crosshair{}
}

// Init initializes the crosshair rendering system
func (c *Crosshair) Init() error {
	// Create shader
	var err error
	c.shader, err = graphics.NewShader(VertShader, FragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO
	c.setupCrosshairVAO()

	return nil
}

// Render renders the crosshair
func (c *Crosshair) Render(ctx renderer.RenderContext) {
	func() {
		defer profiling.Track("renderer.renderCrosshair")()
		c.renderCrosshair(ctx.Camera.AspectRatio)
	}()
}

// Dispose cleans up OpenGL resources
func (c *Crosshair) Dispose() {
	if c.vao != 0 {
		gl.DeleteVertexArrays(1, &c.vao)
	}
	if c.vbo != 0 {
		gl.DeleteBuffers(1, &c.vbo)
	}
}

func (c *Crosshair) setupCrosshairVAO() {
	gl.GenVertexArrays(1, &c.vao)
	gl.BindVertexArray(c.vao)

	gl.GenBuffers(1, &c.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(Vertices)*4, gl.Ptr(Vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 2*4, 0)
}

func (c *Crosshair) renderCrosshair(aspectRatio float32) {
	c.shader.Use()
	c.shader.SetFloat("aspectRatio", aspectRatio)

	gl.BindVertexArray(c.vao)
	gl.LineWidth(1.0)
	gl.DrawArrays(gl.LINES, 0, 4)
}
