package wireframe

import (
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/profiling"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/wireframe"
)

var (
	WireframeVertShader = filepath.Join(ShadersDir, "wireframe.vert")
	WireframeFragShader = filepath.Join(ShadersDir, "wireframe.frag")
)

// Wireframe implements wireframe rendering for highlighted blocks
type Wireframe struct {
	shader *graphics.Shader
	vao    uint32
	vbo    uint32
}

// NewWireframe creates a new wireframe renderable
func NewWireframe() *Wireframe {
	return &Wireframe{}
}

// Init initializes the wireframe rendering system
func (w *Wireframe) Init() error {
	// Create shader
	var err error
	w.shader, err = graphics.NewShader(WireframeVertShader, WireframeFragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO
	w.setupWireframeVAO()

	return nil
}

// Render renders the wireframe for highlighted blocks
func (w *Wireframe) Render(ctx renderer.RenderContext) {
	if ctx.Player.HasHoveredBlock {
		func() {
			defer profiling.Track("renderer.renderHighlightedBlock")()
			w.renderHighlightedBlock(ctx.Player.HoveredBlock, ctx.View, ctx.Proj)
		}()
	}
}

// Dispose cleans up OpenGL resources
func (w *Wireframe) Dispose() {
	if w.vao != 0 {
		gl.DeleteVertexArrays(1, &w.vao)
	}
	if w.vbo != 0 {
		gl.DeleteBuffers(1, &w.vbo)
	}
}

func (w *Wireframe) setupWireframeVAO() {
	gl.GenVertexArrays(1, &w.vao)
	gl.BindVertexArray(w.vao)

	gl.GenBuffers(1, &w.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)

	// Cube wireframe vertices from world.CubeWireframeVertices
	vertices := []float32{
		// Front face
		-0.5, -0.5, 0.5, 0.5, -0.5, 0.5,
		0.5, -0.5, 0.5, 0.5, 0.5, 0.5,
		0.5, 0.5, 0.5, -0.5, 0.5, 0.5,
		-0.5, 0.5, 0.5, -0.5, -0.5, 0.5,

		// Back face
		-0.5, -0.5, -0.5, 0.5, -0.5, -0.5,
		0.5, -0.5, -0.5, 0.5, 0.5, -0.5,
		0.5, 0.5, -0.5, -0.5, 0.5, -0.5,
		-0.5, 0.5, -0.5, -0.5, -0.5, -0.5,

		// Connecting edges
		-0.5, -0.5, 0.5, -0.5, -0.5, -0.5,
		0.5, -0.5, 0.5, 0.5, -0.5, -0.5,
		0.5, 0.5, 0.5, 0.5, 0.5, -0.5,
		-0.5, 0.5, 0.5, -0.5, 0.5, -0.5,
	}

	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 3*4, 0)
}

func (w *Wireframe) renderHighlightedBlock(blockPos [3]int, view, projection mgl32.Mat4) {
	w.shader.Use()
	w.shader.SetMatrix4("proj", &projection[0])
	w.shader.SetMatrix4("view", &view[0])

	// Create model matrix for the highlighted block
	model := mgl32.Translate3D(
		float32(blockPos[0]),
		float32(blockPos[1]),
		float32(blockPos[2]),
	).Mul4(mgl32.Scale3D(1.01, 1.01, 1.01))

	w.shader.SetMatrix4("model", &model[0])
	w.shader.SetVector3("color", 0.0, 0.0, 0.0) // Black outline

	gl.BindVertexArray(w.vao)
	gl.LineWidth(1.0)
	gl.DrawArrays(gl.LINES, 0, 24) // 24 vertices for cube wireframe
}
