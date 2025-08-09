package world

import (
	"github.com/go-gl/mathgl/mgl32"
)

const (
	WorldSizeX   = 17
	WorldSizeY   = 17
	WorldSizeZ   = 17
	WorldOffsetX = 8
	WorldOffsetY = 0
	WorldOffsetZ = 8
)

type World struct {
	blocks [WorldSizeX][WorldSizeY][WorldSizeZ]bool
}

func New() *World {
	w := &World{}

	// Initialize a flat world
	for x := -8; x <= 8; x++ {
		for z := -8; z <= 8; z++ {
			w.Set(x, 0, z, true)
		}
	}

	return w
}

func (w *World) Get(x, y, z int) bool {
	ix, iy, iz := w.toIndex(x, y, z)
	if ix < 0 || ix >= WorldSizeX || iy < 0 || iy >= WorldSizeY || iz < 0 || iz >= WorldSizeZ {
		return false
	}
	return w.blocks[ix][iy][iz]
}

func (w *World) Set(x, y, z int, val bool) {
	ix, iy, iz := w.toIndex(x, y, z)
	if ix < 0 || ix >= WorldSizeX || iy < 0 || iy >= WorldSizeY || iz < 0 || iz >= WorldSizeZ {
		return
	}
	w.blocks[ix][iy][iz] = val
}

func (w *World) toIndex(x, y, z int) (int, int, int) {
	return x + WorldOffsetX, y + WorldOffsetY, z + WorldOffsetZ
}

func (w *World) fromIndex(ix, iy, iz int) (int, int, int) {
	return ix - WorldOffsetX, iy - WorldOffsetY, iz - WorldOffsetZ
}

func (w *World) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3
	for ix := 0; ix < WorldSizeX; ix++ {
		for iy := 0; iy < WorldSizeY; iy++ {
			for iz := 0; iz < WorldSizeZ; iz++ {
				if w.blocks[ix][iy][iz] {
					x, y, z := w.fromIndex(ix, iy, iz)
					positions = append(positions, mgl32.Vec3{float32(x), float32(y), float32(z)})
				}
			}
		}
	}
	return positions
}
