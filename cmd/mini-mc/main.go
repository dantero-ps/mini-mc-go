package main

import (
	"runtime"

	"mini-mc/internal/graphics/renderables/blocks"

	"github.com/go-gl/glfw/v3.3/glfw"
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

	for {
		// Show Game Mode Selection Menu
		gameMode := showMenu(window)

		// Initialize game components
		game, err := setupGame(gameMode)
		if err != nil {
			panic(err)
		}

		// Create game loop
		gameLoop := NewGameLoop(
			window,
			game.Renderer,
			game.UIRenderer,
			game.HUDRenderer,
			game.Player,
			game.World,
		)

		// Setup input handlers
		setupInputHandlers(window, gameLoop, game.Renderer, game.HUDRenderer, game.Player, gameLoop.Paused())

		// Run the game
		shouldRestart := gameLoop.Run()

		// Cleanup
		game.World.Close()
		blocks.ShutdownMeshSystem()
		game.Renderer.Dispose()

		if !shouldRestart {
			break
		}
	}
}
