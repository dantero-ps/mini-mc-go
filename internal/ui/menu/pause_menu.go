package menu

import (
	"fmt"
	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/ui/widget"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type PauseMenu struct {
	buttons      []*widget.Button
	renderDist   *widget.Slider
	fpsLimit     *widget.Slider
	bobbing      *widget.Toggle
	shouldResume bool
	shouldQuit   bool
}

func NewPauseMenu() *PauseMenu {
	pm := &PauseMenu{}

	// Initialize Sliders & Toggles with current config
	// Render Distance: Range 5-50. Slider 0-1 mapped to this.
	curDist := config.GetRenderDistance()
	distVal := float32(curDist-5) / float32(50-5)
	pm.renderDist = widget.NewSlider(0, 0, 200, 20, distVal, 46, "renderDist", func(val float32) {
		chunks := int(5 + val*45 + 0.5)
		config.SetRenderDistance(chunks)
	})

	// FPS Limit: Range 30-240 (mapped), 0 (uncapped) at max.
	// Logic: 0.0-0.9 -> 30-240. >0.9 -> Uncapped.
	curFPS := config.GetFPSLimit()
	var fpsVal float32
	if curFPS <= 0 {
		fpsVal = 1.0
	} else {
		fpsVal = float32(curFPS-30) / float32(240-30)
		if fpsVal > 0.95 {
			fpsVal = 0.95 // Keep it within range to avoid accidental uncapped visual jump
		}
	}
	pm.fpsLimit = widget.NewSlider(0, 0, 200, 20, fpsVal, 211, "fpsLimit", func(val float32) {
		if val > 0.99 {
			config.SetFPSLimit(0)
		} else {
			limit := int(30 + val*210 + 0.5)
			config.SetFPSLimit(limit)
		}
	})

	// View Bobbing
	pm.bobbing = widget.NewToggle("View Bobbing", 0, 0, 40, 20, config.GetViewBobbing(), func(isOn bool) {
		config.SetViewBobbing(isOn)
	})

	// Resume Button
	resumeBtn := widget.NewButton("Devam Et", 0, 0, 200, 40, func() {
		pm.shouldResume = true
	})
	resumeBtn.NormalColor = mgl32.Vec3{0.2, 0.2, 0.2}
	resumeBtn.HoverColor = mgl32.Vec3{0.3, 0.3, 0.3}
	pm.buttons = append(pm.buttons, resumeBtn)

	// Quit Button
	quitBtn := widget.NewButton("Ana Menü", 0, 0, 200, 40, func() {
		pm.shouldQuit = true
	})
	quitBtn.NormalColor = mgl32.Vec3{0.2, 0.2, 0.2}
	quitBtn.HoverColor = mgl32.Vec3{0.3, 0.3, 0.3}
	pm.buttons = append(pm.buttons, quitBtn)

	return pm
}

func (p *PauseMenu) Update(window *glfw.Window, justPressedLeft bool) Action {
	p.shouldResume = false
	p.shouldQuit = false

	// Update sync with config (in case changed externally)
	// For sliders, we trust internal state unless we want full bi-directional sync every frame.
	// For toggle, it's safer to sync to visual if changed by keybind?
	p.bobbing.IsOn = config.GetViewBobbing()

	// Update components
	// Render handles slider input (DrawSlider), but we need to propagate clicks for buttons/toggles
	p.bobbing.HandleInput(window, justPressedLeft)
	for _, btn := range p.buttons {
		btn.HandleInput(window, justPressedLeft)
	}

	if p.shouldResume {
		return ActionResume
	}
	if p.shouldQuit {
		return ActionQuitToMenu
	}
	return ActionNone
}

func (p *PauseMenu) Render(u *ui.UI, window *glfw.Window) {
	// Draw background overlay
	winW, winH := window.GetSize()
	fWinW, fWinH := float32(winW), float32(winH)
	u.DrawFilledRect(0, 0, fWinW, fWinH, mgl32.Vec3{0, 0, 0}, 0.5)

	centerX := fWinW / 2

	// Title
	title := "PAUSED"
	tw, _ := u.MeasureText(title, 1.0)
	u.DrawText(title, centerX-tw/2, 80, 1.0, mgl32.Vec3{1, 1, 1})

	// Layout Constants
	startY := float32(150.0)
	spacing := float32(70.0)
	sliderW := float32(200.0)
	sliderH := float32(20.0)

	// 1. Render Distance
	// Label
	rdTitle := "Render Distance"
	rdW, _ := u.MeasureText(rdTitle, 0.4)
	u.DrawText(rdTitle, centerX-rdW/2, startY-15, 0.4, mgl32.Vec3{1, 1, 1})
	// Slider
	p.renderDist.X = centerX - sliderW/2
	p.renderDist.Y = startY
	p.renderDist.W = sliderW
	p.renderDist.H = sliderH
	p.renderDist.Render(u, window)
	// Value Text
	distVal := int(5 + p.renderDist.Value*45 + 0.5)
	u.DrawText(fmt.Sprintf("%d chunks", distVal), p.renderDist.X+sliderW+10, startY+15, 0.35, mgl32.Vec3{0.8, 0.8, 0.8})

	startY += spacing

	// 2. FPS Limit
	// Label
	fpsTitle := "FPS Limit"
	fpsW, _ := u.MeasureText(fpsTitle, 0.4)
	u.DrawText(fpsTitle, centerX-fpsW/2, startY-15, 0.4, mgl32.Vec3{1, 1, 1})
	// Slider
	p.fpsLimit.X = centerX - sliderW/2
	p.fpsLimit.Y = startY
	p.fpsLimit.W = sliderW
	p.fpsLimit.H = sliderH
	p.fpsLimit.Render(u, window)
	// Value Text
	var fpsText string
	if p.fpsLimit.Value > 0.99 {
		fpsText = "Uncapped"
	} else {
		limit := int(30 + p.fpsLimit.Value*210 + 0.5)
		fpsText = fmt.Sprintf("%d FPS", limit)
	}
	u.DrawText(fpsText, p.fpsLimit.X+sliderW+10, startY+15, 0.35, mgl32.Vec3{0.8, 0.8, 0.8})

	startY += spacing

	// 3. View Bobbing
	// Label
	bobTitle := "View Bobbing"
	bobW, _ := u.MeasureText(bobTitle, 0.4)
	u.DrawText(bobTitle, centerX-bobW/2, startY-15, 0.4, mgl32.Vec3{1, 1, 1})
	// Toggle
	toggleW := float32(40.0)
	p.bobbing.X = centerX - toggleW/2
	p.bobbing.Y = startY
	p.bobbing.W = toggleW
	p.bobbing.H = float32(20.0)
	p.bobbing.Render(u, window)
	// Status Text
	statusText := "Kapalı"
	if p.bobbing.IsOn {
		statusText = "Açık"
	}
	u.DrawText(statusText, p.bobbing.X+toggleW+10, startY+15, 0.35, mgl32.Vec3{0.8, 0.8, 0.8})

	startY += spacing

	// 4. Resume Button
	p.buttons[0].SetPosition(centerX-100, startY)
	p.buttons[0].Render(u, window)

	startY += 50

	// 5. Quit Button
	p.buttons[1].SetPosition(centerX-100, startY)
	p.buttons[1].Render(u, window)
}
