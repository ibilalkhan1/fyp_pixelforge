package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// TimelineOptions configures a Timeline.
type TimelineOptions struct {
	Frames int

	// Pixels per "major" tick. Default 30 frames per tick (= 1s @ 30 TPS).
	FramesPerTick int

	BgColor       pixelforge.Color
	PlayheadColor pixelforge.Color
	TickColor     pixelforge.Color
	MarkColor     pixelforge.Color
}

// Timeline is a scrubbable horizontal strip widget.
//
// API mirrors the surface the M4 plan specifies:
//
//	tl := widgets.NewTimeline(x, y, w, h, opts)
//	tl.SetFrames(n)
//	tl.SetPosition(idx)
//	tl.OnScrub = func(idx int) { ... }
//	tl.OnMarkRange = func(start, end int) { ... }
//
// Drag handling:
//   - Pointer down inside the strip + drag horizontally → OnScrub(idx)
//   - Shift+drag (caller-controlled via SetShiftHeld) → OnMarkRange
//
// The widget is canvas-resident; rendering uses engine primitives
// (RectFill + cofont) so the Capture workspace + Animator + M5
// Behavior recorder share a single implementation.
type Timeline struct {
	*pgui.Element

	frames        int
	framesPerTick int
	position      int
	markStart     int
	markEnd       int

	bgColor       pixelforge.Color
	playheadColor pixelforge.Color
	tickColor     pixelforge.Color
	markColor     pixelforge.Color

	dragging   bool
	dragMark   bool
	dragStartF int

	shiftHeld bool

	// OnScrub fires when the playhead position changes via drag.
	OnScrub func(idx int)
	// OnMarkRange fires on shift-drag end with start <= end.
	OnMarkRange func(start, end int)
}

// NewTimeline constructs a Timeline.
func NewTimeline(x, y, w, h int, opts TimelineOptions) *Timeline {
	if opts.FramesPerTick <= 0 {
		opts.FramesPerTick = 30
	}
	if opts.BgColor == 0 {
		opts.BgColor = 5
	}
	if opts.PlayheadColor == 0 {
		opts.PlayheadColor = 12
	}
	if opts.TickColor == 0 {
		opts.TickColor = 6
	}
	if opts.MarkColor == 0 {
		opts.MarkColor = 9
	}
	t := &Timeline{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		frames:        opts.Frames,
		framesPerTick: opts.FramesPerTick,
		markStart:     -1,
		markEnd:       -1,
		bgColor:       opts.BgColor,
		playheadColor: opts.PlayheadColor,
		tickColor:     opts.TickColor,
		markColor:     opts.MarkColor,
	}
	t.Element.OnDraw = func(ev pgui.DrawEvent) { t.draw() }
	t.Element.OnPress = func(ev pgui.Event) { t.handlePress() }
	t.Element.OnRelease = func(ev pgui.Event) { t.handleRelease() }
	return t
}

// Frames returns the total frame count the timeline represents.
func (t *Timeline) Frames() int { return t.frames }

// SetFrames updates the frame count and clamps the playhead position
// into the new range.
func (t *Timeline) SetFrames(n int) {
	if n < 0 {
		n = 0
	}
	t.frames = n
	if t.position >= n {
		t.position = max0(n - 1)
	}
	if t.markStart >= n {
		t.markStart = max0(n - 1)
	}
	if t.markEnd >= n {
		t.markEnd = max0(n - 1)
	}
}

// Position returns the current playhead frame index.
func (t *Timeline) Position() int { return t.position }

// SetPosition moves the playhead. Out-of-range values clamp.
func (t *Timeline) SetPosition(p int) {
	if t.frames <= 0 {
		t.position = 0
		return
	}
	if p < 0 {
		p = 0
	}
	if p >= t.frames {
		p = t.frames - 1
	}
	t.position = p
}

// MarkRange returns the current marked range. (-1, -1) means no mark.
func (t *Timeline) MarkRange() (int, int) { return t.markStart, t.markEnd }

