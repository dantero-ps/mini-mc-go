package hud

import (
	"fmt"
	"mini-mc/internal/inventory"
	"mini-mc/internal/player"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

// ContainerScreen is a base struct for screens that display a container inventory
type ContainerScreen struct {
	HUD       *HUD
	Container *inventory.Container
	Player    *player.Player

	// Screen dimensions and position (usually centered)
	X, Y          float32
	Width, Height float32
	Scale         float32

	backgroundTex uint32
	backgroundW   float32
	backgroundH   float32

	hoveredSlotIndex int

	// Double click tracking
	lastClickSlotIndex int
	lastClickTime      time.Time
}

func NewContainerScreen(hud *HUD, p *player.Player, c *inventory.Container, tex uint32, w, h float32) *ContainerScreen {
	scale := float32(2.0)
	// Center on screen
	screenW := hud.width
	screenH := hud.height
	invW := w * scale
	invH := h * scale

	// Calculate screen position to center the container
	x := (screenW - invW) / 2
	y := (screenH - invH) / 2

	return &ContainerScreen{
		HUD:                hud,
		Container:          c,
		Player:             p,
		X:                  x,
		Y:                  y,
		Width:              invW,
		Height:             invH,
		Scale:              scale,
		backgroundTex:      tex,
		backgroundW:        w,
		backgroundH:        h,
		hoveredSlotIndex:   -1,
		lastClickSlotIndex: -1,
	}
}

func (s *ContainerScreen) Init() {
	s.Resize()
}

func (s *ContainerScreen) Resize() {
	screenW := s.HUD.width
	screenH := s.HUD.height
	s.Width = s.backgroundW * s.Scale
	s.Height = s.backgroundH * s.Scale
	s.X = (screenW - s.Width) / 2
	s.Y = (screenH - s.Height) / 2
}

func (s *ContainerScreen) Render(mouseX, mouseY float64) {
	// Draw Background
	u1 := s.backgroundW / 256.0
	v1 := s.backgroundH / 256.0
	color := mgl32.Vec3{1.0, 1.0, 1.0}

	s.HUD.uiRenderer.DrawTexturedRect(s.X, s.Y, s.Width, s.Height, s.backgroundTex, 0, 0, u1, v1, color, 1.0)

	// Flush background so items draw on top
	s.HUD.uiRenderer.Flush()

	itemSize := 16 * s.Scale
	s.hoveredSlotIndex = -1

	mx := float32(mouseX)
	my := float32(mouseY)

	for i, slot := range s.Container.Slots {
		slotX := s.X + float32(slot.X)*s.Scale
		slotY := s.Y + float32(slot.Y)*s.Scale

		stack := slot.GetStack()
		if stack != nil {
			s.HUD.itemRenderer.RenderGUI(stack, slotX, slotY, itemSize)

			if stack.Count > 1 {
				countText := fmt.Sprintf("%d", stack.Count)
				tx := slotX + itemSize/2
				ty := slotY + itemSize/2
				s.HUD.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
			}
		}

		if mx >= slotX && mx < slotX+itemSize && my >= slotY && my < slotY+itemSize {
			s.hoveredSlotIndex = i
			s.HUD.uiRenderer.DrawFilledRect(slotX, slotY, itemSize, itemSize, mgl32.Vec3{1, 1, 1}, 0.5)
		}
	}

	// Flush overlays (so they are drawn over items but UNDER cursor)
	s.HUD.uiRenderer.Flush()

	cursor := s.Player.Inventory.CursorStack
	if cursor != nil {
		s.HUD.itemRenderer.RenderGUI(cursor, mx-itemSize/2, my-itemSize/2, itemSize)

		if cursor.Count > 1 {
			countText := fmt.Sprintf("%d", cursor.Count)
			tx := mx + itemSize/4
			ty := my + itemSize/4
			s.HUD.fontRenderer.Render(countText, tx, ty, 0.3, mgl32.Vec3{1, 1, 1})
		}
	}
}

func (s *ContainerScreen) HandleClick(x, y float64, button glfw.MouseButton, action glfw.Action) bool {
	if action != glfw.Press {
		return false
	}

	mx := float32(x)
	my := float32(y)
	itemSize := 16 * s.Scale

	clickedSlotIndex := -1

	for i, slot := range s.Container.Slots {
		slotX := s.X + float32(slot.X)*s.Scale
		slotY := s.Y + float32(slot.Y)*s.Scale

		if mx >= slotX && mx < slotX+itemSize && my >= slotY && my < slotY+itemSize {
			clickedSlotIndex = i
			break
		}
	}

	if clickedSlotIndex != -1 {
		// Map glfw button to inventory button
		var invBtn inventory.MouseButton
		if button == glfw.MouseButtonLeft {
			invBtn = inventory.MouseButtonLeft
		} else if button == glfw.MouseButtonRight {
			invBtn = inventory.MouseButtonRight
		} else {
			return false // Unknown button
		}

		// Double click detection
		isDoubleClick := false
		if button == glfw.MouseButtonLeft {
			if clickedSlotIndex == s.lastClickSlotIndex && time.Since(s.lastClickTime) < 300*time.Millisecond {
				isDoubleClick = true
			}
			s.lastClickSlotIndex = clickedSlotIndex
			s.lastClickTime = time.Now()
		}

		s.Container.SlotClick(clickedSlotIndex, invBtn, isDoubleClick, s.Player.Inventory)
		return true
	}

	return false
}

func (s *ContainerScreen) Close() {}

func (s *ContainerScreen) Update() {}

func (s *ContainerScreen) IsPauseScreen() bool {
	return false
}

func (s *ContainerScreen) GetHoveredSlot() int {
	return s.hoveredSlotIndex
}

func (s *ContainerScreen) IsActive() bool {
	return true
}

func (s *ContainerScreen) GetContainer() *inventory.Container {
	return s.Container
}
