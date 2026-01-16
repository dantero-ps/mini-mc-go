package main

import (
	"fmt"
	"time"

	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/ui"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/input"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// GameLoop manages the main game loop state
type GameLoop struct {
	window       *glfw.Window
	renderer     *renderer.Renderer
	uiRenderer   *ui.UI
	hudRenderer  *hud.HUD
	player       *player.Player
	world        *world.World
	inputManager *input.InputManager

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
func NewGameLoop(window *glfw.Window, r *renderer.Renderer, uiRenderer *ui.UI, hudRenderer *hud.HUD, p *player.Player, w *world.World, im *input.InputManager) *GameLoop {
	return &GameLoop{
		window:           window,
		renderer:         r,
		uiRenderer:       uiRenderer,
		hudRenderer:      hudRenderer,
		player:           p,
		world:            w,
		inputManager:     im,
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
// Returns true if the game should return to the main menu
func (gl *GameLoop) Run() bool {
	for !gl.window.ShouldClose() {
		if action := gl.tick(); action == MenuActionQuitToMenu {
			return true
		}
	}
	return false
}

func (gl *GameLoop) tick() MenuAction {
	profiling.ResetFrame()
	now := time.Now()
	dt := now.Sub(gl.lastTime).Seconds()
	gl.lastTime = now

	// Poll events at start
	func() { defer profiling.Track("glfw.PollEvents")(); glfw.PollEvents() }()

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
	if action := gl.handlePauseMenu(); action != MenuActionNone {
		return action
	}

	// Present and pump events
	func() { defer profiling.Track("glfw.SwapBuffers")(); gl.window.SwapBuffers() }()

	// Clear edge flags at end of frame
	gl.inputManager.PostUpdate()

	// Update profiling
	gl.updateProfiling(now, updateDur, renderDur, playerDur, worldDur, glfwDur, physicsDur)

	// FPS limiting
	gl.fpsLimiter.Wait(gl.paused)

	return MenuActionNone
}

func (gl *GameLoop) updateGameState(dt float64) time.Duration {
	var updateDur time.Duration
	if !gl.paused {
		updateStart := time.Now()
		func() { defer profiling.Track("player.Update")(); gl.player.Update(dt, gl.inputManager) }()
		func() { defer profiling.Track("world.UpdateEntities")(); gl.world.UpdateEntities(dt) }()
		updateDur = time.Since(updateStart)
	}

	gl.handleInputActions()

	return updateDur
}

func (gl *GameLoop) handleInputActions() {
	// Hotbar selection
	if gl.inputManager.JustPressed(input.ActionHotbar1) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 0)
		} else {
			gl.player.HandleNumKey(0)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar2) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 1)
		} else {
			gl.player.HandleNumKey(1)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar3) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 2)
		} else {
			gl.player.HandleNumKey(2)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar4) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 3)
		} else {
			gl.player.HandleNumKey(3)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar5) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 4)
		} else {
			gl.player.HandleNumKey(4)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar6) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 5)
		} else {
			gl.player.HandleNumKey(5)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar7) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 6)
		} else {
			gl.player.HandleNumKey(6)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar8) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 7)
		} else {
			gl.player.HandleNumKey(7)
		}
	}
	if gl.inputManager.JustPressed(input.ActionHotbar9) {
		if gl.player.IsInventoryOpen {
			gl.hudRenderer.MoveHoveredItemToHotbar(gl.player, 8)
		} else {
			gl.player.HandleNumKey(8)
		}
	}

	// Drop Item (with Ctrl modifier check)
	if gl.inputManager.JustPressed(input.ActionDropItem) {
		if !gl.paused && !gl.player.IsInventoryOpen {
			dropStack := gl.inputManager.IsActive(input.ActionModControl)
			gl.player.DropHeldItem(dropStack)
		}
	}

	// Inventory Toggle
	if gl.inputManager.JustPressed(input.ActionInventory) {
		if !gl.paused {
			gl.player.IsInventoryOpen = !gl.player.IsInventoryOpen
			if gl.player.IsInventoryOpen {
				gl.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
				width, height := gl.window.GetSize()
				gl.window.SetCursorPos(float64(width)/2, float64(height)/2)
			} else {
				gl.player.DropCursorItem()
				gl.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				gl.player.FirstMouse = true
			}
		}
	}

	// Pause Toggle
	if gl.inputManager.JustPressed(input.ActionPause) {
		if gl.player.IsInventoryOpen {
			gl.player.IsInventoryOpen = false
			gl.player.DropCursorItem()
			gl.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			gl.player.FirstMouse = true
		} else {
			gl.paused = !gl.paused
			if gl.paused {
				gl.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
			} else {
				gl.window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				gl.player.FirstMouse = true
			}
		}
	}

	// Toggle Wireframe
	if gl.inputManager.JustPressed(input.ActionToggleWireframe) {
		config.ToggleWireframeMode()
	}

	// Toggle Profiling
	if gl.inputManager.JustPressed(input.ActionToggleProfiling) {
		gl.hudRenderer.ToggleProfiling()
	}
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

// RefreshRender renders a frame without updating game state (used during window resize)
func (gl *GameLoop) RefreshRender() {
	// Use a small dt to keep FOV transitions smooth, but don't update game state
	dt := 0.016 // ~60fps
	gl.renderer.Render(gl.world, gl.player, dt)

	// Render pause menu if paused
	if gl.paused {
		gl.pauseMenu.Render(gl.window, gl.uiRenderer, gl.hudRenderer, gl.player)
	}

	gl.window.SwapBuffers()
}

func (gl *GameLoop) handlePauseMenu() MenuAction {
	if gl.paused {
		action := gl.pauseMenu.Render(gl.window, gl.uiRenderer, gl.hudRenderer, gl.player)
		switch action {
		case MenuActionResume:
			gl.paused = false
		case MenuActionQuitToMenu:
			return MenuActionQuitToMenu
		default:
			return MenuActionNone
		}
	}
	return MenuActionNone
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
