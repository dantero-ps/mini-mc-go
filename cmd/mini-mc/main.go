package main

import (
	"runtime"
	"time"

	"mini-mc/internal/graphics"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"mini-mc/internal/physics"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
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

	// Generate a smaller initial spawn area synchronously to keep startup smooth
	spawnX, spawnZ := float32(0), float32(0)
	gameWorld.StreamChunksAroundSync(spawnX, spawnZ, 2)

	// Compute ground level at spawn
	tempPos := mgl32.Vec3{spawnX, 300, spawnZ}
	groundY := physics.FindGroundLevel(spawnX, spawnZ, tempPos, gameWorld)

	// Initialize player at safe ground
	gamePlayer := player.New(gameWorld)
	gamePlayer.Position[0] = spawnX
	gamePlayer.Position[2] = spawnZ
	gamePlayer.Position[1] = groundY
	gamePlayer.OnGround = true

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
	})
}

func runGameLoop(window *glfw.Window, renderer *graphics.Renderer, p *player.Player, w *world.World) {
	frames := 0
	lastFPSCheckTime := time.Now()
	lastTime := time.Now()

	for !window.ShouldClose() {
		profiling.ResetFrame()
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		func() { defer profiling.Track("player.Update")(); p.Update(dt, window) }()

		// Enqueue async streaming around player every frame (idempotent due to pending dedup)
		func() {
			defer profiling.Track("world.StreamChunksAroundAsync")()
			// Lower initial pressure; far chunks will fill over time
			w.StreamChunksAroundAsync(p.Position[0], p.Position[2], 24)
		}()
		func() { defer profiling.Track("renderer.Render")(); renderer.Render(w, p, dt) }()
		frames++

		if time.Since(lastFPSCheckTime) >= time.Second {
			frames = 0
			lastFPSCheckTime = time.Now()
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
