package ui

import (
	"mini-mc/internal/graphics"
	renderer "mini-mc/internal/graphics/renderer"
	"path/filepath"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	ShadersDir = "assets/shaders/ui"
	WinWidth   = 900
	WinHeight  = 600
)

var (
	UIVertShader = filepath.Join(ShadersDir, "ui.vert")
	UIFragShader = filepath.Join(ShadersDir, "ui.frag")
	// Textured UI shaders
	TexUIVertShader = filepath.Join(ShadersDir, "textured_ui.vert")
	TexUIFragShader = filepath.Join(ShadersDir, "textured_ui.frag")
)

type drawCmdKind uint8

const (
	cmdFilledRect drawCmdKind = iota
	cmdTexturedRect
	cmdText
)

type drawCmd struct {
	kind drawCmdKind

	// Geometry (pixels, top-left origin)
	x, y, w, h float32

	// Filled rect color / textured tint
	color mgl32.Vec3
	alpha float32

	// Textured rect
	textureID uint32
	u0, v0    float32
	u1, v1    float32

	// Resolved draw-range (set during Flush build)
	first int32
	count int32

	// Text
	text      string
	textScale float32
}

// UI implements UI rendering for rectangles and text
type UI struct {
	shader           *graphics.Shader
	texShader        *graphics.Shader
	fontRenderer     *graphics.FontRenderer
	vao              uint32
	vbo              uint32
	texVao           uint32
	texVbo           uint32
	isDraggingSlider bool
	activeSliderID   string

	// FIFO draw list (strict draw order)
	cmds []drawCmd

	// Built per-flush buffers (uploaded once, drawn many)
	filledVerts     []float32 // pos only: 2 floats per vertex
	unifiedTexVerts []float32 // pos(2)+uv(2)+rgba(4) = 8 floats per vertex

	// Internal
	frameActive bool
}

// NewUI creates a new UI renderable
func NewUI() *UI {
	return &UI{
		cmds:            make([]drawCmd, 0, 2048),
		filledVerts:     make([]float32, 0, 2048),
		unifiedTexVerts: make([]float32, 0, 8192),
	}
}

// Init initializes the UI rendering system
func (u *UI) Init() error {
	// Create flat color shader
	var err error
	u.shader, err = graphics.NewShader(UIVertShader, UIFragShader)
	if err != nil {
		return err
	}

	// Create textured shader
	u.texShader, err = graphics.NewShader(TexUIVertShader, TexUIFragShader)
	if err != nil {
		return err
	}

	// Setup VAO and VBO for flat rects (pos: 2 floats per vertex)
	gl.GenVertexArrays(1, &u.vao)
	gl.GenBuffers(1, &u.vbo)
	gl.BindVertexArray(u.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 6*2*4*200, nil, gl.DYNAMIC_DRAW) // allocate for ~200 rects
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	// Setup VAO and VBO for textured rects
	// Pos(2) + UV(2) + Color(4) = 8 floats per vertex
	gl.GenVertexArrays(1, &u.texVao)
	gl.GenBuffers(1, &u.texVbo)
	gl.BindVertexArray(u.texVao)
	gl.BindBuffer(gl.ARRAY_BUFFER, u.texVbo)
	gl.BufferData(gl.ARRAY_BUFFER, 6*8*4*1024, nil, gl.DYNAMIC_DRAW) // allocate for ~1024 rects

	// Pos
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(0))
	// UV
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(2*4))
	// Color (RGBA)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, 8*4, gl.PtrOffset(4*4))

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	return nil
}

// Render handles UI rendering - this is mainly called for text and HUD rendering
func (u *UI) Render(ctx renderer.RenderContext) {
	// UI rendering is mainly handled by HUD feature now
}

// BeginFrame starts a new FIFO UI frame. Optional but recommended.
func (u *UI) BeginFrame() {
	u.cmds = u.cmds[:0]
	u.frameActive = true
}

func (u *UI) SetFontRenderer(fr *graphics.FontRenderer) {
	u.fontRenderer = fr
}

func (u *UI) DrawText(text string, x, y float32, scale float32, color mgl32.Vec3) {
	if !u.frameActive {
		u.BeginFrame()
	}
	u.cmds = append(u.cmds, drawCmd{
		kind:      cmdText,
		x:         x,
		y:         y,
		color:     color,
		alpha:     1.0,
		text:      text,
		textScale: scale,
	})
}

// Dispose cleans up OpenGL resources
func (u *UI) Dispose() {
	if u.vao != 0 {
		gl.DeleteVertexArrays(1, &u.vao)
	}
	if u.vbo != 0 {
		gl.DeleteBuffers(1, &u.vbo)
	}
	if u.texVao != 0 {
		gl.DeleteVertexArrays(1, &u.texVao)
	}
	if u.texVbo != 0 {
		gl.DeleteBuffers(1, &u.texVbo)
	}
}

