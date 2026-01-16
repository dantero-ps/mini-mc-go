package main

import (
	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func setupInputHandlers(window *glfw.Window, gameLoop *GameLoop, r *renderer.Renderer, hudRenderer *hud.HUD, p *player.Player, paused *bool) {
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

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key >= glfw.Key1 && key <= glfw.Key9 && action == glfw.Press {
			if p.IsInventoryOpen {
				// In inventory: move hovered item to hotbar slot
				hotbarSlot := int(key - glfw.Key1)
				hudRenderer.MoveHoveredItemToHotbar(p, hotbarSlot)
			} else {
				p.HandleNumKey(int(key - glfw.Key1))
			}
		}
		if key == glfw.KeyF && action == glfw.Press {
			config.ToggleWireframeMode()
		}
		if key == glfw.KeyV && action == glfw.Press {
			hudRenderer.ToggleProfiling()
		}

		// Drop Item
		if key == glfw.KeyQ && action == glfw.Press {
			if !*paused && !p.IsInventoryOpen {
				// Check for Ctrl key to drop entire stack
				dropStack := (mods & glfw.ModControl) != 0
				p.DropHeldItem(dropStack)
			}
		}

		// Inventory Toggle
		if key == glfw.KeyE && action == glfw.Press {
			if !*paused {
				p.IsInventoryOpen = !p.IsInventoryOpen
				if p.IsInventoryOpen {
					w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
					width, height := w.GetSize()
					w.SetCursorPos(float64(width)/2, float64(height)/2)
				} else {
					p.DropCursorItem()
					w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
					p.FirstMouse = true
				}
			}
		}

		if key == glfw.KeyEscape && action == glfw.Press {
			if p.IsInventoryOpen {
				// Close inventory if open
				p.IsInventoryOpen = false
				p.DropCursorItem()
				w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				p.FirstMouse = true
			} else {
				// Toggle pause
				*paused = !*paused
				if *paused {
					w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
				} else {
					w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
					p.FirstMouse = true
				}
			}
		}
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
