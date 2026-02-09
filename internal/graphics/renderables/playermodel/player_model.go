package playermodel

import (
	"math"
	"mini-mc/internal/graphics"
	"mini-mc/internal/player"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/hand"
)

var (
	PlayerVertShader = filepath.Join(ShadersDir, "hand.vert")
	PlayerFragShader = filepath.Join(ShadersDir, "hand.frag")
)

// PlayerModel renders the player character
type PlayerModel struct {
	shader *graphics.Shader

	// Body Parts
	torsoVAO         uint32
	torsoVBO         uint32
	torsoVertexCount int32

	rightArmVAO         uint32
	rightArmVBO         uint32
	rightArmVertexCount int32

	leftArmVAO         uint32
	leftArmVBO         uint32
	leftArmVertexCount int32

	rightLegVAO         uint32
	rightLegVBO         uint32
	rightLegVertexCount int32

	leftLegVAO         uint32
	leftLegVBO         uint32
	leftLegVertexCount int32

	// Head mesh
	headVAO         uint32
	headVBO         uint32
	headVertexCount int32

	texture uint32
}

// NewPlayerModel creates a new player model renderable
func NewPlayerModel() *PlayerModel {
	return &PlayerModel{}
}

// Init initializes the player rendering system
func (m *PlayerModel) Init() error {
	var err error
	m.shader, err = graphics.NewShader(PlayerVertShader, PlayerFragShader)
	if err != nil {
		return err
	}

	// Setup VAOs
	m.setupTorsoVO()
	m.setupRightArmVO()
	m.setupLeftArmVO()
	m.setupRightLegVO()
	m.setupLeftLegVO()
	m.setupHeadVO()

	// Load skin (Steve)
	m.texture, _, _, err = graphics.LoadTexture("assets/textures/entity/steve.png")
	if err != nil {
		return err
	}

	return nil
}

