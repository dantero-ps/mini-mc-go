package config

import "sync"

// RenderSettings holds render configuration
type RenderSettings struct {
	mu             sync.RWMutex
	renderDistance int  // in chunks
	fpsLimit       int  // 0 means uncapped, otherwise target FPS
	wireframeMode  bool // wireframe rendering mode
}

var globalRenderSettings = &RenderSettings{
	renderDistance: 25,  // default value
	fpsLimit:       180, // default FPS cap
	wireframeMode:  false,
}

// GetRenderDistance returns the current render distance in chunks
func GetRenderDistance() int {
	globalRenderSettings.mu.RLock()
	defer globalRenderSettings.mu.RUnlock()
	return globalRenderSettings.renderDistance
}

// SetRenderDistance sets the render distance in chunks
func SetRenderDistance(distance int) {
	globalRenderSettings.mu.Lock()
	defer globalRenderSettings.mu.Unlock()

	// Clamp to reasonable values
	if distance < 5 {
		distance = 5
	}
	if distance > 50 {
		distance = 50
	}

	globalRenderSettings.renderDistance = distance
}

// GetFPSLimit returns the configured FPS cap (0 means uncapped)
func GetFPSLimit() int {
	globalRenderSettings.mu.RLock()
	defer globalRenderSettings.mu.RUnlock()
	return globalRenderSettings.fpsLimit
}

// SetFPSLimit sets the FPS cap; 0 disables the cap (uncapped)
func SetFPSLimit(limit int) {
	globalRenderSettings.mu.Lock()
	defer globalRenderSettings.mu.Unlock()
	if limit < 0 {
		limit = 0
	}
	if limit > 240 {
		limit = 240
	}
	globalRenderSettings.fpsLimit = limit
}

// GetChunkLoadRadius returns radius for chunk loading (slightly larger than render distance)
func GetChunkLoadRadius() int {
	return GetRenderDistance()
}

// GetChunkEvictRadius returns radius for chunk eviction (larger than load radius)
func GetChunkEvictRadius() int {
	return GetRenderDistance() * 2
}

// GetMaxRenderRadius returns maximum render radius for pre-culling
func GetMaxRenderRadius() int {
	rd := GetRenderDistance()
	return rd
}

// GetWireframeMode returns whether wireframe mode is enabled
func GetWireframeMode() bool {
	globalRenderSettings.mu.RLock()
	defer globalRenderSettings.mu.RUnlock()
	return globalRenderSettings.wireframeMode
}

// SetWireframeMode sets the wireframe mode
func SetWireframeMode(enabled bool) {
	globalRenderSettings.mu.Lock()
	defer globalRenderSettings.mu.Unlock()
	globalRenderSettings.wireframeMode = enabled
}

// ToggleWireframeMode toggles wireframe mode
func ToggleWireframeMode() {
	globalRenderSettings.mu.Lock()
	defer globalRenderSettings.mu.Unlock()
	globalRenderSettings.wireframeMode = !globalRenderSettings.wireframeMode
}