// DrawTexturedRect batches a textured rectangle (with color tint).
// Vertices accumulate in per-texture buckets until Flush() is called.
// u0, v0, u1, v1 are UV coordinates (0.0-1.0).
func (u *UI) DrawTexturedRect(x, y, w, h float32, textureID uint32, u0, v0, u1, v1 float32, color mgl32.Vec3, alpha float32) {
	if !u.frameActive {
		u.BeginFrame()
	}
	u.cmds = append(u.cmds, drawCmd{
		kind:      cmdTexturedRect,
		x:         x,
		y:         y,
		w:         w,
		h:         h,
		color:     color,
		alpha:     alpha,
		textureID: textureID,
		u0:        u0,
		v0:        v0,
		u1:        u1,
		v1:        v1,
	})
}

// DrawFilledRect batches a solid-color rectangle.
// Vertices accumulate until Flush() is called.
func (u *UI) DrawFilledRect(x, y, w, h float32, color mgl32.Vec3, alpha float32) {
	if !u.frameActive {
		u.BeginFrame()
	}
	u.cmds = append(u.cmds, drawCmd{
		kind:  cmdFilledRect,
		x:     x,
		y:     y,
		w:     w,
		h:     h,
		color: color,
		alpha: alpha,
	})
}

// DrawSlider draws a horizontal slider with the given value (0.0-1.0) and returns the new value. Supports drag capture and optional step snapping with tick marks.
// sliderID must uniquely identify this slider so that only one slider is active during a drag.
func (u *UI) DrawSlider(x, y, w, h float32, value float32, window interface{}, steps int, sliderID string) float32 {
	// Draw slider track
	trackColor := mgl32.Vec3{0.3, 0.3, 0.3}
	u.DrawFilledRect(x, y, w, h, trackColor, 0.8)

	// Draw step ticks if requested (downsample to ~10 ticks max to reduce clutter)
	if steps > 1 {
		tickHeight := h * 0.6
		tickY := y + (h-tickHeight)*0.5
		tickWidth := float32(2)
		tickColor := mgl32.Vec3{0.9, 0.9, 0.9}
		visibleTicks := 10
		if visibleTicks < 2 {
			visibleTicks = 2
		}
		stepSpacing := steps / visibleTicks
		if stepSpacing < 1 {
			stepSpacing = 1
		}
		for i := 0; i < steps; i++ {
			if i != 0 && i != steps-1 && (i%stepSpacing) != 0 {
				continue
			}
			var ratio float32
			if steps <= 1 {
				ratio = 0
			} else {
				ratio = float32(i) / float32(steps-1)
			}
			tx := x + ratio*w - tickWidth*0.5
			u.DrawFilledRect(tx, tickY, tickWidth, tickHeight, tickColor, 0.18)
		}
	}

	// Thumb size
	thumbWidth := float32(20)
	thumbHeight := h

	// Mouse interaction with drag capture and snapping
	if glfwWindow, ok := window.(*glfw.Window); ok {
		cx, cy := glfwWindow.GetCursorPos()
		mouseX, mouseY := float32(cx), float32(cy)
		leftDown := glfwWindow.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press

		inside := mouseY >= y && mouseY <= y+h && mouseX >= x && mouseX <= x+w

		if u.isDraggingSlider && u.activeSliderID == sliderID {
			if leftDown {
				v := (mouseX - x) / w
				if v < 0 {
					v = 0
				}
				if v > 1 {
					v = 1
				}
				if steps > 1 {
					denom := float32(steps - 1)
					stepIndex := int(v*denom + 0.5)
					if stepIndex < 0 {
						stepIndex = 0
					}
					if stepIndex > steps-1 {
						stepIndex = steps - 1
					}
					v = float32(stepIndex) / denom
				}
				value = v
			} else {
				u.isDraggingSlider = false
				u.activeSliderID = ""
			}
		} else if !u.isDraggingSlider && leftDown && inside {
			// Begin drag
			u.isDraggingSlider = true
			u.activeSliderID = sliderID
			v := (mouseX - x) / w
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			if steps > 1 {
				denom := float32(steps - 1)
				stepIndex := int(v*denom + 0.5)
				if stepIndex < 0 {
					stepIndex = 0
				}
				if stepIndex > steps-1 {
					stepIndex = steps - 1
				}
				v = float32(stepIndex) / denom
			}
			value = v
		}
	}

	thumbX := x + (w-thumbWidth)*value
	thumbColor := mgl32.Vec3{0.6, 0.6, 0.6}
	u.DrawFilledRect(thumbX, y, thumbWidth, thumbHeight, thumbColor, 0.9)

	return value
}

