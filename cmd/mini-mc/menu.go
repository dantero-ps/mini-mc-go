package main

import (
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/player"

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
	// Put text into the same FIFO UI queue (correct z-order, single Flush).
	uiRenderer.SetFontRenderer(menuHUD.FontRenderer())
	defer uiRenderer.Dispose()

	var selectedMode player.GameMode
	confirmed := false
	wasMouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
	fpsLimiter := NewFPSLimiter()

	renderUI := func(w *glfw.Window, mx, my float64, isClick bool) {
		winW, winH := w.GetSize()
		fWinW, fWinH := float32(winW), float32(winH)

		// Hem UI hem Font projeksiyonlarını eşitle
		uiRenderer.SetViewport(winW, winH)
		menuHUD.SetViewport(winW, winH)

		scaleX := fWinW / 900.0
		scaleY := fWinH / 600.0
		scale := scaleX
		if scaleY < scale {
			scale = scaleY
		}

		centerX := fWinW / 2
		centerY := fWinH / 2

		btnW := 400.0 * scale
		btnH := 80.0 * scale
		btnX := centerX - (btnW / 2)

		// Yardımcı: Metni yatayda ortalayarak çizer
		drawTextCentered := func(text string, y float32, s float32, col mgl32.Vec3) {
			tw, _ := menuHUD.FontRenderer().Measure(text, s)
			uiRenderer.DrawText(text, centerX-(tw/2), y, s, col)
		}

		// 1. Başlıklar
		drawTextCentered("MINI MC", centerY-(200*scale), 1.0*scale, mgl32.Vec3{1, 1, 1})
		drawTextCentered("Select Game Mode:", centerY-(140*scale), 0.5*scale, mgl32.Vec3{0.8, 0.8, 0.8})

		// 2. Survival Butonu
		sBtnY := centerY - (40 * scale)
		btn1Hovered := mx >= float64(btnX) && mx <= float64(btnX+btnW) && my >= float64(sBtnY) && my <= float64(sBtnY+btnH)

		b1Col := mgl32.Vec3{0.3, 0.3, 0.3}
		if btn1Hovered {
			b1Col = mgl32.Vec3{0.4, 0.4, 0.4}
			if isClick {
				selectedMode = player.GameModeSurvival
				confirmed = true
			}
		}
		uiRenderer.DrawFilledRect(btnX, sBtnY, btnW, btnH, b1Col, 1.0)
		uiRenderer.DrawText("Survival", btnX+(20*scale), sBtnY+(30*scale), 0.6*scale, mgl32.Vec3{0, 1, 0})
		uiRenderer.DrawText("No Flying, Normal Mining", btnX+(20*scale), sBtnY+(60*scale), 0.35*scale, mgl32.Vec3{0.8, 0.8, 0.8})

		// 3. Creative Butonu
		cBtnY := centerY + (60 * scale)
		btn2Hovered := mx >= float64(btnX) && mx <= float64(btnX+btnW) && my >= float64(cBtnY) && my <= float64(cBtnY+btnH)

		b2Col := mgl32.Vec3{0.3, 0.3, 0.3}
		if btn2Hovered {
			b2Col = mgl32.Vec3{0.4, 0.4, 0.4}
			if isClick {
				selectedMode = player.GameModeCreative
				confirmed = true
			}
		}
		uiRenderer.DrawFilledRect(btnX, cBtnY, btnW, btnH, b2Col, 1.0)
		uiRenderer.DrawText("Creative", btnX+(20*scale), cBtnY+(30*scale), 0.6*scale, mgl32.Vec3{0, 0.8, 1})
		uiRenderer.DrawText("Flying, Instant Break", btnX+(20*scale), cBtnY+(60*scale), 0.35*scale, mgl32.Vec3{0.8, 0.8, 0.8})

		uiRenderer.Flush()
	}

	// Main Loop ve Callback
	window.SetRefreshCallback(func(w *glfw.Window) {
		gl.ClearColor(0.1, 0.1, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		mx, my := w.GetCursorPos()
		renderUI(w, mx, my, false)
		w.SwapBuffers()
	})

	for !window.ShouldClose() && !confirmed {
		gl.ClearColor(0.1, 0.1, 0.1, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		mx, my := window.GetCursorPos()
		mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
		isClick := mouseDown && !wasMouseDown

		renderUI(window, mx, my, isClick)

		window.SwapBuffers()
		glfw.PollEvents()
		fpsLimiter.Wait(true)
		wasMouseDown = mouseDown
	}

	window.SetRefreshCallback(nil)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	return selectedMode
}
