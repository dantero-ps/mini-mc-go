package game

import (
	"mini-mc/internal/entity"
	"mini-mc/internal/world"
)

func init() {
	// Set up the ItemEntity configurator to inject the GetNearbyItems callback
	// This avoids circular imports by setting the callback at runtime
	world.ItemEntityConfigurator = func(item world.Ticker, w interface{}) {
		// Type assert to ItemEntity
		itemEnt, ok := item.(*entity.ItemEntity)
		if !ok {
			return
		}

		// Type assert world
		worldPtr, ok := w.(*world.World)
		if !ok {
			return
		}

		// Set the GetNearbyItems callback function
		itemEnt.GetNearbyItems = func(cx, cy, cz, rx, ry, rz float32) []interface{} {
			return worldPtr.GetNearbyEntities(cx, cy, cz, rx, ry, rz)
		}
	}
}
