package main

import (
	"fmt"
	"mini-mc/internal/graphics/renderables/crosshair"
	"mini-mc/internal/graphics/renderables/direction"
	"mini-mc/internal/graphics/renderables/hand"
	"mini-mc/internal/graphics/renderables/wireframe"
	"runtime"
	"time"

	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	renderer "mini-mc/internal/graphics/renderer"
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

	// Initialize renderable features
	blocksRenderer := blocks.NewBlocks()
	wireframeRenderer := wireframe.NewWireframe()
	crosshairRenderer := crosshair.NewCrosshair()
	directionRenderer := direction.NewDirection()
	handRenderer := hand.NewHand()
	uiRenderer := ui.NewUI()
	hudRenderer := hud.NewHUD()

	// Initialize renderer with all features
	r, err := renderer.NewRenderer(
		blocksRenderer,
		wireframeRenderer,
		crosshairRenderer,
		directionRenderer,
		handRenderer,
		uiRenderer,
		hudRenderer,
	)
	if err != nil {
		panic(err)
	}

	// Create world
	gameWorld := world.New()

	// Initialize mesh worker pool system (4 workers for mesh generation)
	blocks.InitMeshSystem(4)
	defer blocks.ShutdownMeshSystem()

	// Generate a smaller initial spawn area synchronously to keep startup smooth
	spawnX, spawnZ := float32(0), float32(0)
	gameWorld.StreamChunksAroundSync(spawnX, spawnZ, 50)

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
	setupInputHandlers(window, r, hudRenderer, gamePlayer, gameWorld, &paused)

	// Main game loop
	runGameLoop(window, r, uiRenderer, hudRenderer, gamePlayer, gameWorld, &paused)
}

