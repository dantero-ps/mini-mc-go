package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	windowWidth  = 800
	windowHeight = 600
)

func init() {
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.False)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "OpenGL 4.1 - Single Triangle (Max Perf)", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	// Disable VSync for max raw framerate; comment this if you want vsync.
	glfw.SwapInterval(0)

	if err := gl.Init(); err != nil {
		panic(err)
	}

	// Minimal shaders
	vertexSrc := `#version 410 core
layout(location = 0) in vec2 position;
void main() {
	gl_Position = vec4(position, 0.0, 1.0);
}` + "\x00"

	fragmentSrc := `#version 410 core
out vec4 fragColor;
void main() {
	fragColor = vec4(0.0, 1.0, 0.0, 1.0);
}` + "\x00"

	program, err := newProgram(vertexSrc, fragmentSrc)
	if err != nil {
		panic(err)
	}
	defer gl.DeleteProgram(program)

	// Triangle vertex positions (NDC): highly cache-friendly small buffer
	vertices := []float32{
		0.0, 0.5,
		-0.5, -0.5,
		0.5, -0.5,
	}

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// setup attribute: location 0, 2 floats per vertex
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 0, gl.PtrOffset(0))

	// unbind to reduce accidental state changes
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)

	// Minimal GL state for fastest clear+draw
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	// FPS counter variables
	frames := 0
	last := time.Now()
	fpsTicker := time.NewTicker(time.Second)
	defer fpsTicker.Stop()
	gl.UseProgram(program)
	gl.BindVertexArray(vao)

	// Main loop
	for !window.ShouldClose() {
		// close on Esc
		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}

		// render
		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.DrawArrays(gl.TRIANGLES, 0, 3)
		// Keep state changes minimal: unbinding not required here for a single draw

		window.SwapBuffers()
		glfw.PollEvents()

		frames++

		select {
		case <-fpsTicker.C:
			// print fps once per second to console
			now := time.Now()
			elapsed := now.Sub(last).Seconds()
			if elapsed > 0 {
				fmt.Printf("FPS: %d\n", int(float64(frames)/elapsed+0.5))
			}
			frames = 0
			last = now
		default:
			// continue rendering
		}
	}

	// cleanup
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteVertexArrays(1, &vao)
}

// newProgram compiles shaders and links them into a program.
func newProgram(vertexSrc, fragmentSrc string) (uint32, error) {
	v := gl.CreateShader(gl.VERTEX_SHADER)
	cvertex, freeVertex := gl.Strs(vertexSrc)
	gl.ShaderSource(v, 1, cvertex, nil)
	freeVertex()
	gl.CompileShader(v)

	var status int32
	gl.GetShaderiv(v, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(v, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetShaderInfoLog(v, logLength, nil, &log[0])
		return 0, fmt.Errorf("vertex shader compile error: %s", string(log))
	}

	f := gl.CreateShader(gl.FRAGMENT_SHADER)
	cfragment, freeFragment := gl.Strs(fragmentSrc)
	gl.ShaderSource(f, 1, cfragment, nil)
	freeFragment()
	gl.CompileShader(f)
	gl.GetShaderiv(f, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(f, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetShaderInfoLog(f, logLength, nil, &log[0])
		return 0, fmt.Errorf("fragment shader compile error: %s", string(log))
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, v)
	gl.AttachShader(program, f)
	gl.LinkProgram(program)
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
		log := make([]byte, logLength+1)
		gl.GetProgramInfoLog(program, logLength, nil, &log[0])
		return 0, fmt.Errorf("program link error: %s", string(log))
	}

	// shaders can be deleted after linking
	gl.DeleteShader(v)
	gl.DeleteShader(f)
	return program, nil
}
