package input

import (
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Action represents a logical game action, not a physical key
type Action int

// Action constants using iota
const (
	ActionMoveForward Action = iota
	ActionMoveBackward
	ActionMoveLeft
	ActionMoveRight
	ActionJump
	ActionSprint
	ActionSneak
	ActionInventory
	ActionPause
	ActionDropItem
	ActionHotbar1
	ActionHotbar2
	ActionHotbar3
	ActionHotbar4
	ActionHotbar5
	ActionHotbar6
	ActionHotbar7
	ActionHotbar8
	ActionHotbar9
	ActionToggleWireframe
	ActionToggleProfiling
	ActionMouseLeft
	ActionMouseRight
	ActionMouseMiddle
	ActionModControl
	ActionModShift
	ActionModAlt
	ActionModSuper
	ActionCount // Sentinel value for array sizing
)

// InputManager manages keyboard and mouse input state and maps physical keys/buttons to logical actions
type InputManager struct {
	mu sync.RWMutex

	// Key to action mapping (one key can map to multiple actions)
	keyToActions map[glfw.Key][]Action

	// Mouse button to action mapping
	mouseButtonToActions map[glfw.MouseButton][]Action

	// Current frame state (indexed by Action)
	currentState [ActionCount]bool

	// Previous frame state (for edge detection)
	prevState [ActionCount]bool

	// Just pressed/released flags (reset each frame)
	justPressed  [ActionCount]bool
	justReleased [ActionCount]bool
}

// NewInputManager creates a new InputManager with default key bindings
func NewInputManager() *InputManager {
	im := &InputManager{
		keyToActions:         make(map[glfw.Key][]Action),
		mouseButtonToActions: make(map[glfw.MouseButton][]Action),
	}

	// Set default key bindings
	im.BindKey(glfw.KeyW, ActionMoveForward)
	im.BindKey(glfw.KeyS, ActionMoveBackward)
	im.BindKey(glfw.KeyA, ActionMoveLeft)
	im.BindKey(glfw.KeyD, ActionMoveRight)
	im.BindKey(glfw.KeySpace, ActionJump)
	im.BindKey(glfw.KeyLeftControl, ActionSprint)
	im.BindKey(glfw.KeyLeftShift, ActionSneak)
	im.BindKey(glfw.KeyE, ActionInventory)
	im.BindKey(glfw.KeyEscape, ActionPause)
	im.BindKey(glfw.KeyQ, ActionDropItem)
	im.BindKey(glfw.Key1, ActionHotbar1)
	im.BindKey(glfw.Key2, ActionHotbar2)
	im.BindKey(glfw.Key3, ActionHotbar3)
	im.BindKey(glfw.Key4, ActionHotbar4)
	im.BindKey(glfw.Key5, ActionHotbar5)
	im.BindKey(glfw.Key6, ActionHotbar6)
	im.BindKey(glfw.Key7, ActionHotbar7)
	im.BindKey(glfw.Key8, ActionHotbar8)
	im.BindKey(glfw.Key9, ActionHotbar9)
	im.BindKey(glfw.KeyF, ActionToggleWireframe)
	im.BindKey(glfw.KeyV, ActionToggleProfiling)

	// Set default mouse button bindings
	im.BindMouseButton(glfw.MouseButtonLeft, ActionMouseLeft)
	im.BindMouseButton(glfw.MouseButtonRight, ActionMouseRight)
	im.BindMouseButton(glfw.MouseButtonMiddle, ActionMouseMiddle)

	// Set default modifier key bindings
	im.BindKey(glfw.KeyLeftControl, ActionModControl)
	im.BindKey(glfw.KeyRightControl, ActionModControl)
	im.BindKey(glfw.KeyLeftShift, ActionModShift)
	im.BindKey(glfw.KeyRightShift, ActionModShift)
	im.BindKey(glfw.KeyLeftAlt, ActionModAlt)
	im.BindKey(glfw.KeyRightAlt, ActionModAlt)
	im.BindKey(glfw.KeyLeftSuper, ActionModSuper)
	im.BindKey(glfw.KeyRightSuper, ActionModSuper)

	return im
}

// BindKey binds a physical key to a logical action
// Multiple keys can be bound to the same action (e.g., WASD and arrow keys)
func (im *InputManager) BindKey(key glfw.Key, action Action) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if action < 0 || action >= ActionCount {
		return
	}

	im.keyToActions[key] = append(im.keyToActions[key], action)
}

