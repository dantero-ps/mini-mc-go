package config

import "sync"

// WorldGenSettings holds world generation configuration
type WorldGenSettings struct {
	mu              sync.RWMutex
	useAuthenticGen bool
	seaLevel        int
	caves           bool
}

var globalWorldGenSettings = &WorldGenSettings{
	useAuthenticGen: false, // Default to existing generator
	seaLevel:        63,    // Standard sea level
	caves:           true,  // Caves enabled by default
}

// GetUseAuthenticGen returns whether to use the authentic 1.8.9 generator
func GetUseAuthenticGen() bool {
	globalWorldGenSettings.mu.RLock()
	defer globalWorldGenSettings.mu.RUnlock()
	return globalWorldGenSettings.useAuthenticGen
}

// SetUseAuthenticGen sets the generator type
func SetUseAuthenticGen(enabled bool) {
	globalWorldGenSettings.mu.Lock()
	defer globalWorldGenSettings.mu.Unlock()
	globalWorldGenSettings.useAuthenticGen = enabled
}

// GetSeaLevel returns the configured sea level
func GetSeaLevel() int {
	globalWorldGenSettings.mu.RLock()
	defer globalWorldGenSettings.mu.RUnlock()
	return globalWorldGenSettings.seaLevel
}

// SetSeaLevel sets the sea level
func SetSeaLevel(level int) {
	globalWorldGenSettings.mu.Lock()
	defer globalWorldGenSettings.mu.Unlock()
	globalWorldGenSettings.seaLevel = level
}

// GetCaves returns whether caves are enabled
func GetCaves() bool {
	globalWorldGenSettings.mu.RLock()
	defer globalWorldGenSettings.mu.RUnlock()
	return globalWorldGenSettings.caves
}

// SetCaves sets whether caves are enabled
func SetCaves(enabled bool) {
	globalWorldGenSettings.mu.Lock()
	defer globalWorldGenSettings.mu.Unlock()
	globalWorldGenSettings.caves = enabled
}
