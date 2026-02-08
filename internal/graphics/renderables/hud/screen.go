package hud

import (
	"mini-mc/internal/inventory"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Screen represents a GUI screen
type Screen interface {
	// Init initializes the screen
	Init()
	// Render renders the screen
	Render(mouseX, mouseY float64)
	// HandleClick handles mouse clicks
	HandleClick(x, y float64, button glfw.MouseButton, action glfw.Action) bool
	// Close cleans up resources when screen is closed
	Close()
	// Update is called every frame for logic updates
	Update()
	// IsPauseScreen returns whether this screen pauses the game
	IsPauseScreen() bool
	// GetHoveredSlot returns the currently hovered slot index or -1
	GetHoveredSlot() int
	// GetContainer returns the underlying container if available, or nil
	GetContainer() *inventory.Container
	// IsActive returns whether this screen is currently active and should be rendered/processed
	IsActive() bool
}
