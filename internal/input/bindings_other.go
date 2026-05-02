//go:build !darwin

package input

import "github.com/go-gl/glfw/v3.3/glfw"

func (im *InputManager) bindSprintKey() {
	im.BindKey(glfw.KeyLeftControl, ActionSprint)
}