// UnbindKey removes all action bindings for a key
func (im *InputManager) UnbindKey(key glfw.Key) {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.keyToActions, key)
}

// BindMouseButton binds a mouse button to a logical action
func (im *InputManager) BindMouseButton(button glfw.MouseButton, action Action) {
	im.mu.Lock()
	defer im.mu.Unlock()

	if action < 0 || action >= ActionCount {
		return
	}

	im.mouseButtonToActions[button] = append(im.mouseButtonToActions[button], action)
}

// UnbindMouseButton removes all action bindings for a mouse button
func (im *InputManager) UnbindMouseButton(button glfw.MouseButton) {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.mouseButtonToActions, button)
}

// HandleKeyEvent processes a key event and updates internal state
// This can be called from a custom key callback
func (im *InputManager) HandleKeyEvent(key glfw.Key, action glfw.Action) {
	im.mu.RLock()
	actions, exists := im.keyToActions[key]
	im.mu.RUnlock()

	if !exists {
		return
	}

	isPressed := action == glfw.Press || action == glfw.Repeat

	im.mu.Lock()
	for _, act := range actions {
		if act >= 0 && act < ActionCount {
			// Detect edges immediately when event arrives
			if isPressed && !im.currentState[act] {
				im.justPressed[act] = true
			}
			if !isPressed && im.currentState[act] {
				im.justReleased[act] = true
			}
			im.currentState[act] = isPressed
		}
	}
	im.mu.Unlock()
}

// HandleMouseButtonEvent processes a mouse button event and updates internal state
// This can be called from a custom mouse button callback
func (im *InputManager) HandleMouseButtonEvent(button glfw.MouseButton, action glfw.Action) {
	im.mu.RLock()
	actions, exists := im.mouseButtonToActions[button]
	im.mu.RUnlock()

	if !exists {
		return
	}

	isPressed := action == glfw.Press

	im.mu.Lock()
	for _, act := range actions {
		if act >= 0 && act < ActionCount {
			// Detect edges immediately when event arrives
			if isPressed && !im.currentState[act] {
				im.justPressed[act] = true
			}
			if !isPressed && im.currentState[act] {
				im.justReleased[act] = true
			}
			im.currentState[act] = isPressed
		}
	}
	im.mu.Unlock()
}

// SetKeyCallback sets up the GLFW key callback for this input manager
// This should be called once during initialization
func (im *InputManager) SetKeyCallback(window *glfw.Window) {
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		im.HandleKeyEvent(key, action)
	})
}

// PostUpdate must be called at the end of each frame to update edge detection states
// This should be called after all input checks are done
func (im *InputManager) PostUpdate() {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Reset edge flags and update prev state
	for i := range ActionCount {
		im.justPressed[i] = false
		im.justReleased[i] = false
		im.prevState[i] = im.currentState[i]
	}
}

// IsActive returns true if the action is currently being held down
func (im *InputManager) IsActive(action Action) bool {
	if action < 0 || action >= ActionCount {
		return false
	}

	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.currentState[action]
}

// JustPressed returns true only if the action was pressed in the current frame
func (im *InputManager) JustPressed(action Action) bool {
	if action < 0 || action >= ActionCount {
		return false
	}

	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.justPressed[action]
}

// JustReleased returns true only if the action was released in the current frame
func (im *InputManager) JustReleased(action Action) bool {
	if action < 0 || action >= ActionCount {
		return false
	}

	im.mu.RLock()
	defer im.mu.RUnlock()

	return im.justReleased[action]
}
