package main

import (
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/player"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

func showMenu(window *glfw.Window) player.GameMode {
	// Temporarily unlock cursor for menu
	window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)

	// Create a temporary HUD for text rendering
	menuHUD := hud.NewHUD()
	if err := menuHUD.Init(); err != nil {
		panic(err)
	}
	defer menuHUD.Dispose()

	// Create UI renderer for buttons
	uiRenderer := ui.NewUI()
	if err := uiRenderer.Init(); err != nil {
		panic(err)
	}
	defer uiRenderer.Dispose()

	var selectedMode player.GameMode
	confirmed := false

	// Helper to check button hover/click
	isHovered := func(mx, my, x, y, w, h float64) bool {
		return mx >= x && mx <= x+w && my >= y && my <= y+h
	}

	// Track previous mouse state to prevent accidental clicks from previous scene
	wasMouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press

	for !window.ShouldClose() && !confirmed {
		// Clear screen
		gl.ClearColor(0.1, 0.1, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		mx, my := window.GetCursorPos()
		mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press

		// Only register click if mouse was NOT down in previous frame and IS down now (fresh click)
		// Also wait until initial mouse down is released if it started pressed
		isClick := mouseDown && !wasMouseDown

		// Survival Button
		btnW, btnH := 400.0, 80.0
		btn1X, btn1Y := (900.0-btnW)/2, 220.0

		btn1Color := mgl32.Vec3{0.3, 0.3, 0.3}
		if isHovered(mx, my, btn1X, btn1Y, btnW, btnH) {
			btn1Color = mgl32.Vec3{0.4, 0.4, 0.4}
			if isClick {
				selectedMode = player.GameModeSurvival
				confirmed = true
			}
		}
		uiRenderer.DrawFilledRect(float32(btn1X), float32(btn1Y), float32(btnW), float32(btnH), btn1Color, 1.0)

		// Creative Button
		btn2X, btn2Y := (900.0-btnW)/2, 320.0

		btn2Color := mgl32.Vec3{0.3, 0.3, 0.3}
		if isHovered(mx, my, btn2X, btn2Y, btnW, btnH) {
			btn2Color = mgl32.Vec3{0.4, 0.4, 0.4}
			if isClick {
				selectedMode = player.GameModeCreative
				confirmed = true
			}
		}
		uiRenderer.DrawFilledRect(float32(btn2X), float32(btn2Y), float32(btnW), float32(btnH), btn2Color, 1.0)

		// 1) Flush UI geometry first so it stays behind text.
		uiRenderer.Flush()

		// 2) Draw text on top (font renderer is immediate-mode).
		menuHUD.RenderText("MINI MC", 350, 100, 1.0, mgl32.Vec3{1, 1, 1})
		menuHUD.RenderText("Select Game Mode:", 340, 160, 0.5, mgl32.Vec3{0.8, 0.8, 0.8})
		menuHUD.RenderText("Survival", float32(btn1X)+20, float32(btn1Y)+37, 0.6, mgl32.Vec3{0, 1, 0})
		menuHUD.RenderText("No Flying, Normal Mining", float32(btn1X)+20, float32(btn1Y)+65, 0.35, mgl32.Vec3{0.8, 0.8, 0.8})
		menuHUD.RenderText("Creative", float32(btn2X)+20, float32(btn2Y)+37, 0.6, mgl32.Vec3{0, 0.8, 1})
		menuHUD.RenderText("Flying, Instant Break", float32(btn2X)+20, float32(btn2Y)+65, 0.35, mgl32.Vec3{0.8, 0.8, 0.8})

		window.SwapBuffers()
		glfw.PollEvents()

		// Update mouse state
		wasMouseDown = mouseDown

		// Limit CPU usage in menu
		runtime.Gosched()
	}

	// Re-lock cursor before returning to game
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	return selectedMode
}
