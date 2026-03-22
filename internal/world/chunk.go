package world

import (
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	// Chunk dimensions
	ChunkSizeX = 16
	ChunkSizeY = 256
	ChunkSizeZ = 16

	// Section dimensions
	SectionHeight = 16
	NumSections   = ChunkSizeY / SectionHeight
	SectionVolume = ChunkSizeX * SectionHeight * ChunkSizeZ
)

// Section represents a 16x16x16 sub-volume of a chunk
type Section struct {
	blocks   []BlockType
	basePtr  unsafe.Pointer // &blocks[0] tutuluyor (nil slice durumunda nil kalır)
	metadata []uint8
	metaPtr  unsafe.Pointer // &metadata[0] tutuluyor; nil → tüm metadata sıfır (kaynak su gibi)
}

// Chunk represents a 16x256x16 section of the world
type Chunk struct {
	X, Y, Z    int
	sections   [NumSections]*Section
	dirty      bool
	generation uint64 // incremented on each block change; used to detect stale mesh jobs
}

// Generation returns the current generation counter.
func (c *Chunk) Generation() uint64 {
	return c.generation
}

// NewChunk creates a new chunk at the specified chunk coordinates
func NewChunk(x, y, z int) *Chunk {
	return &Chunk{
		X:     x,
		Y:     y,
		Z:     z,
		dirty: true,
	}
}

// indexInSection converts local section coordinates (x, localY, z) → flat index
func indexInSection(x, localY, z int) int {
	return x*SectionHeight*ChunkSizeZ + localY*ChunkSizeZ + z
}

// GetBlock returns the block type at the specified local coordinates
func (c *Chunk) GetBlock(x, y, z int) BlockType {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return BlockTypeAir
	}

	secIdx := y / SectionHeight
	sec := c.sections[secIdx]
	if sec == nil || sec.basePtr == nil {
		return BlockTypeAir
	}

	localY := y % SectionHeight
	idx := indexInSection(x, localY, z)

	blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)*unsafe.Sizeof(BlockType(0))))
	return *blockPtr
}

// SetBlock sets the block type at the specified local coordinates
func (c *Chunk) SetBlock(x, y, z int, blockType BlockType) {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return
	}

	secIdx := y / SectionHeight
	localY := y % SectionHeight
	idx := indexInSection(x, localY, z)

	sec := c.sections[secIdx]

	if blockType == BlockTypeAir {
		if sec != nil && sec.basePtr != nil {
			blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)*unsafe.Sizeof(BlockType(0))))
			old := *blockPtr

			if old != BlockTypeAir {
				*blockPtr = BlockTypeAir
				c.dirty = true
				c.generation++

				// Blok air yapılırken o pozisyondaki metadata'yı da temizle
				if sec.metaPtr != nil {
					metaPtr := (*uint8)(unsafe.Pointer(uintptr(sec.metaPtr) + uintptr(idx)))
					*metaPtr = 0
					// Tüm metadata sıfır olduysa diziyi serbest bırak
					allZero := true
					for _, v := range sec.metadata {
						if v != 0 {
							allZero = false
							break
						}
					}
					if allZero {
						sec.metadata = nil
						sec.metaPtr = nil
					}
				}

				if len(sec.blocks) <= 0 {
					sec.blocks = nil
					sec.basePtr = nil
					sec.metadata = nil
					sec.metaPtr = nil
					c.sections[secIdx] = nil
				}
			}
		}
		return
	}

	// non-air blok → section yoksa oluştur
	if sec == nil {
		sec = &Section{}
		c.sections[secIdx] = sec
	}

	if sec.blocks == nil {
		sec.blocks = make([]BlockType, SectionVolume)
		sec.basePtr = unsafe.Pointer(&sec.blocks[0])
	}

	blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)*unsafe.Sizeof(BlockType(0))))
	old := *blockPtr

	if old != blockType {
		*blockPtr = blockType
		c.dirty = true
		c.generation++
	}
}

// GetMeta returns metadata at local coordinates.
// Metadata dizisi nil ise 0 döner — kaynak su gibi tüm okyanuslar için sıfır tahsisat.
func (c *Chunk) GetMeta(x, y, z int) uint8 {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return 0
	}

	secIdx := y / SectionHeight
	sec := c.sections[secIdx]
	if sec == nil || sec.metaPtr == nil {
		return 0
	}

	localY := y % SectionHeight
	idx := indexInSection(x, localY, z)

	metaPtr := (*uint8)(unsafe.Pointer(uintptr(sec.metaPtr) + uintptr(idx)))
	return *metaPtr
}

