package hud

import (
	"fmt"
	"math"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/font"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/inventory"
	"mini-mc/internal/item"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/registry"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

// HUD implements HUD rendering including text and profiling
type HUD struct {
	fontAtlas     *font.FontAtlasInfo
	fontRenderer  *font.FontRenderer
	uiRenderer    *ui.UI
	itemRenderer  *items.Items
	showProfiling bool

	// Viewport dimensions
	width  float32
	height float32

	// Textures
	widgetsTexture   uint32
	inventoryTexture uint32
	iconsTexture     uint32

	// Inventory state
	HoveredSlot   int       // -1 if no hover, otherwise slot index (0-35)
	lastClickSlot int       // Last clicked slot
	lastClickTime time.Time // Time of last click

	// Profiling state
	frames       int
	lastFPSCheck time.Time
	currentFPS   int

	// Enhanced profiling metrics
	profilingStats struct {
		frameStartTime    time.Time
		frameEndTime      time.Time
		frameDuration     time.Duration
		lastFrameDuration time.Duration

		totalDrawCalls int
		totalVertices  int
		totalTriangles int
		visibleChunks  int
		culledChunks   int
		meshedChunks   int

		gpuMemoryUsage     int64
		textureMemoryUsage int64
		bufferMemoryUsage  int64

		frustumCullTime time.Duration
		meshingTime     time.Duration
		shaderBindTime  time.Duration
		vaoBindTime     time.Duration

		frameTimeHistory []time.Duration
		maxFrameTime     time.Duration
		minFrameTime     time.Duration
		avgFrameTime     time.Duration

		lastTotalFrameDuration time.Duration
		lastUpdateDuration     time.Duration

		lastPlayerDuration  time.Duration
		lastWorldDuration   time.Duration
		lastGlfwDuration    time.Duration
		lastHudDuration     time.Duration
		lastPruneDuration   time.Duration
		lastOtherDuration   time.Duration
		lastPhysicsDuration time.Duration

		lastPreRenderDuration  time.Duration
		lastSwapEventsDuration time.Duration
	}
}

// NewHUD creates a new HUD renderable
func NewHUD() *HUD {
	return &HUD{
		showProfiling: false,
		HoveredSlot:   -1,
		width:         900,
		height:        600,
	}
}

// Init initializes the HUD rendering system
func (h *HUD) Init() error {
	// Load font atlas and renderer
	fontPath := filepath.Join("assets", "fonts", "OpenSans-Regular.ttf")
	atlas, err := font.BuildFontAtlas(fontPath, 48)
	if err != nil {
		return err
	}

	fontRenderer, err := font.NewFontRenderer(atlas)
	if err != nil {
		return err
	}

	// Create UI renderer
	uiRenderer := ui.NewUI()
	if err := uiRenderer.Init(); err != nil {
		return err
	}
	// Allow FIFO UI to also render text when used directly.
	uiRenderer.SetFontRenderer(fontRenderer)

	// Create Item renderer for GUI
	itemRenderer := items.NewItems()
	if err := itemRenderer.Init(); err != nil {
		return err
	}

	h.fontAtlas = atlas
	h.fontRenderer = fontRenderer
	h.uiRenderer = uiRenderer
	h.itemRenderer = itemRenderer

	// Load Textures
	widgetsPath := "assets/textures/gui/widgets.png"
	tex, _, _, err := graphics.LoadTexture(widgetsPath)
	if err != nil {
		return fmt.Errorf("failed to load widgets texture: %v", err)
	}
	h.widgetsTexture = tex

	// Inventory texture loading
	invPath := "assets/textures/gui/inventory.png"
	texInv, _, _, err := graphics.LoadTexture(invPath)
	if err != nil {
		return fmt.Errorf("failed to load inventory texture: %v", err)
	}
	h.inventoryTexture = texInv

	// Icons texture loading
	iconsPath := "assets/textures/gui/icons.png"
	texIcons, _, _, err := graphics.LoadTexture(iconsPath)
	if err != nil {
		return fmt.Errorf("failed to load icons texture: %v", err)
	}
	h.iconsTexture = texIcons

	return nil
}

// Render renders the HUD elements
func (h *HUD) Render(ctx renderer.RenderContext) {
	// Update FPS tracking
	h.frames++
	if time.Since(h.lastFPSCheck) >= time.Second {
		h.currentFPS = h.frames
		h.lastFPSCheck = time.Now()
		h.frames = 0
	}

	// Render player position
	h.renderPlayerPosition(ctx.Player)

	// Render FPS
	h.renderFPS()

	if ctx.Player.IsInventoryOpen {
		// Dim background
		h.uiRenderer.DrawFilledRect(0, 0, h.width, h.height, mgl32.Vec3{0, 0, 0}, 0.70)

		h.renderInventory(ctx.Player)
	}
	// Render Hotbar always
	h.renderHotbar(ctx.Player)
	h.renderHealth(ctx.Player)
	h.renderFood(ctx.Player)

	// Render profiling info if enabled
	if h.showProfiling {
		func() {
			defer profiling.Track("renderer.hud")()
			h.RenderProfilingInfo()
		}()
	}

	// Flush any remaining UI commands (should be minimal; main flush points are inside renderHotbar/renderInventory)
	h.uiRenderer.Flush()
}

func (h *HUD) renderHotbar(p *player.Player) {
	if p.Inventory == nil {
		return
	}

	// Screen dimensions
	screenWidth := h.width
	screenHeight := h.height

	// Hotbar dimensions (widgets.png)
	// Original: 182x22 pixels. Scaled x2 for visibility -> 364x44
	scale := float32(2.0)
	hbW := 182 * scale
	hbH := 22 * scale

	x := (screenWidth - hbW) / 2
	y := screenHeight - hbH - 10 // 10px padding from bottom

	// Draw Hotbar Background
	// UVs: 0,0 to 182/256, 22/256
	u1 := float32(182) / 256.0
	v1 := float32(22) / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.uiRenderer.DrawTexturedRect(x, y, hbW, hbH, h.widgetsTexture, 0, 0, u1, v1, color, 1.0)

	// Draw Selected Slot
	// Selector is 24x24 pixels in texture at 0,22
	selW := 24 * scale
	selH := 24 * scale

	// Slot index (0-8)
	slotIdx := p.Inventory.CurrentItem
	// Each slot is 20px wide in texture grid (approx)
	// Actually:
	// Slot 0 starts at x=3 (relative to 182).
	// Spacing is 20px.
	// Slot x pos = 3 + 20*i
	// But the selector is centered on the slot.
	// Selector texture is at 0, 22. Size 24x24.
	// Selector pos relative to hotbar: (slotX - 1, -1)
	// Let's calculate screen pos directly.

	// Offset for selector logic: -1px in texture space relative to slot start
	slotXTex := 3 + 20*slotIdx
	selXTex := slotXTex - 2 // -2 pixels adjustment to align 24px box over 20px slot?
	// Actually Minecraft logic:
	// Hotbar: 182 wide.
	// Slot 0: x=6 (in 182 width coords? No, let's look at texture)
	// Texture check:
	// First slot seems to be at x=3 (border is 3px).
	// Slot inner is 16x16.
	// Selector is 24x24.
	// Selector draws OVER the hotbar.

	selXScreen := x + float32(selXTex)*scale - float32(1)*scale // Fine tune alignment
	selYScreen := y - float32(1)*scale                          // -1px up

	// Selector UVs
	selU0 := float32(0.0)
	selV0 := float32(22) / 256.0
	selU1 := float32(24) / 256.0
	selV1 := float32(22+24) / 256.0

	h.uiRenderer.DrawTexturedRect(selXScreen, selYScreen, selW, selH, h.widgetsTexture, selU0, selV0, selU1, selV1, color, 1.0)

	// IMPORTANT: UI quads are FIFO-batched; flush backgrounds before rendering items/text so they don't draw over items.
	h.uiRenderer.Flush()

	// Draw Items
	for i := range 9 {
		stack := p.Inventory.MainInventory[i]
		if stack != nil {
			// Calculate slot position
			// Slot is 16x16 items
			// x = 3 + 20*i + 3 (padding to center 16 in 20? No)
			// Let's approximate: 3 + 20*i is left edge of slot area.
			slotX := x + float32(3+20*i)*scale
			slotY := y + float32(3)*scale

			// Render Item
			itemSize := 16 * scale
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			// Render Count if > 1
			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				// Bottom right of slot
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
	}

	// Draw item name text above hotbar if selected
	selItem := p.Inventory.GetCurrentItem()
	if selItem != nil {
		name := "Unknown"
		if def, ok := registry.Blocks[selItem.Type]; ok {
			name = def.Name
		}
		// Center text
		w, _ := h.fontRenderer.Measure(name, 0.4)
		tx := (screenWidth - w) / 2
		ty := y - 30
		h.fontRenderer.Render(name, tx, ty, 0.4, mgl32.Vec3{1, 1, 1})
	}
}

func (h *HUD) renderHealth(p *player.Player) {
	screenWidth := h.width
	screenHeight := h.height
	scale := float32(2.0)

	hbH := 22.0 * scale
	yHotbar := screenHeight - hbH - 10.0
	y := yHotbar - 17.0*scale

	// Start X: Same as hotbar left
	hbW := 182.0 * scale
	startBaseX := (screenWidth - hbW) / 2

	// Textures
	texW := float32(256.0)
	texH := float32(256.0)

	heartW := float32(9.0) * scale
	heartH := float32(9.0) * scale

	uEmpty := float32(16.0) / texW
	vEmpty := float32(0.0) / texH
	uFull := float32(52.0) / texW
	uHalf := float32(61.0) / texW

	uWidth := float32(9.0) / texW
	vHeight := float32(9.0) / texH

	color := mgl32.Vec3{1.0, 1.0, 1.0}

	maxHearts := int(math.Ceil(float64(p.MaxHealth) / 2.0))
	currentHealth := int(math.Ceil(float64(p.Health)))

	for i := 0; i < maxHearts; i++ {
		x := startBaseX + float32(i*8)*scale

		// Draw Empty Heart (Background)
		h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uEmpty, vEmpty, uEmpty+uWidth, vEmpty+vHeight, color, 1.0)

		// Draw Full or Half
		if (i*2 + 1) < currentHealth {
			h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uFull, vEmpty, uFull+uWidth, vEmpty+vHeight, color, 1.0)
		} else if (i*2 + 1) == currentHealth {
			h.uiRenderer.DrawTexturedRect(x, y, heartW, heartH, h.iconsTexture, uHalf, vEmpty, uHalf+uWidth, vEmpty+vHeight, color, 1.0)
		}
	}
}

