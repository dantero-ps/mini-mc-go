package blocks

import (
	"path/filepath"
)

const (
	ShadersDir = "assets/shaders/blocks"
)

var (
	MainVertShader = filepath.Join(ShadersDir, "main.vert")
	MainFragShader = filepath.Join(ShadersDir, "main.frag")
)

type chunkMesh struct {
	vao         uint32 // legacy per-chunk VAO (kept for now)
	vbo         uint32 // legacy per-chunk VBO (kept for cleanup)
	vertexCount int32
	cpuVerts    []float32 // kept for atlas updates
	firstFloat  int       // offset into atlas in floats (pos+normal interleaved)
}

type columnMesh struct {
	cpuVerts    []float32
	vertexCount int32
	firstFloat  int
	dirty       bool
}

type plane struct {
	a, b, c, d float32
}