// SetMeta sets metadata at local coordinates.
// Değer sıfırsa ve metadata dizisi henüz yoksa tahsisat YAPILMAZ (kaynak su için kritik optimizasyon).
// Değer sıfır dışıysa metadata dizisi lazy olarak tahsis edilir.
// Bölümdeki tüm metadata sıfır olduğunda dizi serbest bırakılır.
func (c *Chunk) SetMeta(x, y, z int, meta uint8) {
	if x < 0 || x >= ChunkSizeX || y < 0 || y >= ChunkSizeY || z < 0 || z >= ChunkSizeZ {
		return
	}

	secIdx := y / SectionHeight
	localY := y % SectionHeight
	idx := indexInSection(x, localY, z)

	sec := c.sections[secIdx]

	if meta == 0 {
		// Sıfır yazarken: dizi yoksa hiçbir şey yapma (tahsisat önlenir)
		if sec == nil || sec.metaPtr == nil {
			return
		}
		metaPtr := (*uint8)(unsafe.Pointer(uintptr(sec.metaPtr) + uintptr(idx)))
		*metaPtr = 0

		// Tüm metadata sıfır olduysa diziyi serbest bırak
		allZero := true
		for _, v := range sec.metadata {
			if v != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			sec.metadata = nil
			sec.metaPtr = nil
		}
		c.dirty = true
		return
	}

	// Sıfır dışı değer: gerekirse section ve metadata dizisini oluştur
	if sec == nil {
		sec = &Section{}
		c.sections[secIdx] = sec
	}
	if sec.metadata == nil {
		sec.metadata = make([]uint8, SectionVolume)
		sec.metaPtr = unsafe.Pointer(&sec.metadata[0])
	}

	metaPtr := (*uint8)(unsafe.Pointer(uintptr(sec.metaPtr) + uintptr(idx)))
	*metaPtr = meta
	c.dirty = true
}

// SetBlockFast sets block without bounds checking. Caller must ensure valid coordinates.
// For use during initial chunk generation only — skips dirty flag, bounds check, and old-value comparison.
func (c *Chunk) SetBlockFast(x, y, z int, blockType BlockType) {
	secIdx := y >> 4 // y / 16
	sec := c.sections[secIdx]

	if blockType == BlockTypeAir {
		if sec != nil && sec.basePtr != nil {
			localY := y & 0xF // y % 16
			idx := x*SectionHeight*ChunkSizeZ + localY*ChunkSizeZ + z
			blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)))
			*blockPtr = BlockTypeAir
		}
		return
	}

	if sec == nil {
		sec = &Section{}
		c.sections[secIdx] = sec
	}
	if sec.blocks == nil {
		sec.blocks = make([]BlockType, SectionVolume)
		sec.basePtr = unsafe.Pointer(&sec.blocks[0])
	}

	localY := y & 0xF
	idx := x*SectionHeight*ChunkSizeZ + localY*ChunkSizeZ + z
	blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)))
	*blockPtr = blockType
}

// IsSectionEmpty returns true if the section at the given Y index has no blocks allocated.
func (c *Chunk) IsSectionEmpty(sectionIdx int) bool {
	if sectionIdx < 0 || sectionIdx >= NumSections {
		return true
	}
	sec := c.sections[sectionIdx]
	return sec == nil || sec.basePtr == nil
}

// IsAir checks if the block at the specified local coordinates is air
func (c *Chunk) IsAir(x, y, z int) bool {
	return c.GetBlock(x, y, z) == BlockTypeAir
}

// IsDirty returns whether the chunk has been modified since last render
func (c *Chunk) IsDirty() bool {
	return c.dirty
}

// SetClean marks the chunk as clean (not modified)
func (c *Chunk) SetClean() {
	c.dirty = false
}

// GetActiveBlocks returns world-space positions of non-air blocks
func (c *Chunk) GetActiveBlocks() []mgl32.Vec3 {
	var positions []mgl32.Vec3

	worldOffsetX := float32(c.X * ChunkSizeX)
	worldOffsetY := float32(c.Y * ChunkSizeY)
	worldOffsetZ := float32(c.Z * ChunkSizeZ)

	for secIdx := range NumSections {
		sec := c.sections[secIdx]
		if sec == nil || sec.basePtr == nil {
			continue
		}

		sectionBaseY := secIdx * SectionHeight
		sizeof := unsafe.Sizeof(BlockType(0))

		for lx := range ChunkSizeX {
			for ly := range SectionHeight {
				for lz := range ChunkSizeZ {
					idx := indexInSection(lx, ly, lz)
					blockPtr := (*BlockType)(unsafe.Pointer(uintptr(sec.basePtr) + uintptr(idx)*sizeof))

					if *blockPtr != BlockTypeAir {
						wx := worldOffsetX + float32(lx)
						wy := worldOffsetY + float32(sectionBaseY+ly)
						wz := worldOffsetZ + float32(lz)
						positions = append(positions, mgl32.Vec3{wx, wy, wz})
					}
				}
			}
		}
	}

	return positions
}
