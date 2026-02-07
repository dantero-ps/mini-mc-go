package hud

import (
	"fmt"
	"math"
	"strings"
	"time"

	"mini-mc/internal/player"
	"mini-mc/internal/profiling"

	"github.com/go-gl/mathgl/mgl32"
)

type ProfilingStats struct {
	frameStartTime    time.Time
	frameEndTime      time.Time
	frameDuration     time.Duration
	lastFrameDuration time.Duration

	totalDrawCalls int
	totalVertices  int
	totalTriangles int
	visibleChunks  int
	culledChunks   int
	meshedChunks   int

	gpuMemoryUsage     int64
	textureMemoryUsage int64
	bufferMemoryUsage  int64

	frustumCullTime time.Duration
	meshingTime     time.Duration
	shaderBindTime  time.Duration
	vaoBindTime     time.Duration

	frameTimeHistory []time.Duration
	maxFrameTime     time.Duration
	minFrameTime     time.Duration
	avgFrameTime     time.Duration

	lastTotalFrameDuration time.Duration
	lastUpdateDuration     time.Duration

	lastPlayerDuration  time.Duration
	lastWorldDuration   time.Duration
	lastGlfwDuration    time.Duration
	lastHudDuration     time.Duration
	lastPruneDuration   time.Duration
	lastOtherDuration   time.Duration
	lastPhysicsDuration time.Duration

	lastPreRenderDuration  time.Duration
	lastSwapEventsDuration time.Duration
}

// Profiling methods for external updates
func (h *HUD) ProfilingSetLastTotalFrameDuration(d time.Duration) {
	h.profilingStats.lastTotalFrameDuration = d
}

func (h *HUD) ProfilingSetLastUpdateDuration(d time.Duration) {
	h.profilingStats.lastUpdateDuration = d
}

func (h *HUD) ProfilingSetBreakdown(player, world, glfw, hud, prune time.Duration) {
	h.profilingStats.lastPlayerDuration = player
	h.profilingStats.lastWorldDuration = world
	h.profilingStats.lastGlfwDuration = glfw
	h.profilingStats.lastHudDuration = hud
	h.profilingStats.lastPruneDuration = prune
}

func (h *HUD) ProfilingSetPhysics(physics time.Duration) {
	h.profilingStats.lastPhysicsDuration = physics
}

func (h *HUD) ProfilingSetPhases(preRender, swapEvents time.Duration) {
	h.profilingStats.lastPreRenderDuration = preRender
	h.profilingStats.lastSwapEventsDuration = swapEvents
}

