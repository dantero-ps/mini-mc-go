package hand

import (
	"math"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/items"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/hand"
)

var (
	HandVertShader = filepath.Join(ShadersDir, "hand.vert")
	HandFragShader = filepath.Join(ShadersDir, "hand.frag")
)

// Hand implements first-person hand rendering
type Hand struct {
	shader      *graphics.Shader
	vao         uint32
	vbo         uint32
	vertexCount int32
	items       *items.Items
}

// NewHand creates a new hand renderable
func NewHand(items *items.Items) *Hand {
	return &Hand{
		items: items,
	}
}

// Init initializes the hand rendering system
func (h *Hand) Init() error {
	// Create shader
	var err error
	h.shader, err = graphics.NewShader(HandVertShader, HandFragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO
	h.setupHandVAO()

	return nil
}

// Render renders the first-person hand
func (h *Hand) Render(ctx renderer.RenderContext) {
	func() {
		defer profiling.Track("renderer.renderHand")()
		h.renderHand(ctx.Player, ctx.DT, ctx.Camera)
	}()
}

// Dispose cleans up OpenGL resources
func (h *Hand) Dispose() {
	if h.vao != 0 {
		gl.DeleteVertexArrays(1, &h.vao)
	}
	if h.vbo != 0 {
		gl.DeleteBuffers(1, &h.vbo)
	}
}

// setupHandVAO creates a simple rectangular prism to represent the right arm
func (h *Hand) setupHandVAO() {
	l, rgt := float32(-2), float32(2)
	b, top := float32(-2), float32(10)
	bk, ft := float32(-2), float32(2)

	// pos only (no normals) to keep shader minimal
	vertexData := []float32{
		// Front face - normal: (0,0,1)
		l, b, ft, 0, 0, 1, rgt, b, ft, 0, 0, 1, rgt, top, ft, 0, 0, 1,
		rgt, top, ft, 0, 0, 1, l, top, ft, 0, 0, 1, l, b, ft, 0, 0, 1,

		// Back face - normal: (0,0,-1)
		rgt, b, bk, 0, 0, -1, l, b, bk, 0, 0, -1, l, top, bk, 0, 0, -1,
		l, top, bk, 0, 0, -1, rgt, top, bk, 0, 0, -1, rgt, b, bk, 0, 0, -1,

		// Left face - normal: (-1,0,0)
		l, b, bk, -1, 0, 0, l, b, ft, -1, 0, 0, l, top, ft, -1, 0, 0,
		l, top, ft, -1, 0, 0, l, top, bk, -1, 0, 0, l, b, bk, -1, 0, 0,

		// Right face - normal: (1,0,0)
		rgt, b, ft, 1, 0, 0, rgt, b, bk, 1, 0, 0, rgt, top, bk, 1, 0, 0,
		rgt, top, bk, 1, 0, 0, rgt, top, ft, 1, 0, 0, rgt, b, ft, 1, 0, 0,

		// Top face - normal: (0,1,0)
		l, top, ft, 0, 1, 0, rgt, top, ft, 0, 1, 0, rgt, top, bk, 0, 1, 0,
		rgt, top, bk, 0, 1, 0, l, top, bk, 0, 1, 0, l, top, ft, 0, 1, 0,

		// Bottom face - normal: (0,-1,0)
		l, b, bk, 0, -1, 0, rgt, b, bk, 0, -1, 0, rgt, b, ft, 0, -1, 0,
		rgt, b, ft, 0, -1, 0, l, b, ft, 0, -1, 0, l, b, bk, 0, -1, 0,
	}

	gl.GenVertexArrays(1, &h.vao)
	gl.GenBuffers(1, &h.vbo)
	gl.BindVertexArray(h.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, h.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertexData)*4, gl.Ptr(vertexData), gl.STATIC_DRAW)

	// Position attribute (location = 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))

	// Normal attribute (location = 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	h.vertexCount = int32(len(vertexData) / 6)
}