func setupWindow() (*glfw.Window, error) {
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	window, err := glfw.CreateWindow(900, 600, "mini-mc", nil, nil)
	if err != nil {
		return nil, err
	}
	window.MakeContextCurrent()

	// Disable V-Sync; we'll use our own FPS limiter
	glfw.SwapInterval(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	return window, nil
}

func setupInputHandlers(window *glfw.Window, r *renderer.Renderer, hudRenderer *hud.HUD, p *player.Player, gameWorld *world.World, paused *bool) {
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

func runGameLoop(window *glfw.Window, r *renderer.Renderer, uiRenderer *ui.UI, hudRenderer *hud.HUD, p *player.Player, w *world.World, paused *bool) {
	frames := 0
	lastFPSCheckTime := time.Now()
	lastTime := time.Now()
	lastPrune := time.Now()
	mouseWasDown := false
	var btnX, btnY, btnW, btnH float32
	var fpsLimiterNext time.Time

	// Render distance slider state
	renderDistanceSlider := float32(config.GetRenderDistance()-5) / float32(50-5) // Normalize to 0-1
	// FPS limiter slider state: left=minCap (finite), right=uncapped
	const minFPSCap = 30
	const maxFPSCap = 240
	fpsLimit := config.GetFPSLimit()
	var fpsLimitSlider float32
	if fpsLimit <= 0 {
		fpsLimitSlider = 1.0
	} else {
		if fpsLimit < minFPSCap {
			fpsLimit = minFPSCap
		}
		if fpsLimit > maxFPSCap {
			fpsLimit = maxFPSCap
		}
		fpsLimitSlider = float32(fpsLimit-minFPSCap) / float32(maxFPSCap-minFPSCap)
	}

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
				w.StreamChunksAroundAsync(p.Position[0], p.Position[2], config.GetChunkLoadRadius())
			}()
		}

		// Process completed mesh results from worker pool
		func() {
			defer profiling.Track("blocks.ProcessMeshResults")()
			blocks.ProcessMeshResults()
		}()

		// Periodically evict far chunks and prune renderer meshes
		if !*paused && time.Since(lastPrune) > 750*time.Millisecond {
			func() {
				defer profiling.Track("world.EvictFarChunks")()
				w.EvictFarChunks(p.Position[0], p.Position[2], config.GetChunkEvictRadius())
			}()
			func() {
				defer profiling.Track("renderer.PruneMeshesByWorld")()
				_ = blocks.PruneMeshesByWorld(w, p.Position[0], p.Position[2], config.GetChunkEvictRadius())
			}()
			lastPrune = time.Now()
		}

		// Measure render duration precisely
		renderStart := time.Now()
		r.Render(w, p, dt)
		renderDur := time.Since(renderStart)
		hudRenderer.ProfilingSetRenderDuration(renderDur)
		frames++

		if time.Since(lastFPSCheckTime) >= time.Second {
			fmt.Println("FPS: ", frames)
			frames = 0
			lastFPSCheckTime = time.Now()
		}

		// Pause overlay and Resume button
		if *paused {
			// Dim background
			uiRenderer.DrawFilledRect(0, 0, 900, 600, mgl32.Vec3{0, 0, 0}, 0.45)

			// Render Distance Slider
			sliderLabel := "Render Distance"
			sliderScale := float32(0.8)
			_, sliderTH := hudRenderer.MeasureText(sliderLabel, sliderScale)
			sliderX := float32(350)
			sliderY := float32(200)
			sliderW := float32(200)
			sliderH := float32(20)

			// Draw slider label
			hudRenderer.RenderText(sliderLabel, sliderX, sliderY-10, sliderScale, mgl32.Vec3{1, 1, 1})

			// Draw and handle slider (snap to steps equal to render distance range)
			steps := 50 - 5 + 1 // inclusive range [5..50]
			newSliderValue := uiRenderer.DrawSlider(sliderX, sliderY, sliderW, sliderH, renderDistanceSlider, window, steps, "renderDistance")
			if newSliderValue != renderDistanceSlider {
				renderDistanceSlider = newSliderValue
				// Convert slider value to render distance (5-50 range)
				newDistance := int(float32(5) + renderDistanceSlider*float32(50-5) + 0.5)
				config.SetRenderDistance(newDistance)
			}

			// Show current render distance value
			currentDistance := config.GetRenderDistance()
			distanceText := fmt.Sprintf("%d chunks", currentDistance)
			hudRenderer.RenderText(distanceText, sliderX+sliderW+10, sliderY+sliderH/2+sliderTH/2, sliderScale, mgl32.Vec3{0.8, 0.8, 0.8})

			// Read mouse once for UI interactions in pause menu
			cx, cy := window.GetCursorPos()
			mouseDown := window.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press

			// FPS Limit Slider
			fpsLabel := "FPS Limit"
			fpsScale := float32(0.9)
			_, fpsTH := hudRenderer.MeasureText(fpsLabel, fpsScale)
			fpsX := sliderX
			fpsY := sliderY + 50
			fpsW := sliderW
			fpsH := sliderH
			hudRenderer.RenderText(fpsLabel, fpsX, fpsY-10, fpsScale, mgl32.Vec3{1, 1, 1})
			// 30..240 inclusive -> snapping, rightmost => uncapped
			newFPSValue := uiRenderer.DrawSlider(fpsX, fpsY, fpsW, fpsH, fpsLimitSlider, window, (maxFPSCap-minFPSCap)+1, "fpsLimit")
			if newFPSValue != fpsLimitSlider {
				fpsLimitSlider = newFPSValue
				// Map slider: rightmost => uncapped (0), else 30..240
				var newLimit int
				if fpsLimitSlider >= 0.999 {
					newLimit = 0
				} else {
					newLimit = int(float32(minFPSCap) + fpsLimitSlider*float32(maxFPSCap-minFPSCap) + 0.5)
				}
				config.SetFPSLimit(newLimit)
			}
			// Show current FPS limit value
			currentFPSLimit := config.GetFPSLimit()
			var limitText string
			if currentFPSLimit <= 0 {
				limitText = "Uncapped"
			} else {
				limitText = fmt.Sprintf("%d FPS", currentFPSLimit)
			}
			hudRenderer.RenderText(limitText, fpsX+fpsW+10, fpsY+fpsH/2+fpsTH/2, fpsScale, mgl32.Vec3{0.8, 0.8, 0.8})

			// Resume button
			label := "Devam Et"
			scale := float32(1.0)
			tw, th := hudRenderer.MeasureText(label, scale)
			paddingX := float32(24)
			paddingY := float32(14)
			btnW = tw + paddingX*2
			btnH = th + paddingY*2
			btnX = (900 - btnW) / 2
			btnY = float32(300) // Move button lower to make space for slider

			hover := float32(cx) >= btnX && float32(cx) <= btnX+btnW && float32(cy) >= btnY && float32(cy) <= btnY+btnH
			base := mgl32.Vec3{0.20, 0.20, 0.20}
			if hover {
				base = mgl32.Vec3{0.30, 0.30, 0.30}
			}
			uiRenderer.DrawFilledRect(btnX, btnY, btnW, btnH, base, 0.85)
			// Text
			tx := btnX + paddingX
			ty := btnY + paddingY + th
			hudRenderer.RenderText(label, tx, ty, scale, mgl32.Vec3{1, 1, 1})

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

		// Optional: warn if processing exceeds target frame time when limiter is active
		{
			effectiveLimit := config.GetFPSLimit()
			if *paused {
				effectiveLimit = 120
			}
			if effectiveLimit > 0 {
				processingDur := renderDur + updateDur
				targetFrameTime := time.Second / time.Duration(effectiveLimit)
				if processingDur > targetFrameTime {
					fmt.Printf("Frame processing too slow: %.2fms (target: %.2fms)\n",
						float64(processingDur.Nanoseconds())/1000000.0,
						float64(targetFrameTime.Nanoseconds())/1000000.0)
				}
			}
		}
		preRenderDur := totalFrameDur - swapEventsDur - renderDur
		if preRenderDur < 0 {
			preRenderDur = 0
		}
		hudRenderer.ProfilingSetLastTotalFrameDuration(totalFrameDur)
		hudRenderer.ProfilingSetLastUpdateDuration(updateDur)
		// Update breakdown fields (main thread only)
		hudRenderer.ProfilingSetBreakdown(playerDur, worldDur, glfwDur, 0, profiling.SumWithPrefix("renderer.PruneMeshesByWorld"))
		hudRenderer.ProfilingSetPhysics(physicsDur)
		hudRenderer.ProfilingSetPhases(preRenderDur, swapEventsDur)
		// Console timing after swap

		// High-precision FPS limiter (after swap+events): schedule next frame deadline,
		// sleep most of the remaining time, then spin for the last ~200µs to improve accuracy.
		{
			effectiveLimit := config.GetFPSLimit()
			if *paused {
				effectiveLimit = 120
			}
			if effectiveLimit > 0 {
				target := time.Second / time.Duration(effectiveLimit)
				if fpsLimiterNext.IsZero() {
					fpsLimiterNext = time.Now().Add(target)
				} else {
					fpsLimiterNext = fpsLimiterNext.Add(target)
				}
				for {
					remaining := time.Until(fpsLimiterNext)
					if remaining <= 0 {
						break
					}
					if remaining > 200*time.Microsecond {
						time.Sleep(remaining - 200*time.Microsecond)
					} else {
						// busy-wait for the final few microseconds
						// yields substantially better precision on high FPS caps
					}
					if time.Until(fpsLimiterNext) <= 0 {
						break
					}
				}
				// If we're significantly late (e.g., hitch), resync to avoid drift
				if late := -time.Until(fpsLimiterNext); late > target {
					fpsLimiterNext = time.Now().Add(target)
				}
			} else {
				fpsLimiterNext = time.Time{}
			}
		}
	}
}
