package main

import (
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"mini-mc/internal/game"
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

	// Create App (Manages Lifecycle)
	app := game.NewApp(window)

	// Setup input handlers (routes low level callbacks to App/Session)
	game.SetupInputHandlers(app)

	// Run the app loop
	app.Run()

	// App.Run() only returns when Main Loop exits (window closed)
	// Cleanup is handled within App/Session
}
