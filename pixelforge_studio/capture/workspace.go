package capture

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Workspace is the Capture workspace M4 promotes in place of the M3
// placeholder. It implements editor.CanvasWorkspace so the canvas-resident
// chrome path renders the workspace alongside Scene and Palette.
//
// The workspace owns one Recorder and re-registers by name during
// RegisterWith — the existing M3 placeholder is replaced cleanly via
// the idempotent editor.RegisterWorkspace seam.
type Workspace struct {
	recorder *Recorder

	// Capture state surfaced through the workspace UI.
	markStart int // -1 when no mark set
	markEnd   int
	scrubPos  int // -1 means "follow live"

	// Tool palette buttons surfaced as click regions in DrawCanvas;
	// the slice mirrors the order rendered so Update can hit-test
	// against them without reflowing layout.
	toolButtons []toolButton

	// timeline is the canvas-resident scrub strip the user drags.
	// Owned here; drawn into the reserved timeline region of the
	// workspace by DrawCanvas (U37).
	timeline *pguiwidgets.Timeline

	// Status line published from tool actions ("clip saved", "GIF
	// exported", etc.). Renders in the workspace footer.
	statusLine string
}

type toolButton struct {
	label   string
	rect    widgets.Rect
	onPress func(*editor.Editor)
}

// NewWorkspace constructs a Capture workspace with a fresh recorder.
// budget is the recorder's ring size; pass editor.Settings().CaptureBudgetFrames
// (or DefaultBudgetFrames when bootstrapping outside an Editor).
func NewWorkspace(budget int) *Workspace {
	if budget <= 0 {
		budget = DefaultBudgetFrames
	}
	w := &Workspace{
		recorder:  New(budget),
		markStart: -1,
		markEnd:   -1,
		scrubPos:  -1,
	}
	w.timeline = pguiwidgets.NewTimeline(0, 0, 0, 0, pguiwidgets.TimelineOptions{})
	AttachTimeline(w.timeline, w.recorder, func(s, e int) {
		w.SetMarkRange(s, e)
		w.statusLine = fmt.Sprintf("mark: [%d, %d]", s, e)
	})
	return w
}

// Timeline exposes the workspace's underlying timeline widget for
// tests and for caller-driven scrubbing.
func (w *Workspace) Timeline() *pguiwidgets.Timeline { return w.timeline }

// Recorder exposes the workspace's recorder so other capture-package
// units (timeline scrub, regression promotion, exports) can reach it.
func (w *Workspace) Recorder() *Recorder { return w.recorder }

// Name returns the stable workspace identifier.
func (w *Workspace) Name() string { return "capture" }

// DisplayName returns the tab strip label.
func (w *Workspace) DisplayName() string { return "Capture" }

// MarkRange returns the current marked range. (-1, -1) means no mark.
func (w *Workspace) MarkRange() (start, end int) { return w.markStart, w.markEnd }

// SetMarkRange records a marked range. Out-of-order endpoints swap so
// callers always get start <= end downstream.
func (w *Workspace) SetMarkRange(start, end int) {
	if start > end {
		start, end = end, start
	}
	w.markStart, w.markEnd = start, end
}

// ClearMark removes the marked range.
func (w *Workspace) ClearMark() { w.markStart, w.markEnd = -1, -1 }

// ScrubPos returns the current scrub position, or -1 when following live.
func (w *Workspace) ScrubPos() int { return w.scrubPos }

// SetScrubPos updates the scrub position. -1 follows live.
func (w *Workspace) SetScrubPos(p int) { w.scrubPos = p }

// SetStatus surfaces a one-liner in the workspace footer (e.g. "clip
// saved", "GIF exported"). Cleared by the next Update tick once the
// caller's action completes, so calling sites should re-publish each
// time they want the line to stay visible.
func (w *Workspace) SetStatus(msg string) { w.statusLine = msg }

// Status returns the most recent status message for tests.
func (w *Workspace) Status() string { return w.statusLine }

