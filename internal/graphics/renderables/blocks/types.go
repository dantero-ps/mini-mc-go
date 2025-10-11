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
	firstFloat  int       // offset into atlas in floats (pos+encodedNormal interleaved)
	firstVertex int32     // offset into atlas in vertices (firstFloat/4)
}

type columnMesh struct {
	x            int
	z            int
	cpuVerts     []float32
	vertexCount  int32
	firstFloat   int
	dirty        bool
	firstVertex  int32  // offset into atlas in vertices (firstFloat/4)
	drawnFrame   uint64 // last frame this column participated in a merged draw call
	visibleFrame uint64 // last frame this column was marked visible
}

type plane struct {
	a, b, c, d float32
}