// RenderInventoryPlayer renders the player model for the inventory screen (looking at mouse)
func (m *PlayerModel) RenderInventoryPlayer(p *player.Player, startX, startY float32, scale float32, mouseX, mouseY float32, screenWidth, screenHeight float32, timeSeconds float64) {
	// 51 is the center of the black box relative to guiLeft
	// 75 is the bottom of the black box relative to guiTop
	posX := startX
	posY := startY

	// Fix HiDPI scalling: Use logic screen dimensions for Ortho
	// This ensures our logical coordinates (startX, startY) map correctly to the viewport
	proj := mgl32.Ortho(0, screenWidth, screenHeight, 0, -1000, 1000)

	// Calculate rotations
	// RelX/Y calculation based on 'Standard' logical coordinates
	relX := posX - mouseX
	// Adjusted for the fact that Y is relative to feet? No, explicit offset.
	// Actually MC uses: (i + 51) - mouseX. My posX IS (i+51). So relX is correct.
	// y: (j + 75 - 50) - mouseY. My posY IS (j+75). So posY - 50*scale/30?
	// MC Code: float f = (float)(i + 51) - mouseX;
	// MC Code: float f1 = (float)(j + 75 - 50) - mouseY;
	// Wait, MC scale is passed as 30.
	// My scale passed is ~60 (30 * UI_SCALE).
	// I need the "UI Scale". Let's assume standard density for rotation calc or approximate.
	// 50 pixels is roughly eye height offset in GUI space.
	// Let's deduce HUD scale: scale / 30.0.
	// relY calculation:
	// MC Y is inverted in screenspace relative to model look vector logic often.
	// But let's stick to the derived logic.
	// HUD scale is implicitly 2.0 based to 'scale' 60.
	hudScale := scale / 30.0
	relY := posY - (50 * hudScale) - mouseY

	// Body Yaw: Negated to fix looking direction (Mouse Left -> Look Left)
	renderYawOffset := -float32(math.Atan(float64(relX/40.0))) * 20.0

	// Head Yaw: Negated
	rotationYawHead := -float32(math.Atan(float64(relX/40.0))) * 40.0

	// Head Pitch: follows mouse Y (20.0 factor)
	rotationPitch := -float32(math.Atan(float64(relY/40.0))) * 20.0

	// Common transformations
	baseModel := mgl32.Ident4()
	baseModel = baseModel.Mul4(mgl32.Translate3D(posX, posY, 50.0))

	// Final Scale implementation
	// "scale" passed in is ~60 (30 * 2).
	// MC renders model with scale argument.
	// Internal Model is 1/16 (0.0625) scale of pixels.
	// So we scale by scale * 0.0625.
	finalScale := scale * 0.0625
	baseModel = baseModel.Mul4(mgl32.Scale3D(-finalScale, finalScale, finalScale))

	// Flip (180 deg Z) and Standard Lighting Orientation logic
	baseModel = baseModel.Mul4(mgl32.HomogRotate3D(mgl32.DegToRad(180), mgl32.Vec3{0, 0, 1}))

	// Apply Whole Body Pitch (tilting backwards/forwards to look at mouse Y)
	baseModel = baseModel.Mul4(mgl32.HomogRotate3D(mgl32.DegToRad(rotationPitch), mgl32.Vec3{1, 0, 0}))

	// Apply Body Yaw (following mouse X roughly)
	bodyModel := baseModel.Mul4(mgl32.HomogRotate3D(mgl32.DegToRad(renderYawOffset), mgl32.Vec3{0, 1, 0}))

	// Setup Shader
	m.shader.Use()
	m.shader.SetMatrix4("proj", &proj[0])

	// Set Light (Directly overhead 90 degrees)
	lightPos := mgl32.Vec3{posX, posY - 500, 50}
	m.shader.SetVector3("lightPos", lightPos.X(), lightPos.Y(), lightPos.Z())
	m.shader.SetVector3("viewPos", posX, posY, 200)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, m.texture)
	m.shader.SetInt("skinTexture", 0)

	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)

	// --- 1. TORSO ---
	m.shader.SetMatrix4("model", &bodyModel[0])
	gl.BindVertexArray(m.torsoVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.torsoVertexCount)

	// --- 2. LEGS (Static for now, detached from body sway logic maybe? No, legs rotate with body yaw) ---
	gl.BindVertexArray(m.rightLegVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.rightLegVertexCount)

	gl.BindVertexArray(m.leftLegVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.leftLegVertexCount)

	// --- 3. ARMS (Animated) ---
	// Idle Animation Math
	// rightArm.rotateAngleZ += Math.cos(ageInTicks * 0.09) * 0.05 + 0.05;
	// rightArm.rotateAngleX += Math.sin(ageInTicks * 0.067) * 0.05;
	age := timeSeconds * 20.0 // MC ticks usually 20 tps.

	runCos := float32(math.Cos(age * 0.09))
	runSin := float32(math.Sin(age * 0.067))

	rightArmRozZ := -(runCos*0.05 + 0.05)
	rightArmRotX := runSin * 0.05

	leftArmRozZ := runCos*0.05 + 0.05 // Positive to mirror outwards
	leftArmRotX := -runSin * 0.05

	// RIGHT ARM
	// Pivot: (-5, 22, 0).
	rArmModel := bodyModel.Mul4(mgl32.Translate3D(-5, 22, 0))
	rArmModel = rArmModel.Mul4(mgl32.HomogRotate3D(rightArmRotX, mgl32.Vec3{1, 0, 0}))
	rArmModel = rArmModel.Mul4(mgl32.HomogRotate3D(rightArmRozZ, mgl32.Vec3{0, 0, 1}))
	rArmModel = rArmModel.Mul4(mgl32.Translate3D(5, -22, 0)) // Translate back. Wait, vertices are relative to 0,0,0 feet?
	// In setupRightArmVO, box is defined at -8, 12, -2 (Width 4). Center X is -6. Top Y is 24.
	// We need to verify vertex setup relative to pivot.
	// Currently vertices are absolute world positions relative to feet origin.
	// So we translate pivot (-5,22,0) to origin, rotate, then translate back. Correct.

	m.shader.SetMatrix4("model", &rArmModel[0])
	gl.BindVertexArray(m.rightArmVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.rightArmVertexCount)

	// LEFT ARM
	// Pivot: (5, 22, 0)
	lArmModel := bodyModel.Mul4(mgl32.Translate3D(5, 22, 0))
	lArmModel = lArmModel.Mul4(mgl32.HomogRotate3D(leftArmRotX, mgl32.Vec3{1, 0, 0}))
	lArmModel = lArmModel.Mul4(mgl32.HomogRotate3D(leftArmRozZ, mgl32.Vec3{0, 0, 1}))
	lArmModel = lArmModel.Mul4(mgl32.Translate3D(-5, -22, 0))

	m.shader.SetMatrix4("model", &lArmModel[0])
	gl.BindVertexArray(m.leftArmVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.leftArmVertexCount)

	// --- 4. HEAD ---
	// Head Pivot: (0, 24, 0) relative to Body
	headModel := bodyModel.Mul4(mgl32.Translate3D(0, 24, 0))
	headModel = headModel.Mul4(mgl32.HomogRotate3D(mgl32.DegToRad(rotationYawHead), mgl32.Vec3{0, 1, 0}))
	headModel = headModel.Mul4(mgl32.HomogRotate3D(mgl32.DegToRad(rotationPitch), mgl32.Vec3{1, 0, 0}))
	headModel = headModel.Mul4(mgl32.Translate3D(0, -24, 0))

	m.shader.SetMatrix4("model", &headModel[0])
	gl.BindVertexArray(m.headVAO)
	gl.DrawArrays(gl.TRIANGLES, 0, m.headVertexCount)

	gl.BindVertexArray(0)
	gl.Disable(gl.DEPTH_TEST)
}

