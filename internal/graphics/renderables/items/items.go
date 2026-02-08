package items

import (
	"math"
	"mini-mc/internal/entity"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/item"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
	"mini-mc/pkg/blockmodel"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Items struct {
	shader *graphics.Shader

	// Cache for generated item meshes
	meshCache map[world.BlockType]*ItemMesh

	// Viewport dimensions for GUI rendering
	width  float32
	height float32
}

func NewItems() *Items {
	return &Items{
		meshCache: make(map[world.BlockType]*ItemMesh),
		width:     900,
		height:    600,
	}
}

func (i *Items) Init() error {
	var err error
	i.shader, err = graphics.NewShader("assets/shaders/item/item.vert", "assets/shaders/item/item.frag")
	if err != nil {
		return err
	}

	// Generate meshes for all registered blocks/items
	for bType, def := range registry.Blocks {
		var elements []blockmodel.Element

		// 1. Try to load specific item model first
		// This handles cases where item looks different than block (e.g. saplings, flats)
		// registry.ModelLoader is available.
		itemModelName := def.Name

		// Determine if we should check for item model?
		// registry.ModelLoader.LoadItemModel will look for "item/<name>"
		model, err := registry.ModelLoader.LoadItemModel(itemModelName)

		// Using the loaded model's elements if found
		if err == nil && len(model.Elements) > 0 {
			elements = model.Elements
		} else {
			// Fallback: Use the block definition's elements (which were loaded from block model)
			elements = def.Elements
		}

		// Skip if no elements (e.g. Air)
		if len(elements) == 0 {
			continue
		}

		mesh, err := BuildItemMesh(elements)
		if err != nil {
			// fmt.Printf("Failed to build mesh for %s: %v\n", def.Name, err)
			continue
		}

		i.meshCache[bType] = mesh
	}

	return nil
}

func (i *Items) Render(ctx renderer.RenderContext) {
	entities := ctx.World.GetEntities()
	if len(entities) == 0 {
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

	gl.BindVertexArray(0)

	for _, ent := range entities {
		itemEnt, ok := ent.(*entity.ItemEntity)
		if !ok {
			continue
		}

		// Check if we have a mesh for this item
		mesh, exists := i.meshCache[itemEnt.Stack.Type]
		if !exists || mesh == nil {
			continue
		}

		// Calculate how many items to render based on stack count (Minecraft style)
		// 1 item: 1 copy
		// 2-16 items: 2 copies
		// 17-32 items: 3 copies
		// 33-48 items: 4 copies
		// 49-64 items: 5 copies
		renderCount := getStackRenderCount(itemEnt.Stack.Count)

		// Animation logic (bobbing & rotation)
		age := float32(itemEnt.Age * 20.0) // Convert seconds to ticks approx
		hover := float32(math.Sin(float64(age/10.0+float32(itemEnt.HoverStart))))*0.1 + 0.25
		rot := (age/20.0 + float32(itemEnt.HoverStart)) * (180.0 / math.Pi)

		pos := itemEnt.Position()

		// Render multiple items for stacks
		for j := 0; j < renderCount; j++ {
			// Offset each item slightly for visual stacking effect
			// Use deterministic offsets based on index for consistent appearance
			offsetX := float32(0)
			offsetY := float32(j) * 0.03 // Stack vertically
			offsetZ := float32(0)

			// Add slight random-like horizontal offset for items beyond first
			if j > 0 {
				// Use sine/cosine for pseudo-random but deterministic offsets
				angle := float32(j) * 2.39996 // Golden angle for nice distribution
				offsetX = float32(math.Sin(float64(angle))) * 0.05
				offsetZ = float32(math.Cos(float64(angle))) * 0.05
			}

			// Translate
			model := mgl32.Translate3D(pos.X()+offsetX, pos.Y()+hover+offsetY, pos.Z()+offsetZ)

			// Rotate (around Y) - each layer rotates slightly differently
			layerRot := rot + float32(j)*15.0
			model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(layerRot)))

			// Scale (0.25 size block)
			model = model.Mul4(mgl32.Scale3D(0.25, 0.25, 0.25))

			// Center the mesh (0..1 -> -0.5..0.5)
			model = model.Mul4(mgl32.Translate3D(-0.5, -0.5, -0.5))

			i.shader.SetMatrix4("model", &model[0])

			i.drawBlock(itemEnt.Stack.Type, mesh)
		}
	}
}

// getStackRenderCount returns how many item copies to render based on stack count
// Matches Minecraft's visual stacking behavior
func getStackRenderCount(count int) int {
	switch {
	case count <= 1:
		return 1
	case count <= 16:
		return 2
	case count <= 32:
		return 3
	case count <= 48:
		return 4
	default:
		return 5
	}
}

