package hand

import (
	"math"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderer"
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
	texture     uint32
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

	// Load skin texture
	var dpth int
	h.texture, _, dpth, err = graphics.LoadTexture("assets/textures/entity/steve.png")
	if err != nil {
		return err
	}
	_ = dpth

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

// SetViewport updates viewport dimensions (not needed for hand)
func (h *Hand) SetViewport(width, height int) {
	// Hand rendering doesn't need viewport dimensions
}

// setupHandVAO creates a simple rectangular prism to represent the right arm
func (h *Hand) setupHandVAO() {
	// Minecraft Steve Right Arm Geometry
	// Box Dimensions: 4x12x4
	// Origin (Bottom-Top-Right corner of box relative to pivot?):
	// MC uses: addBox(-3, -2, -2, 4, 12, 4)
	// X: -3 to 1
	// Y: -2 to 10
	// Z: -2 to 2

	// Pivot point: (-5, 2, 0)
	// The Pivot translation is applied by ModelRenderer before drawing.
	// Since we want our Model matrix to match the one in ItemRenderer (before renderRightArm),
	// we must bake the Pivot translation AND the Global Scale (0.0625) into the vertices.

	scale := float32(0.0625)
	pX, pY, pZ := float32(-5.0), float32(2.0), float32(0.0)

	// Box coordinates
	x1, y1, z1 := float32(-3.0), float32(-2.0), float32(-2.0)
	x2, y2, z2 := float32(1.0), float32(10.0), float32(2.0)

	// Bake transform: v_final = (v_box + pivot) * scale
	l := (x1 + pX) * scale
	rgt := (x2 + pX) * scale

	b := (y1 + pY) * scale
	top := (y2 + pY) * scale

	bk := (z1 + pZ) * scale
	ft := (z2 + pZ) * scale

	// UV Mapping (Steve Skin - 64x64)
	s := float32(1.0 / 64.0)
	uOut1, uOut2 := 40*s, 44*s
	uFr1, uFr2 := 44*s, 48*s
	uIn1, uIn2 := 48*s, 52*s
	uBk1, uBk2 := 52*s, 56*s
	uTop1, uTop2 := 44*s, 48*s
	uBot1, uBot2 := 48*s, 52*s

	// V coordinates (Swapped to fix vertical inversion)
	vTop1, vTop2 := 20*s, 16*s
	vBody1, vBody2 := 32*s, 20*s

	// Note on Face Orientation:
	// We map "Front" UV to +Z geometry.
	// We map "Right" UV (Outside) to +X geometry (rgt).
	// We map "Left" UV (Inside) to -X geometry (l).

	vertexData := []float32{
		// Front face (+Z) - normal: (0,0,1)
		l, b, ft, 0, 0, 1, uFr2, vBody2,
		rgt, b, ft, 0, 0, 1, uFr1, vBody2,
		rgt, top, ft, 0, 0, 1, uFr1, vBody1,
		rgt, top, ft, 0, 0, 1, uFr1, vBody1,
		l, top, ft, 0, 0, 1, uFr2, vBody1,
		l, b, ft, 0, 0, 1, uFr2, vBody2,

		// Back face (-Z) - normal: (0,0,-1)
		rgt, b, bk, 0, 0, -1, uBk2, vBody2,
		l, b, bk, 0, 0, -1, uBk1, vBody2,
		l, top, bk, 0, 0, -1, uBk1, vBody1,
		l, top, bk, 0, 0, -1, uBk1, vBody1,
		rgt, top, bk, 0, 0, -1, uBk2, vBody1,
		rgt, b, bk, 0, 0, -1, uBk2, vBody2,

		// Left face (-X) (Inside) - normal: (-1,0,0)
		l, b, bk, -1, 0, 0, uIn2, vBody2,
		l, b, ft, -1, 0, 0, uIn1, vBody2,
		l, top, ft, -1, 0, 0, uIn1, vBody1,
		l, top, ft, -1, 0, 0, uIn1, vBody1,
		l, top, bk, -1, 0, 0, uIn2, vBody1,
		l, b, bk, -1, 0, 0, uIn2, vBody2,

		// Right face (+X) (Outside) - normal: (1,0,0)
		rgt, b, ft, 1, 0, 0, uOut2, vBody2,
		rgt, b, bk, 1, 0, 0, uOut1, vBody2,
		rgt, top, bk, 1, 0, 0, uOut1, vBody1,
		rgt, top, bk, 1, 0, 0, uOut1, vBody1,
		rgt, top, ft, 1, 0, 0, uOut2, vBody1,
		rgt, b, ft, 1, 0, 0, uOut2, vBody2,

		// Top face (+Y) - normal: (0,1,0)
		l, top, ft, 0, 1, 0, uTop1, vTop2,
		rgt, top, ft, 0, 1, 0, uTop2, vTop2,
		rgt, top, bk, 0, 1, 0, uTop2, vTop1,
		rgt, top, bk, 0, 1, 0, uTop2, vTop1,
		l, top, bk, 0, 1, 0, uTop1, vTop1,
		l, top, ft, 0, 1, 0, uTop1, vTop2,

		// Bottom face (-Y) - normal: (0,-1,0)
		l, b, bk, 0, -1, 0, uBot1, vTop1,
		rgt, b, bk, 0, -1, 0, uBot2, vTop1,
		rgt, b, ft, 0, -1, 0, uBot2, vTop2,
		rgt, b, ft, 0, -1, 0, uBot2, vTop2,
		l, b, ft, 0, -1, 0, uBot1, vTop2,
		l, b, bk, 0, -1, 0, uBot1, vTop1,
	}

	gl.GenVertexArrays(1, &h.vao)
	gl.GenBuffers(1, &h.vbo)
	gl.BindVertexArray(h.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, h.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertexData)*4, gl.Ptr(vertexData), gl.STATIC_DRAW)

	// Position attribute (location = 0)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))

	// Normal attribute (location = 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))

	// TexCoord attribute (location = 2)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(6*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	h.vertexCount = int32(len(vertexData) / 8)
}

