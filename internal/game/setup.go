package game

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func SetupWindow() (*glfw.Window, error) {
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	window, err := glfw.CreateWindow(900, 600, "mini-mc", nil, nil)
	if err != nil {
		return nil, err
	}
	window.MakeContextCurrent()

	// Initialize OpenGL bindings
	if err := gl.Init(); err != nil {
		return nil, err
	}

	// Disable V-Sync; we'll use our own FPS limiter
	glfw.SwapInterval(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorNormal) // Start with normal cursor for Menu

	return window, nil
}