func (h *HUD) renderFood(p *player.Player) {
	screenWidth := h.width
	screenHeight := h.height
	scale := float32(2.0)

	hbH := 22.0 * scale
	yHotbar := screenHeight - hbH - 10.0
	y := yHotbar - 17.0*scale

	// Center point
	centerX := screenWidth / 2.0
	rightEdge := centerX + 91.0*scale

	// Textures
	texW := float32(256.0)
	texH := float32(256.0)

	iconW := float32(9.0) * scale
	iconH := float32(9.0) * scale

	// Icon UVs for Food (Y=27)
	vBase := float32(27.0) / texH
	uEmpty := float32(16.0) / texW
	uFull := float32(52.0) / texW
	uHalf := float32(61.0) / texW

	uWidth := float32(9.0) / texW
	vHeight := float32(9.0) / texH

	color := mgl32.Vec3{1.0, 1.0, 1.0}

	maxFood := int(math.Ceil(float64(p.MaxFoodLevel) / 2.0))
	currentFood := int(math.Ceil(float64(p.FoodLevel)))

	for i := 0; i < maxFood; i++ {
		// x = rightEdge - i*8*scale - 9*scale
		x := rightEdge - float32(i*8)*scale - 9.0*scale

		// Draw Empty Food (Background)
		h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uEmpty, vBase, uEmpty+uWidth, vBase+vHeight, color, 1.0)

		// Draw Full or Half
		if (i*2 + 1) < currentFood {
			h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uFull, vBase, uFull+uWidth, vBase+vHeight, color, 1.0)
		} else if (i*2 + 1) == currentFood {
			h.uiRenderer.DrawTexturedRect(x, y, iconW, iconH, h.iconsTexture, uHalf, vBase, uHalf+uWidth, vBase+vHeight, color, 1.0)
		}
	}
}