func (h *Hand) renderHand(p *player.Player, dt float64, camera *graphics.Camera) {
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	// Minecraft uses a fixed 70.0 FOV for hand rendering, ignoring game settings.
	// It also uses 0.05 for near plane.
	proj := mgl32.Perspective(mgl32.DegToRad(70.0), camera.AspectRatio, 0.05, camera.FarPlane)

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
		itemModel = h.setupHandSway(p, itemModel, dt)

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
		model = h.setupHandSway(p, model, dt)
		model = model.Mul4(mgl32.Translate3D(asX, asY, asZ))

		// Hand position/rotation (renderPlayerArm)
		model = model.Mul4(mgl32.Translate3D(0.64000005, -0.6, -0.72))
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

		h.shader.Use()
		h.shader.SetMatrix4("proj", &proj[0])
		h.shader.SetMatrix4("model", &model[0])

		lightPos := mgl32.Vec3{2, 55, 2}
		h.shader.SetVector3("lightPos", lightPos.X(), lightPos.Y(), lightPos.Z())

		viewPos := mgl32.Vec3{0, 50, 0}
		h.shader.SetVector3("viewPos", viewPos.X(), viewPos.Y(), viewPos.Z())

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, h.texture)
		h.shader.SetInt("skinTexture", 0)

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

	angX2 := float32(f3) * float32(deg2rad) / 15
	model = model.Mul4(mgl32.HomogRotate3D(angX2, mgl32.Vec3{1, 0, 0}))

	return model
}

func (h *Hand) setupHandSway(p *player.Player, model mgl32.Mat4, dt float64) mgl32.Mat4 {
	interpPitch := p.PrevRenderArmPitch + (p.RenderArmPitch-p.PrevRenderArmPitch)*float32(dt)
	interpYaw := p.PrevRenderArmYaw + (p.RenderArmYaw-p.PrevRenderArmYaw)*float32(dt)

	// Minecraft uses rotationPitch/Yaw - interp
	// (entityplayerspIn.rotationPitch - f) * 0.1F
	swayPitch := (float32(p.CamPitch) - interpPitch) * 0.1
	swayYaw := (float32(p.CamYaw) - interpYaw) * 0.1

	model = model.Mul4(mgl32.HomogRotate3DX(mgl32.DegToRad(swayPitch)))
	model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(swayYaw)))

	return model
}
