package renderer

import (
	"mini-mc/internal/graphics"
	"mini-mc/internal/player"
	"mini-mc/internal/world"

	"github.com/go-gl/mathgl/mgl32"
)

// RenderContext provides shared context for all renderables
type RenderContext struct {
	Camera *graphics.Camera
	World  *world.World
	Player *player.Player
	DT     float64
	View   mgl32.Mat4
	Proj   mgl32.Mat4
}

// Renderable interface defines the lifecycle for renderable features
type Renderable interface {
	Init() error
	Render(ctx RenderContext)
	Dispose()
	SetViewport(width, height int)
}
