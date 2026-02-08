package graphics

import (
	"sync"
)

var (
	textureCache = make(map[string]uint32)
	cacheMutex   sync.RWMutex
)

// GetTexture returns a cached texture ID for the given path.
// If the texture is already loaded, it returns the cached ID.
// Otherwise, it loads the texture from disk and caches it.
func GetTexture(path string) (uint32, error) {
	cacheMutex.RLock()
	if tex, ok := textureCache[path]; ok {
		cacheMutex.RUnlock()
		return tex, nil
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Double check locking
	if tex, ok := textureCache[path]; ok {
		return tex, nil
	}

	tex, _, _, err := LoadTexture(path)
	if err != nil {
		return 0, err
	}

	textureCache[path] = tex
	return tex, nil
}