func (m *PlayerModel) Dispose() {
	if m.torsoVAO != 0 {
		gl.DeleteVertexArrays(1, &m.torsoVAO)
		gl.DeleteBuffers(1, &m.torsoVBO)
	}
	if m.rightArmVAO != 0 {
		gl.DeleteVertexArrays(1, &m.rightArmVAO)
		gl.DeleteBuffers(1, &m.rightArmVBO)
	}
	if m.leftArmVAO != 0 {
		gl.DeleteVertexArrays(1, &m.leftArmVAO)
		gl.DeleteBuffers(1, &m.leftArmVBO)
	}
	if m.rightLegVAO != 0 {
		gl.DeleteVertexArrays(1, &m.rightLegVAO)
		gl.DeleteBuffers(1, &m.rightLegVBO)
	}
	if m.leftLegVAO != 0 {
		gl.DeleteVertexArrays(1, &m.leftLegVAO)
		gl.DeleteBuffers(1, &m.leftLegVBO)
	}
	if m.headVAO != 0 {
		gl.DeleteVertexArrays(1, &m.headVAO)
		gl.DeleteBuffers(1, &m.headVBO)
	}
}

// Helper to add a box to a vertex list
func addBox(vertices *[]float32, x, y, z, w, h, d float32, u, v float32) {
	// Box coordinates
	x1, y1, z1 := x, y, z
	x2, y2, z2 := x+w, y+h, z+d

	// Scale UV to 0-1 (texture 64x64)
	s := float32(1.0 / 64.0)

	// Texture Step Sizes
	dW := w * s
	dH := h * s
	dD := d * s

	// Tex coords
	uMin, vMin := u*s, v*s

	// Top Face (Y+) - u+d, v. Size w, d.
	uT1, vT1 := uMin+dD, vMin
	uT2, vT2 := uT1+dW, vT1+dD

	// Bottom Face (Y-) - u+d+w, v.
	uB1, vB1 := uT2, vMin
	uB2, vB2 := uB1+dW, vB1+dD

	// Right Face (X+) (MC: Outside) - u, v+d
	uR1, vR1 := uMin, vMin+dD
	uR2, vR2 := uR1+dD, vR1+dH

	// Front Face (Z+) - u+d, v+d
	uF1, vF1 := uR2, vR1
	uF2, vF2 := uF1+dW, vF1+dH

	// Left Face (X-) - u+d+w, v+d
	uL1, vL1 := uF2, vR1
	uL2, vL2 := uL1+dD, vL1+dH

	// Back Face (Z-) - u+d+w+d, v+d
	uBk1, vBk1 := uL2, vR1
	uBk2, vBk2 := uBk1+dW, vBk1+dH

	// Vertices
	// Front Face (+Z)
	v_fr := []float32{
		x1, y1, z2, 0, 0, 1, uF1, vF2,
		x2, y1, z2, 0, 0, 1, uF2, vF2,
		x2, y2, z2, 0, 0, 1, uF2, vF1,
		x2, y2, z2, 0, 0, 1, uF2, vF1,
		x1, y2, z2, 0, 0, 1, uF1, vF1,
		x1, y1, z2, 0, 0, 1, uF1, vF2,
	}

	// Back Face (-Z)
	v_bk := []float32{
		x2, y1, z1, 0, 0, -1, uBk1, vBk2,
		x1, y1, z1, 0, 0, -1, uBk2, vBk2,
		x1, y2, z1, 0, 0, -1, uBk2, vBk1,
		x1, y2, z1, 0, 0, -1, uBk2, vBk1,
		x2, y2, z1, 0, 0, -1, uBk1, vBk1,
		x2, y1, z1, 0, 0, -1, uBk1, vBk2,
	}

	// Left Face (-X)
	v_lf := []float32{
		x1, y1, z1, -1, 0, 0, uL2, vL2,
		x1, y1, z2, -1, 0, 0, uL1, vL2,
		x1, y2, z2, -1, 0, 0, uL1, vL1,
		x1, y2, z2, -1, 0, 0, uL1, vL1,
		x1, y2, z1, -1, 0, 0, uL2, vL1,
		x1, y1, z1, -1, 0, 0, uL2, vL2,
	}

	// Right Face (+X)
	v_rt := []float32{
		x2, y1, z2, 1, 0, 0, uR1, vR2,
		x2, y1, z1, 1, 0, 0, uR2, vR2,
		x2, y2, z1, 1, 0, 0, uR2, vR1,
		x2, y2, z1, 1, 0, 0, uR2, vR1,
		x2, y2, z2, 1, 0, 0, uR1, vR1,
		x2, y1, z2, 1, 0, 0, uR1, vR2,
	}

	// Top Face (+Y)
	v_top := []float32{
		x1, y2, z2, 0, 1, 0, uT1, vT2,
		x2, y2, z2, 0, 1, 0, uT2, vT2,
		x2, y2, z1, 0, 1, 0, uT2, vT1,
		x2, y2, z1, 0, 1, 0, uT2, vT1,
		x1, y2, z1, 0, 1, 0, uT1, vT1,
		x1, y2, z2, 0, 1, 0, uT1, vT2,
	}

	// Bottom Face (-Y)
	v_bot := []float32{
		x1, y1, z1, 0, -1, 0, uB1, vB1,
		x2, y1, z1, 0, -1, 0, uB2, vB1,
		x2, y1, z2, 0, -1, 0, uB2, vB2,
		x2, y1, z2, 0, -1, 0, uB2, vB2,
		x1, y1, z2, 0, -1, 0, uB1, vB2,
		x1, y1, z1, 0, -1, 0, uB1, vB1,
	}

	*vertices = append(*vertices, v_fr...)
	*vertices = append(*vertices, v_bk...)
	*vertices = append(*vertices, v_lf...)
	*vertices = append(*vertices, v_rt...)
	*vertices = append(*vertices, v_top...)
	*vertices = append(*vertices, v_bot...)
}

