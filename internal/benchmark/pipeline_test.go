// Package benchmark provides end-to-end benchmarks for the chunk generation pipeline.
//
// The pipeline under test has four stages:
//
//  1. Chunk generation  – terrain noise + biome placement via ChunkProvider189
//  2. Greedy meshing    – face visibility + quad merging → packed uint32 vertices
//  3. Vertex unpacking  – packed uint32 → int16 atlas format ready for GPU
//  4. GPU upload        – GenBuffers / MapBufferRange / copy / UnmapBuffer cycle
//
// Run all benchmarks:
//
//	go test -bench=. -benchmem ./internal/benchmark/
//
// Run a single stage with memory stats:
//
//	go test -bench=BenchmarkChunkGeneration -benchmem ./internal/benchmark/
package benchmark

import (
	"os"
	"runtime"
	"testing"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"mini-mc/internal/meshing"
	"mini-mc/internal/registry"
	"mini-mc/internal/world"
)

// glWindow holds the hidden GLFW window so that benchmarks running on worker
// goroutines (different OS threads) can call MakeContextCurrent before issuing
// OpenGL commands.
var glWindow *glfw.Window

// ---------------------------------------------------------------------------
// TestMain – one-time setup shared by all benchmarks in this package.
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	// The registry reads block model JSON from the assets/ directory at the
	// project root. Chdir ensures the path is correct regardless of where
	// `go test` is invoked from.
	if err := os.Chdir("../.."); err != nil {
		panic("cannot chdir to project root: " + err.Error())
	}

	// GLFW init + window creation must happen on the main thread.
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		panic("glfw.Init: " + err.Error())
	}
	defer glfw.Terminate()

	// Request a core-profile OpenGL 4.1 context (the minimum we target on macOS).
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Visible, glfw.False) // headless – no window appears

	var err error
	glWindow, err = glfw.CreateWindow(1, 1, "bench", nil, nil)
	if err != nil {
		panic("glfw.CreateWindow: " + err.Error())
	}
	glWindow.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		panic("gl.Init: " + err.Error())
	}

	// Block registry must be initialised before any meshing call so that
	// texture IDs and block definitions are available.
	registry.InitRegistry()

	// Release the GL context from the main thread so benchmark goroutines
	// can acquire it via ensureGLContext().
	glfw.DetachCurrentContext()

	os.Exit(m.Run())
}

