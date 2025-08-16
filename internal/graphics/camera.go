package graphics

import (
	"mini-mc/internal/player"

	"github.com/go-gl/mathgl/mgl32"
)

// Camera handles the view and projection matrices
type Camera struct {
	AspectRatio float32
	FOV         float32
	NearPlane   float32
	FarPlane    float32
}

func NewCamera(width, height int) *Camera {
	return &Camera{
		AspectRatio: float32(width) / float32(height),
		FOV:         60.0,
		NearPlane:   0.1,
		FarPlane:    1000.0,
	}
}

func (c *Camera) GetProjectionMatrix() mgl32.Mat4 {
	return mgl32.Perspective(mgl32.DegToRad(c.FOV), c.AspectRatio, c.NearPlane, c.FarPlane)
}

func (c *Camera) GetViewMatrix(player *player.Player) mgl32.Mat4 {
	return player.GetViewMatrix()
}
