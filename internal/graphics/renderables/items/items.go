package items

import (
	"math"
	"mini-mc/internal/entity"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/blocks"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/item"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Items struct {
	shader *graphics.Shader
	vao    uint32
	vbo    uint32
	cube   []float32
}

func NewItems() *Items {
	return &Items{}
}

func (i *Items) Init() error {
	var err error
	i.shader, err = graphics.NewShader("assets/shaders/item/item.vert", "assets/shaders/item/item.frag")
	if err != nil {
		return err
	}

	// Cube vertices ... (kept same)
	vertices := []float32{
		// Back face (North, -Z)
		-0.5, -0.5, -0.5, 1.0, 1.0, 0.0, 0.0, -1.0,
		0.5, 0.5, -0.5, 0.0, 0.0, 0.0, 0.0, -1.0,
		0.5, -0.5, -0.5, 0.0, 1.0, 0.0, 0.0, -1.0,
		0.5, 0.5, -0.5, 0.0, 0.0, 0.0, 0.0, -1.0,
		-0.5, -0.5, -0.5, 1.0, 1.0, 0.0, 0.0, -1.0,
		-0.5, 0.5, -0.5, 1.0, 0.0, 0.0, 0.0, -1.0,

		// Front face (South, +Z)
		-0.5, -0.5, 0.5, 0.0, 1.0, 0.0, 0.0, 1.0,
		0.5, -0.5, 0.5, 1.0, 1.0, 0.0, 0.0, 1.0,
		0.5, 0.5, 0.5, 1.0, 0.0, 0.0, 0.0, 1.0,
		0.5, 0.5, 0.5, 1.0, 0.0, 0.0, 0.0, 1.0,
		-0.5, 0.5, 0.5, 0.0, 0.0, 0.0, 0.0, 1.0,
		-0.5, -0.5, 0.5, 0.0, 1.0, 0.0, 0.0, 1.0,

		// Left face (West, -X)
		-0.5, 0.5, 0.5, 1.0, 0.0, -1.0, 0.0, 0.0,
		-0.5, 0.5, -0.5, 0.0, 0.0, -1.0, 0.0, 0.0,
		-0.5, -0.5, -0.5, 0.0, 1.0, -1.0, 0.0, 0.0,
		-0.5, -0.5, -0.5, 0.0, 1.0, -1.0, 0.0, 0.0,
		-0.5, -0.5, 0.5, 1.0, 1.0, -1.0, 0.0, 0.0,
		-0.5, 0.5, 0.5, 1.0, 0.0, -1.0, 0.0, 0.0,

		// Right face (East, +X)
		0.5, 0.5, 0.5, 0.0, 0.0, 1.0, 0.0, 0.0,
		0.5, -0.5, -0.5, 1.0, 1.0, 1.0, 0.0, 0.0,
		0.5, 0.5, -0.5, 1.0, 0.0, 1.0, 0.0, 0.0,
		0.5, -0.5, -0.5, 1.0, 1.0, 1.0, 0.0, 0.0,
		0.5, 0.5, 0.5, 0.0, 0.0, 1.0, 0.0, 0.0,
		0.5, -0.5, 0.5, 0.0, 1.0, 1.0, 0.0, 0.0,

		// Bottom face (-Y)
		-0.5, -0.5, -0.5, 0.0, 0.0, 0.0, -1.0, 0.0,
		0.5, -0.5, -0.5, 1.0, 0.0, 0.0, -1.0, 0.0,
		0.5, -0.5, 0.5, 1.0, 1.0, 0.0, -1.0, 0.0,
		0.5, -0.5, 0.5, 1.0, 1.0, 0.0, -1.0, 0.0,
		-0.5, -0.5, 0.5, 0.0, 1.0, 0.0, -1.0, 0.0,
		-0.5, -0.5, -0.5, 0.0, 0.0, 0.0, -1.0, 0.0,

		// Top face (+Y)
		-0.5, 0.5, -0.5, 0.0, 1.0, 0.0, 1.0, 0.0,
		0.5, 0.5, 0.5, 1.0, 0.0, 0.0, 1.0, 0.0,
		0.5, 0.5, -0.5, 1.0, 1.0, 0.0, 1.0, 0.0,
		0.5, 0.5, 0.5, 1.0, 0.0, 0.0, 1.0, 0.0,
		-0.5, 0.5, -0.5, 0.0, 1.0, 0.0, 1.0, 0.0,
		-0.5, 0.5, 0.5, 0.0, 0.0, 0.0, 1.0, 0.0,
	}
	i.cube = vertices

	gl.GenVertexArrays(1, &i.vao)
	gl.GenBuffers(1, &i.vbo)

	gl.BindVertexArray(i.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, i.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Pos
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	// UV
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(3*4))
	// Normal
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(5*4))

	return nil
}

