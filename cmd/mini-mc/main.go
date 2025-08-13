package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"mini-mc/internal/graphics"
	"mini-mc/internal/player"
	"mini-mc/internal/world"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// Window setup
	window, err := setupWindow()
	if err != nil {
		panic(err)
	}

	// Initialize renderer
	renderer, err := graphics.NewRenderer()
	if err != nil {
		panic(err)
	}

	// Create world
	gameWorld := world.New()

	// Initialize player
	gamePlayer := player.New(gameWorld)

	// Setup input handlers
	setupInputHandlers(window, gamePlayer, gameWorld)

	// Main game loop
	runGameLoop(window, renderer, gamePlayer, gameWorld)
}

func setupWindow() (*glfw.Window, error) {
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	window, err := glfw.CreateWindow(graphics.WinWidth, graphics.WinHeight, "mini-mc", nil, nil)
	if err != nil {
		return nil, err
	}
	window.MakeContextCurrent()

	glfw.SwapInterval(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	return window, nil
}

func setupInputHandlers(window *glfw.Window, p *player.Player, w *world.World) {
	// Mouse position callback
	window.SetCursorPosCallback(p.HandleMouseMovement)

	// Mouse button callback
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		p.HandleMouseButton(button, action, mods)
	})

	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if key == glfw.KeyF && action == glfw.Press {
			p.ToggleWireframeMode()
		}
		if key == glfw.KeyC && action == glfw.Press {
			// Toggle chunk boundaries
			graphics.ShowChunkBoundaries = !graphics.ShowChunkBoundaries
			fmt.Printf("Chunk boundaries: %v\n", graphics.ShowChunkBoundaries)
		}
	})
}

func runGameLoop(window *glfw.Window, renderer *graphics.Renderer, p *player.Player, w *world.World) {
	frames := 0
	lastFPSCheckTime := time.Now()
	lastTime := time.Now()

	for !window.ShouldClose() {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		p.Update(dt, window)
		renderer.Render(w, p, dt)
		frames++

		if time.Since(lastFPSCheckTime) >= time.Second {
			window.SetTitle(fmt.Sprintf("mini-mc | FPS: %d",
				frames))

			frames = 0
			lastFPSCheckTime = time.Now()
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
