package main

import (
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/player"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func setupInputHandlers(window *glfw.Window, hudRenderer *hud.HUD, p *player.Player, paused *bool) {
	// Mouse position callback
	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		if !*paused {
			p.HandleMouseMovement(w, xpos, ypos)
		}
	})

	// Mouse button callback (game interactions disabled when paused)
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		if !*paused {
			p.HandleMouseButton(button, action, mods)
		}
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key == glfw.KeyF && action == glfw.Press {
			p.ToggleWireframeMode()
		}
		if key == glfw.KeyV && action == glfw.Press {
			hudRenderer.ToggleProfiling()
		}
		if key == glfw.KeyEscape && action == glfw.Press {
			*paused = !*paused
			if *paused {
				w.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			} else {
				w.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				p.FirstMouse = true
			}
		}
	})
}
