package world

const (
	WaterTickRate      = 5
	LavaTickRate       = 30
	WaterSpreadCost    = 1
	LavaSpreadCost     = 2 // In overworld. Nether would be 1.
	MaxFlowSearchDepth = 4
)

// 4 horizontal directions: +X, -X, +Z, -Z
var horizontalDirs = [4][3]int{
	{1, 0, 0},  // East (+X)
	{-1, 0, 0}, // West (-X)
	{0, 0, 1},  // South (+Z)
	{0, 0, -1}, // North (-Z)
}

// oppositeDir returns the index of the opposite horizontal direction.
// 0<->1 (East<->West), 2<->3 (South<->North)
func oppositeDir(dirIdx int) int {
	return dirIdx ^ 1
}

// GetLiquidHeightPercent returns the fractional drop amount for rendering.
// MC formula: (meta+1)/9.0 gives how much of the block is "empty" from the top.
// Renderer uses: fluid height = 1.0 - GetLiquidHeightPercent(level)
// Level 0 (source) → 0.111 drop → 0.889 height
// Level 7 → 0.889 drop → 0.111 height
func GetLiquidHeightPercent(meta int) float32 {
	if meta >= 8 {
		meta = 0
	}
	return float32(meta+1) / 9.0
}

// getFluidLevel returns the level (0-15) of a fluid block at (x,y,z).
// Returns -1 if the block is not the specified fluid type.
// Mirrors Java's BlockLiquid.getLevel()
func getFluidLevel(w *World, x, y, z int, fluidType BlockType) int {
	bt := w.Get(x, y, z)
	if bt != fluidType {
		return -1
	}
	return int(w.GetMeta(x, y, z))
}

// isFluidBlocked checks if a block position blocks fluid flow.
// Uses BlockSolidTable (populated by registry after init) to avoid import cycle.
func isFluidBlocked(w *World, x, y, z int) bool {
	bt := w.Get(x, y, z)
	if bt == BlockTypeAir {
		return false
	}
	return BlockSolidTable[bt]
}

// canFluidFlowInto checks if fluid can flow into position (x,y,z).
// MC: material != this fluid AND material != other fluid AND !isBlocked
func canFluidFlowInto(w *World, x, y, z int, fluidType BlockType) bool {
	bt := w.Get(x, y, z)
	if bt == fluidType {
		return false // Can't flow into same fluid type
	}
	if BlockFluidTable[bt] {
		return false // Can't flow into any other fluid type directly (mixing is separate)
	}
	return !isFluidBlocked(w, x, y, z)
}

// checkAdjacentBlock is MC's checkAdjacentBlock.
// Returns updated minimum adjacent level. Increments adjacentSources if a source is found.
func checkAdjacentBlock(w *World, x, y, z int, currentMin int, adjacentSources *int, fluidType BlockType) int {
	level := getFluidLevel(w, x, y, z, fluidType)
	if level < 0 {
		return currentMin
	}
	if level == 0 {
		*adjacentSources++
	}
	if level >= 8 {
		level = 0
	}
	// -100 is the sentinel "no valid neighbor seen yet"
	if currentMin != -100 && level >= currentMin {
		return currentMin
	}
	return level
}

// flowCost is MC's func_176374_a - recursive flow cost search.
// Finds the shortest distance to a drop-off (a hole where water can fall) within
// MaxFlowSearchDepth blocks. excludeDir is the direction index we came FROM to
// avoid backtracking.
func flowCost(w *World, x, y, z int, distance int, excludeDir int, fluidType BlockType) int {
	cost := 1000

	for dirIdx, dir := range horizontalDirs {
		if dirIdx == excludeDir {
			continue
		}
		nx, ny, nz := x+dir[0], y+dir[1], z+dir[2]

		// If neighbor is blocked or is a same-fluid source, don't search through it
		if isFluidBlocked(w, nx, ny, nz) {
			continue
		}
		bt := w.Get(nx, ny, nz)
		if bt == fluidType && int(w.GetMeta(nx, ny, nz)) == 0 {
			continue
		}

		// Check if there is a drop-off below the neighbor
		if !isFluidBlocked(w, nx, ny-1, nz) {
			return distance // Found a drop-off at this distance
		}

		// Recurse if within the allowed search depth
		if distance < MaxFlowSearchDepth {
			j := flowCost(w, nx, ny, nz, distance+1, oppositeDir(dirIdx), fluidType)
			if j < cost {
				cost = j
			}
		}
	}

	return cost
}

