package hud

import (
	"fmt"
	"mini-mc/internal/graphics"
	"mini-mc/internal/graphics/renderables/font"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderables/playermodel"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/profiling"
	"path/filepath"
	"time"

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
	profilingStats ProfilingStats
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

	// Create Player Model
	playerModel := playermodel.NewPlayerModel()
	if err := playerModel.Init(); err != nil {
		return err
	}
	h.playerModel = playerModel

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

func (h *HUD) Dispose() {
	h.uiRenderer.Dispose()
	h.itemRenderer.Dispose()
	if h.playerModel != nil {
		h.playerModel.Dispose()
	}
	// Font resources are managed by graphics package
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