// Update keeps the recorder fed and routes capture-specific keymap
// actions. The recorder runs whenever there's a project loaded; the
// workspace transitions automatically.
func (w *Workspace) Update(e *editor.Editor) {
	if e == nil {
		return
	}
	// Drive the recorder. piloop's EventLateDraw subscription handles
	// most captures; but during tests and on builds where the
	// loop tick isn't firing, we also save here so the workspace is
	// observable in headless contexts.
	if e.Project() == nil {
		return
	}
	if !w.recorder.Running() {
		w.recorder.Start()
	}
	SyncTimelineFrames(w.timeline, w.recorder)
	w.timeline.UpdateDrag()

	// Keymap-driven mark toggle.
	if e.KeyMap() != nil && e.KeyMap().JustPressed("capture.toggle_mark") {
		recent := w.recorder.FrameCount() - 1
		if w.markStart < 0 {
			w.markStart = recent
		} else {
			w.markEnd = recent
			if w.markStart > w.markEnd {
				w.markStart, w.markEnd = w.markEnd, w.markStart
			}
		}
	}
	if e.KeyMap() != nil && e.KeyMap().JustPressed("capture.clear_mark") {
		w.ClearMark()
	}

	// Tool buttons (single-shot left-click).
	if buttonJustClicked(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for _, b := range w.toolButtons {
			if b.rect.Contains(mx, my) {
				b.onPress(e)
				return
			}
		}
	}
}

// buttonJustClicked indirection lets the workspace tests stub mouse
// input — the editor's tests also use this idiom.
var buttonJustClicked = func(b ebiten.MouseButton) bool {
	// Inline ebitenutil-style check; the workspace doesn't depend on
	// inpututil to keep the import surface minimal in tests.
	return ebiten.IsMouseButtonPressed(b)
}

// Draw renders the workspace via the native overlay path. M3+M4
// hybrid: the canvas path (DrawCanvas) is the primary surface; this
// is here so non-cart code paths see something too.
func (w *Workspace) Draw(dst *ebiten.Image, area widgets.Rect, e *editor.Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	if e == nil || e.Project() == nil {
		ebitenutil.DebugPrintAt(dst, "Capture — (no project)", area.X+8, area.Y+8)
		return
	}
	ebitenutil.DebugPrintAt(dst, "Capture", area.X+8, area.Y+4)
	ebitenutil.DebugPrintAt(dst,
		fmt.Sprintf("%d / %d frames", w.recorder.FrameCount(), w.recorder.Budget()),
		area.X+8, area.Y+18)
}

