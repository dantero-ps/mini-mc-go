//go:build darwin

package input

import "github.com/go-gl/glfw/v3.3/glfw"

// bindSprintKey binds sprint on macOS.
// macOS intercepts Ctrl+Space at OS level (Input Source switch shortcut),
// so LeftAlt (Option) is used as the primary sprint key.
// LeftControl is kept as a secondary binding for users who disable the OS shortcut.
func (im *InputManager) bindSprintKey() {
	im.BindKey(glfw.KeyLeftAlt, ActionSprint)
	im.BindKey(glfw.KeyLeftControl, ActionSprint)
}
