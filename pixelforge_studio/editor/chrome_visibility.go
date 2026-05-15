package editor

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
)

// chromeVisibility owns the Esc-toggle that hides the workspace chrome
// to reveal the game canvas full-screen inside the Scene workspace
// region. The toggle has modal-precedence: while any modal is open, Esc
// dismisses the modal and the chrome state stays as-is.
type chromeVisibility struct {
	hidden bool

	// gameCanvas is the project's render target. It updates at TPS
	// regardless of editor activity. M3 ships the always-on render
	// path; M5+ adds behaviour execution on top.
	gameCanvas pixelforge.Canvas
}

func newChromeVisibility() *chromeVisibility {
	return &chromeVisibility{}
}

// Hidden reports whether workspace chrome is currently hidden.
func (c *chromeVisibility) Hidden() bool { return c.hidden }

// Toggle flips the hidden state.
func (c *chromeVisibility) Toggle() { c.hidden = !c.hidden }

// EnsureGameCanvas allocates the game canvas to match the project's
// screen size if not yet allocated or if the size has changed.
func (c *chromeVisibility) EnsureGameCanvas(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	if c.gameCanvas.W() == w && c.gameCanvas.H() == h {
		return
	}
	c.gameCanvas = pixelforge.NewCanvas(w, h)
}

// GameCanvas returns the project's game render target.
func (c *chromeVisibility) GameCanvas() pixelforge.Canvas { return c.gameCanvas }