// ensureGLContext locks the current goroutine to its OS thread and makes the
// GL context current. Must be called at the top of any benchmark that issues
// OpenGL calls.
func ensureGLContext() {
	runtime.LockOSThread()
	glWindow.MakeContextCurrent()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupBenchWorld creates a world and populates a (2*radius+1)² grid of
// chunks around the origin. All chunks are seeded with terrain data so that
// cross-border face visibility works correctly during meshing.
func setupBenchWorld(seed int64, radius int) (*world.World, []*world.Chunk) {
	w := world.New()
	provider := world.NewChunkProvider189(seed)

	var chunks []*world.Chunk
	for x := -radius; x <= radius; x++ {
		for z := -radius; z <= radius; z++ {
			c := w.GetChunk(x, 0, z, true) // allocates and registers the chunk
			provider.PopulateChunk(c)      // fills with terrain noise
			chunks = append(chunks, c)
		}
	}
	return w, chunks
}

// unpackVertices converts the greedy mesher's packed uint32 output into the
// int16 atlas format expected by the GPU shader. Two uint32 values encode one
// vertex; the output is six int16 values per vertex.
//
// Bit layout:
//
//	V1: X(5) | Y(9) | Z(5) | Normal(3) | Brightness(8)
//	V2: TextureID(16) | Tint(16)
//
// Output per vertex: [worldX, worldY, worldZ, info, texID, tint] as int16,
// where info = normal | (brightness << 8).
func unpackVertices(cpuVerts []uint32, baseX, baseY, baseZ int) []int16 {
	count := len(cpuVerts) / 2
	buf := make([]int16, 0, count*6)
	for i := range count {
		v1 := cpuVerts[i*2]
		v2 := cpuVerts[i*2+1]

		lx := int(v1 & 0x1F)
		ly := int((v1 >> 5) & 0x1FF)
		lz := int((v1 >> 14) & 0x1F)
		norm := int((v1 >> 19) & 0x7)
		brightness := int((v1 >> 22) & 0xFF)
		texID := int(v2 & 0xFFFF)
		tint := int((v2 >> 16) & 0xFFFF)

		wx := int16(baseX + lx)
		wy := int16(baseY + ly)
		wz := int16(baseZ + lz)
		info := int16(norm | (brightness << 8))

		buf = append(buf, wx, wy, wz, info, int16(texID), int16(tint))
	}
	return buf
}

// gpuUpload performs a complete VBO upload cycle for the given int16 slice.
// The VBO is deleted after upload so that each iteration is independent.
func gpuUpload(data []int16) {
	if len(data) == 0 {
		return
	}

	capacityBytes := len(data) * 2 // int16 = 2 bytes
	dataBytes := capacityBytes     // upload everything

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	// Allocate GPU memory without uploading data yet (nil pointer).
	gl.BufferData(gl.ARRAY_BUFFER, capacityBytes, nil, gl.DYNAMIC_DRAW)

	ptr := gl.MapBufferRange(gl.ARRAY_BUFFER, 0, dataBytes,
		gl.MAP_WRITE_BIT|gl.MAP_INVALIDATE_BUFFER_BIT)
	if ptr != nil {
		dst := unsafe.Slice((*int16)(ptr), len(data))
		copy(dst, data)
		gl.UnmapBuffer(gl.ARRAY_BUFFER)
	}

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.DeleteBuffers(1, &vbo)
}

// ---------------------------------------------------------------------------
// Stage 1: Chunk Generation
// ---------------------------------------------------------------------------

// BenchmarkChunkGeneration measures the cost of generating terrain for a
// single chunk using the authentic Minecraft 1.8.9 noise pipeline.
func BenchmarkChunkGeneration(b *testing.B) {
	const seed = int64(8675309)
	provider := world.NewChunkProvider189(seed)

	b.ReportAllocs()
	b.ResetTimer()

	i := 0
	for b.Loop() {
		// Use varying coordinates so each iteration exercises different noise
		// values and we avoid any internal cache hits.
		x := i % 64
		z := i / 64 % 64
		c := world.NewChunk(x, 0, z)
		provider.PopulateChunk(c)
		i++
	}
}

// ---------------------------------------------------------------------------
// Stage 2: Greedy Meshing
// ---------------------------------------------------------------------------

// BenchmarkMeshing measures the greedy meshing pass for a single chunk that
// has all six neighbours present in the world so that cross-border visibility
// is fully exercised.
func BenchmarkMeshing(b *testing.B) {
	const seed = int64(8675309)

	// radius=2 guarantees the center chunk (0,0,0) has all direct neighbours.
	w, _ := setupBenchWorld(seed, 2)
	defer w.Close()

	center := w.GetChunk(0, 0, 0, false)
	if center == nil {
		b.Fatal("center chunk not found after world setup")
	}

	// The direction pool is shared across iterations, mirroring production use.
	pool := meshing.NewDirectionWorkerPool(6, 32)
	pool.Start()

	var lastVertCount int

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		verts := meshing.BuildGreedyMeshForChunk(w, center, pool)
		lastVertCount = len(verts) / 2
	}

	b.ReportMetric(float64(lastVertCount), "vertices/op")
}

// ---------------------------------------------------------------------------
// Stage 3: Vertex Unpacking
// ---------------------------------------------------------------------------

// BenchmarkVertexUnpack measures the cost of converting the packed uint32
// greedy-mesh output into the int16 atlas format expected by the GPU shader.
func BenchmarkVertexUnpack(b *testing.B) {
	const seed = int64(8675309)

	w, _ := setupBenchWorld(seed, 2)
	defer w.Close()

	center := w.GetChunk(0, 0, 0, false)
	if center == nil {
		b.Fatal("center chunk not found after world setup")
	}

	pool := meshing.NewDirectionWorkerPool(6, 32)
	pool.Start()

	// Mesh once to produce the packed vertices we will unpack in each iteration.
	cpuVerts := meshing.BuildGreedyMeshForChunk(w, center, pool)
	if len(cpuVerts) == 0 {
		b.Skip("chunk produced no vertices – nothing to unpack")
	}

	vertCount := len(cpuVerts) / 2

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = unpackVertices(cpuVerts, 0, 0, 0)
	}

	b.ReportMetric(float64(vertCount), "vertices/op")
}

// ---------------------------------------------------------------------------
// Stage 4: GPU Upload
// ---------------------------------------------------------------------------

// BenchmarkGPUUpload measures the cost of mapping a VBO, copying int16 vertex
// data, and unmapping it. Allocation and deletion of the VBO are included
// because they happen every time a dirty chunk is re-uploaded.
func BenchmarkGPUUpload(b *testing.B) {
	ensureGLContext()

	const seed = int64(8675309)

	w, _ := setupBenchWorld(seed, 2)
	defer w.Close()

	center := w.GetChunk(0, 0, 0, false)
	if center == nil {
		b.Fatal("center chunk not found after world setup")
	}

	pool := meshing.NewDirectionWorkerPool(6, 32)
	pool.Start()

	cpuVerts := meshing.BuildGreedyMeshForChunk(w, center, pool)
	int16Data := unpackVertices(cpuVerts, 0, 0, 0)
	if len(int16Data) == 0 {
		b.Skip("chunk produced no int16 data – nothing to upload")
	}

	byteCount := len(int16Data) * 2

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		gpuUpload(int16Data)
	}

	b.ReportMetric(float64(byteCount), "bytes/op")
}

