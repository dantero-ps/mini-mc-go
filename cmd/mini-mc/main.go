package main

import (
	"runtime"

	"mini-mc/internal/game"
	"mini-mc/internal/input"

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
	window, err := game.SetupWindow()
	if err != nil {
		panic(err)
	}

	// Create input manager
	inputManager := input.NewInputManager()
	inputManager.SetKeyCallback(window)

	// Create App (Manages Lifecycle)
	app := game.NewApp(window, inputManager)

	// Setup input handlers (routes low level callbacks to App/Session)
	game.SetupInputHandlers(app)

	// Run the app loop
	app.Run()

	// App.Run() only returns when Main Loop exits (window closed)
	// Cleanup is handled within App/Session
}