// RenderGUI draws an item stack at 2D screen coordinates (x,y) with given size
func (i *Items) RenderGUI(stack *item.ItemStack, x, y, size float32) {
	i.RenderGUIScaled(stack, x, y, size, size)
}

// RenderGUIScaled draws an item stack at 2D screen coordinates with separate width/height
// This allows for animated scaling effects (squeeze/stretch)
func (i *Items) RenderGUIScaled(stack *item.ItemStack, x, y, width, height float32) {
	if stack == nil {
		return
	}

	i.shader.Use()

	// Orthographic projection for UI
	proj := mgl32.Ortho(0, i.width, 0, i.height, -100, 100)
	i.shader.SetMatrix4("proj", &proj[0])

	// Identity view for UI
	view := mgl32.Ident4()
	i.shader.SetMatrix4("view", &view[0])

	if blocks.GlobalTextureAtlas != nil {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D_ARRAY, blocks.GlobalTextureAtlas.TextureID)
		i.shader.SetInt("textureArray", 0)
	}

	mesh, exists := i.meshCache[stack.Type]
	if !exists || mesh == nil {
		return
	}

	gl.Enable(gl.DEPTH_TEST)      // GUI items need depth test for 3D look
	gl.Clear(gl.DEPTH_BUFFER_BIT) // Clear depth so items don't clip with world or each other

	// Model matrix for positioning
	// 1. Scale to size
	// 2. Rotate for isometric view (45 deg Y, 30 deg X approx)
	// 3. Translate to screen pos

	// Center of item is at (0,0,0) in model space
	// Screen coords (x,y) are top-left usually, let's adjust to center
	cx := x + width/2
	cy := i.height - (y + height/2)

	model := mgl32.Translate3D(cx, cy, 0)

	// Minecraft GUI item scale and rotation
	// We use -size for Y because the UI coordinate system has Y increasing downwards
	// The 0.65 factor compensates for the isometric expansion to fit inside 16x16 slot area
	// Use the smaller dimension to maintain aspect ratio of the block model
	guiScaleX := width * 0.65
	guiScaleY := height * 0.65
	guiScaleZ := (width + height) / 2 * 0.65 // Average for Z depth
	model = model.Mul4(mgl32.Scale3D(guiScaleX, guiScaleY, guiScaleZ))

	// Minecraft standard GUI item rotation:
	// 1. Rotate 45 degrees around Y
	// 2. Rotate 30 degrees around X (positive since Y is flipped)
	model = model.Mul4(mgl32.HomogRotate3DX(mgl32.DegToRad(30)))
	model = model.Mul4(mgl32.HomogRotate3DY(mgl32.DegToRad(45)))

	// Center mesh
	model = model.Mul4(mgl32.Translate3D(-0.5, -0.5, -0.5))

	i.shader.SetMatrix4("model", &model[0])

	i.drawBlock(stack.Type, mesh)

	gl.BindVertexArray(0)
	gl.Disable(gl.DEPTH_TEST)
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

	mesh, exists := i.meshCache[stack.Type]
	if !exists || mesh == nil {
		return
	}

	// Center mesh mismatch adjustment
	model = model.Mul4(mgl32.Translate3D(-0.5, -0.5, -0.5))
	i.shader.SetMatrix4("model", &model[0])

	i.drawBlock(stack.Type, mesh)
	gl.BindVertexArray(0)
}

func (i *Items) drawBlock(blockType world.BlockType, mesh *ItemMesh) {
	// Set tint color for the whole item
	// Individual faces will apply it based on TintIndex attribute
	def, hasDef := registry.Blocks[blockType]

	r, g, b := float32(1.0), float32(1.0), float32(1.0)
	if hasDef && def.TintColor != 0 {
		r = float32((def.TintColor>>16)&0xFF) / 255.0
		g = float32((def.TintColor>>8)&0xFF) / 255.0
		b = float32(def.TintColor&0xFF) / 255.0
	}
	i.shader.SetVector3("tintColor", r, g, b)

	gl.BindVertexArray(mesh.VAO)
	gl.DrawArrays(gl.TRIANGLES, 0, mesh.VertexCount)
}

// SetViewport updates the items renderer viewport dimensions for GUI rendering
func (i *Items) SetViewport(width, height int) {
	i.width = float32(width)
	i.height = float32(height)
}

func (i *Items) Dispose() {
	for _, mesh := range i.meshCache {
		gl.DeleteVertexArrays(1, &mesh.VAO)
		gl.DeleteBuffers(1, &mesh.VBO)
	}
	i.meshCache = nil
}
