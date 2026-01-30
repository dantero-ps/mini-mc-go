package widget

import (
	"mini-mc/internal/graphics/renderables/ui"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Slider struct {
	BaseComponent
	Value    float32 // 0.0 to 1.0
	Steps    int
	ID       string
	Label    string // For internal use or debugging
	OnChange func(val float32)
}

func NewSlider(x, y, w, h float32, initialVal float32, steps int, id string, onChange func(val float32)) *Slider {
	return &Slider{
		BaseComponent: BaseComponent{X: x, Y: y, W: w, H: h},
		Value:         initialVal,
		Steps:         steps,
		ID:            id,
		OnChange:      onChange,
	}
}

func (s *Slider) Render(u *ui.UI, window *glfw.Window) {
	// DrawSlider updates the value based on input and returns it
	newValue := u.DrawSlider(s.X, s.Y, s.W, s.H, s.Value, window, s.Steps, s.ID)

	if newValue != s.Value {
		s.Value = newValue
		if s.OnChange != nil {
			s.OnChange(s.Value)
		}
	}
}

func (s *Slider) HandleInput(window *glfw.Window, justPressedLeft bool) bool {
	// Logic is handled in DrawSlider
	return false
}
