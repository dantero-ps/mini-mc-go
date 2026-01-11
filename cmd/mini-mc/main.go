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

	// Initialize game components
	game, err := setupGame()
	if err != nil {
		panic(err)
	}
	defer blocks.ShutdownMeshSystem()

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
	setupInputHandlers(window, game.HUDRenderer, game.Player, gameLoop.Paused())

	// Run the game
	gameLoop.Run()
}
