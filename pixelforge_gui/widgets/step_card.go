package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// StepCardOptions configures a StepCard widget.
type StepCardOptions struct {
	Kind       string
	Label      string
	IsActive   bool
	Selected   bool
	BgColor    pixelforge.Color
	AccentColor pixelforge.Color
	TextColor  pixelforge.Color
	OnSelect   func()
	OnDragMove func(dx, dy int)
}

// StepCard is the visual representation of one BehaviorGraph.StepNode
// in the lane editor: a ~64x56 px rectangle with the Kind label, an
// abbreviated args summary, and a footer "active step" indicator
// when IsActive is true.
//
// The card composes a Draggable so the host (Lane editor) gets
// cumulative drag deltas without the card knowing about its peers.
// A press-release with zero net movement fires OnSelect; anything
// else is reported as a drag.
type StepCard struct {
	*pgui.Element

	Kind        string
	Label       string
	IsActive    bool
	Selected    bool
	BgColor     pixelforge.Color
	AccentColor pixelforge.Color
	TextColor   pixelforge.Color

	OnSelect   func()
	OnDragMove func(dx, dy int)

	drag         *Draggable
	cumulativeDX int
	cumulativeDY int
	hadAnyMove   bool
}

// NewStepCard constructs a StepCard rooted at (x, y, w, h).
//
// Defaults: 64×56 px when w/h are zero; theme colours fall back to
// palette indices that work with the studio's default theme.
func NewStepCard(x, y, w, h int, opts StepCardOptions) *StepCard {
	if w <= 0 {
		w = 64
	}
	if h <= 0 {
		h = 56
	}
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.AccentColor == 0 {
		opts.AccentColor = 12
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	c := &StepCard{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Kind:        opts.Kind,
		Label:       opts.Label,
		IsActive:    opts.IsActive,
		Selected:    opts.Selected,
		BgColor:     opts.BgColor,
		AccentColor: opts.AccentColor,
		TextColor:   opts.TextColor,
		OnSelect:    opts.OnSelect,
		OnDragMove:  opts.OnDragMove,
	}
	c.drag = NewDraggable(IntRect{X: x, Y: y, W: w, H: h})
	c.drag.OnDrag = func(dx, dy int) {
		c.cumulativeDX += dx
		c.cumulativeDY += dy
		if dx != 0 || dy != 0 {
			c.hadAnyMove = true
		}
		if c.OnDragMove != nil {
			c.OnDragMove(dx, dy)
		}
	}
	c.Element.OnDraw = func(_ pgui.DrawEvent) {
		c.draw()
	}
	return c
}

// Press forwards a pointer-down at (mx, my) to the underlying Draggable.
// Returns true when the press hit the card and started a drag.
func (c *StepCard) Press(mx, my int) bool {
	if c == nil {
		return false
	}
	c.cumulativeDX = 0
	c.cumulativeDY = 0
	c.hadAnyMove = false
	c.drag.SetRegion(IntRect{X: c.X, Y: c.Y, W: c.W, H: c.H})
	return c.drag.Press(mx, my)
}

// Move forwards a pointer-move while pressed.
func (c *StepCard) Move(mx, my int) {
	if c == nil {
		return
	}
	c.drag.Move(mx, my)
}

// Release ends a drag. Returns true if the press resolved as a click
// (no net movement); the host invokes OnSelect in that case.
//
// A drag-then-release-back-to-zero is still treated as a drag, not
// a click, because hadAnyMove latches.
func (c *StepCard) Release() bool {
	if c == nil {
		return false
	}
	wasPressed := c.drag.Pressed()
	c.drag.Release()
	if !wasPressed {
		return false
	}
	if !c.hadAnyMove && c.cumulativeDX == 0 && c.cumulativeDY == 0 {
		if c.OnSelect != nil {
			c.OnSelect()
		}
		return true
	}
	return false
}

// CumulativeDX returns the net drag delta in X since the most recent
// Press call.
func (c *StepCard) CumulativeDX() int {
	if c == nil {
		return 0
	}
	return c.cumulativeDX
}

// CumulativeDY returns the net drag delta in Y since the most recent
// Press call.
func (c *StepCard) CumulativeDY() int {
	if c == nil {
		return 0
	}
	return c.cumulativeDY
}

func (c *StepCard) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	bg := c.BgColor
	if c.IsActive {
		bg = c.AccentColor
	}
	if c.Selected {
		bg = c.AccentColor
	}
	pixelforge.SetColor(bg)
	pixelforge.RectFill(0, 0, c.W-1, c.H-1)

	// Header (Kind) and body (label).
	font := pgui.DefaultFont()
	pixelforge.SetColor(c.TextColor)
	_, _ = font.Print(c.Kind, 4, 4)
	_, _ = font.Print(c.Label, 4, 16)

	if c.IsActive {
		pixelforge.SetColor(c.TextColor)
		pixelforge.RectFill(0, c.H-2, c.W-1, c.H-1)
	}
}