func (i *Items) Render(ctx renderer.RenderContext) {
	if len(ctx.World.Entities) == 0 {
		return
	}

	i.shader.Use()
	i.shader.SetMatrix4("view", &ctx.View[0])
	i.shader.SetMatrix4("proj", &ctx.Proj[0])

	// Bind global texture atlas
	if blocks.GlobalTextureAtlas != nil {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, blocks.GlobalTextureAtlas.TextureID)
		i.shader.SetInt("textureArray", 0)
	}

	gl.BindVertexArray(i.vao)

	for _, ent := range ctx.World.Entities {
		itemEnt, ok := ent.(*entity.ItemEntity)
		if !ok {
			continue
		}

		// Animation logic (bobbing & rotation)
		age := float32(itemEnt.Age * 20.0) // Convert seconds to ticks approx
		hover := float32(math.Sin(float64(age/10.0+float32(itemEnt.HoverStart))))*0.1 + 0.25
		rot := (age/20.0 + float32(itemEnt.HoverStart)) * (180.0 / math.Pi)

		// Translate
		pos := itemEnt.Position()
		model := mgl32.Translate3D(pos.X()-0.5, pos.Y()+hover-1.0, pos.Z()-0.5)

		// Rotate (around Y)
		model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(rot)))

		// Scale (0.25 size block)
		model = model.Mul4(mgl32.Scale3D(0.25, 0.25, 0.25))

		i.shader.SetMatrix4("model", &model[0])

		i.drawBlock(itemEnt.Stack.Type)
	}
}

// RenderGUI draws an item stack at 2D screen coordinates (x,y) with given size
func (i *Items) RenderGUI(stack *item.ItemStack, x, y, size float32) {
	if stack == nil {
		return
	}

	gl.Enable(gl.DEPTH_TEST)      // GUI items need depth test for 3D look
	gl.Clear(gl.DEPTH_BUFFER_BIT) // Clear depth so items don't clip with world or each other

	i.shader.Use()

	// Orthographic projection for UI
	// Assuming 900x600 window for now, needs to match UI context
	proj := mgl32.Ortho(0, 900, 600, 0, -100, 100)
	i.shader.SetMatrix4("proj", &proj[0])

	// Identity view for UI
	view := mgl32.Ident4()
	i.shader.SetMatrix4("view", &view[0])

	if blocks.GlobalTextureAtlas != nil {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, blocks.GlobalTextureAtlas.TextureID)
		i.shader.SetInt("textureArray", 0)
	}

	gl.BindVertexArray(i.vao)

	// Model matrix for positioning
	// 1. Scale to size
	// 2. Rotate for isometric view (45 deg Y, 30 deg X approx)
	// 3. Translate to screen pos

	// Center of item is at (0,0,0) in model space
	// Screen coords (x,y) are top-left usually, let's adjust to center
	cx := x + size/2
	cy := y + size/2

	model := mgl32.Translate3D(cx, cy, 0)

	// Minecraft GUI item scale and rotation
	// We use -size for Y because the UI coordinate system has Y increasing downwards
	// The 0.65 factor compensates for the isometric expansion to fit inside 16x16 slot area
	guiScale := size * 0.65
	model = model.Mul4(mgl32.Scale3D(guiScale, -guiScale, guiScale))

	// Minecraft standard GUI item rotation:
	// 1. Rotate 45 degrees around Y
	// 2. Rotate 30 degrees around X (positive since Y is flipped)
	model = model.Mul4(mgl32.HomogRotate3DX(mgl32.DegToRad(30)))
	model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(45)))

	i.shader.SetMatrix4("model", &model[0])

	i.drawBlock(stack.Type)

	gl.BindVertexArray(0)
}

// RenderHand draws an item stack in the player's hand using provided projection and model matrices
func (i *Items) RenderHand(stack *item.ItemStack, proj mgl32.Mat4, model mgl32.Mat4) {
	if stack == nil {
		return
	}

	i.shader.Use()
	i.shader.SetMatrix4("proj", &proj[0])

	// For hand rendering, view matrix is usually identity as the model matrix
	// is already calculated in view space
	view := mgl32.Ident4()
	i.shader.SetMatrix4("view", &view[0])
	i.shader.SetMatrix4("model", &model[0])

	if blocks.GlobalTextureAtlas != nil {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, blocks.GlobalTextureAtlas.TextureID)
		i.shader.SetInt("textureArray", 0)
	}

	gl.BindVertexArray(i.vao)
	i.drawBlock(stack.Type)
	gl.BindVertexArray(0)
}

func (i *Items) drawBlock(blockType world.BlockType) {
	// Draw each face with correct texture
	def, hasDef := registry.Blocks[blockType]

	faces := []world.BlockFace{
		world.FaceNorth,
		world.FaceSouth,
		world.FaceWest,
		world.FaceEast,
		world.FaceBottom,
		world.FaceTop,
	}

	for fIdx, face := range faces {
		texID := registry.GetTextureLayer(blockType, face)
		i.shader.SetInt("textureID", int32(texID))

		r, g, b := float32(1.0), float32(1.0), float32(1.0)
		if hasDef && def.TintColor != 0 {
			if def.TintFaces[face] {
				r = float32((def.TintColor>>16)&0xFF) / 255.0
				g = float32((def.TintColor>>8)&0xFF) / 255.0
				b = float32(def.TintColor&0xFF) / 255.0
			}
		}
		i.shader.SetVector3("tintColor", r, g, b)

		gl.DrawArrays(gl.TRIANGLES, int32(fIdx*6), 6)
	}
}

func (i *Items) Dispose() {
	gl.DeleteVertexArrays(1, &i.vao)
	gl.DeleteBuffers(1, &i.vbo)
}