func (m *PlayerModel) setupHeadVO() {
	var vertices []float32
	// Head: 8x8x8, Texture (0,0)
	// Position: -4, 24, -4
	addBox(&vertices, -4, 24, -4, 8, 8, 8, 0, 0)
	m.headVertexCount = createVAO(&m.headVAO, &m.headVBO, vertices)
}

func (m *PlayerModel) setupTorsoVO() {
	var vertices []float32
	// Body: 8x12x4, Texture (16,16)
	// Position: -4, 12, -2
	addBox(&vertices, -4, 12, -2, 8, 12, 4, 16, 16)
	m.torsoVertexCount = createVAO(&m.torsoVAO, &m.torsoVBO, vertices)
}

func (m *PlayerModel) setupRightArmVO() {
	var vertices []float32
	// Right Arm (40, 16)
	addBox(&vertices, -8, 12, -2, 4, 12, 4, 40, 16)
	m.rightArmVertexCount = createVAO(&m.rightArmVAO, &m.rightArmVBO, vertices)
}

func (m *PlayerModel) setupLeftArmVO() {
	var vertices []float32
	// Left Arm (32, 48)
	addBox(&vertices, 4, 12, -2, 4, 12, 4, 32, 48)
	m.leftArmVertexCount = createVAO(&m.leftArmVAO, &m.leftArmVBO, vertices)
}

func (m *PlayerModel) setupRightLegVO() {
	var vertices []float32
	// Right Leg (0, 16)
	addBox(&vertices, -4, 0, -2, 4, 12, 4, 0, 16)
	m.rightLegVertexCount = createVAO(&m.rightLegVAO, &m.rightLegVBO, vertices)
}

func (m *PlayerModel) setupLeftLegVO() {
	var vertices []float32
	// Left Leg (16, 48)
	addBox(&vertices, 0, 0, -2, 4, 12, 4, 16, 48)
	m.leftLegVertexCount = createVAO(&m.leftLegVAO, &m.leftLegVBO, vertices)
}

func createVAO(vao, vbo *uint32, vertices []float32) int32 {
	gl.GenVertexArrays(1, vao)
	gl.GenBuffers(1, vbo)

	gl.BindVertexArray(*vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, *vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 8*4, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, 8*4, 3*4)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, 8*4, 6*4)

	gl.BindVertexArray(0)
	return int32(len(vertices) / 8)
}
