package blocks

import (
	"path/filepath"
)

const (
	ShadersDir = "assets/shaders/blocks"
)

var (
	MainVertShader  = filepath.Join(ShadersDir, "main.vert")
	MainFragShader  = filepath.Join(ShadersDir, "main.frag")
	FluidVertShader = filepath.Join(ShadersDir, "fluid.vert")
	FluidFragShader = filepath.Join(ShadersDir, "fluid.frag")
)

type atlasWrite struct {
	offsetBytes int
	data        []int16
}

// freeSpan tracks a contiguous hole in an atlasRegion's VBO, measured in int16 units.
type freeSpan struct {
	offsetShorts int
	sizeShorts   int
}

type atlasRegion struct {
	key            [2]int
	vao            uint32
	vbo            uint32
	capacityBytes  int
	totalFloats    int        // high-water mark: next append offset in int16 units
	freeList       []freeSpan // holes inside [0, totalFloats), sorted by offsetShorts, coalesced
	orderedColumns []*columnMesh
	pendingWrites  []atlasWrite
	lastCompact    uint64
	growthCount    int
	activeColumns  int // number of columnMeshes with vertexCount>0 referencing this region
}

type chunkMesh struct {
	vertexCount int32
	cpuVerts    []uint32 // Packed vertices
	fluidVerts  []float32
	firstFloat  int    // offset into atlas in shorts
	firstVertex int32  // offset into atlas in vertices
	regionKey   [2]int // atlas region owning this mesh data
}

type columnMesh struct {
	x            int
	z            int
	vertexCount  int32
	firstFloat   int
	dirty        bool
	firstVertex  int32  // offset into atlas in vertices (firstFloat/4)
	drawnFrame   uint64 // last frame this column participated in a merged draw call
	visibleFrame uint64 // last frame this column was marked visible
	regionKey    [2]int // atlas region owning this column data
	retryFrame   uint64 // earliest frame at which a failed alloc may be retried
}

type plane struct {
	a, b, c, d float32
}
