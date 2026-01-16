package renderer

import (
	"mini-mc/internal/graphics"
	"mini-mc/internal/player"
	"mini-mc/internal/world"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Renderer orchestrates rendering via renderable features
type Renderer struct {
	renderables []Renderable
	camera      *graphics.Camera

	// FOV transition
	targetFOV  float32
	currentFOV float32
}

// NewRenderer creates a new renderer with the given renderables
func NewRenderer(rs ...Renderable) (*Renderer, error) {
	// Configure OpenGL
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CCW)

	// Create camera
	camera := graphics.NewCamera(900, 600)

	renderer := &Renderer{
		renderables: rs,
		camera:      camera,
		targetFOV:   60.0,
		currentFOV:  60.0,
	}

	// Initialize all renderables
	for _, r := range rs {
		if err := r.Init(); err != nil {
			return nil, err
		}
	}

	return renderer, nil
}

// Render executes the main render loop
func (r *Renderer) Render(w *world.World, p *player.Player, dt float64) {
	// Clear the screen
	gl.ClearColor(0.53, 0.81, 0.92, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// Update FOV smoothly based on sprinting and horizontal speed
	{
		// Base and sprint FOVs
		normalFOV := float32(60.0)
		sprintFOV := float32(70.0)
		// Horizontal speed magnitude
		hs := float32(p.Velocity[0]*p.Velocity[0] + p.Velocity[2]*p.Velocity[2])
		isMovingFast := hs > 0.01
		if p.IsSprinting && isMovingFast {
			r.targetFOV = sprintFOV
		} else {
			r.targetFOV = normalFOV
		}
		// Interpolate
		transitionSpeed := float32(100.0)
		step := float32(dt) * transitionSpeed
		if r.currentFOV < r.targetFOV {
			r.currentFOV += step
			if r.currentFOV > r.targetFOV {
				r.currentFOV = r.targetFOV
			}
		} else if r.currentFOV > r.targetFOV {
			r.currentFOV -= step
			if r.currentFOV < r.targetFOV {
				r.currentFOV = r.targetFOV
			}
		}
		// Apply
		r.camera.FOV = r.currentFOV
	}

	// Compute view and projection matrices
	view := p.GetViewMatrix()
	projection := r.camera.GetProjectionMatrix()

	// Create render context
	ctx := RenderContext{
		Camera: r.camera,
		World:  w,
		Player: p,
		DT:     dt,
		View:   view,
		Proj:   projection,
	}

	// Render all features
	for _, renderable := range r.renderables {
		renderable.Render(ctx)
	}
}

// Dispose cleans up all renderables in reverse order
func (r *Renderer) Dispose() {
	// Dispose in reverse order
	for i := len(r.renderables) - 1; i >= 0; i-- {
		r.renderables[i].Dispose()
	}
}

// GetCamera returns the camera instance
func (r *Renderer) GetCamera() *graphics.Camera {
	return r.camera
}

// UpdateViewport updates the camera's viewport dimensions
func (r *Renderer) UpdateViewport(width, height int) {
	r.camera.SetViewport(width, height)
}