func (h *HUD) renderInventory(p *player.Player) {
	if p.Inventory == nil {
		return
	}

	// Screen dimensions
	screenWidth := h.width
	screenHeight := h.height

	scale := float32(2.0)
	invW := 176 * scale
	invH := 166 * scale

	x := (screenWidth - invW) / 2
	y := (screenHeight - invH) / 2

	// Draw Background
	// UVs: 0,0 to 176/256, 166/256
	u1 := float32(176) / 256.0
	v1 := float32(166) / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}

	// Use inventoryTexture
	h.uiRenderer.DrawTexturedRect(x, y, invW, invH, h.inventoryTexture, 0, 0, u1, v1, color, 1.0)

	// Flush background UI first so items/text draw on top
	h.uiRenderer.Flush()

	// Render "Crafting" text (on top of background)
	craftingX := x + 86*scale
	craftingY := y + 16*scale
	h.fontRenderer.Render("Crafting", craftingX, craftingY, 0.35, mgl32.Vec3{0.3, 0.3, 0.3})

	// Draw Items
	itemSize := 16 * scale

	// Reset hover slot
	h.HoveredSlot = -1

	// Helper to check hover and draw overlay
	drawHoverOverlay := func(slotIdx int, slotX, slotY float32) {
		mx := float32(p.MouseX)
		my := float32(p.MouseY)
		if mx >= slotX && mx < slotX+itemSize && my >= slotY && my < slotY+itemSize {
			// Draw semi-transparent white overlay
			// 0x80FFFFFF in ARGB is 1,1,1,0.5
			h.uiRenderer.DrawFilledRect(slotX, slotY, itemSize, itemSize, mgl32.Vec3{1, 1, 1}, 0.5)
			h.HoveredSlot = slotIdx
		}
	}

	// 1. Main Inventory (Indices 9-35)
	// Java: x=8, y=84
	for i := 9; i < 36; i++ {
		stack := p.Inventory.MainInventory[i]
		// Grid: 9 cols, 3 rows
		// i-9 to normalize to 0-26
		normIdx := i - 9
		col := normIdx % 9
		row := normIdx / 9

		slotX := x + float32(8+col*18)*scale
		slotY := y + float32(84+row*18)*scale

		if stack != nil {
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
		// Draw hover overlay after item (so it's on top)
		drawHoverOverlay(i, slotX, slotY)
	}

	// 2. Hotbar (Indices 0-8)
	// Java: x=8, y=142
	for i := range 9 {
		stack := p.Inventory.MainInventory[i]
		slotX := x + float32(8+i*18)*scale
		slotY := y + float32(142)*scale

		if stack != nil {
			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}
		drawHoverOverlay(i, slotX, slotY)
	}

	// 3. Armor (Indices 0-3 in ArmorInventory)
	// Java: x=8, y=8 start, vertical step 18
	for i := range 4 {
		stack := p.Inventory.ArmorInventory[i]
		if stack != nil {
			slotX := x + float32(8)*scale
			slotY := y + float32(8+i*18)*scale

			h.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)
		}
	}

	// Flush hover overlays (and any other filled UI drawn during inventory) above items.
	h.uiRenderer.Flush()

	// Render Cursor Stack
	if p.Inventory.CursorStack != nil {
		mx := float32(p.MouseX)
		my := float32(p.MouseY)
		// Center item on cursor
		h.itemRenderer.RenderGUI(p.Inventory.CursorStack, mx-itemSize/2, my-itemSize/2, itemSize)

		if p.Inventory.CursorStack.Count > 1 {
			countText := fmt.Sprintf("%d", p.Inventory.CursorStack.Count)
			tx := mx + itemSize/4
			ty := my + itemSize/4
			h.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
		}
	}
}

