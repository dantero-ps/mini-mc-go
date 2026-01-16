package main

import (
	"fmt"

	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/player"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type MenuAction int

const (
	MenuActionNone MenuAction = iota
	MenuActionResume
	MenuActionQuitToMenu
)

const (
	minFPSCap = 30
	maxFPSCap = 240
)

// PauseMenu handles the pause menu UI and state
type PauseMenu struct {
	renderDistanceSlider float32
	fpsLimitSlider       float32
	mouseWasDown         bool
}

// NewPauseMenu creates a new pause menu with initial slider values from config
func NewPauseMenu() *PauseMenu {
	pm := &PauseMenu{}
	pm.initSliders()
	return pm
}

func (pm *PauseMenu) initSliders() {
	// Render distance slider: normalize to 0-1
	pm.renderDistanceSlider = float32(config.GetRenderDistance()-5) / float32(50-5)

	// FPS limit slider: rightmost => uncapped
	fpsLimit := config.GetFPSLimit()
	if fpsLimit <= 0 {
		pm.fpsLimitSlider = 1.0
	} else {
		if fpsLimit < minFPSCap {
			fpsLimit = minFPSCap
		}
		if fpsLimit > maxFPSCap {
			fpsLimit = maxFPSCap
		}
		pm.fpsLimitSlider = float32(fpsLimit-minFPSCap) / float32(maxFPSCap-minFPSCap)
	}
}

// Render draws the pause menu and handles interactions.
// Returns the action taken by the user.
func (pm *PauseMenu) Render(window *glfw.Window, uiRenderer *ui.UI, hudRenderer *hud.HUD, p *player.Player) MenuAction {
	// Ensure UI text draws through the FIFO UI renderer (one Flush, correct order).
	uiRenderer.SetFontRenderer(hudRenderer.FontRenderer())

	// Get window size for dynamic positioning
	winWidth, winHeight := window.GetSize()
	uiRenderer.SetViewport(winWidth, winHeight)
	hudRenderer.SetViewport(winWidth, winHeight)

	// Dim background
	uiRenderer.DrawFilledRect(0, 0, float32(winWidth), float32(winHeight), mgl32.Vec3{0, 0, 0}, 0.45)

	// Render Distance Slider
	pm.renderRenderDistanceSlider(window, uiRenderer, hudRenderer)

	// FPS Limit Slider
	pm.renderFPSLimitSlider(window, uiRenderer, hudRenderer)

	// Resume button
	resume := pm.renderResumeButton(window, uiRenderer, hudRenderer, p)

	// Main Menu button
	mainMenu := pm.renderMainMenuButton(window, uiRenderer, hudRenderer)

	action := MenuActionNone
	if resume {
		pm.mouseWasDown = true
		action = MenuActionResume
	} else if mainMenu {
		pm.mouseWasDown = true
		action = MenuActionQuitToMenu
	}

	// Single flush per pause-menu frame.
	uiRenderer.Flush()

	// Update mouse state at the end of frame
	pm.mouseWasDown = window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press

	return action
}

func (pm *PauseMenu) renderMainMenuButton(window *glfw.Window, uiRenderer *ui.UI, hudRenderer *hud.HUD) bool {
	// Read mouse once for UI interactions
	cx, cy := window.GetCursorPos()
	mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	winWidth, _ := window.GetSize()

	label := "Ana Menü"
	scale := float32(0.5)
	tw, th := hudRenderer.MeasureText(label, scale)
	paddingX := float32(24)
	paddingY := float32(14)
	btnW := tw + paddingX*2
	btnH := th + paddingY*2
	btnX := (float32(winWidth) - btnW) / 2
	// Position below Resume button
	renderDistanceSliderY := float32(200)
	renderDistanceSliderH := float32(20)
	fpsY := renderDistanceSliderY + renderDistanceSliderH + 60
	fpsH := float32(20)
	resumeBtnY := fpsY + fpsH + 60
	// Calculate Resume button height using same scale and padding
	resumeLabel := "Devam Et"
	resumeScale := float32(0.5)
	_, resumeTH := hudRenderer.MeasureText(resumeLabel, resumeScale)
	resumePaddingY := float32(14)
	resumeBtnH := resumeTH + resumePaddingY*2
	btnY := resumeBtnY + resumeBtnH + 20

	hover := float32(cx) >= btnX && float32(cx) <= btnX+btnW && float32(cy) >= btnY && float32(cy) <= btnY+btnH
	base := mgl32.Vec3{0.20, 0.20, 0.20}
	if hover {
		base = mgl32.Vec3{0.30, 0.30, 0.30}
	}
	uiRenderer.DrawFilledRect(btnX, btnY, btnW, btnH, base, 0.85)

	shouldQuit := false
	if mouseDown && !pm.mouseWasDown && hover {
		shouldQuit = true
	}

	renderMainMenuButtonText(uiRenderer, hudRenderer, winWidth, btnY)

	return shouldQuit
}

func (pm *PauseMenu) renderRenderDistanceSlider(window *glfw.Window, uiRenderer *ui.UI, hudRenderer *hud.HUD) {
	winWidth, _ := window.GetSize()
	sliderLabel := "Render Distance"
	sliderScale := float32(0.4)
	_ = sliderLabel
	_ = sliderScale
	sliderW := float32(200)
	sliderH := float32(20)
	sliderX := (float32(winWidth) - sliderW) / 2
	sliderY := float32(200)

	// Draw and handle slider (snap to steps equal to render distance range)
	steps := 50 - 5 + 1 // inclusive range [5..50]
	newSliderValue := uiRenderer.DrawSlider(sliderX, sliderY, sliderW, sliderH, pm.renderDistanceSlider, window, steps, "renderDistance")
	if newSliderValue != pm.renderDistanceSlider {
		pm.renderDistanceSlider = newSliderValue
		// Convert slider value to render distance (5-50 range)
		newDistance := int(float32(5) + pm.renderDistanceSlider*float32(50-5) + 0.5)
		config.SetRenderDistance(newDistance)
	}

	renderRenderDistanceSliderText(uiRenderer, hudRenderer, winWidth)
}

func (pm *PauseMenu) renderFPSLimitSlider(window *glfw.Window, uiRenderer *ui.UI, hudRenderer *hud.HUD) {
	winWidth, _ := window.GetSize()
	fpsLabel := "FPS Limit"
	fpsScale := float32(0.45)
	_ = fpsLabel
	_ = fpsScale
	fpsW := float32(200)
	fpsH := float32(20)
	fpsX := (float32(winWidth) - fpsW) / 2
	// Position below Render Distance slider
	renderDistanceSliderY := float32(200)
	renderDistanceSliderH := float32(20)
	fpsY := renderDistanceSliderY + renderDistanceSliderH + 60

	// 30..240 inclusive -> snapping, rightmost => uncapped
	newFPSValue := uiRenderer.DrawSlider(fpsX, fpsY, fpsW, fpsH, pm.fpsLimitSlider, window, (maxFPSCap-minFPSCap)+1, "fpsLimit")
	if newFPSValue != pm.fpsLimitSlider {
		pm.fpsLimitSlider = newFPSValue
		// Map slider: rightmost => uncapped (0), else 30..240
		var newLimit int
		if pm.fpsLimitSlider >= 0.999 {
			newLimit = 0
		} else {
			newLimit = int(float32(minFPSCap) + pm.fpsLimitSlider*float32(maxFPSCap-minFPSCap) + 0.5)
		}
		config.SetFPSLimit(newLimit)
	}

	renderFPSLimitSliderText(uiRenderer, hudRenderer, winWidth)
}

func (pm *PauseMenu) renderResumeButton(window *glfw.Window, uiRenderer *ui.UI, hudRenderer *hud.HUD, p *player.Player) bool {
	// Read mouse once for UI interactions
	cx, cy := window.GetCursorPos()
	mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	winWidth, _ := window.GetSize()

	label := "Devam Et"
	scale := float32(0.5)
	tw, th := hudRenderer.MeasureText(label, scale)
	paddingX := float32(24)
	paddingY := float32(14)
	btnW := tw + paddingX*2
	btnH := th + paddingY*2
	btnX := (float32(winWidth) - btnW) / 2
	// Position below FPS Limit slider
	renderDistanceSliderY := float32(200)
	renderDistanceSliderH := float32(20)
	fpsY := renderDistanceSliderY + renderDistanceSliderH + 60
	fpsH := float32(20)
	btnY := fpsY + fpsH + 60

	hover := float32(cx) >= btnX && float32(cx) <= btnX+btnW && float32(cy) >= btnY && float32(cy) <= btnY+btnH
	base := mgl32.Vec3{0.20, 0.20, 0.20}
	if hover {
		base = mgl32.Vec3{0.30, 0.30, 0.30}
	}
	uiRenderer.DrawFilledRect(btnX, btnY, btnW, btnH, base, 0.85)

	shouldResume := false
	if mouseDown && !pm.mouseWasDown && hover {
		shouldResume = true
		window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		p.FirstMouse = true
	}

	renderResumeButtonText(uiRenderer, hudRenderer, winWidth, btnY)

	return shouldResume
}

func renderRenderDistanceSliderText(uiRenderer *ui.UI, hudRenderer *hud.HUD, winWidth int) {
	sliderLabel := "Render Distance"
	sliderScale := float32(0.4)
	_, sliderTH := hudRenderer.MeasureText(sliderLabel, sliderScale)
	sliderW := float32(200)
	sliderH := float32(20)
	sliderX := (float32(winWidth) - sliderW) / 2
	sliderY := float32(200)
	tw, _ := hudRenderer.MeasureText(sliderLabel, sliderScale)
	labelX := (float32(winWidth) - tw) / 2
	uiRenderer.DrawText(sliderLabel, labelX, sliderY-10, sliderScale, mgl32.Vec3{1, 1, 1})
	currentDistance := config.GetRenderDistance()
	distanceText := fmt.Sprintf("%d chunks", currentDistance)
	uiRenderer.DrawText(distanceText, sliderX+sliderW+10, sliderY+sliderH/2+sliderTH/2, sliderScale, mgl32.Vec3{0.8, 0.8, 0.8})
}

func renderFPSLimitSliderText(uiRenderer *ui.UI, hudRenderer *hud.HUD, winWidth int) {
	fpsLabel := "FPS Limit"
	fpsScale := float32(0.45)
	_, fpsTH := hudRenderer.MeasureText(fpsLabel, fpsScale)
	fpsW := float32(200)
	fpsH := float32(20)
	fpsX := (float32(winWidth) - fpsW) / 2
	// Position below Render Distance slider
	renderDistanceSliderY := float32(200)
	renderDistanceSliderH := float32(20)
	fpsY := renderDistanceSliderY + renderDistanceSliderH + 60
	tw, _ := hudRenderer.MeasureText(fpsLabel, fpsScale)
	labelX := (float32(winWidth) - tw) / 2
	uiRenderer.DrawText(fpsLabel, labelX, fpsY-10, fpsScale, mgl32.Vec3{1, 1, 1})
	currentFPSLimit := config.GetFPSLimit()
	var limitText string
	if currentFPSLimit <= 0 {
		limitText = "Uncapped"
	} else {
		limitText = fmt.Sprintf("%d FPS", currentFPSLimit)
	}
	uiRenderer.DrawText(limitText, fpsX+fpsW+10, fpsY+fpsH/2+fpsTH/2, fpsScale, mgl32.Vec3{0.8, 0.8, 0.8})
}

func renderResumeButtonText(uiRenderer *ui.UI, hudRenderer *hud.HUD, winWidth int, btnY float32) {
	label := "Devam Et"
	scale := float32(0.5)
	tw, th := hudRenderer.MeasureText(label, scale)
	paddingX := float32(24)
	paddingY := float32(14)
	btnW := tw + paddingX*2
	btnX := (float32(winWidth) - btnW) / 2
	tx := btnX + paddingX
	ty := btnY + paddingY + th
	uiRenderer.DrawText(label, tx, ty, scale, mgl32.Vec3{1, 1, 1})
}

func renderMainMenuButtonText(uiRenderer *ui.UI, hudRenderer *hud.HUD, winWidth int, btnY float32) {
	label := "Ana Menü"
	scale := float32(0.5)
	tw, th := hudRenderer.MeasureText(label, scale)
	paddingX := float32(24)
	paddingY := float32(14)
	btnW := tw + paddingX*2
	btnX := (float32(winWidth) - btnW) / 2
	tx := btnX + paddingX
	ty := btnY + paddingY + th
	uiRenderer.DrawText(label, tx, ty, scale, mgl32.Vec3{1, 1, 1})
}
