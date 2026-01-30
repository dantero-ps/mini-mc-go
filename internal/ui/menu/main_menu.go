package menu

import (
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/ui/widget"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type MainMenu struct {
	buttons             []*widget.Button
	shouldStartSurvival bool
	shouldStartCreative bool
}

func NewMainMenu() *MainMenu {
	mm := &MainMenu{}

	// Survival Button
	survivalBtn := widget.NewButton("Survival", 0, 0, 0, 0, func() {
		mm.shouldStartSurvival = true
	})
	survivalBtn.TextColor = mgl32.Vec3{0, 1, 0}
	survivalBtn.Subtitle = "No Flying, Normal Mining"
	mm.buttons = append(mm.buttons, survivalBtn)

	// Creative Button
	creativeBtn := widget.NewButton("Creative", 0, 0, 0, 0, func() {
		mm.shouldStartCreative = true
	})
	creativeBtn.TextColor = mgl32.Vec3{0, 0.8, 1}
	creativeBtn.Subtitle = "Flying, Instant Break"
	mm.buttons = append(mm.buttons, creativeBtn)

	return mm
}

func (m *MainMenu) Update(window *glfw.Window, justPressedLeft bool) Action {
	m.shouldStartSurvival = false
	m.shouldStartCreative = false

	for _, btn := range m.buttons {
		btn.HandleInput(window, justPressedLeft)
	}

	if m.shouldStartSurvival {
		return ActionStartSurvival
	}
	if m.shouldStartCreative {
		return ActionStartCreative
	}
	return ActionNone
}

func (m *MainMenu) Render(u *ui.UI, window *glfw.Window) {
	width, height := window.GetSize()
	fWinW, fWinH := float32(width), float32(height)

	// Calculate scale
	scaleX := fWinW / 900.0
	scaleY := fWinH / 600.0
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	centerX := fWinW / 2
	centerY := fWinH / 2

	// Button dimensions
	btnW := 400.0 * scale
	btnH := 80.0 * scale
	btnX := centerX - (btnW / 2)

	// Update Button 1 (Survival)
	// Position: centerY - (40 * scale)
	sBtnY := centerY - (40 * scale)
	m.buttons[0].SetPosition(btnX, sBtnY)
	m.buttons[0].SetSize(btnW, btnH)

	// Update Button 2 (Creative)
	// Position: centerY + (60 * scale)
	cBtnY := centerY + (60 * scale)
	m.buttons[1].SetPosition(btnX, cBtnY)
	m.buttons[1].SetSize(btnW, btnH)

	// Draw background
	u.DrawFilledRect(0, 0, fWinW, fWinH, mgl32.Vec3{0.1, 0.1, 0.1}, 1.0)

	// Title: MINI MC
	title := "MINI MC"
	titleScale := 1.0 * scale
	tw, _ := u.MeasureText(title, titleScale)
	u.DrawText(title, centerX-tw/2, centerY-(200*scale), titleScale, mgl32.Vec3{1, 1, 1})

	// Subtitle: Select Game Mode
	subTitle := "Select Game Mode:"
	subScale := 0.5 * scale
	sw, _ := u.MeasureText(subTitle, subScale)
	u.DrawText(subTitle, centerX-sw/2, centerY-(140*scale), subScale, mgl32.Vec3{0.8, 0.8, 0.8})

	// Render buttons (they now have updated positions)
	for _, btn := range m.buttons {
		btn.Render(u, window)
	}
}