// HandleInventoryClick handles mouse clicks in the inventory
func (h *HUD) HandleInventoryClick(p *player.Player, x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	if action != glfw.Press {
		return false
	}

	// Coordinates match renderInventory
	scale := float32(2.0)
	invW := 176 * scale
	invH := 166 * scale

	// Screen dimensions
	screenW := h.width
	screenH := h.height

	startX := (screenW - invW) / 2
	startY := (screenH - invH) / 2

	mouseX := float32(x)
	mouseY := float32(y)

	checkSlot := func(idx int, slotX, slotY float32) {
		// slotX, slotY are relative to startX, startY
		sx := startX + slotX
		sy := startY + slotY
		size := 16 * scale

		if mouseX >= sx && mouseX < sx+size && mouseY >= sy && mouseY < sy+size {
			h.handleSlotClick(p, idx, button)
		}
	}

	// Main Inventory (9-35)
	for i := 9; i < 36; i++ {
		col := (i - 9) % 9
		row := (i - 9) / 9

		// Java coords: x=8 + col*18, y=84 + row*18
		checkSlot(i, float32(8+col*18)*scale, float32(84+row*18)*scale)
	}

	// Hotbar (0-8)
	for i := range 9 {
		// Java coords: x=8 + i*18, y=142
		checkSlot(i, float32(8+i*18)*scale, float32(142)*scale)
	}

	return true
}

