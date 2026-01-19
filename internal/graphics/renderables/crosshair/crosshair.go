package crosshair

import (
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/profiling"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	ShadersDir  = "assets/shaders/crosshair"
	TexturesDir = "assets/textures/gui"
)

var (
	VertShader   = filepath.Join(ShadersDir, "crosshair.vert")
	FragShader   = filepath.Join(ShadersDir, "crosshair.frag")
	IconsTexture = filepath.Join(TexturesDir, "icons.png")
)

// Crosshair vertices with position (x,y) and texture coordinates (u,v)
// Positioned at screen center with 32x32 pixel size (2x scale)
var Vertices = []float32{
	// pos.x, pos.y, tex.u, tex.v
	-16.0, -16.0, 0.0, 0.0, // Bottom-left
	16.0, -16.0, 16.0, 0.0, // Bottom-right
	16.0, 16.0, 16.0, 16.0, // Top-right
	-16.0, 16.0, 0.0, 16.0, // Top-left
}

var Indices = []uint32{
	0, 1, 2, // First triangle
	2, 3, 0, // Second triangle
}

// Crosshair implements crosshair rendering
type Crosshair struct {
	shader    *graphics.Shader
	vao       uint32
	vbo       uint32
	ebo       uint32
	textureID uint32

	// Viewport dimensions
	width  float32
	height float32
}

// NewCrosshair creates a new crosshair renderable
func NewCrosshair() *Crosshair {
	return &Crosshair{
		width:  900,
		height: 600,
	}
}

// Init initializes the crosshair rendering system
func (c *Crosshair) Init() error {
	// Create shader
	var err error
	c.shader, err = graphics.NewShader(VertShader, FragShader)
	if err != nil {
		return err
	}

	// Load icons texture (Minecraft's GUI texture with crosshair)
	c.textureID, _, _, err = graphics.LoadTexture(IconsTexture)
	if err != nil {
		return err
	}

	// Setup VAO, VBO and EBO
	c.setupCrosshairVAO()

	return nil
}

// Render renders the crosshair
func (c *Crosshair) Render(ctx renderer.RenderContext) {
	func() {
		defer profiling.Track("renderer.renderCrosshair")()
		c.renderCrosshair(int32(c.width), int32(c.height))
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
	if c.ebo != 0 {
		gl.DeleteBuffers(1, &c.ebo)
	}
	if c.textureID != 0 {
		gl.DeleteTextures(1, &c.textureID)
	}
}

// SetViewport updates the crosshair viewport dimensions
func (c *Crosshair) SetViewport(width, height int) {
	c.width = float32(width)
	c.height = float32(height)
}

func (c *Crosshair) setupCrosshairVAO() {
	gl.GenVertexArrays(1, &c.vao)
	gl.BindVertexArray(c.vao)

	// Create VBO
	gl.GenBuffers(1, &c.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, c.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(Vertices)*4, gl.Ptr(Vertices), gl.STATIC_DRAW)

	// Create EBO
	gl.GenBuffers(1, &c.ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, c.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(Indices)*4, gl.Ptr(Indices), gl.STATIC_DRAW)

	// Position attribute (location = 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 4*4, 0)

	// Texture coordinate attribute (location = 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, 2*4)

	gl.BindVertexArray(0)
}

func (c *Crosshair) renderCrosshair(screenWidth, screenHeight int32) {
	// Enable blending for crosshair
	gl.Enable(gl.BLEND)

	// Minecraft's special blend function for crosshair visibility
	// GL_ONE_MINUS_DST_COLOR (775) and GL_ONE_MINUS_SRC_ALPHA (769)
	// This creates an inverse effect that makes crosshair visible on any background
	gl.BlendFuncSeparate(gl.ONE_MINUS_DST_COLOR, gl.ONE_MINUS_SRC_ALPHA, gl.ONE, gl.ZERO)

	c.shader.Use()

	// Set screen dimensions for proper positioning
	c.shader.SetInt("screenWidth", screenWidth)
	c.shader.SetInt("screenHeight", screenHeight)
	c.shader.SetInt("crosshairTexture", 0)

	// Bind texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, c.textureID)

	// Draw crosshair
	gl.BindVertexArray(c.vao)
	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)

	gl.Disable(gl.BLEND)
}
