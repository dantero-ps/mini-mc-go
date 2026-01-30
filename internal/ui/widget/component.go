package widget

import (
	"mini-mc/internal/graphics/renderables/ui"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Component interface {
	Render(u *ui.UI, window *glfw.Window)
	HandleInput(window *glfw.Window, justPressedLeft bool) bool
	SetPosition(x, y float32)
	SetSize(w, h float32)
	GetSize() (float32, float32)
}

type BaseComponent struct {
	X, Y, W, H float32
}

func (b *BaseComponent) SetPosition(x, y float32)    { b.X, b.Y = x, y }
func (b *BaseComponent) SetSize(w, h float32)        { b.W, b.H = w, h }
func (b *BaseComponent) GetSize() (float32, float32) { return b.W, b.H }
