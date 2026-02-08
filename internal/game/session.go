package game

import (
	"runtime"
	"time"

	"mini-mc/internal/config"
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderables/breaking"
	"mini-mc/internal/graphics/renderables/crosshair"
	"mini-mc/internal/graphics/renderables/hand"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/graphics/renderables/wireframe"
	"mini-mc/internal/graphics/renderer"
	standardInput "mini-mc/internal/input"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/ui/menu"
	"mini-mc/internal/world"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Session struct {
	Window      *glfw.Window
	Renderer    *renderer.Renderer
	UIRenderer  *ui.UI
	HUDRenderer *hud.HUD
	Player      *player.Player
	World       *world.World

	Paused    bool
	PauseMenu *menu.PauseMenu

	Frames           int
	LastFPSCheckTime time.Time
	lastEviction     time.Time
}

func NewSession(window *glfw.Window, mode player.GameMode) (*Session, error) {
	// Initialize renderable features
	blocksRenderer := blocks.NewBlocks()
	itemsRenderer := items.NewItems()
	breakingRenderer := breaking.NewBreaking()
	wireframeRenderer := wireframe.NewWireframe()
	crosshairRenderer := crosshair.NewCrosshair()
	handRenderer := hand.NewHand(itemsRenderer)
	uiRenderer := ui.NewUI()
	hudRenderer := hud.NewHUD()

	// Initialize renderer with all features
	r, err := renderer.NewRenderer(
		blocksRenderer,
		itemsRenderer,
		breakingRenderer,
		wireframeRenderer,
		crosshairRenderer,
		handRenderer,
		uiRenderer,
		hudRenderer,
	)
	if err != nil {
		return nil, err
	}

	uiRenderer.SetFontRenderer(hudRenderer.FontRenderer())

	// Create world
	gameWorld := world.New()

	// Initialize (or re-initialize) mesh system
	blocks.InitMeshSystem(runtime.NumCPU() - 1)

	// Create player
	gamePlayer := player.New(gameWorld, mode)

	// Fix spawn position: find ground level at 0,0
	// Ensure the chunk at 0,0 is generated or height is known
	spawnX, spawnZ := 0, 0
	groundY := gameWorld.SurfaceHeightAt(spawnX, spawnZ)
	// Set player Y to ground + height + slight buffer
	gamePlayer.Position[1] = float32(groundY) + 2.0
	// Reset velocity just in case
	gamePlayer.Velocity = [3]float32{0, 0, 0}

	// Set cursor disabled for game
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	width, height := window.GetSize()
	r.UpdateViewport(width, height)

	// Connect inventory state changes to HUD
	gamePlayer.OnInventoryStateChange = func(isOpen bool) {
		hudRenderer.SetInventoryOpen(isOpen, gamePlayer)
	}

	return &Session{
		Window:           window,
		Renderer:         r,
		UIRenderer:       uiRenderer,
		HUDRenderer:      hudRenderer,
		World:            gameWorld,
		Player:           gamePlayer,
		PauseMenu:        menu.NewPauseMenu(),
		LastFPSCheckTime: time.Now(),
	}, nil
}

func (s *Session) Cleanup() {
	s.World.Close()
	blocks.ShutdownMeshSystem()
	s.Renderer.Dispose()

	// Explicilty nil out
	s.World = nil
	s.Player = nil
	s.Renderer = nil
	s.UIRenderer = nil
	s.HUDRenderer = nil
}

func (s *Session) Update(dt float64, im *standardInput.InputManager) menu.Action {
	// Handle Menu Logic if paused
	if s.Paused {
		action := s.PauseMenu.Update(s.Window, im.JustPressed(standardInput.ActionMouseLeft))
		switch action {
		case menu.ActionResume:
			s.SetPaused(false)
			return menu.ActionNone
		case menu.ActionQuitToMenu:
			return menu.ActionQuitToMenu
		case menu.ActionQuitGame:
			return menu.ActionQuitGame
		}
	}

	if !s.Paused {
		profiling.Track("player.Update")
		s.Player.Update(dt, im)
		profiling.Track("world.UpdateEntities")
		s.World.UpdateEntities(dt)
	}

	s.handleInputActions(im)
	s.processWorldUpdates()

	return menu.ActionNone
}

func (s *Session) Render(dt float64) (time.Duration, time.Duration, time.Duration) {
	renderStart := time.Now()
	s.Renderer.Render(s.World, s.Player, dt)

	// Render Pause Menu
	if s.Paused {
		s.UIRenderer.BeginFrame()
		s.PauseMenu.Render(s.UIRenderer, s.Window)
		s.UIRenderer.Flush()
	}

	renderDur := time.Since(renderStart)
	s.HUDRenderer.ProfilingSetRenderDuration(renderDur)

	s.Frames++
	if time.Since(s.LastFPSCheckTime) >= time.Second {
		s.Frames = 0
		s.LastFPSCheckTime = time.Now()
	}

	return renderDur, 0, 0
}

func (s *Session) SetPaused(paused bool) {
	s.Paused = paused
	if s.Paused {
		s.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		w, h := s.Window.GetSize()
		s.Window.SetCursorPos(float64(w)/2, float64(h)/2)
	} else {
		s.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		s.Player.FirstMouse = true
	}
}

func (s *Session) processWorldUpdates() {
	if !s.Paused {
		func() {
			s.World.StreamChunksAroundAsync(s.Player.Position[0], s.Player.Position[2], config.GetChunkLoadRadius())
		}()
	}

	func() {
		defer profiling.Track("blocks.ProcessMeshResults")()
		blocks.ProcessMeshResults()
	}()

	// Periodic cleanup (every 1 second)
	if time.Since(s.lastEviction) > time.Second {
		func() {
			defer profiling.Track("world.EvictFarChunks")()
			// Use EvictRadius (e.g. 2x render distance) to avoid thrashing
			evictRadius := config.GetChunkEvictRadius()
			s.World.EvictFarChunks(s.Player.Position[0], s.Player.Position[2], evictRadius)
			blocks.PruneMeshesByWorld(s.World, s.Player.Position[0], s.Player.Position[2], evictRadius)
		}()
		s.lastEviction = time.Now()
	}
}

func (s *Session) handleInputActions(im *standardInput.InputManager) {
	p := s.Player

	if im.JustPressed(standardInput.ActionHotbar1) {
		s.handleHotbar(0)
	}
	if im.JustPressed(standardInput.ActionHotbar2) {
		s.handleHotbar(1)
	}
	if im.JustPressed(standardInput.ActionHotbar3) {
		s.handleHotbar(2)
	}
	if im.JustPressed(standardInput.ActionHotbar4) {
		s.handleHotbar(3)
	}
	if im.JustPressed(standardInput.ActionHotbar5) {
		s.handleHotbar(4)
	}
	if im.JustPressed(standardInput.ActionHotbar6) {
		s.handleHotbar(5)
	}
	if im.JustPressed(standardInput.ActionHotbar7) {
		s.handleHotbar(6)
	}
	if im.JustPressed(standardInput.ActionHotbar8) {
		s.handleHotbar(7)
	}
	if im.JustPressed(standardInput.ActionHotbar9) {
		s.handleHotbar(8)
	}

	if im.JustPressed(standardInput.ActionDropItem) {
		if !s.Paused && !p.IsInventoryOpen {
			dropStack := im.IsActive(standardInput.ActionModControl)
			p.DropHeldItem(dropStack)
		}
	}

	if im.JustPressed(standardInput.ActionInventory) {
		if !s.Paused {
			newState := !p.IsInventoryOpen
			p.SetInventoryOpen(newState)
			if newState {
				s.Window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
				w, h := s.Window.GetSize()
				s.Window.SetCursorPos(float64(w)/2, float64(h)/2)
			} else {
				p.DropCursorItem()
				s.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
				p.FirstMouse = true
			}
		}
	}

	if im.JustPressed(standardInput.ActionPause) {
		if p.IsInventoryOpen {
			p.SetInventoryOpen(false)
			p.DropCursorItem()
			s.Window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
			p.FirstMouse = true
		} else {
			s.SetPaused(!s.Paused)
		}
	}

	if im.JustPressed(standardInput.ActionToggleWireframe) {
		config.ToggleWireframeMode()
	}

	if im.JustPressed(standardInput.ActionToggleProfiling) {
		s.HUDRenderer.ToggleProfiling()
	}
}

func (s *Session) handleHotbar(slot int) {
	if s.Player.IsInventoryOpen {
		s.HUDRenderer.MoveHoveredItemToHotbar(slot)
	} else {
		s.Player.HandleNumKey(slot)
	}
}

func (s *Session) RefreshRender() {
	dt := 0.016
	s.Renderer.Render(s.World, s.Player, dt)
	if s.Paused {
		s.UIRenderer.BeginFrame()
		s.PauseMenu.Render(s.UIRenderer, s.Window)
		s.UIRenderer.Flush()
	}
	s.Window.SwapBuffers()
}
