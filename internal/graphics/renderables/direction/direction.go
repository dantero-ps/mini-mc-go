package direction

import (
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	WinWidth  = 900
	WinHeight = 600
)

const (
	ShadersDir = "assets/shaders/direction"
)

var (
	DirectionVertShader = filepath.Join(ShadersDir, "direction.vert")
	DirectionFragShader = filepath.Join(ShadersDir, "direction.frag")
)

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

// Direction implements direction indicator rendering
type Direction struct {
	shader    *graphics.Shader
	vao       uint32
	vbo       uint32
	letterVAO uint32
	letterVBO uint32
}

// NewDirection creates a new direction renderable
func NewDirection() *Direction {
	return &Direction{}
}

// Init initializes the direction rendering system
func (d *Direction) Init() error {
	// Create shader
	var err error
	d.shader, err = graphics.NewShader(DirectionVertShader, DirectionFragShader)
	if err != nil {
		return err
	}

	// Setup VAOs and VBOs
	d.setupDirectionVAO()
	d.setupLetterVAO()

	return nil
}

// Render renders the direction indicator
func (d *Direction) Render(ctx renderer.RenderContext) {
	func() {
		defer profiling.Track("renderer.renderDirection")()
		// Use camera aspect to keep correct proportions
		aspectRatio := ctx.Camera.AspectRatio
		d.shader.Use()
		d.shader.SetFloat("aspectRatio", aspectRatio)
		d.renderDirection(ctx.Player)
	}()
}

// Dispose cleans up OpenGL resources
func (d *Direction) Dispose() {
	if d.vao != 0 {
		gl.DeleteVertexArrays(1, &d.vao)
	}
	if d.vbo != 0 {
		gl.DeleteBuffers(1, &d.vbo)
	}
	if d.letterVAO != 0 {
		gl.DeleteVertexArrays(1, &d.letterVAO)
	}
	if d.letterVBO != 0 {
		gl.DeleteBuffers(1, &d.letterVBO)
	}
}

func (d *Direction) setupDirectionVAO() {
	gl.GenVertexArrays(1, &d.vao)
	gl.BindVertexArray(d.vao)

	gl.GenBuffers(1, &d.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, d.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(DirectionVertices)*4, gl.Ptr(DirectionVertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
}

func (d *Direction) setupLetterVAO() {
	gl.GenVertexArrays(1, &d.letterVAO)
	gl.BindVertexArray(d.letterVAO)

	gl.GenBuffers(1, &d.letterVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, d.letterVBO)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
}

// drawDirectionLetter draws the specified direction letter
func (d *Direction) drawDirectionLetter(letter string) {
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
	gl.BindBuffer(gl.ARRAY_BUFFER, d.letterVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	// Draw the letter
	gl.BindVertexArray(d.letterVAO)
	gl.LineWidth(1.0)
	gl.DrawArrays(gl.LINES, 0, int32(len(vertices)/2))
}

func (d *Direction) renderDirection(p *player.Player) {
	// Set direction color (red)
	d.shader.SetVector3("directionColor", 1.0, 0.0, 0.0)

	// Position at the bottom of the screen
	d.shader.SetFloat("positionX", 0.0)
	d.shader.SetFloat("positionY", -0.85)

	// Rotate based on player's yaw (convert to radians)
	// Subtract 90 degrees because the arrow points up by default, but yaw 0 is north
	yawRadians := float32(mgl32.DegToRad(float32(p.CamYaw + 90.0)))
	d.shader.SetFloat("rotation", yawRadians)

	// Draw the direction indicator
	gl.BindVertexArray(d.vao)
	gl.LineWidth(1.0)
	// Draw arrow body (rectangle)
	gl.DrawArrays(gl.LINE_LOOP, 0, 4)
	// Draw arrow head (triangle)
	gl.DrawArrays(gl.LINE_LOOP, 4, 3)

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
	d.shader.SetFloat("positionX", 0.0)
	d.shader.SetFloat("positionY", -0.75)
	d.shader.SetFloat("rotation", 0.0) // No rotation for text

	// Draw the direction letter
	d.drawDirectionLetter(directionLetter)
}
