package hud

import (
	"fmt"
	"mini-mc/internal/graphics"
	"mini-mc/internal/inventory"
	"mini-mc/internal/player"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type InventoryScreen struct {
	*ContainerScreen
	playerModel interface{} // interface{} to avoid circular dependency if needed? No, logic is in hud package.
	// Actually HUD in ContainerScreen has playerModel references.
}

func NewInventoryScreen(hud *HUD, p *player.Player) *InventoryScreen {
	// Create logic container
	container := inventory.NewPlayerContainer(p.Inventory)

	// Load inventory texture
	tex, err := graphics.GetTexture("assets/textures/gui/inventory.png")
	if err != nil {
		panic(fmt.Errorf("failed to load inventory texture: %v", err))
	}

	// Create visual screen (176x166 pixels)
	base := NewContainerScreen(hud, p, container, tex, 176, 166)

	s := &InventoryScreen{
		ContainerScreen: base,
	}
	// Initial init
	s.Init()
	return s
}

func (s *InventoryScreen) Render(mouseX, mouseY float64) {
	// 1. Base Render (Background + Slots + Cursor Item)
	s.ContainerScreen.Render(mouseX, mouseY)

	// 2. Custom Draw: "Crafting" Text
	// Position relative to screen
	// In original code: craftingX := x + 86*scale, craftingY := y + 16*scale
	scale := s.Scale
	craftingX := s.X + 86*scale
	craftingY := s.Y + 16*scale
	s.HUD.fontRenderer.Render("Crafting", craftingX, craftingY, 0.35, mgl32.Vec3{0.3, 0.3, 0.3})

	// 3. Custom Draw: Player Model
	// Position in MC: x + 51, y + 75.
	playerX := s.X + 51*scale
	playerY := s.Y + 75*scale
	playerScale := 30.0 * scale

	s.HUD.playerModel.RenderInventoryPlayer(s.Player, playerX, playerY, playerScale, float32(mouseX), float32(mouseY), s.HUD.width, s.HUD.height, glfw.GetTime())

	// Note: We might be drawing over items if items overlap these areas (they shouldn't in standard inventory)
}

// Override HandleClick if custom logic needed, or use Base
func (s *InventoryScreen) HandleClick(x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	// Base handles slot clicks
	handled := s.ContainerScreen.HandleClick(x, y, button, action)

	// If base didn't handle it (clicked outside slots), we could handle custom buttons here
	return handled
}
