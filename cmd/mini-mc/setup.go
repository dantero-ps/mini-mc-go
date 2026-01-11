package main

import (
	"mini-mc/internal/graphics/renderables/blocks"
	"mini-mc/internal/graphics/renderables/breaking"
	"mini-mc/internal/graphics/renderables/crosshair"
	"mini-mc/internal/graphics/renderables/direction"
	"mini-mc/internal/graphics/renderables/hand"
	"mini-mc/internal/graphics/renderables/hud"
	"mini-mc/internal/graphics/renderables/items"
	"mini-mc/internal/graphics/renderables/ui"
	"mini-mc/internal/graphics/renderables/wireframe"
	renderer "mini-mc/internal/graphics/renderer"
	"mini-mc/internal/physics"
	"mini-mc/internal/player"
	"mini-mc/internal/world"
	"runtime"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

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

	// Initialize OpenGL bindings
	if err := gl.Init(); err != nil {
		return nil, err
	}

	// Disable V-Sync; we'll use our own FPS limiter
	glfw.SwapInterval(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)

	return window, nil
}

// GameComponents holds all the initialized game components
type GameComponents struct {
	Renderer    *renderer.Renderer
	UIRenderer  *ui.UI
	HUDRenderer *hud.HUD
	World       *world.World
	Player      *player.Player
}

func setupGame(mode player.GameMode) (*GameComponents, error) {
	// Initialize renderable features
	blocksRenderer := blocks.NewBlocks()
	itemsRenderer := items.NewItems()
	breakingRenderer := breaking.NewBreaking()
	wireframeRenderer := wireframe.NewWireframe()
	crosshairRenderer := crosshair.NewCrosshair()
	directionRenderer := direction.NewDirection()
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
		directionRenderer,
		handRenderer,
		uiRenderer,
		hudRenderer,
	)
	if err != nil {
		return nil, err
	}

	// Create world
	gameWorld := world.New()

	// Initialize mesh worker pool system (4 workers for mesh generation)
	blocks.InitMeshSystem(runtime.NumCPU() / 2)

	// Generate a smaller initial spawn area synchronously to keep startup smooth
	spawnX, spawnZ := float32(0), float32(0)
	gameWorld.StreamChunksAroundSync(spawnX, spawnZ, 50)

	// Compute ground level at spawn
	tempPos := mgl32.Vec3{spawnX, 300, spawnZ}
	groundY := physics.FindGroundLevel(spawnX, spawnZ, tempPos, gameWorld)

	// Initialize player at safe ground
	gamePlayer := player.New(gameWorld, mode)
	gamePlayer.Position[0] = spawnX
	gamePlayer.Position[2] = spawnZ
	gamePlayer.Position[1] = groundY
	gamePlayer.OnGround = true

	return &GameComponents{
		Renderer:    r,
		UIRenderer:  uiRenderer,
		HUDRenderer: hudRenderer,
		World:       gameWorld,
		Player:      gamePlayer,
	}, nil
}
