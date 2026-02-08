package hud

import (
	"mini-mc/internal/graphics/renderables/font"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderables/playermodel"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"path/filepath"
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
	playerModel   *playermodel.PlayerModel
	showProfiling bool

	// Viewport dimensions
	width  float32
	height float32

	// Profiling state
	frames       int
	lastFPSCheck time.Time
	currentFPS   int

	// Enhanced profiling metrics
	profilingStats ProfilingStats

	// Current active screen (e.g. inventory)
	currentScreen Screen
}

// NewHUD creates a new HUD renderable
func NewHUD() *HUD {
	return &HUD{
		showProfiling: false,
		width:         900,
		height:        600,
		currentScreen: &NullScreen{},
	}
}

// SetInventoryOpen handles inventory state changes
func (h *HUD) SetInventoryOpen(open bool, p *player.Player) {
	if open {
		if !h.currentScreen.IsActive() {
			h.currentScreen = NewInventoryScreen(h, p)
		}
	} else {
		if h.currentScreen.IsActive() {
			h.currentScreen.Close()
			h.currentScreen = &NullScreen{}
		}
	}
}

// Init initializes the HUD rendering system
func (h *HUD) Init() error {
	// Load font atlas and renderer
	fontPath := filepath.Join("assets", "fonts", "Minecraft.otf")
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

	// Create Player Model
	playerModel := playermodel.NewPlayerModel()
	if err := playerModel.Init(); err != nil {
		return err
	}
	h.playerModel = playerModel

	return nil
}

// Render renders the HUD elements
func (h *HUD) Render(ctx renderer.RenderContext) {
	h.frames++
	if time.Since(h.lastFPSCheck) >= time.Second {
		h.currentFPS = h.frames
		h.lastFPSCheck = time.Now()
		h.frames = 0
	}

	// Render World-Level HUD elements (Hotbar, Health, Food) which should be dimmed by menus
	h.renderHotbar(ctx.Player)
	h.renderHealth(ctx.Player)
	h.renderFood(ctx.Player)

	if ctx.Player.IsInventoryOpen {
		// Dim background
		h.uiRenderer.DrawFilledRect(0, 0, h.width, h.height, mgl32.Vec3{0, 0, 0}, 0.70)

		h.currentScreen.Render(ctx.Player.MouseX, ctx.Player.MouseY)
	} else {
		if h.currentScreen.IsActive() {
			h.currentScreen.Close()
			h.currentScreen = &NullScreen{}
		}
	}

	// Render Debug Info (FPS, Coords) - Always on top
	h.renderPlayerPosition(ctx.Player)
	h.renderFPS()

	// Render profiling info if enabled
	if h.showProfiling {
		func() {
			defer profiling.Track("renderer.hud")()
			h.RenderProfilingInfo()
		}()
	}

	// Flush any remaining UI commands
	h.uiRenderer.Flush()
}

func (h *HUD) HandleInventoryClick(x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	return h.currentScreen.HandleClick(x, y, button, action)
}

// MoveHoveredItemToHotbar moves the hovered item to the specified hotbar slot
func (h *HUD) MoveHoveredItemToHotbar(hotbarSlot int) {
	hoveredSlot := h.currentScreen.GetHoveredSlot()
	if hoveredSlot == -1 {
		return
	}

	container := h.currentScreen.GetContainer()
	if container == nil {
		return
	}

	targetSlotIndex := 27 + hotbarSlot

	if hoveredSlot < 0 || hoveredSlot >= len(container.Slots) {
		return
	}
	if targetSlotIndex < 0 || targetSlotIndex >= len(container.Slots) {
		return
	}

	sourceSlot := container.Slots[hoveredSlot]
	targetSlot := container.Slots[targetSlotIndex]

	sourceStack := sourceSlot.GetStack()
	targetStack := targetSlot.GetStack()

	// Swap them
	sourceSlot.PutStack(targetStack)
	targetSlot.PutStack(sourceStack)
}

func (h *HUD) Dispose() {
	h.uiRenderer.Dispose()
	h.itemRenderer.Dispose()
	if h.playerModel != nil {
		h.playerModel.Dispose()
	}
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
