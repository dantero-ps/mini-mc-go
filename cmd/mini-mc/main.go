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

	// Pause state
	paused := false

	// Setup input handlers
	setupInputHandlers(window, gamePlayer, gameWorld, &paused)

	// Main game loop
	runGameLoop(window, renderer, gamePlayer, gameWorld, &paused)
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

func setupInputHandlers(window *glfw.Window, p *player.Player, gameWorld *world.World, paused *bool) {
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

func runGameLoop(window *glfw.Window, renderer *graphics.Renderer, p *player.Player, w *world.World, paused *bool) {
	frames := 0
	lastFPSCheckTime := time.Now()
	lastTime := time.Now()
	lastPrune := time.Now()
	mouseWasDown := false
	var btnX, btnY, btnW, btnH float32

	for !window.ShouldClose() {
		profiling.ResetFrame()
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		var updateDur time.Duration
		if !*paused {
			updateStart := time.Now()
			func() { defer profiling.Track("player.Update")(); p.Update(dt, window) }()
			updateDur = time.Since(updateStart)
		}

		// Measure non-render buckets for HUD breakdown
		playerDur := profiling.SumWithPrefix("player.")
		worldDur := profiling.SumWithPrefix("world.")
		glfwDur := profiling.SumWithPrefix("glfw.")
		physicsDur := profiling.SumWithPrefix("physics.")

		// Enqueue async streaming when not paused
		if !*paused {
			func() {
				defer profiling.Track("world.StreamChunksAroundAsync")()
				w.StreamChunksAroundAsync(p.Position[0], p.Position[2], 24)
			}()
		}

		// Periodically evict far chunks and prune renderer meshes
		if !*paused && time.Since(lastPrune) > 750*time.Millisecond {
			func() {
				defer profiling.Track("world.EvictFarChunks")()
				w.EvictFarChunks(p.Position[0], p.Position[2], 40)
			}()
			func() {
				defer profiling.Track("renderer.PruneMeshesByWorld")()
				renderer.PruneMeshesByWorld(w, p.Position, 40)
			}()
			lastPrune = time.Now()
		}

		renderer.Render(w, p, dt)
		frames++

		if time.Since(lastFPSCheckTime) >= time.Second {
			frames = 0
			lastFPSCheckTime = time.Now()
		}

		renderer.RenderProfilingInfo()

		// Pause overlay and Resume button
		if *paused {
			// Dim background
			renderer.DrawFilledRect(0, 0, float32(graphics.WinWidth), float32(graphics.WinHeight), mgl32.Vec3{0, 0, 0}, 0.45)
			label := "Devam Et"
			scale := float32(1.0)
			tw, th := renderer.MeasureText(label, scale)
			paddingX := float32(24)
			paddingY := float32(14)
			btnW = tw + paddingX*2
			btnH = th + paddingY*2
			btnX = (float32(graphics.WinWidth) - btnW) / 2
			btnY = (float32(graphics.WinHeight) - btnH) / 2

			cx, cy := window.GetCursorPos()
			hover := float32(cx) >= btnX && float32(cx) <= btnX+btnW && float32(cy) >= btnY && float32(cy) <= btnY+btnH
			base := mgl32.Vec3{0.20, 0.20, 0.20}
			if hover {
				base = mgl32.Vec3{0.30, 0.30, 0.30}
			}
			renderer.DrawFilledRect(btnX, btnY, btnW, btnH, base, 0.85)
			// Text
			tx := btnX + paddingX
			ty := btnY + paddingY + th
			renderer.RenderText(label, tx, ty, scale, mgl32.Vec3{1, 1, 1})

			mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press
			if mouseDown && !mouseWasDown && hover {
				*paused = false
				window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				p.FirstMouse = true
			}
			mouseWasDown = mouseDown
		}

		// Present and pump events
		func() { defer profiling.Track("glfw.SwapBuffers")(); window.SwapBuffers() }()
		func() { defer profiling.Track("glfw.PollEvents")(); glfw.PollEvents() }()

		// After presenting, record durations for next frame's HUD
		totalFrameDur := time.Since(now)
		swapEventsDur := profiling.SumWithPrefix("glfw.")
		preRenderDur := totalFrameDur - swapEventsDur - time.Duration(float64(renderer.GetProfilingStats()["frameDuration"].(time.Duration)))
		if preRenderDur < 0 {
			preRenderDur = 0
		}
		renderer.ProfilingSetLastTotalFrameDuration(totalFrameDur)
		renderer.ProfilingSetLastUpdateDuration(updateDur)
		// Update breakdown fields (main thread only)
		renderer.ProfilingSetBreakdown(playerDur, worldDur, glfwDur, 0, profiling.SumWithPrefix("renderer.PruneMeshesByWorld"))
		renderer.ProfilingSetPhysics(physicsDur)
		// store physics separately
		stats := renderer.GetProfilingStats()
		_ = stats
		renderer.ProfilingSetPhases(preRenderDur, swapEventsDur)
		// Console timing after swap
	}
}
