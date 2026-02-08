package hud

import (
	"mini-mc/internal/inventory"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// NullScreen is a null object pattern implementation of Screen interface
type NullScreen struct{}

// Init implements Screen
func (s *NullScreen) Init() {}

// Render implements Screen
func (s *NullScreen) Render(mouseX, mouseY float64) {}

// HandleClick implements Screen
func (s *NullScreen) HandleClick(x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	return false
}

// Close implements Screen
func (s *NullScreen) Close() {}

// Update implements Screen
func (s *NullScreen) Update() {}

// IsPauseScreen implements Screen
func (s *NullScreen) IsPauseScreen() bool {
	return false
}

// GetHoveredSlot implements Screen
func (s *NullScreen) GetHoveredSlot() int {
	return -1
}

// IsActive implements Screen
func (s *NullScreen) IsActive() bool {
	return false
}

// GetContainer implements Screen
func (s *NullScreen) GetContainer() *inventory.Container {
	return nil
}