// SetMarkRange sets the marked range. start > end is corrected.
func (t *Timeline) SetMarkRange(start, end int) {
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if end >= t.frames {
		end = max0(t.frames - 1)
	}
	t.markStart, t.markEnd = start, end
}

// ClearMark removes any marked range.
func (t *Timeline) ClearMark() { t.markStart, t.markEnd = -1, -1 }

// SetShiftHeld controls whether the next drag is treated as a
// mark-range drag. Callers feed this from their input layer (the
// editor's KeyMap knows about shift state already).
func (t *Timeline) SetShiftHeld(held bool) { t.shiftHeld = held }

// Scrub is the imperative path callers can use when they want to drive
// the playhead without going through a drag — e.g. a "rewind to most
// recent" button. Fires OnScrub.
func (t *Timeline) Scrub(idx int) {
	t.SetPosition(idx)
	if t.OnScrub != nil {
		t.OnScrub(t.position)
	}
}

// handlePress begins a drag. Position is set from the click x.
func (t *Timeline) handlePress() {
	if t.frames <= 0 {
		return
	}
	t.dragging = true
	t.dragMark = t.shiftHeld
	mx, _ := pguiPointerLocal(t.Element)
	idx := t.xToFrame(mx)
	t.dragStartF = idx
	if t.dragMark {
		t.markStart = idx
		t.markEnd = idx
	} else {
		t.SetPosition(idx)
		if t.OnScrub != nil {
			t.OnScrub(t.position)
		}
	}
}

// handleRelease completes a drag.
func (t *Timeline) handleRelease() {
	if !t.dragging {
		return
	}
	t.dragging = false
	if t.dragMark {
		// Normalise mark range: start <= end.
		if t.markStart > t.markEnd {
			t.markStart, t.markEnd = t.markEnd, t.markStart
		}
		if t.OnMarkRange != nil {
			t.OnMarkRange(t.markStart, t.markEnd)
		}
		t.dragMark = false
	}
}

// UpdateDrag is called by the host (typically from Update) each tick
// while a drag is active. It updates the playhead (or mark range) based
// on the current pointer x.
func (t *Timeline) UpdateDrag() {
	if !t.dragging || t.frames <= 0 {
		return
	}
	mx, _ := pguiPointerLocal(t.Element)
	idx := t.xToFrame(mx)
	if t.dragMark {
		t.markEnd = idx
		return
	}
	t.SetPosition(idx)
	if t.OnScrub != nil {
		t.OnScrub(t.position)
	}
}

func (t *Timeline) xToFrame(x int) int {
	if t.W <= 0 || t.frames <= 0 {
		return 0
	}
	if x < 0 {
		return 0
	}
	if x >= t.W {
		return t.frames - 1
	}
	return x * t.frames / t.W
}

func (t *Timeline) frameToX(f int) int {
	if t.frames <= 0 {
		return 0
	}
	return f * t.W / t.frames
}

func (t *Timeline) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(t.bgColor)
	pixelforge.RectFill(0, 0, t.W-1, t.H-1)

	if t.frames <= 0 {
		pixelforge.SetColor(t.tickColor)
		pixelforge_cofont.Print("(no frames)", 4, t.H/2-4)
		return
	}

	// Tick marks every framesPerTick frames.
	pixelforge.SetColor(t.tickColor)
	for f := 0; f < t.frames; f += t.framesPerTick {
		x := t.frameToX(f)
		pixelforge.Line(x, 0, x, t.H/2-1)
	}

	// Mark range overlay.
	if t.markStart >= 0 && t.markEnd >= 0 {
		ms := t.frameToX(t.markStart)
		me := t.frameToX(t.markEnd)
		pixelforge.SetColor(t.markColor)
		pixelforge.RectFill(ms, 0, me, t.H-1)
	}

	// Playhead.
	px := t.frameToX(t.position)
	pixelforge.SetColor(t.playheadColor)
	pixelforge.Line(px, 0, px, t.H-1)
}

// Dragging reports whether a drag is currently in progress.
func (t *Timeline) Dragging() bool { return t.dragging }

func max0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}