func (h *HUD) handleSlotClick(p *player.Player, slotIdx int, button glfw.MouseButton) {
	cursor := p.Inventory.CursorStack
	slot := p.Inventory.MainInventory[slotIdx]

	// Double-click detection (within 300ms on the same slot)
	isDoubleClick := false
	if button == glfw.MouseButtonLeft && slotIdx == h.lastClickSlot && time.Since(h.lastClickTime) < 300*time.Millisecond {
		isDoubleClick = true
	}
	h.lastClickSlot = slotIdx
	h.lastClickTime = time.Now()

	// Handle double-click: collect all items of same type
	if isDoubleClick {
		// If cursor has an item, collect all matching items from inventory
		if cursor != nil {
			totalCount := cursor.Count

			// If clicked slot has same item, add it
			if slot != nil && slot.IsItemEqual(*cursor) {
				totalCount += slot.Count
				p.Inventory.MainInventory[slotIdx] = nil
			}

			// Loop through all slots and collect matching items
			for i := range len(p.Inventory.MainInventory) {
				if i == slotIdx {
					continue // Already handled above
				}

				otherSlot := p.Inventory.MainInventory[i]
				if otherSlot != nil && otherSlot.IsItemEqual(*cursor) {
					totalCount += otherSlot.Count
					p.Inventory.MainInventory[i] = nil // Clear the slot
				}
			}

			// Update cursor with total
			cursor.Count = totalCount
			return
		} else if slot != nil {
			// Cursor empty, slot has item: collect all matching items
			totalCount := slot.Count

			// Loop through all slots and collect matching items
			for i := range len(p.Inventory.MainInventory) {
				if i == slotIdx {
					continue // Skip the slot we're clicking
				}

				otherSlot := p.Inventory.MainInventory[i]
				if otherSlot != nil && otherSlot.IsItemEqual(*slot) {
					totalCount += otherSlot.Count
					p.Inventory.MainInventory[i] = nil // Clear the slot
				}
			}

			// Put all collected items in cursor
			if totalCount > 0 {
				cursorStack := item.NewItemStack(slot.Type, totalCount)
				p.Inventory.CursorStack = &cursorStack
				p.Inventory.MainInventory[slotIdx] = nil
			}
			return
		}
	}

	if button == glfw.MouseButtonRight && cursor != nil {
		// Right click: place one item from cursor stack into slot
		if slot == nil {
			// Empty slot: place one item
			newStack := item.NewItemStack(cursor.Type, 1)
			p.Inventory.MainInventory[slotIdx] = &newStack
			cursor.Count--
			if cursor.Count <= 0 {
				p.Inventory.CursorStack = nil
			}
		} else if slot.IsItemEqual(*cursor) {
			// Same item: merge one item if there's space
			if slot.Count < slot.GetMaxStackSize() {
				slot.Count++
				cursor.Count--
				if cursor.Count <= 0 {
					p.Inventory.CursorStack = nil
				}
			}
		}
		return
	}

	// Left click (normal behavior)
	if cursor == nil {
		if slot != nil {
			// Pick up
			p.Inventory.CursorStack = slot
			p.Inventory.MainInventory[slotIdx] = nil
		}
	} else {
		if slot == nil {
			// Place into empty slot
			p.Inventory.MainInventory[slotIdx] = cursor
			p.Inventory.CursorStack = nil
		} else if slot.IsItemEqual(*cursor) {
			// Same item type: merge stacks
			space := slot.GetMaxStackSize() - slot.Count
			if space > 0 {
				toAdd := min(cursor.Count, space)
				slot.Count += toAdd
				cursor.Count -= toAdd
				if cursor.Count <= 0 {
					p.Inventory.CursorStack = nil
				}
			}
		} else {
			// Different items: swap
			p.Inventory.MainInventory[slotIdx] = cursor
			p.Inventory.CursorStack = slot
		}
	}
}

