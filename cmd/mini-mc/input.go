package main

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/input"
	"mini-mc/internal/player"
)

func setupInputHandlers(window *glfw.Window, gameLoop *GameLoop, r *renderer.Renderer, hudRenderer *hud.HUD, p *player.Player, paused *bool, im *input.InputManager) {
	// Mouse position callback
	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		p.MouseX = xpos
		p.MouseY = ypos
		if !*paused && !p.IsInventoryOpen {
			p.HandleMouseMovement(w, xpos, ypos)
		}
	})

	// Mouse button callback (game interactions disabled when paused or inventory open)
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		// Update InputManager state first
		im.HandleMouseButtonEvent(button, action)

		if !*paused {
			if p.IsInventoryOpen {
				hudRenderer.HandleInventoryClick(p, p.MouseX, p.MouseY, button, action)
			} else {
				p.HandleMouseButton(button, action, mods)
			}
		}
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		if !*paused && !p.IsInventoryOpen {
			p.HandleScroll(w, xoff, yoff)
		}
	})

	// Handle keyboard actions with InputManager integration
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Update InputManager state first
		im.HandleKeyEvent(key, action)
	})

	// Framebuffer size callback
	window.SetFramebufferSizeCallback(func(w *glfw.Window, fbWidth, fbHeight int) {
		gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))
		winW, winH := w.GetSize()
		r.UpdateViewport(winW, winH)
		hudRenderer.SetViewport(winW, winH)
	})

	// Window size callback
	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		r.UpdateViewport(width, height)
		hudRenderer.SetViewport(width, height)
	})

	// Refresh callback (called during window resize to prevent visual glitches)
	window.SetRefreshCallback(func(w *glfw.Window) {
		gameLoop.RefreshRender()
	})
}