// ---------------------------------------------------------------------------
// Stage 5: End-to-End Pipeline
// ---------------------------------------------------------------------------

// BenchmarkPipelineE2E measures the full pipeline (gen → mesh → unpack →
// upload) both as individual sub-benchmarks and as a combined sequential run.
//
// The sub-benchmarks share the same world setup so that neighbour lookups
// during meshing are representative.
func BenchmarkPipelineE2E(b *testing.B) {
	ensureGLContext()

	const seed = int64(8675309)

	w, _ := setupBenchWorld(seed, 2)
	defer w.Close()

	provider := world.NewChunkProvider189(seed)

	pool := meshing.NewDirectionWorkerPool(6, 32)
	pool.Start()

	center := w.GetChunk(0, 0, 0, false)
	if center == nil {
		b.Fatal("center chunk not found after world setup")
	}

	// --- Sub-benchmark: generation only ---
	b.Run("Stage1_ChunkGen", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			x := i%32 + 100 // offset to avoid overwriting the neighbour grid
			z := i / 32 % 32
			c := world.NewChunk(x, 0, z)
			provider.PopulateChunk(c)
			i++
		}
	})

	// --- Sub-benchmark: meshing only ---
	b.Run("Stage2_Meshing", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var last int
		for b.Loop() {
			verts := meshing.BuildGreedyMeshForChunk(w, center, pool)
			last = len(verts) / 2
		}
		b.ReportMetric(float64(last), "vertices/op")
	})

	// --- Sub-benchmark: unpack only ---
	cpuVerts := meshing.BuildGreedyMeshForChunk(w, center, pool)
	b.Run("Stage3_Unpack", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var last int
		for b.Loop() {
			data := unpackVertices(cpuVerts, 0, 0, 0)
			last = len(data) / 6
		}
		b.ReportMetric(float64(last), "vertices/op")
	})

	// --- Sub-benchmark: GPU upload only ---
	int16Data := unpackVertices(cpuVerts, 0, 0, 0)
	b.Run("Stage4_GPUUpload", func(b *testing.B) {
		ensureGLContext()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			gpuUpload(int16Data)
		}
		b.ReportMetric(float64(len(int16Data)*2), "bytes/op")
	})

	// --- Sub-benchmark: all four stages in sequence ---
	// Each iteration generates a fresh chunk at varying coords to prevent any
	// re-use of cached block data from prior iterations.
	b.Run("AllStages", func(b *testing.B) {
		ensureGLContext()
		b.ReportAllocs()
		b.ResetTimer()
		i := 0
		for b.Loop() {
			// Stage 1: generate a fresh chunk and inject it into the world so
			// neighbouring chunks see correct blocks during meshing.
			cx := (i%16 + 50) // stay away from the pre-seeded neighbour region
			cz := (i / 16 % 16) + 50
			c := w.GetChunk(cx, 0, cz, true)
			provider.PopulateChunk(c)

			// Stage 2: mesh (uses neighbouring chunks already in the world)
			cpuV := meshing.BuildGreedyMeshForChunk(w, c, pool)

			// Stage 3: unpack
			int16V := unpackVertices(cpuV, cx*world.ChunkSizeX, 0, cz*world.ChunkSizeZ)

			// Stage 4: GPU upload
			gpuUpload(int16V)
			i++
		}
	})
}

// ---------------------------------------------------------------------------
// Stage 6: Parallel Meshing Scalability
// ---------------------------------------------------------------------------

// BenchmarkMeshingParallel shows how meshing throughput scales with GOMAXPROCS
// when multiple goroutines share a single DirectionWorkerPool.
//
// Run with -cpu 1,2,4,8 to produce a scaling chart:
//
//	go test -bench=BenchmarkMeshingParallel -cpu 1,2,4,8 ./internal/benchmark/
func BenchmarkMeshingParallel(b *testing.B) {
	const seed = int64(8675309)

	w, _ := setupBenchWorld(seed, 3) // radius 3 gives more neighbour combinations
	defer w.Close()

	center := w.GetChunk(0, 0, 0, false)
	if center == nil {
		b.Fatal("center chunk not found after world setup")
	}

	// Scale the pool's direction workers with the CPU count so it does not
	// become the bottleneck when testing concurrency.
	workerCount := runtime.GOMAXPROCS(0) * 6
	pool := meshing.NewDirectionWorkerPool(workerCount, workerCount*8)
	pool.Start()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// All goroutines mesh the same center chunk; this is intentional –
			// it maximises contention on the direction pool's job queue and
			// reflects the production scenario where many dirty chunks compete
			// for the same pool.
			verts := meshing.BuildGreedyMeshForChunk(w, center, pool)
			_ = verts
		}
	})
}
