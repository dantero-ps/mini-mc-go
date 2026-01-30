package widget

import (
	"mini-mc/internal/graphics/renderables/ui"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Toggle struct {
	BaseComponent
	Label     string
	IsOn      bool
	OnToggle  func(isOn bool)
	IsHovered bool
}

func NewToggle(label string, x, y, w, h float32, initial bool, onToggle func(isOn bool)) *Toggle {
	return &Toggle{
		BaseComponent: BaseComponent{X: x, Y: y, W: w, H: h},
		Label:         label,
		IsOn:          initial,
		OnToggle:      onToggle,
	}
}

func (t *Toggle) Render(u *ui.UI, window *glfw.Window) {
	mx, my := window.GetCursorPos()
	mx32, my32 := float32(mx), float32(my)

	t.IsHovered = mx32 >= t.X && mx32 <= t.X+t.W && my32 >= t.Y && my32 <= t.Y+t.H

	// Background color logic
	bgColor := mgl32.Vec3{0.2, 0.2, 0.2}
	if t.IsOn {
		bgColor = mgl32.Vec3{0.2, 0.5, 0.2} // Green when enabled
	} else {
		bgColor = mgl32.Vec3{0.5, 0.2, 0.2} // Red when disabled
	}
	if t.IsHovered {
		bgColor = bgColor.Mul(1.2) // Brighten on hover
	}

	u.DrawFilledRect(t.X, t.Y, t.W, t.H, bgColor, 0.85)

	// Draw label (centered above or inside? Old code put label outside and status inside/outside)
	// Old code snippet: Draw status text inside/next to it?
	// "View Bobbing" label was above, "Açık/Kapalı" was to the right.
	// We'll rely on the caller to draw external text if needed, or we simple render the toggle box here.
	// But let's try to match the self-contained widget style.

	// For now, just DrawFilledRect. The caller (PauseMenu) will handle specific text placement to match exactly.
}

func (t *Toggle) HandleInput(window *glfw.Window, justPressedLeft bool) bool {
	if t.IsHovered && justPressedLeft {
		t.IsOn = !t.IsOn
		if t.OnToggle != nil {
			t.OnToggle(t.IsOn)
		}
		return true
	}
	return false
}
