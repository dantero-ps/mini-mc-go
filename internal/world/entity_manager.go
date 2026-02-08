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
	em.mu.Lock()
	defer em.mu.Unlock()

	activeCount := 0
	for i := 0; i < len(em.entities); i++ {
		e := em.entities[i]
		if !e.IsDead() {
			e.Update(dt)
			if !e.IsDead() {
				// Keep alive
				em.entities[activeCount] = e
				activeCount++
			}
		}
	}
	// Trim slice to remove dead entities
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