// getPossibleFlowDirections determines which horizontal directions water should spread into.
// Uses BFS-like cost search to find the shortest path to a drop-off.
// Returns a [4]bool array: index 0=East(+X), 1=West(-X), 2=South(+Z), 3=North(-Z).
func getPossibleFlowDirections(w *World, x, y, z int, fluidType BlockType) [4]bool {
	minCost := 1000
	var result [4]bool

	for dirIdx, dir := range horizontalDirs {
		nx, ny, nz := x+dir[0], y+dir[1], z+dir[2]

		// If blocked or same-fluid source, can't flow this way
		if isFluidBlocked(w, nx, ny, nz) {
			continue
		}
		bt := w.Get(nx, ny, nz)
		if bt == fluidType && int(w.GetMeta(nx, ny, nz)) == 0 {
			continue
		}

		var cost int
		if isFluidBlocked(w, nx, ny-1, nz) {
			// Block below neighbor is solid - search further for a drop-off
			cost = flowCost(w, nx, ny, nz, 1, oppositeDir(dirIdx), fluidType)
		} else {
			// Immediate drop-off below neighbor
			cost = 0
		}

		if cost < minCost {
			minCost = cost
			result = [4]bool{} // Reset: only directions with the new minimum count
		}
		if cost <= minCost {
			result[dirIdx] = true
		}
	}

	return result
}

// tryFlowInto attempts to flow fluid into position (x,y,z) with the given level.
// Schedules a tick on the new fluid block so it can continue propagating.
func tryFlowInto(w *World, x, y, z int, level int, fluidType BlockType) {
	if !canFluidFlowInto(w, x, y, z, fluidType) {
		return
	}
	// Non-air blocks are overwritten (items would drop in full MC - skipped here)
	w.SetWithMeta(x, y, z, fluidType, uint8(level))
	tickRate := WaterTickRate
	if fluidType == BlockTypeLava {
		tickRate = LavaTickRate
	}
	w.ScheduleBlockTick(x, y, z, tickRate, 0)
}

// FluidTick is called when a scheduled tick fires for a fluid block.
// This is the Go port of MC's BlockDynamicLiquid.updateTick().
func FluidTick(w *World, x, y, z int) {
	bt := w.Get(x, y, z)
	if bt != BlockTypeWater && bt != BlockTypeLava {
		return
	}
	fluidType := bt

	// Lava-water mixing check must happen first for lava blocks
	if fluidType == BlockTypeLava {
		if checkForMixing(w, x, y, z) {
			return
		}
	}

	currentLevel := int(w.GetMeta(x, y, z))
	spreadCost := WaterSpreadCost
	tickRate := WaterTickRate
	if fluidType == BlockTypeLava {
		spreadCost = LavaSpreadCost
		tickRate = LavaTickRate
	}

	// === LEVEL RECALCULATION (only for non-source flowing blocks) ===
	if currentLevel > 0 {
		minAdjacentLevel := -100 // Sentinel: no valid neighbor seen yet
		adjacentSources := 0

		for _, dir := range horizontalDirs {
			minAdjacentLevel = checkAdjacentBlock(w,
				x+dir[0], y+dir[1], z+dir[2],
				minAdjacentLevel, &adjacentSources, fluidType)
		}

		newLevel := minAdjacentLevel + spreadCost
		if newLevel >= 8 || minAdjacentLevel < 0 {
			newLevel = -1 // No valid feeder, block should disappear
		}

		// If there is the same fluid directly above, inherit its level
		aboveLevel := getFluidLevel(w, x, y+1, z, fluidType)
		if aboveLevel >= 0 {
			if aboveLevel >= 8 {
				newLevel = aboveLevel
			} else {
				newLevel = aboveLevel + 8
			}
		}

		// Infinite water source rule: 2+ adjacent source blocks can regenerate a source
		if adjacentSources >= 2 && fluidType == BlockTypeWater {
			belowBt := w.Get(x, y-1, z)
			isBelowSolid := BlockSolidTable[belowBt]
			isBelowWaterSource := belowBt == fluidType && int(w.GetMeta(x, y-1, z)) == 0
			if isBelowSolid || isBelowWaterSource {
				newLevel = 0
			}
		}

		if newLevel != currentLevel {
			currentLevel = newLevel

			if newLevel < 0 {
				// Block dried up / starved of fluid
				w.Set(x, y, z, BlockTypeAir)
				notifyFluidNeighbors(w, x, y, z)
				return
			}

			// Propagate updated level and reschedule
			w.SetMeta(x, y, z, uint8(newLevel))
			w.ScheduleBlockTick(x, y, z, tickRate, 0)
			notifyFluidNeighbors(w, x, y, z)
		}
		// When newLevel == currentLevel, level is stable — don't return.
		// Fall through to downward/horizontal spread (matches MC behavior
		// where stable blocks still complete their current tick's spread).
	}
	// currentLevel == 0 means source block: level is always stable (no change).
	// We still fall through to handle downward/horizontal spread from the source.

	// === DOWNWARD FLOW ===
	belowBt := w.Get(x, y-1, z)
	if canFluidFlowInto(w, x, y-1, z, fluidType) {
		// Lava flowing down into water → stone
		if fluidType == BlockTypeLava && belowBt == BlockTypeWater {
			w.SetWithMeta(x, y-1, z, BlockTypeStone, 0)
			return
		}
		if currentLevel >= 8 {
			tryFlowInto(w, x, y-1, z, currentLevel, fluidType)
		} else {
			// Set the falling bit (bit 3) by adding 8
			tryFlowInto(w, x, y-1, z, currentLevel+8, fluidType)
		}
		// After flowing down we still continue to spread horizontally if needed
	}

	// === HORIZONTAL SPREAD ===
	// Spread horizontally only when we can't flow straight down OR we're a source.
	belowBlocked := isFluidBlocked(w, x, y-1, z) || belowBt == fluidType
	if currentLevel == 0 || belowBlocked {
		directions := getPossibleFlowDirections(w, x, y, z, fluidType)

		spreadLevel := currentLevel + spreadCost
		if currentLevel >= 8 {
			// Falling water spreads at level 1
			spreadLevel = 1
		}

		if spreadLevel >= 8 {
			return // Would exceed maximum flow distance, stop
		}

		for dirIdx, shouldFlow := range directions {
			if shouldFlow {
				dir := horizontalDirs[dirIdx]
				nx, nz := x+dir[0], z+dir[2]
				tryFlowInto(w, nx, y, nz, spreadLevel, fluidType)
			}
		}
	}
}