func (h *Hand) renderHand(p *player.Player, dt float64, camera *graphics.Camera) {
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	proj := mgl32.Perspective(mgl32.DegToRad(camera.FOV), camera.AspectRatio, camera.NearPlane, camera.FarPlane)

	swing := p.GetHandSwingProgress()
	equip := p.GetHandEquipProgress()

	// Arm swing constants (renderPlayerArm)
	asX := float32(-0.3 * math.Sin(math.Sqrt(float64(swing))*math.Pi))
	asY := float32(0.4 * math.Sin(math.Sqrt(float64(swing))*math.Pi*2.0))
	asZ := float32(-0.4 * math.Sin(float64(swing)*math.Pi))

	// Item swing constants (doItemUsedTransformations)
	isX := float32(-0.4 * math.Sin(math.Sqrt(float64(swing))*math.Pi))
	isY := float32(0.2 * math.Sin(math.Sqrt(float64(swing))*math.Pi*2.0))
	isZ := float32(-0.2 * math.Sin(float64(swing)*math.Pi))

	// Rotation components
	rotS1 := float32(math.Sin(float64(swing*swing) * math.Pi))
	rotS2 := float32(math.Sin(math.Sqrt(float64(swing)) * math.Pi))

	// Render either item or hand (like Minecraft ItemRenderer.java:406-411)
	if p.EquippedItem != nil && h.items != nil {
		itemModel := mgl32.Ident4()
		itemModel = h.setupViewBobbing(p, itemModel, dt)

		// Item used transformations (bobbing during swing)
		itemModel = itemModel.Mul4(mgl32.Translate3D(isX, isY, isZ))

		// transformFirstPersonItem (Minecraft ItemRenderer.java:296)
		itemModel = itemModel.Mul4(mgl32.Translate3D(0.48, -0.46, -0.85))
		itemModel = itemModel.Mul4(mgl32.Translate3D(0.0, (1.0-equip)*-0.6, 0.0))
		itemModel = itemModel.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(45.0)))

		// Item rotations during swing
		itemModel = itemModel.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(rotS1 * -20.0)))
		itemModel = itemModel.Mul4(mgl32.HomogRotate3DZ(mgl32.DegToRad(rotS2 * -20.0)))
		itemModel = itemModel.Mul4(mgl32.HomogRotate3DX(mgl32.DegToRad(rotS2 * -80.0)))

		// Minecraft block in hand scale
		itemModel = itemModel.Mul4(mgl32.Scale3D(0.4, 0.4, 0.4))

		h.items.RenderHand(p.EquippedItem, proj, itemModel)
	} else { // Show hand even when sneaking
		model := mgl32.Ident4()
		model = h.setupViewBobbing(p, model, dt)
		model = model.Mul4(mgl32.Translate3D(asX, asY, asZ))

		// Hand position/rotation (renderPlayerArm)
		model = model.Mul4(mgl32.Translate3D(0.64000005, -0.3, -0.72))
		model = model.Mul4(mgl32.Translate3D(0.0, (1.0-equip)*-0.6, 0.0))
		model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(45.0)))

		// Swing rotations
		model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(rotS2 * 70.0)))
		model = model.Mul4(mgl32.HomogRotate3DZ(mgl32.DegToRad(rotS1 * -20.0)))

		// Final hand orientations
		model = model.Mul4(mgl32.Translate3D(-1.0, 3.6, 3.5))
		model = model.Mul4(mgl32.HomogRotate3DZ(mgl32.DegToRad(120.0)))
		model = model.Mul4(mgl32.HomogRotate3DX(mgl32.DegToRad(200.0)))
		model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(-135.0)))
		model = model.Mul4(mgl32.Translate3D(5.6, 0.0, 0.0))
		model = model.Mul4(mgl32.Scale3D(0.0625, 0.0625, 0.0625))

		h.shader.Use()
		h.shader.SetMatrix4("proj", &proj[0])
		h.shader.SetMatrix4("model", &model[0])

		lightPos := mgl32.Vec3{2, 55, 2}
		h.shader.SetVector3("lightPos", lightPos.X(), lightPos.Y(), lightPos.Z())

		viewPos := mgl32.Vec3{0, 50, 0}
		h.shader.SetVector3("viewPos", viewPos.X(), viewPos.Y(), viewPos.Z())

		gl.Disable(gl.CULL_FACE)
		gl.BindVertexArray(h.vao)
		gl.DrawArrays(gl.TRIANGLES, 0, h.vertexCount)
		gl.BindVertexArray(0)
		gl.Enable(gl.CULL_FACE)
	}
}

func (h *Hand) setupViewBobbing(p *player.Player, model mgl32.Mat4, dt float64) mgl32.Mat4 {
	f := p.DistanceWalkedModified - p.PrevDistanceWalkedModified
	f1 := -(p.DistanceWalkedModified + f*dt)
	f2 := p.PrevHeadBobYaw + (p.HeadBobYaw-p.PrevHeadBobYaw)*dt
	f3 := p.PrevHeadBobPitch + (p.HeadBobPitch-p.PrevHeadBobPitch)*dt

	const deg2rad = math.Pi / 180.0

	tx := float32(math.Sin(f1*math.Pi) * f2 * 0.5)
	ty := float32(-math.Abs(math.Cos(f1*math.Pi) * f2))
	model = model.Mul4(mgl32.Translate3D(tx, ty, 0))

	angZ := float32(math.Sin(f1*math.Pi)*f2*3.0) * float32(deg2rad)
	model = model.Mul4(mgl32.HomogRotate3D(angZ, mgl32.Vec3{0, 0, 1}))

	angX1 := float32(math.Abs(math.Cos(f1*math.Pi-0.2)*f2)*5.0) * float32(deg2rad)
	model = model.Mul4(mgl32.HomogRotate3D(angX1, mgl32.Vec3{1, 0, 0}))

	angX2 := float32(f3) * float32(deg2rad) / 50
	model = model.Mul4(mgl32.HomogRotate3D(angX2, mgl32.Vec3{1, 0, 0}))

	return model
}
