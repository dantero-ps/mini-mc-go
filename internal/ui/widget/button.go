package widget

import (
	"mini-mc/internal/graphics/renderables/ui"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type Button struct {
	BaseComponent
	Text      string
	Subtitle  string
	OnClick   func()
	IsHovered bool

	NormalColor   mgl32.Vec3
	HoverColor    mgl32.Vec3
	TextColor     mgl32.Vec3
	SubtitleColor mgl32.Vec3
}

func NewButton(text string, x, y, w, h float32, onClick func()) *Button {
	return &Button{
		BaseComponent: BaseComponent{X: x, Y: y, W: w, H: h},
		Text:          text,
		OnClick:       onClick,
		NormalColor:   mgl32.Vec3{0.3, 0.3, 0.3},
		HoverColor:    mgl32.Vec3{0.4, 0.4, 0.4},
		TextColor:     mgl32.Vec3{1, 1, 1},
		SubtitleColor: mgl32.Vec3{0.8, 0.8, 0.8},
	}
}

func (b *Button) Render(u *ui.UI, window *glfw.Window) {
	mx, my := window.GetCursorPos()
	mx32, my32 := float32(mx), float32(my)

	b.IsHovered = mx32 >= b.X && mx32 <= b.X+b.W && my32 >= b.Y && my32 <= b.Y+b.H

	color := b.NormalColor
	if b.IsHovered {
		color = b.HoverColor
	}

	u.DrawFilledRect(b.X, b.Y, b.W, b.H, color, 1.0)

	// Calculate text positioning and scaling
	// 1. Initial Height Estimation
	// If we have a subtitle, we share the vertical space.
	// Main text gets ~30%, Subtitle gets ~20%, Spacing ~5% => Total ~55% of button height
	// If no subtitle, Main text gets ~40%
	mainTextHeightRatio := float32(0.4)
	if b.Subtitle != "" {
		mainTextHeightRatio = 0.3
	}

	_, rawH := u.MeasureText(b.Text, 1.0)
	if rawH == 0 {
		rawH = 20
	}

	targetH := b.H * mainTextHeightRatio
	textScale := targetH / rawH

	// 2. Width Constraint (Main Text)
	textW, _ := u.MeasureText(b.Text, textScale)
	maxW := b.W * 0.90 // 90% of button width

	if textW > maxW {
		correction := maxW / textW
		textScale *= correction
		targetH *= correction // Height reduces as we scale down
		textW = maxW
	}

	// 3. Subtitle Calculation
	var subScale, subW, subH, spacing float32
	if b.Subtitle != "" {
		subScale = textScale * 0.6
		subW, _ = u.MeasureText(b.Subtitle, subScale)

		// Check Subtitle Width Constraint
		if subW > maxW {
			correction := maxW / subW
			subScale *= correction
			subW = maxW
		}

		// Calculate Subtitle Height
		_, rawSubH := u.MeasureText(b.Subtitle, 1.0)
		subH = rawSubH * subScale
		spacing = b.H * 0.05
	}

	// 4. Calculate Total Content Height
	totalContentH := targetH
	if b.Subtitle != "" {
		totalContentH += spacing + subH
	}

	// 5. Vertical Centering
	// Calculate top-left Y position of the whole content block
	contentTopY := b.Y + (b.H-totalContentH)/2

	// We need to shift down to get to the baseline.
	// Approximate baseline offset: ~75% of the line height.
	mainBaselineOffset := targetH * 0.75
	subBaselineOffset := subH * 0.75

	// 6. Draw Main Text
	textX := b.X + (b.W-textW)/2
	u.DrawText(b.Text, textX, contentTopY+mainBaselineOffset, textScale, b.TextColor)

	// 7. Draw Subtitle
	if b.Subtitle != "" {
		subX := b.X + (b.W-subW)/2
		subY := contentTopY + targetH + spacing + subBaselineOffset
		u.DrawText(b.Subtitle, subX, subY, subScale, b.SubtitleColor)
	}
}

func (b *Button) HandleInput(window *glfw.Window, justPressedLeft bool) bool {
	if b.IsHovered && justPressedLeft {
		if b.OnClick != nil {
			b.OnClick()
		}
		return true
	}
	return false
}