// ProfilingSetRenderDuration stores the render() call duration for this frame
func (h *HUD) ProfilingSetRenderDuration(d time.Duration) {
	h.profilingStats.frameDuration = d
	h.profilingStats.lastFrameDuration = d
	// update rolling history and stats
	if len(h.profilingStats.frameTimeHistory) >= 60 {
		h.profilingStats.frameTimeHistory = h.profilingStats.frameTimeHistory[1:]
	}
	h.profilingStats.frameTimeHistory = append(h.profilingStats.frameTimeHistory, d)
	// recompute min/max/avg
	var total time.Duration
	min := d
	max := d
	for _, v := range h.profilingStats.frameTimeHistory {
		total += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	h.profilingStats.avgFrameTime = 0
	if len(h.profilingStats.frameTimeHistory) > 0 {
		h.profilingStats.avgFrameTime = total / time.Duration(len(h.profilingStats.frameTimeHistory))
	}
	h.profilingStats.minFrameTime = min
	h.profilingStats.maxFrameTime = max
}

func (h *HUD) renderPlayerPosition(p *player.Player) {
	// Build text and draw at top-left
	// Calculate horiz speed (m/s)
	speed := math.Sqrt(float64(p.Velocity[0]*p.Velocity[0] + p.Velocity[2]*p.Velocity[2]))
	text := fmt.Sprintf("Pos: %.2f, %.2f, %.2f | Speed: %.2f m/s | MaxJump: %.3f", p.Position[0], p.Position[1], p.Position[2], speed, p.MaxJumpHeight)
	x := float32(10)
	y := float32(30)
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.fontRenderer.Render(text, x, y, 0.35, color)
}

// renderFPS renders the current FPS value on screen
func (h *HUD) renderFPS() {
	text := fmt.Sprintf("FPS: %d", h.currentFPS)
	x := float32(10)
	y := float32(46)
	color := mgl32.Vec3{1.0, 1.0, 1.0}
	h.fontRenderer.Render(text, x, y, 0.3, color)
}

// RenderProfilingInfo renders the current profiling information on screen
func (h *HUD) RenderProfilingInfo() {
	lines := make([]string, 0, 64)

	// Frame timing
	tracked := profiling.SumWithPrefix("renderer.")
	frameMs := float64(h.profilingStats.frameDuration.Microseconds()) / 1000.0
	trackedMs := float64(tracked.Microseconds()) / 1000.0
	avgMs := float64(h.profilingStats.avgFrameTime.Microseconds()) / 1000.0
	lines = append(lines, fmt.Sprintf("Frame(render): %.2fms (%.2fms avg) | Tracked(render): %.2fms", frameMs, avgMs, trackedMs))

	// Update from main loop
	if h.profilingStats.lastUpdateDuration > 0 {
		updateMs := float64(h.profilingStats.lastUpdateDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Frame(update): %.2fms", updateMs))
	}

	// Previous frame total and overhead
	if h.profilingStats.lastTotalFrameDuration > 0 {
		totalMs := float64(h.profilingStats.lastTotalFrameDuration.Microseconds()) / 1000.0
		overheadMs := totalMs - frameMs
		if overheadMs < 0 {
			overheadMs = 0
		}
		lines = append(lines, fmt.Sprintf("Frame(total): %.2fms | Overhead(non-render): %.2fms", totalMs, overheadMs))

		// Overhead breakdown
		playerMs := float64(h.profilingStats.lastPlayerDuration.Microseconds()) / 1000.0
		worldMs := float64(h.profilingStats.lastWorldDuration.Microseconds()) / 1000.0
		glfwMs := float64(h.profilingStats.lastGlfwDuration.Microseconds()) / 1000.0
		hudMs := float64(h.profilingStats.lastHudDuration.Microseconds()) / 1000.0
		pruneMs := float64(h.profilingStats.lastPruneDuration.Microseconds()) / 1000.0
		physicsMs := float64(h.profilingStats.lastPhysicsDuration.Microseconds()) / 1000.0
		otherMs := overheadMs - (playerMs + worldMs + glfwMs + hudMs + pruneMs + physicsMs)
		if otherMs < 0 {
			otherMs = 0
		}
		lines = append(lines, fmt.Sprintf("Overhead breakdown -> player: %.2fms, world: %.2fms, glfw: %.2fms, hud: %.2fms, prune: %.2fms, physics: %.2fms, other: %.2fms", playerMs, worldMs, glfwMs, hudMs, pruneMs, physicsMs, otherMs))

		// Phase durations (prev)
		preRenderMs := float64(h.profilingStats.lastPreRenderDuration.Microseconds()) / 1000.0
		swapEventsMs := float64(h.profilingStats.lastSwapEventsDuration.Microseconds()) / 1000.0
		lines = append(lines, fmt.Sprintf("Phases -> preRender: %.2fms, swap+events: %.2fms", preRenderMs, swapEventsMs))
	}

	// Renderer breakdown from profiling trackers
	frustumMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.frustumSetup").Microseconds()) / 1000.0
	collectMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.collectVisible").Microseconds()) / 1000.0
	ensureMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.ensureMeshes").Microseconds()) / 1000.0
	drawMs := float64(profiling.SumWithPrefix("renderer.renderBlocks.drawAtlas").Microseconds()) / 1000.0
	highlightMs := float64(profiling.SumWithPrefix("renderer.renderHighlightedBlock").Microseconds()) / 1000.0
	handMs := float64(profiling.SumWithPrefix("renderer.renderHand").Microseconds()) / 1000.0
	crossMs := float64(profiling.SumWithPrefix("renderer.renderCrosshair").Microseconds()) / 1000.0
	dirMs := float64(profiling.SumWithPrefix("renderer.renderDirection").Microseconds()) / 1000.0
	if frustumMs+collectMs+ensureMs+drawMs+highlightMs+handMs+crossMs+dirMs > 0 {
		lines = append(lines, fmt.Sprintf("Blocks -> frustum: %.2fms, collect: %.2fms, ensure: %.2fms, draw: %.2fms", frustumMs, collectMs, ensureMs, drawMs))
		lines = append(lines, fmt.Sprintf("Overlays -> highlight: %.2fms, hand: %.2fms, crosshair: %.2fms, direction: %.2fms", highlightMs, handMs, crossMs, dirMs))
	}

	// Top N tracked lines
	if top := profiling.TopN(10); top != "" {
		for line := range strings.SplitSeq(top, ", ") {
			if line != "" && !strings.Contains(line, ":0ms") {
				lines = append(lines, line)
			}
		}
	}

	textColor := mgl32.Vec3{1.0, 1.0, 1.0}
	startY := float32(60)
	lineStep := float32(17)
	h.fontRenderer.RenderLines(lines, 10, startY, lineStep, 0.375, textColor)
}

// Helper methods for profiling data management
func (h *HUD) startFrameProfiling() {
	h.profilingStats.frameStartTime = time.Now()
}

func (h *HUD) endFrameProfiling() {
	h.profilingStats.frameEndTime = time.Now()
	h.profilingStats.frameDuration = h.profilingStats.frameEndTime.Sub(h.profilingStats.frameStartTime)

	// Update frame time history (keep last 60 frames for averaging)
	if len(h.profilingStats.frameTimeHistory) >= 60 {
		h.profilingStats.frameTimeHistory = h.profilingStats.frameTimeHistory[1:]
	}
	h.profilingStats.frameTimeHistory = append(h.profilingStats.frameTimeHistory, h.profilingStats.frameDuration)

	// Calculate statistics
	h.calculateFrameTimeStats()

	// Update GPU memory usage
	h.updateGPUMemoryUsage()

	// Store last frame duration for next frame
	h.profilingStats.lastFrameDuration = h.profilingStats.frameDuration
}

func (h *HUD) resetFrameCounters() {
	h.profilingStats.totalDrawCalls = 0
	h.profilingStats.totalVertices = 0
	h.profilingStats.totalTriangles = 0
	h.profilingStats.visibleChunks = 0
	h.profilingStats.culledChunks = 0
	h.profilingStats.meshedChunks = 0
	h.profilingStats.frustumCullTime = 0
	h.profilingStats.meshingTime = 0
	h.profilingStats.shaderBindTime = 0
	h.profilingStats.vaoBindTime = 0
}

func (h *HUD) calculateFrameTimeStats() {
	if len(h.profilingStats.frameTimeHistory) == 0 {
		return
	}

	var total time.Duration
	h.profilingStats.maxFrameTime = 0
	h.profilingStats.minFrameTime = h.profilingStats.frameTimeHistory[0]

	for _, duration := range h.profilingStats.frameTimeHistory {
		total += duration
		if duration > h.profilingStats.maxFrameTime {
			h.profilingStats.maxFrameTime = duration
		}
		if duration < h.profilingStats.minFrameTime {
			h.profilingStats.minFrameTime = duration
		}
	}

	h.profilingStats.avgFrameTime = total / time.Duration(len(h.profilingStats.frameTimeHistory))
}

func (h *HUD) updateGPUMemoryUsage() {
	h.profilingStats.bufferMemoryUsage = h.estimateBufferMemoryUsage()
	h.profilingStats.textureMemoryUsage = h.estimateTextureMemoryUsage()
	h.profilingStats.gpuMemoryUsage = h.profilingStats.bufferMemoryUsage + h.profilingStats.textureMemoryUsage
}

func (h *HUD) estimateBufferMemoryUsage() int64 {
	// This is a simplified estimation - in a real implementation,
	// you'd track actual buffer allocations
	return 0
}

func (h *HUD) estimateTextureMemoryUsage() int64 {
	return int64(h.fontAtlas.AtlasW * h.fontAtlas.AtlasH * 4) // RGBA
}

// ToggleProfiling toggles profiling HUD visibility
func (h *HUD) ToggleProfiling() {
	h.showProfiling = !h.showProfiling
}

// ShowProfiling returns whether profiling is enabled
func (h *HUD) ShowProfiling() bool {
	return h.showProfiling
}
