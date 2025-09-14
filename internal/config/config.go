package config

import "sync"

// RenderSettings holds render configuration
type RenderSettings struct {
	mu             sync.RWMutex
	renderDistance int // in chunks
}

var globalRenderSettings = &RenderSettings{
	renderDistance: 25, // default value
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