// DrawCanvas renders the workspace into the editor cart canvas. R1
// dogfooding: panel header, frame counter, mark range markers, tool
// palette, status footer — all engine primitives.
func (w *Workspace) DrawCanvas(rel widgets.Rect, e *editor.Editor) {
	if rel.W <= 0 || rel.H <= 0 || e == nil {
		return
	}
	theme := editor.DefaultEditorTheme()
	if c := e.Cart(); c != nil && c.Theme() != nil {
		theme = c.Theme()
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Background.
	pixelforge.SetColor(theme.BackgroundSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	// Header strip.
	pixelforge.SetColor(theme.PanelHeaderSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+15)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("CAPTURE", rel.X+8, rel.Y+4)

	if e.Project() == nil {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(no project)", rel.X+8, rel.Y+24)
		return
	}

	// Frame counter, top-right.
	counter := fmt.Sprintf("%d / %d", w.recorder.FrameCount(), w.recorder.Budget())
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(counter, rel.X+rel.W-len(counter)*4-8, rel.Y+4)

	// Timeline strip — owned by the canvas-resident Timeline widget
	// (U37). Reposition and re-size each tick so the workspace
	// remains responsive to layout changes.
	timelineY := rel.Y + 28
	timelineH := 20
	w.timeline.X = rel.X + 8
	w.timeline.Y = timelineY
	w.timeline.W = rel.W - 16
	w.timeline.H = timelineH
	w.drawTimelineWidget(theme)

	// Tool palette. The buttons stack horizontally below the timeline.
	w.layoutToolPalette(rel.X+8, timelineY+timelineH+8, rel.W-16, theme)
	for _, b := range w.toolButtons {
		pixelforge.SetColor(theme.AccentSlot)
		pixelforge.RectFill(b.rect.X, b.rect.Y, b.rect.X+b.rect.W-1, b.rect.Y+b.rect.H-1)
		pixelforge.SetColor(theme.TextSlot)
		pixelforge_cofont.Print(b.label, b.rect.X+6, b.rect.Y+4)
	}

	// Status footer.
	if w.statusLine != "" {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print(w.statusLine, rel.X+8, rel.Y+rel.H-12)
	}
}

// drawTimelineWidget paints the Timeline widget into the canvas. The
// widget draws in element-local coordinates, so we shift the camera
// the same way pgui.Element.Update does and call its OnDraw directly.
// The workspace's M3 hybrid render pipeline doesn't yet dispatch
// widget Draws automatically, so this is the explicit bridge.
func (w *Workspace) drawTimelineWidget(theme *editor.EditorTheme) {
	prevCamX, prevCamY := pixelforge.Camera.X, pixelforge.Camera.Y
	defer func() {
		pixelforge.Camera.X, pixelforge.Camera.Y = prevCamX, prevCamY
	}()
	pixelforge.Camera.X -= w.timeline.X
	pixelforge.Camera.Y -= w.timeline.Y
	if w.timeline.OnDraw != nil {
		w.timeline.OnDraw(pgui.DrawEvent{Element: w.timeline.Element})
	}
	// Sync mark range from workspace into the widget so the overlay
	// reflects user-driven mark via the keymap path too.
	if w.markStart >= 0 && w.markEnd >= 0 {
		w.timeline.SetMarkRange(w.markStart, w.markEnd)
	}
	_ = theme
}

func (w *Workspace) layoutToolPalette(x, y, width int, _ *editor.EditorTheme) {
	w.toolButtons = w.toolButtons[:0]
	labels := []struct {
		label  string
		action func(*editor.Editor)
	}{
		{"MARK", func(e *editor.Editor) {
			recent := w.recorder.FrameCount() - 1
			if w.markStart < 0 {
				w.markStart = recent
			} else if w.markEnd < 0 {
				w.markEnd = recent
				if w.markStart > w.markEnd {
					w.markStart, w.markEnd = w.markEnd, w.markStart
				}
			} else {
				w.markStart = recent
				w.markEnd = -1
			}
			w.statusLine = fmt.Sprintf("mark: [%d, %d]", w.markStart, w.markEnd)
		}},
		{"CLEAR", func(e *editor.Editor) {
			w.ClearMark()
			w.statusLine = "mark cleared"
		}},
		{"CLIP", func(e *editor.Editor) {
			// U38 wires this to PromoteRangeToClip; here we surface
			// a status hint so the workspace is observable today.
			w.statusLine = "clip: select a sprite first"
		}},
		{"GIF", func(e *editor.Editor) {
			w.statusLine = "GIF: use Ctrl+Shift+G"
		}},
		{"MP4", func(e *editor.Editor) {
			w.statusLine = "MP4: ffmpeg required"
		}},
		{"REG", func(e *editor.Editor) {
			w.statusLine = "regression: pick a frame"
		}},
		{"REPORT", func(e *editor.Editor) {
			w.statusLine = "bug-repro: use Ctrl+Shift+B"
		}},
	}
	btnW := 56
	btnH := 18
	gap := 4
	cx := x
	for _, l := range labels {
		if cx+btnW > x+width {
			break
		}
		rect := widgets.Rect{X: cx, Y: y, W: btnW, H: btnH}
		// capture loop variable safely (Go 1.22 closes by iteration).
		action := l.action
		w.toolButtons = append(w.toolButtons, toolButton{
			label:   l.label,
			rect:    rect,
			onPress: action,
		})
		cx += btnW + gap
	}
}

// RegisterWith installs the Capture workspace on the editor and starts
// its recorder. Idempotent: re-registering by name replaces the M3
// placeholder via the editor's RegisterWorkspace seam.
func RegisterWith(e *editor.Editor) *Workspace {
	budget := DefaultBudgetFrames
	if e.Settings() != nil && e.Settings().CaptureBudgetFrames > 0 {
		budget = e.Settings().CaptureBudgetFrames
	}
	w := NewWorkspace(budget)
	w.recorder.Start()
	e.RegisterWorkspace(w)
	registerKeymap(e)
	return w
}

// registerKeymap installs the capture.* action namespace. The plan
// reserves the namespace; individual bindings live next to the
// features that need them (U38, U39, U40, U41, U42).
func registerKeymap(e *editor.Editor) {
	km := e.KeyMap()
	if km == nil {
		return
	}
	// Mark / clear are bound now so the workspace is interactive at
	// U36. Save-clip / promote-regression / export / report bindings
	// land alongside their owning units; we register them as
	// reserved actions here so accidental double-bindings surface
	// early.
	km.Register("capture.toggle_mark", editor.Binding{Key: ebiten.KeyM})
	km.Register("capture.clear_mark", editor.Binding{Mods: editor.ModShift, Key: ebiten.KeyM})
	km.Register("capture.save_clip", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyC})
	km.Register("capture.export_gif", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyG})
	km.Register("capture.export_mp4", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyV})
	km.Register("capture.promote_regression", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyR})
	km.Register("capture.bug_report", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyB})
}
