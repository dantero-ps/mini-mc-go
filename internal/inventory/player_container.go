package inventory

// NewPlayerContainer creates a container for the player's inventory
func NewPlayerContainer(inv *Inventory) *Container {
	c := NewContainer()

	// Add Main Inventory Slots (Indices 9-35)
	// Grid: 9 columns, 3 rows
	// Starting at index 9 because 0-8 is hotbar
	for i := 0; i < 3; i++ { // rows
		for j := 0; j < 9; j++ { // cols
			index := j + (i+1)*9 // j + (row+1)*9 -> row=0: 9-17, row=1: 18-26, row=2: 27-35
			x := 8 + j*18
			y := 84 + i*18
			c.AddSlot(NewSlot(inv, index, x, y))
		}
	}

	// Add Hotbar Slots (Indices 0-8)
	for i := 0; i < 9; i++ {
		x := 8 + i*18
		y := 142
		c.AddSlot(NewSlot(inv, i, x, y))
	}

	// Add Armor Slots (Indices 36-39)
	// Armor inventory is indices 36-39 in our global view
	for i := 0; i < 4; i++ {
		x := 8
		y := 8 + i*18
		// In inventory.go we defined GetItem to map [36, 39] to ArmorInventory [0, 3]
		c.AddSlot(NewSlot(inv, 36+i, x, y))
	}

	return c
}