// checkForMixing handles lava-water interaction.
// Checks the 5 neighbors excluding DOWN for water.
// Returns true if the lava block was converted (tick should stop).
func checkForMixing(w *World, x, y, z int) bool {
	bt := w.Get(x, y, z)
	if bt != BlockTypeLava {
		return false
	}

	neighbors := [5][3]int{
		{x + 1, y, z},
		{x - 1, y, z},
		{x, y + 1, z},
		{x, y, z + 1},
		{x, y, z - 1},
	}
	waterFound := false
	for _, n := range neighbors {
		if w.Get(n[0], n[1], n[2]) == BlockTypeWater {
			waterFound = true
			break
		}
	}

	if !waterFound {
		return false
	}

	level := int(w.GetMeta(x, y, z))
	if level == 0 {
		// Source lava + adjacent water → obsidian
		w.SetWithMeta(x, y, z, BlockTypeObsidian, 0)
		return true
	}
	if level <= 4 {
		// Flowing lava (close to source) + water → cobblestone
		w.SetWithMeta(x, y, z, BlockTypeCobblestone, 0)
		return true
	}

	// Level > 4: lava too weak to solidify, no conversion
	return false
}

// notifyFluidNeighbors schedules ticks for any fluid blocks in all 6 neighboring positions.
// This is how water reacts to adjacent block changes (placement or removal).
func notifyFluidNeighbors(w *World, x, y, z int) {
	neighbors := [6][3]int{
		{x + 1, y, z},
		{x - 1, y, z},
		{x, y + 1, z},
		{x, y - 1, z},
		{x, y, z + 1},
		{x, y, z - 1},
	}
	for _, n := range neighbors {
		bt := w.Get(n[0], n[1], n[2])
		if bt == BlockTypeWater {
			w.ScheduleBlockTick(n[0], n[1], n[2], WaterTickRate, 0)
		} else if bt == BlockTypeLava {
			w.ScheduleBlockTick(n[0], n[1], n[2], LavaTickRate, 0)
		}
	}
}

// NotifyNeighbors is called when a block is placed or broken to wake up any
// adjacent fluid blocks so they can recalculate their flow.
func (w *World) NotifyNeighbors(x, y, z int) {
	notifyFluidNeighbors(w, x, y, z)
}