// MoveHoveredItemToHotbar moves the hovered item to the specified hotbar slot
func (h *HUD) MoveHoveredItemToHotbar(p *player.Player, hotbarSlot int) {
	if h.HoveredSlot < 0 || h.HoveredSlot >= inventory.MainInventorySize {
		return
	}

	hoveredStack := p.Inventory.MainInventory[h.HoveredSlot]
	if hoveredStack == nil {
		return
	}

	// If hotbar slot has something, move hovered to cursor instead
	hotbarStack := p.Inventory.MainInventory[hotbarSlot]
	if hotbarStack != nil {
		p.Inventory.CursorStack = hoveredStack
		p.Inventory.MainInventory[h.HoveredSlot] = hotbarStack
		p.Inventory.MainInventory[hotbarSlot] = nil
	} else {
		// Hotbar slot is empty: move hovered item there
		p.Inventory.MainInventory[hotbarSlot] = hoveredStack
		p.Inventory.MainInventory[h.HoveredSlot] = nil
	}
}

// ... Rest of file (Dispose, Profiling, etc) kept same ...
func (h *HUD) Dispose() {
	h.uiRenderer.Dispose()
	h.itemRenderer.Dispose()
	// Font resources are managed by graphics package
}

// ToggleProfiling toggles profiling HUD visibility
func (h *HUD) ToggleProfiling() {
	h.showProfiling = !h.showProfiling
}

// ShowProfiling returns whether profiling is enabled
func (h *HUD) ShowProfiling() bool {
	return h.showProfiling
}

// Profiling methods for external updates
func (h *HUD) ProfilingSetLastTotalFrameDuration(d time.Duration) {
	h.profilingStats.lastTotalFrameDuration = d
}

func (h *HUD) ProfilingSetLastUpdateDuration(d time.Duration) {
	h.profilingStats.lastUpdateDuration = d
}

func (h *HUD) ProfilingSetBreakdown(player, world, glfw, hud, prune time.Duration) {
	h.profilingStats.lastPlayerDuration = player
	h.profilingStats.lastWorldDuration = world
	h.profilingStats.lastGlfwDuration = glfw
	h.profilingStats.lastHudDuration = hud
	h.profilingStats.lastPruneDuration = prune
}

func (h *HUD) ProfilingSetPhysics(physics time.Duration) {
	h.profilingStats.lastPhysicsDuration = physics
}

func (h *HUD) ProfilingSetPhases(preRender, swapEvents time.Duration) {
	h.profilingStats.lastPreRenderDuration = preRender
	h.profilingStats.lastSwapEventsDuration = swapEvents
}

