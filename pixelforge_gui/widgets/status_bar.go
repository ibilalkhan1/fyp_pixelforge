package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// StatusBar is the canvas-resident bottom-of-window status strip. It
// renders three text regions (Left / centred Hint / Right) plus a
// chrome-hidden indicator when the editor's chrome is collapsed.
type StatusBar struct {
	Left  string
	Right string
	Hint  string

	X, Y, W, H int

	FgColor pixelforge.Color
	BgColor pixelforge.Color
}

// StatusBarHeight matches the native bar's height so existing chrome
// layouts don't shift when the canvas version replaces it.
const StatusBarHeight = 18

// NewStatusBar constructs a status bar with sensible default colors.
func NewStatusBar() *StatusBar {
	return &StatusBar{FgColor: 7, BgColor: 5}
}

// SetBounds positions the status bar.
func (s *StatusBar) SetBounds(x, y, w, h int) {
	s.X, s.Y, s.W, s.H = x, y, w, h
}

// Draw paints the status bar via engine primitives.
func (s *StatusBar) Draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(s.BgColor)
	pixelforge.RectFill(s.X, s.Y, s.X+s.W-1, s.Y+s.H-1)

	pixelforge.SetColor(s.FgColor)
	// Left.
	if s.Left != "" {
		pixelforge_cofont.Print(s.Left, s.X+8, s.Y+(s.H-pixelforge_cofont.Sheet.Height)/2)
	}
	// Right (right-aligned, 4px-per-glyph).
	if s.Right != "" {
		w := len(s.Right) * 4
		pixelforge_cofont.Print(s.Right, s.X+s.W-w-8, s.Y+(s.H-pixelforge_cofont.Sheet.Height)/2)
	}
	// Hint (centred).
	if s.Hint != "" {
		text := truncateWithEllipsis(s.Hint, max0(s.W-32)/4)
		w := len(text) * 4
		pixelforge_cofont.Print(text, s.X+(s.W-w)/2, s.Y+(s.H-pixelforge_cofont.Sheet.Height)/2)
	}
}

// truncateWithEllipsis returns s shortened to at most maxRunes glyphs
// (1 glyph = 4 px in cofont). Adds "…" suffix if the text was cut.
func truncateWithEllipsis(s string, maxGlyphs int) string {
	if maxGlyphs <= 0 {
		return ""
	}
	if len(s) <= maxGlyphs {
		return s
	}
	if maxGlyphs < 2 {
		return s[:maxGlyphs]
	}
	return s[:maxGlyphs-1] + "."
}