func (u *UI) Flush() {
	if len(u.cmds) == 0 {
		return
	}

	// Build packed buffers + resolve draw ranges
	u.filledVerts = u.filledVerts[:0]
	u.unifiedTexVerts = u.unifiedTexVerts[:0]

	filledVertexCount := int32(0)
	texVertexCount := int32(0)

	for i := range u.cmds {
		cmd := &u.cmds[i]
		switch cmd.kind {
		case cmdFilledRect:
			x0 := (cmd.x/float32(WinWidth))*2 - 1
			y0 := 1 - (cmd.y/float32(WinHeight))*2
			x1 := ((cmd.x+cmd.w)/float32(WinWidth))*2 - 1
			y1 := 1 - ((cmd.y+cmd.h)/float32(WinHeight))*2

			cmd.first = filledVertexCount
			cmd.count = 6
			filledVertexCount += 6

			u.filledVerts = append(u.filledVerts,
				x0, y0,
				x1, y0,
				x1, y1,
				x0, y0,
				x1, y1,
				x0, y1,
			)

		case cmdTexturedRect:
			x0 := (cmd.x/float32(WinWidth))*2 - 1
			y0 := 1 - (cmd.y/float32(WinHeight))*2
			x1 := ((cmd.x+cmd.w)/float32(WinWidth))*2 - 1
			y1 := 1 - ((cmd.y+cmd.h)/float32(WinHeight))*2

			cmd.first = texVertexCount
			cmd.count = 6
			texVertexCount += 6

			r, g, b, a := cmd.color.X(), cmd.color.Y(), cmd.color.Z(), cmd.alpha
			u.unifiedTexVerts = append(u.unifiedTexVerts,
				x0, y0, cmd.u0, cmd.v0, r, g, b, a,
				x1, y0, cmd.u1, cmd.v0, r, g, b, a,
				x1, y1, cmd.u1, cmd.v1, r, g, b, a,
				x0, y0, cmd.u0, cmd.v0, r, g, b, a,
				x1, y1, cmd.u1, cmd.v1, r, g, b, a,
				x0, y1, cmd.u0, cmd.v1, r, g, b, a,
			)
		case cmdText:
			// No geometry to build
		}
	}

	// Upload once per pipeline (filled/textured)
	if len(u.filledVerts) > 0 {
		gl.BindVertexArray(u.vao)
		gl.BindBuffer(gl.ARRAY_BUFFER, u.vbo)
		sizeBytes := len(u.filledVerts) * 4
		gl.BufferData(gl.ARRAY_BUFFER, sizeBytes, nil, gl.DYNAMIC_DRAW) // orphan
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, sizeBytes, gl.Ptr(u.filledVerts))
	}
	if len(u.unifiedTexVerts) > 0 {
		gl.BindVertexArray(u.texVao)
		gl.BindBuffer(gl.ARRAY_BUFFER, u.texVbo)
		sizeBytes := len(u.unifiedTexVerts) * 4
		gl.BufferData(gl.ARRAY_BUFFER, sizeBytes, nil, gl.DYNAMIC_DRAW) // orphan
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, sizeBytes, gl.Ptr(u.unifiedTexVerts))
	}

	// Draw in strict FIFO order
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	currentKind := drawCmdKind(255)
	currentTexture := uint32(0)
	colorLoc := int32(-1)

	for i := range u.cmds {
		cmd := &u.cmds[i]
		if cmd.kind != currentKind {
			currentKind = cmd.kind
			currentTexture = 0
			colorLoc = -1

			switch cmd.kind {
			case cmdFilledRect:
				u.shader.Use()
				gl.BindVertexArray(u.vao)
				gl.BindBuffer(gl.ARRAY_BUFFER, u.vbo)
				colorLoc = gl.GetUniformLocation(u.shader.ID, gl.Str("uColor\x00"))
			case cmdTexturedRect:
				u.texShader.Use()
				gl.BindVertexArray(u.texVao)
				gl.BindBuffer(gl.ARRAY_BUFFER, u.texVbo)
				gl.ActiveTexture(gl.TEXTURE0)
				u.texShader.SetInt("uTexture", 0)
			case cmdText:
				// Font renderer manages its own GL state; we'll just call it.
			}
		}

		switch cmd.kind {
		case cmdFilledRect:
			if colorLoc >= 0 {
				gl.Uniform4f(colorLoc, cmd.color.X(), cmd.color.Y(), cmd.color.Z(), cmd.alpha)
			}
			gl.DrawArrays(gl.TRIANGLES, cmd.first, cmd.count)
		case cmdTexturedRect:
			if cmd.textureID != currentTexture {
				gl.BindTexture(gl.TEXTURE_2D, cmd.textureID)
				currentTexture = cmd.textureID
			}
			gl.DrawArrays(gl.TRIANGLES, cmd.first, cmd.count)
		case cmdText:
			if u.fontRenderer != nil && cmd.text != "" {
				// Ensure UI VAOs don't interfere with font rendering
				gl.BindVertexArray(0)
				gl.BindBuffer(gl.ARRAY_BUFFER, 0)
				u.fontRenderer.Render(cmd.text, cmd.x, cmd.y, cmd.textScale, cmd.color)

				// FontRenderer.Render may change GL state (blend/depth/cull/program/VAO).
				// Re-apply the UI state so subsequent FIFO commands (e.g. button rects) still render correctly.
				gl.Disable(gl.DEPTH_TEST)
				gl.Disable(gl.CULL_FACE)
				gl.Enable(gl.BLEND)
				gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
				gl.ActiveTexture(gl.TEXTURE0)

				// Force rebind of pipeline after font draw
				currentKind = drawCmdKind(255)
				currentTexture = 0
			}
		}
	}

	// Restore state
	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.CULL_FACE)

	// Clear FIFO
	u.cmds = u.cmds[:0]
}
