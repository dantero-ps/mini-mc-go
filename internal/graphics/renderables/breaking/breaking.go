package breaking

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/profiling"
	"os"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/breaking"
)

var (
	BreakingVertShader = filepath.Join(ShadersDir, "breaking.vert")
	BreakingFragShader = filepath.Join(ShadersDir, "breaking.frag")

	// Cube vertices with UVs for breaking overlay
	// Position (x,y,z) + UV (u,v)
	breakingVertices = []float32{
		// NORTH
		-0.5, -0.5, 0.5, 0, 1,
		0.5, -0.5, 0.5, 1, 1,
		0.5, 0.5, 0.5, 1, 0,
		0.5, 0.5, 0.5, 1, 0,
		-0.5, 0.5, 0.5, 0, 0,
		-0.5, -0.5, 0.5, 0, 1,

		// SOUTH
		0.5, -0.5, -0.5, 0, 1,
		-0.5, -0.5, -0.5, 1, 1,
		-0.5, 0.5, -0.5, 1, 0,
		-0.5, 0.5, -0.5, 1, 0,
		0.5, 0.5, -0.5, 0, 0,
		0.5, -0.5, -0.5, 0, 1,

		// WEST
		-0.5, -0.5, -0.5, 0, 1,
		-0.5, -0.5, 0.5, 1, 1,
		-0.5, 0.5, 0.5, 1, 0,
		-0.5, 0.5, 0.5, 1, 0,
		-0.5, 0.5, -0.5, 0, 0,
		-0.5, -0.5, -0.5, 0, 1,

		// EAST
		0.5, -0.5, 0.5, 0, 1,
		0.5, -0.5, -0.5, 1, 1,
		0.5, 0.5, -0.5, 1, 0,
		0.5, 0.5, -0.5, 1, 0,
		0.5, 0.5, 0.5, 0, 0,
		0.5, -0.5, 0.5, 0, 1,

		// TOP
		-0.5, 0.5, 0.5, 0, 1,
		0.5, 0.5, 0.5, 1, 1,
		0.5, 0.5, -0.5, 1, 0,
		0.5, 0.5, -0.5, 1, 0,
		-0.5, 0.5, -0.5, 0, 0,
		-0.5, 0.5, 0.5, 0, 1,

		// BOTTOM
		-0.5, -0.5, -0.5, 0, 1,
		0.5, -0.5, -0.5, 1, 1,
		0.5, -0.5, 0.5, 1, 0,
		0.5, -0.5, 0.5, 1, 0,
		-0.5, -0.5, 0.5, 0, 0,
		-0.5, -0.5, -0.5, 0, 1,
	}
)

// Breaking implements rendering for the block breaking animation
type Breaking struct {
	shader    *graphics.Shader
	vao       uint32
	vbo       uint32
	textureID uint32
}

// NewBreaking creates a new breaking overlay renderable
func NewBreaking() *Breaking {
	return &Breaking{}
}

// Init initializes the breaking rendering system
func (b *Breaking) Init() error {
	// Create shader
	var err error
	b.shader, err = graphics.NewShader(BreakingVertShader, BreakingFragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO
	b.setupBreakingVAO()

	// Load texture array
	if err := b.loadBreakingTextures(); err != nil {
		return err
	}

	return nil
}

func (b *Breaking) loadBreakingTextures() error {
	var images []*image.RGBA
	width, height := 0, 0

	for i := range 10 {
		path := fmt.Sprintf("assets/textures/blocks/destroy_stage_%d.png", i)
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open texture %s: %v", path, err)
		}

		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("failed to decode texture %s: %v", path, err)
		}

		rgba := image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

		if width == 0 {
			width = rgba.Bounds().Dx()
			height = rgba.Bounds().Dy()
		} else if rgba.Bounds().Dx() != width || rgba.Bounds().Dy() != height {
			return fmt.Errorf("texture %s dimensions mismatch", path)
		}

		images = append(images, rgba)
	}

	gl.GenTextures(1, &b.textureID)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, b.textureID)

	gl.TexImage3D(
		gl.TEXTURE_2D_ARRAY,
		0,
		gl.RGBA8,
		int32(width),
		int32(height),
		int32(len(images)),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil,
	)

	for i, img := range images {
		gl.TexSubImage3D(
			gl.TEXTURE_2D_ARRAY,
			0,
			0, 0, int32(i),
			int32(width),
			int32(height),
			1,
			gl.RGBA,
			gl.UNSIGNED_BYTE,
			gl.Ptr(img.Pix),
		)
	}

	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D_ARRAY, gl.TEXTURE_WRAP_T, gl.REPEAT)

	return nil
}

// Render renders the breaking overlay
func (b *Breaking) Render(ctx renderer.RenderContext) {
	if ctx.Player.IsBreaking {
		func() {
			defer profiling.Track("renderer.renderBreaking")()
			b.renderBreakingBlock(ctx.Player.BreakingBlock, ctx.Player.BreakProgress, ctx.View, ctx.Proj)
		}()
	}
}

// Dispose cleans up OpenGL resources
func (b *Breaking) Dispose() {
	if b.vao != 0 {
		gl.DeleteVertexArrays(1, &b.vao)
	}
	if b.vbo != 0 {
		gl.DeleteBuffers(1, &b.vbo)
	}
	if b.textureID != 0 {
		gl.DeleteTextures(1, &b.textureID)
	}
}

// SetViewport updates viewport dimensions (not needed for breaking)
func (b *Breaking) SetViewport(width, height int) {
	// Breaking animation doesn't need viewport dimensions
}

func (b *Breaking) setupBreakingVAO() {
	gl.GenVertexArrays(1, &b.vao)
	gl.BindVertexArray(b.vao)

	gl.GenBuffers(1, &b.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, b.vbo)

	gl.BufferData(gl.ARRAY_BUFFER, len(breakingVertices)*4, gl.Ptr(breakingVertices), gl.STATIC_DRAW)

	// Position attribute (location = 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	// TexCoord attribute (location = 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}

func (b *Breaking) renderBreakingBlock(blockPos [3]int, progress float32, view, projection mgl32.Mat4) {
	b.shader.Use()
	b.shader.SetMatrix4("proj", &projection[0])
	b.shader.SetMatrix4("view", &view[0])

	// Bind texture array
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D_ARRAY, b.textureID)
	b.shader.SetInt("breakingTexture", 0)

	// Calculate stage (0-9)
	stage := min(int(progress*10), 9)
	if stage < 0 {
		stage = 0
	}
	b.shader.SetInt("layer", int32(stage))

	// Create model matrix for the breaking overlay
	// Scale slightly larger than 1.0 to overlay the block (z-fighting fix)
	scale := float32(1.002)
	model := mgl32.Translate3D(
		float32(blockPos[0])+0.5,
		float32(blockPos[1])-0.5,
		float32(blockPos[2])+0.5,
	).Mul4(mgl32.Scale3D(scale, scale, scale))

	b.shader.SetMatrix4("model", &model[0])

	// Enable blending for transparency
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.DST_COLOR, gl.SRC_COLOR)

	// Disable depth write to prevent Z-fighting artifacts but keep depth test
	gl.DepthMask(false)

	gl.BindVertexArray(b.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, 36)
	gl.BindVertexArray(0)

	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}
