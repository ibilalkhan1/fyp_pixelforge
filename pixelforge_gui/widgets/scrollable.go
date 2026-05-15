package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// ScrollStep is the pixel distance one wheel notch moves the content.
const ScrollStep = 12

// ScrollableOptions configures a Scrollable.
type ScrollableOptions struct {
	ContentH       int              // total height of the inner content
	BgColor        pixelforge.Color // viewport background colour
	ScrollBarColor pixelforge.Color // scroll bar foreground
}

// Scrollable is a vertically-scrolling container. It owns an internal
// scroll offset and clips its content to its Area. Callers feed wheel
// input through Scroll(delta) — that keeps the widget independent of
// the Ebitengine wheel API.
type Scrollable struct {
	*pgui.Element

	ContentH       int
	BgColor        pixelforge.Color
	ScrollBarColor pixelforge.Color

	offset int

	// Content is a child element that callers attach their inner widgets
	// to. Its Y coordinate is updated each frame to reflect the offset.
	Content *pgui.Element
}

// NewScrollable constructs a Scrollable rooted at (x, y, w, h) with the
// given total content height.
func NewScrollable(x, y, w, h int, opts ScrollableOptions) *Scrollable {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.ScrollBarColor == 0 {
		opts.ScrollBarColor = 6
	}
	s := &Scrollable{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		ContentH:       opts.ContentH,
		BgColor:        opts.BgColor,
		ScrollBarColor: opts.ScrollBarColor,
	}
	// Content child starts at (0, 0) and is repositioned as offset
	// changes. Width matches the viewport minus the 4px scroll bar gutter.
	contentW := w - 4
	if contentW < 0 {
		contentW = 0
	}
	contentH := opts.ContentH
	if contentH < h {
		contentH = h
	}
	s.Content = pgui.Attach(s.Element, 0, 0, contentW, contentH)
	s.Element.OnDraw = func(ev pgui.DrawEvent) {
		s.draw()
	}
	return s
}

// Scroll advances the internal offset by delta pixels (positive = down).
// The offset is clamped so the content never scrolls past its extents.
func (s *Scrollable) Scroll(delta int) {
	s.offset += delta
	s.clamp()
	s.Content.Y = -s.offset
}

// ScrollBy moves the content by `notches` wheel notches (1 notch =
// ScrollStep pixels). Negative values scroll up.
func (s *Scrollable) ScrollBy(notches int) {
	s.Scroll(notches * ScrollStep)
}

// SetContentH updates the content height; offset is re-clamped.
func (s *Scrollable) SetContentH(h int) {
	s.ContentH = h
	if h > s.H {
		s.Content.H = h
	} else {
		s.Content.H = s.H
	}
	s.clamp()
	s.Content.Y = -s.offset
}

// Offset returns the current scroll offset in pixels.
func (s *Scrollable) Offset() int { return s.offset }

// MaxOffset returns the maximum scroll offset (== content overflow).
func (s *Scrollable) MaxOffset() int {
	if s.ContentH <= s.H {
		return 0
	}
	return s.ContentH - s.H
}

func (s *Scrollable) clamp() {
	if s.offset < 0 {
		s.offset = 0
	}
	max := s.MaxOffset()
	if s.offset > max {
		s.offset = max
	}
}

func (s *Scrollable) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Viewport background.
	pixelforge.SetColor(s.BgColor)
	pixelforge.RectFill(0, 0, s.W-1, s.H-1)

	if s.ContentH <= s.H {
		return
	}
	// Scroll bar at the right edge. Thumb height proportional to viewport
	// over content.
	barX := s.W - 4
	pixelforge.SetColor(s.ScrollBarColor)
	thumbH := s.H * s.H / s.ContentH
	if thumbH < 4 {
		thumbH = 4
	}
	maxOff := s.MaxOffset()
	thumbY := 0
	if maxOff > 0 {
		thumbY = (s.H - thumbH) * s.offset / maxOff
	}
	pixelforge.RectFill(barX, thumbY, s.W-1, thumbY+thumbH-1)
}
