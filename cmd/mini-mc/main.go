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

	glfw.SwapInterval(1)
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
	})
}

func runGameLoop(window *glfw.Window, renderer *graphics.Renderer, p *player.Player, w *world.World) {
	frames := 0
	ticks := 0
	lastFPSCheckTime := time.Now()
	lastTime := time.Now()

	// Performance measurement variables
	var totalTickTime time.Duration
	var totalRenderTime time.Duration
	var tickSamples int
	var renderSamples int

	for !window.ShouldClose() {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		// Measure tick time
		tickStart := time.Now()
		// Update game state (tick)
		p.Update(dt, window)
		tickDuration := time.Since(tickStart)
		totalTickTime += tickDuration
		tickSamples++
		ticks++

		// Measure render time
		renderStart := time.Now()
		// Render frame
		renderer.Render(w, p)
		renderDuration := time.Since(renderStart)
		totalRenderTime += renderDuration
		renderSamples++
		frames++

		// FPS, TPS, and performance counter
		if time.Since(lastFPSCheckTime) >= time.Second {
			avgTickTime := totalTickTime / time.Duration(tickSamples)
			avgRenderTime := totalRenderTime / time.Duration(renderSamples)

			window.SetTitle(fmt.Sprintf("mini-mc | FPS: %d | TPS: %d | Tick: %s | Render: %s",
				frames, ticks, avgTickTime, avgRenderTime))

			// Log which takes longer
			if avgTickTime > avgRenderTime {
				fmt.Printf("Tick takes longer: Tick=%s, Render=%s (Difference: %s)\n",
					avgTickTime, avgRenderTime, avgTickTime-avgRenderTime)
			} else {
				fmt.Printf("Render takes longer: Render=%s, Tick=%s (Difference: %s)\n",
					avgRenderTime, avgTickTime, avgRenderTime-avgTickTime)
			}

			// Reset counters
			frames = 0
			ticks = 0
			totalTickTime = 0
			totalRenderTime = 0
			tickSamples = 0
			renderSamples = 0
			lastFPSCheckTime = time.Now()
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
