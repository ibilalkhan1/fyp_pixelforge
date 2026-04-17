// Package pifont provides functionality for rendering text
// using bitmap fonts.
package pixelforge_font

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
)

// Sheet is a character sheet used for rendering text.
type Sheet struct {
	Chars   map[rune]pixelforge.Sprite
	Height  int
	FgColor pixelforge.Color // font color on sprites
	BgColor pixelforge.Color // background color on sprites
}

var intermediateCanvas pixelforge.Canvas // text is first rendered here to change its color from FgColor to selected color

var prevFgColorTable [pixelforge.MaxColors]pixelforge.Color
var prevBgColorTable [pixelforge.MaxColors]pixelforge.Color

// Print draws text using the current draw color.
//
// Returns the x, y position where you can continue writing text.
func (s Sheet) Print(str string, x, y int) (currentX, currentY int) {
	originalDrawTarget := pixelforge.DrawTarget()
	if intermediateCanvas.W() != originalDrawTarget.W() || intermediateCanvas.H() != originalDrawTarget.H() {
		intermediateCanvas = pixelforge.NewCanvas(originalDrawTarget.W(), originalDrawTarget.H())
	}

	currentColor := pixelforge.GetColor()

	prevFgColorTable = pixelforge.ColorTables[0][s.FgColor]
	prevBgColorTable = pixelforge.ColorTables[0][s.BgColor]

	// create fake bg color to avoid a situation when fg and bg colors are the same
	bgColor := (currentColor + 1) % pixelforge.MaxColors
	intermediateCanvas.Clear(s.BgColor)
	pixelforge.RemapColor(s.FgColor, currentColor)
	pixelforge.RemapColor(s.BgColor, bgColor)
	pixelforge.SetDrawTarget(intermediateCanvas)

	// first draw text in selected color on intermediateCanvas
	currentX, currentY = s.PrintOriginal(str, x, y)

	// revert color tables
	pixelforge.ColorTables[0][s.FgColor] = prevFgColorTable
	pixelforge.ColorTables[0][s.BgColor] = prevBgColorTable

	// make bgColor transparent
	prevBgColorTable = pixelforge.ColorTables[0][bgColor]
	pixelforge.SetTransparency(bgColor, true)

	// now copy text in target color on original draw target
	coloredText := pixelforge.Sprite{
		Area: pixelforge.Area[int]{
			X: x - pixelforge.Camera.X,
			Y: y - pixelforge.Camera.Y,
			W: currentX - x,
			H: currentY - y + s.Height,
		},
		Source: intermediateCanvas,
	}
	pixelforge.SetDrawTarget(originalDrawTarget)
	pixelforge.DrawSprite(coloredText, x, y)

	// revert bgColor transparency
	pixelforge.ColorTables[0][bgColor] = prevBgColorTable

	return
}

// PrintOriginal prints the text using its original colors.
func (s Sheet) PrintOriginal(str string, x, y int) (maxX, currentY int) {
	maxX = x
	currentX := x
	currentY = y
	for _, r := range str {
		if r == '\n' {
			currentX = x
			currentY += s.Height
			continue
		}
		sprite := s.Chars[r]
		pixelforge.DrawSprite(sprite, currentX, currentY)
		currentX += sprite.W
		maxX = max(maxX, currentX)
	}

	return
}

// PrintStroked prints the text with a stroke effect.
//
// The text is drawn using the specified foreground and stroke colors.
func (s Sheet) PrintStroked(text string, x, y int, fgColor, strokeColor pixelforge.Color) (currentX, currentY int) {
	prevColor := pixelforge.SetColor(strokeColor)
	for l := y - 1; l <= y+1; l++ {
		s.Print(text, x-1, l)
		s.Print(text, x, l)
		s.Print(text, x+1, l)
	}

	pixelforge.SetColor(fgColor)
	currentX, currentY = s.Print(text, x, y)

	pixelforge.SetColor(prevColor)

	return
}

// Size returns the dimensions of the text without rendering it to the draw target.
func (s Sheet) Size(text string) (width, height int) {
	originalDrawTarget := pixelforge.SetDrawTarget(intermediateCanvas)
	defer pixelforge.SetDrawTarget(originalDrawTarget)

	return s.PrintOriginal(text, 0, 0)
}
