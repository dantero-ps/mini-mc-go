package world

import (
	"mini-mc/internal/profiling"
	"sync"
)

// EntityManager handles the lifecycle and updates of entities in the world.
type EntityManager struct {
	entities []Ticker
	mu       sync.RWMutex
}

// NewEntityManager creates a new entity manager.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities: make([]Ticker, 0),
	}
}

// Add adds an entity to the manager.
func (em *EntityManager) Add(e Ticker) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.entities = append(em.entities, e)
}

// Update updates all entities and removes dead ones.
func (em *EntityManager) Update(dt float64) {
	defer profiling.Track("world.UpdateEntities")()

	// First, get a copy of entities to update (holding lock briefly)
	em.mu.RLock()
	entitiesToUpdate := make([]Ticker, len(em.entities))
	copy(entitiesToUpdate, em.entities)
	em.mu.RUnlock()

	// Update all entities WITHOUT holding the lock
	// This prevents deadlock when ItemEntity.Update() calls GetEntitiesInAABB()
	for _, e := range entitiesToUpdate {
		if !e.IsDead() {
			e.Update(dt)
		}
	}

	// Now compact the slice to remove dead entities (holding write lock)
	em.mu.Lock()
	defer em.mu.Unlock()

	activeCount := 0
	for i := 0; i < len(em.entities); i++ {
		e := em.entities[i]
		if !e.IsDead() {
			em.entities[activeCount] = e
			activeCount++
		}
	}
	em.entities = em.entities[:activeCount]
}

// GetAll returns a safe copy of the entities slice.
func (em *EntityManager) GetAll() []Ticker {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// Create a copy to avoid race conditions if the caller iterates
	// while we modify the internal slice
	result := make([]Ticker, len(em.entities))
	copy(result, em.entities)
	return result
}

// GetEntitiesInAABB returns all entities within the given axis-aligned bounding box.
// Used for item stacking logic to find nearby items.
func (em *EntityManager) GetEntitiesInAABB(minX, minY, minZ, maxX, maxY, maxZ float32) []Ticker {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var result []Ticker
	for _, e := range em.entities {
		if e.IsDead() {
			continue
		}
		pos := e.Position()
		// Check if entity's center is within the AABB
		if pos.X() >= minX && pos.X() <= maxX &&
			pos.Y() >= minY && pos.Y() <= maxY &&
			pos.Z() >= minZ && pos.Z() <= maxZ {
			result = append(result, e)
		}
	}
	return result
}
