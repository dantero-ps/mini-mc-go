package menu

type Action int

const (
	ActionNone Action = iota
	ActionStartSurvival
	ActionStartCreative
	ActionResume
	ActionQuitToMenu
	ActionQuitGame
)