// ProfilingSetRenderDuration stores the render() call duration for this frame
func (h *HUD) ProfilingSetRenderDuration(d time.Duration) {
	h.profilingStats.frameDuration = d
	h.profilingStats.lastFrameDuration = d
	// update rolling history and stats
	if len(h.profilingStats.frameTimeHistory) >= 60 {
		h.profilingStats.frameTimeHistory = h.profilingStats.frameTimeHistory[1:]
	}
	h.profilingStats.frameTimeHistory = append(h.profilingStats.frameTimeHistory, d)
	// recompute min/max/avg
	var total time.Duration
	min := d
	max := d
	for _, v := range h.profilingStats.frameTimeHistory {
		total += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	h.profilingStats.avgFrameTime = 0
	if len(h.profilingStats.frameTimeHistory) > 0 {
		h.profilingStats.avgFrameTime = total / time.Duration(len(h.profilingStats.frameTimeHistory))
	}
	h.profilingStats.minFrameTime = min
	h.profilingStats.maxFrameTime = max
}

func (h *HUD) renderPlayerPosition(p *player.Player) {
	// Build text and draw at top-left
	// Calculate horiz speed (m/s)
	speed := math.Sqrt(float64(p.Velocity[0]*p.Velocity[0] + p.Velocity[2]*p.Velocity[2]))
	text := fmt.Sprintf("Pos: %.2f, %.2f, %.2f | Speed: %.2f m/s | MaxJump: %.3f", p.Position[0], p.Position[1], p.Position[2], speed, p.MaxJumpHeight)
	x := float32(10)
	y := float32(30)
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.fontRenderer.Render(text, x, y, 0.35, color)
}

// renderFPS renders the current FPS value on screen
func (h *HUD) renderFPS() {
	text := fmt.Sprintf("FPS: %d", h.currentFPS)
	x := float32(10)
	y := float32(46)
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.fontRenderer.Render(text, x, y, 0.3, color)
}

// RenderProfilingInfo renders the current profiling information on screen
func (h *HUD) RenderProfilingInfo() {
	lines := make([]string, 0, 64)

	// Frame timing
	tracked := profiling.SumWithPrefix("renderer.")
	frameMs := float64(h.profilingStats.frameDuration.Microseconds()) / 1000.0
	trackedMs := float64(tracked.Microseconds()) / 1000.0
	avgMs := float64(h.profilingStats.avgFrameTime.Microseconds()) / 1000.0
	lines = append(lines, fmt.Sprintf("Frame(render): %.2fms (%.2fms avg) | Tracked(render): %.2fms", frameMs, avgMs, trackedMs))

	// Update from main loop
	if h.profilingStats.lastUpdateDuration > 0 {
		updateMs := float64(h.profilingStats.lastUpdateDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Frame(update): %.2fms", updateMs))
	}

	// Previous frame total and overhead
	if h.profilingStats.lastTotalFrameDuration > 0 {
		totalMs := float64(h.profilingStats.lastTotalFrameDuration.Microseconds()) / 1000.0
		overheadMs := totalMs - frameMs
		if overheadMs < 0 {
			overheadMs = 0
		}
		lines = append(lines, fmt.Sprintf("Frame(total): %.2fms | Overhead(non-render): %.2fms", totalMs, overheadMs))

		// Overhead breakdown
		playerMs := float64(h.profilingStats.lastPlayerDuration.Microseconds()) / 1000.0
		worldMs := float64(h.profilingStats.lastWorldDuration.Microseconds()) / 1000.0
		glfwMs := float64(h.profilingStats.lastGlfwDuration.Microseconds()) / 1000.0
		hudMs := float64(h.profilingStats.lastHudDuration.Microseconds()) / 1000.0
		pruneMs := float64(h.profilingStats.lastPruneDuration.Microseconds()) / 1000.0
		physicsMs := float64(h.profilingStats.lastPhysicsDuration.Microseconds()) / 1000.0
		otherMs := overheadMs - (playerMs + worldMs + glfwMs + hudMs + pruneMs + physicsMs)
		if otherMs < 0 {
			otherMs = 0
		}
		lines = append(lines, fmt.Sprintf("Overhead breakdown -> player: %.2fms, world: %.2fms, glfw: %.2fms, hud: %.2fms, prune: %.2fms, physics: %.2fms, other: %.2fms", playerMs, worldMs, glfwMs, hudMs, pruneMs, physicsMs, otherMs))

		// Phase durations (prev)
		preRenderMs := float64(h.profilingStats.lastPreRenderDuration.Microseconds()) / 1000.0
		swapEventsMs := float64(h.profilingStats.lastSwapEventsDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Phases -> preRender: %.2fms, swap+events: %.2fms", preRenderMs, swapEventsMs))
	}

	// Renderer breakdown from profiling trackers
	frustumMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.frustumSetup").Microseconds()) / 1000.0
	collectMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.collectVisible").Microseconds()) / 1000.0
	ensureMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.ensureMeshes").Microseconds()) / 1000.0
	drawMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.drawAtlas").Microseconds()) / 1000.0
	highlightMs := float64(profiling.SumWithPrefix("renderer.renderHighlightedBlock").Microseconds()) / 1000.0
	handMs := float64(profiling.SumWithPrefix("renderer.renderHand").Microseconds()) / 1000.0
	crossMs := float64(profiling.SumWithPrefix("renderer.renderCrosshair").Microseconds()) / 1000.0
	dirMs := float64(profiling.SumWithPrefix("renderer.renderDirection").Microseconds()) / 1000.0
	if frustumMs+collectMs+ensureMs+drawMs+highlightMs+handMs+crossMs+dirMs > 0 {
		lines = append(lines, fmt.Sprintf("Blocks -> frustum: %.2fms, collect: %.2fms, ensure: %.2fms, draw: %.2fms", frustumMs, collectMs, ensureMs, drawMs))
		lines = append(lines, fmt.Sprintf("Overlays -> highlight: %.2fms, hand: %.2fms, crosshair: %.2fms, direction: %.2fms", highlightMs, handMs, crossMs, dirMs))
	}

	// Top N tracked lines
	if top := profiling.TopN(10); top != "" {
		for line := range strings.SplitSeq(top, ", ") {
			if line != "" && !strings.Contains(line, ":0ms") {
				lines = append(lines, line)
			}
		}
	}

	textColor := mgl32.Vec3{1.0, 1.0, 1.0}
	startY := float32(60)
	lineStep := float32(17)
	h.fontRenderer.RenderLines(lines, 10, startY, lineStep, 0.375, textColor)
}

// Helper methods for profiling data management
func (h *HUD) startFrameProfiling() {
	h.profilingStats.frameStartTime = time.Now()
}

func (h *HUD) endFrameProfiling() {
	h.profilingStats.frameEndTime = time.Now()
	h.profilingStats.frameDuration = h.profilingStats.frameEndTime.Sub(h.profilingStats.frameStartTime)

	// Update frame time history (keep last 60 frames for averaging)
	if len(h.profilingStats.frameTimeHistory) >= 60 {
		h.profilingStats.frameTimeHistory = h.profilingStats.frameTimeHistory[1:]
	}
	h.profilingStats.frameTimeHistory = append(h.profilingStats.frameTimeHistory, h.profilingStats.frameDuration)

	// Calculate statistics
	h.calculateFrameTimeStats()

	// Update GPU memory usage
	h.updateGPUMemoryUsage()

	// Store last frame duration for next frame
	h.profilingStats.lastFrameDuration = h.profilingStats.frameDuration
}

func (h *HUD) resetFrameCounters() {
	h.profilingStats.totalDrawCalls = 0
	h.profilingStats.totalVertices = 0
	h.profilingStats.totalTriangles = 0
	h.profilingStats.visibleChunks = 0
	h.profilingStats.culledChunks = 0
	h.profilingStats.meshedChunks = 0
	h.profilingStats.frustumCullTime = 0
	h.profilingStats.meshingTime = 0
	h.profilingStats.shaderBindTime = 0
	h.profilingStats.vaoBindTime = 0
}

func (h *HUD) calculateFrameTimeStats() {
	if len(h.profilingStats.frameTimeHistory) == 0 {
		return
	}

	var total time.Duration
	h.profilingStats.maxFrameTime = 0
	h.profilingStats.minFrameTime = h.profilingStats.frameTimeHistory[0]

	for _, duration := range h.profilingStats.frameTimeHistory {
		total += duration
		if duration > h.profilingStats.maxFrameTime {
			h.profilingStats.maxFrameTime = duration
		}
		if duration < h.profilingStats.minFrameTime {
			h.profilingStats.minFrameTime = duration
		}
	}

	h.profilingStats.avgFrameTime = total / time.Duration(len(h.profilingStats.frameTimeHistory))
}

func (h *HUD) updateGPUMemoryUsage() {
	h.profilingStats.bufferMemoryUsage = h.estimateBufferMemoryUsage()
	h.profilingStats.textureMemoryUsage = h.estimateTextureMemoryUsage()
	h.profilingStats.gpuMemoryUsage = h.profilingStats.bufferMemoryUsage + h.profilingStats.textureMemoryUsage
}

func (h *HUD) estimateBufferMemoryUsage() int64 {
	// This is a simplified estimation - in a real implementation,
	// you'd track actual buffer allocations
	return 0
}

func (h *HUD) estimateTextureMemoryUsage() int64 {
	return int64(h.fontAtlas.AtlasW * h.fontAtlas.AtlasH * 4) // RGBA
}

// RenderText renders text using the font renderer
func (h *HUD) RenderText(text string, x, y float32, size float32, color mgl32.Vec3) {
	h.fontRenderer.Render(text, x, y, size, color)
}

// MeasureText returns width and height in pixels for the given text at scale
func (h *HUD) MeasureText(text string, scale float32) (float32, float32) {
	return h.fontRenderer.Measure(text, scale)
}

// FontRenderer exposes the HUD's font renderer for UI systems that want to enqueue text.
func (h *HUD) FontRenderer() *font.FontRenderer {
	return h.fontRenderer
}

// SetViewport updates the HUD viewport dimensions
func (h *HUD) SetViewport(width, height int) {
	h.width = float32(width)
	h.height = float32(height)
	h.uiRenderer.SetViewport(width, height)
	h.itemRenderer.SetViewport(width, height)
	h.fontRenderer.SetViewport(float32(width), float32(height))
}
