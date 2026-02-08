package game

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func SetupInputHandlers(app *App) {
	window := app.window
	im := app.inputManager

	// Mouse position callback
	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		if app.session != nil && !app.session.Paused {
			s := app.session
			s.Player.MouseX = xpos
			s.Player.MouseY = ypos
			if !s.Player.IsInventoryOpen {
				s.Player.HandleMouseMovement(w, xpos, ypos)
			}
		}
	})

	// Mouse button callback
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		// Update InputManager state first (globally tracking inputs)
		im.HandleMouseButtonEvent(button, action)

		if app.session != nil && !app.session.Paused {
			s := app.session
			if s.Player.IsInventoryOpen {
				s.HUDRenderer.HandleInventoryClick(s.Player.MouseX, s.Player.MouseY, button, action)
			} else {
				s.Player.HandleMouseButton(button, action)
			}
		}
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		if app.session != nil && !app.session.Paused {
			s := app.session
			if !s.Player.IsInventoryOpen {
				s.Player.HandleScroll(yoff)
			}
		}
	})

	// Handle keyboard actions
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		im.HandleKeyEvent(key, action)
	})

	// Framebuffer size callback
	window.SetFramebufferSizeCallback(func(w *glfw.Window, fbWidth, fbHeight int) {
		gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))

		// Update App level viewports (Menu)
		// UI logic uses Window (Logical) coordinates for layout, so we must pass Window size to SetViewport
		winW, winH := w.GetSize()
		app.menuUI.SetViewport(winW, winH)
		app.fontRenderer.SetViewport(float32(winW), float32(winH))

		if app.session != nil {
			app.session.Renderer.UpdateViewport(winW, winH)
			app.session.UIRenderer.SetViewport(winW, winH)
		}
		// NOTE: Do not render here. Rely on SetRefreshCallback for smooth resizing on macOS.
	})

	// Window size callback
	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		if app.session != nil {
			app.session.Renderer.UpdateViewport(width, height)
		}
	})

	// Focus callback
	window.SetFocusCallback(func(w *glfw.Window, focused bool) {
		if !focused && app.session != nil && !app.session.Paused {
			s := app.session
			if s.Player.IsInventoryOpen {
				s.Player.SetInventoryOpen(false)
				s.Player.DropCursorItem()
			}
			s.SetPaused(true)
		}
	})

	// Refresh callback
	window.SetRefreshCallback(func(w *glfw.Window) {
		app.RefreshRender()
	})
}
