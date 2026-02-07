package game

import (
	// This is actually standard input package, aliased
	"log"
	"mini-mc/internal/graphics/renderables/font"
	"mini-mc/internal/graphics/renderables/ui"
	standardInput "mini-mc/internal/input"
	"mini-mc/internal/player"
	"mini-mc/internal/profiling"
	"mini-mc/internal/ui/menu"

	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type AppState int

const (
	StateMainMenu AppState = iota
	StatePlaying
)

type App struct {
	window       *glfw.Window
	inputManager *standardInput.InputManager

	state AppState

	// Main Menu components
	mainMenu     *menu.MainMenu
	menuUI       *ui.UI
	fontRenderer *font.FontRenderer

	// Game Session
	session *Session

	fpsLimiter *FPSLimiter
	lastTime   time.Time
}

func NewApp(window *glfw.Window, im *standardInput.InputManager) *App {
	// Initialize UI for Main Menu
	newUI := ui.NewUI()
	if err := newUI.Init(); err != nil {
		panic(err)
	}

	// Create independent font renderer for menu
	atlas, err := font.BuildFontAtlas("assets/fonts/Minecraft.otf", 90)
	if err != nil {
		panic(err)
	}
	fr, err := font.NewFontRenderer(atlas)
	if err != nil {
		panic(err)
	}
	newUI.SetFontRenderer(fr)

	width, height := window.GetSize()
	newUI.SetViewport(width, height)
	fr.SetViewport(float32(width), float32(height))

	return &App{
		window:       window,
		inputManager: im,
		state:        StateMainMenu,
		mainMenu:     menu.NewMainMenu(),
		menuUI:       newUI,
		fontRenderer: fr,
		fpsLimiter:   NewFPSLimiter(),
		lastTime:     time.Now(),
	}
}

func (a *App) Run() {
	for !a.window.ShouldClose() {
		a.tick()
	}
}

func (a *App) tick() {
	profiling.ResetFrame()
	startTick := time.Now() // Measure pure processing time
	now := time.Now()
	dt := now.Sub(a.lastTime).Seconds()
	a.lastTime = now

	glfw.PollEvents()

	switch a.state {
	case StateMainMenu:
		a.updateMainMenu(dt)
		a.renderMainMenu()
	case StatePlaying:
		if a.session != nil {
			action := a.session.Update(dt, a.inputManager)
			a.session.Render(dt)

			if action == menu.ActionQuitToMenu {
				a.EndSession()
			} else if action == menu.ActionQuitGame {
				a.window.SetShouldClose(true)
			}
		}
	}

	a.window.SwapBuffers()

	// Check if frame took too long (> 16ms)
	processingDuration := time.Since(startTick)
	if processingDuration > 16*time.Millisecond {
		log.Printf("Slow frame: %v. Top tasks: %s", processingDuration, profiling.TopNCurrentFrame(5))
	}

	a.inputManager.PostUpdate() // Clear "JustPressed" flags

	// FPS limit
	paused := false
	if a.session != nil {
		paused = a.session.Paused
	}
	a.fpsLimiter.Wait(paused || a.state == StateMainMenu)
}

func (a *App) updateMainMenu(dt float64) {
	// Handle input for menu
	action := a.mainMenu.Update(a.window, a.inputManager.JustPressed(standardInput.ActionMouseLeft))

	if action == menu.ActionStartSurvival {
		a.StartSession(player.GameModeSurvival)
	} else if action == menu.ActionStartCreative {
		a.StartSession(player.GameModeCreative)
	} else if action == menu.ActionResume {
		// Should not happen in main menu, but...
		// Resume logic usually for PauseMenu.
	} else if action == menu.ActionQuitGame {
		a.window.SetShouldClose(true)
	}
}

func (a *App) renderMainMenu() {
	// Clear screen
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// Use menuUI to render
	a.menuUI.BeginFrame()
	a.mainMenu.Render(a.menuUI, a.window)
	a.menuUI.Flush()
}

func (a *App) StartSession(mode player.GameMode) {
	var err error
	a.session, err = NewSession(a.window, mode)
	if err != nil {
		panic(err)
	}
	a.state = StatePlaying
}

func (a *App) EndSession() {
	if a.session != nil {
		a.session.Cleanup()
		a.session = nil
	}
	a.state = StateMainMenu

	// Restore cursor for menu
	a.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
}

// RefreshRender handles window resize repaints
func (a *App) RefreshRender() {
	if a.state == StatePlaying && a.session != nil {
		a.session.RefreshRender()
	} else {
		// Repaint menu
		a.renderMainMenu()
		a.window.SwapBuffers()
	}
}
