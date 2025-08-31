package graphics

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Shader represents an OpenGL shader program
type Shader struct {
	ID uint32
}

// NewShader creates a new shader program from vertex and fragment shader source files
func NewShader(vertexPath, fragmentPath string) (*Shader, error) {
	vertexSource, err := os.ReadFile(vertexPath)
	if err != nil {
		return nil, fmt.Errorf("could not read vertex shader file: %v", err)
	}

	fragmentSource, err := os.ReadFile(fragmentPath)
	if err != nil {
		return nil, fmt.Errorf("could not read fragment shader file: %v", err)
	}

	program, err := compileProgram(string(vertexSource), string(fragmentSource))
	if err != nil {
		return nil, err
	}

	return &Shader{ID: program}, nil
}

// Use activates the shader program
func (s *Shader) Use() {
	gl.UseProgram(s.ID)
}

// SetBool sets a boolean uniform
func (s *Shader) SetBool(name string, value bool) {
	var intValue int32
	if value {
		intValue = 1
	}
	gl.Uniform1i(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")), intValue)
}

// SetInt sets an integer uniform
func (s *Shader) SetInt(name string, value int32) {
	gl.Uniform1i(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")), value)
}

// SetFloat sets a float uniform
func (s *Shader) SetFloat(name string, value float32) {
	gl.Uniform1f(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")), value)
}

// SetVector3 sets a vector3 uniform
func (s *Shader) SetVector3(name string, x, y, z float32) {
	gl.Uniform3f(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")), x, y, z)
}

// SetMatrix4 sets a 4x4 matrix uniform
func (s *Shader) SetMatrix4(name string, value *float32) {
	gl.UniformMatrix4fv(gl.GetUniformLocation(s.ID, gl.Str(name+"\x00")), 1, false, value)
}

// Helper functions
func compileProgram(vertexSrc, fragmentSrc string) (uint32, error) {
	vertexShader, err := compileShader(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}
