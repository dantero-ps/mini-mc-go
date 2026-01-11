package main

import (
	"fmt"
	"time"

	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// GameLoop manages the main game loop state
type GameLoop struct {
	window      *glfw.Window
	renderer    *renderer.Renderer
	uiRenderer  *ui.UI
	hudRenderer *hud.HUD
	player      *player.Player
	world       *world.World

	paused     bool
	pauseMenu  *PauseMenu
	fpsLimiter *FPSLimiter

	// Timing
	frames           int
	lastFPSCheckTime time.Time
	lastTime         time.Time
	lastPrune        time.Time
}

// NewGameLoop creates a new game loop with all components
func NewGameLoop(window *glfw.Window, r *renderer.Renderer, uiRenderer *ui.UI, hudRenderer *hud.HUD, p *player.Player, w *world.World) *GameLoop {
	return &GameLoop{
		window:           window,
		renderer:         r,
		uiRenderer:       uiRenderer,
		hudRenderer:      hudRenderer,
		player:           p,
		world:            w,
		pauseMenu:        NewPauseMenu(),
		fpsLimiter:       NewFPSLimiter(),
		lastFPSCheckTime: time.Now(),
		lastTime:         time.Now(),
		lastPrune:        time.Now(),
	}
}

// Paused returns pointer to pause state for input handlers
func (gl *GameLoop) Paused() *bool {
	return &gl.paused
}

// Run starts the main game loop
func (gl *GameLoop) Run() {
	for !gl.window.ShouldClose() {
		gl.tick()
	}
}

func (gl *GameLoop) tick() {
	profiling.ResetFrame()
	now := time.Now()
	dt := now.Sub(gl.lastTime).Seconds()
	gl.lastTime = now

	// Update game state
	updateDur := gl.updateGameState(dt)

	// Measure non-render buckets for HUD breakdown
	playerDur := profiling.SumWithPrefix("player.")
	worldDur := profiling.SumWithPrefix("world.")
	glfwDur := profiling.SumWithPrefix("glfw.")
	physicsDur := profiling.SumWithPrefix("physics.")

	// Process world updates
	gl.processWorldUpdates()

	// Render frame
	renderDur := gl.renderFrame(dt)

	// Handle pause menu
	gl.handlePauseMenu()

	// Present and pump events
	func() { defer profiling.Track("glfw.SwapBuffers")(); gl.window.SwapBuffers() }()
	func() { defer profiling.Track("glfw.PollEvents")(); glfw.PollEvents() }()

	// Update profiling
	gl.updateProfiling(now, updateDur, renderDur, playerDur, worldDur, glfwDur, physicsDur)

	// FPS limiting
	gl.fpsLimiter.Wait(gl.paused)
}

func (gl *GameLoop) updateGameState(dt float64) time.Duration {
	var updateDur time.Duration
	if !gl.paused {
		updateStart := time.Now()
		func() { defer profiling.Track("player.Update")(); gl.player.Update(dt, gl.window) }()
		updateDur = time.Since(updateStart)
	}
	return updateDur
}

func (gl *GameLoop) processWorldUpdates() {
	// Enqueue async streaming when not paused
	if !gl.paused {
		func() {
			defer profiling.Track("world.StreamChunksAroundAsync")()
			gl.world.StreamChunksAroundAsync(gl.player.Position[0], gl.player.Position[2], config.GetChunkLoadRadius())
		}()
	}

	// Process completed mesh results from worker pool
	func() {
		defer profiling.Track("blocks.ProcessMeshResults")()
		blocks.ProcessMeshResults()
	}()

	// Periodically evict far chunks and prune renderer meshes
	if !gl.paused && time.Since(gl.lastPrune) > 750*time.Millisecond {
		func() {
			defer profiling.Track("world.EvictFarChunks")()
			gl.world.EvictFarChunks(gl.player.Position[0], gl.player.Position[2], config.GetChunkEvictRadius())
		}()
		func() {
			defer profiling.Track("renderer.PruneMeshesByWorld")()
			_ = blocks.PruneMeshesByWorld(gl.world, gl.player.Position[0], gl.player.Position[2], config.GetChunkEvictRadius())
		}()
		gl.lastPrune = time.Now()
	}
}

func (gl *GameLoop) renderFrame(dt float64) time.Duration {
	renderStart := time.Now()
	gl.renderer.Render(gl.world, gl.player, dt)
	renderDur := time.Since(renderStart)
	gl.hudRenderer.ProfilingSetRenderDuration(renderDur)
	gl.frames++

	if time.Since(gl.lastFPSCheckTime) >= time.Second {
		fmt.Println("FPS: ", gl.frames)
		gl.frames = 0
		gl.lastFPSCheckTime = time.Now()
	}

	return renderDur
}

func (gl *GameLoop) handlePauseMenu() {
	if gl.paused {
		if gl.pauseMenu.Render(gl.window, gl.uiRenderer, gl.hudRenderer, gl.player) {
			gl.paused = false
		}
	}
}

func (gl *GameLoop) updateProfiling(frameStart time.Time, updateDur, renderDur time.Duration, playerDur, worldDur, glfwDur, physicsDur time.Duration) {
	totalFrameDur := time.Since(frameStart)
	swapEventsDur := profiling.SumWithPrefix("glfw.")

	// Optional: warn if processing exceeds target frame time when limiter is active
	effectiveLimit := config.GetFPSLimit()
	if gl.paused {
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

	preRenderDur := totalFrameDur - swapEventsDur - renderDur
	if preRenderDur < 0 {
		preRenderDur = 0
	}

	gl.hudRenderer.ProfilingSetLastTotalFrameDuration(totalFrameDur)
	gl.hudRenderer.ProfilingSetLastUpdateDuration(updateDur)
	gl.hudRenderer.ProfilingSetBreakdown(playerDur, worldDur, glfwDur, 0, profiling.SumWithPrefix("renderer.PruneMeshesByWorld"))
	gl.hudRenderer.ProfilingSetPhysics(physicsDur)
	gl.hudRenderer.ProfilingSetPhases(preRenderDur, swapEventsDur)
}
