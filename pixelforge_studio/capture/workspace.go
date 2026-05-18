// Package capture's workspace.go was rewritten in U7 of the ImGui
// migration plan: the M4 pgui-driven canvas chrome retired in favour
// of an ImGui immediate-mode window. The Capture *substrate*
// (Recorder, ring buffer, event subscriptions) is unchanged — only
// the surface the user sees moves to ImGui primitives.
package capture

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Workspace is the Capture workspace. Implements editor.Workspace so
// it slots into the dockspace, and exposes Render to build the
// ImGui Capture window each frame.
type Workspace struct {
	recorder *Recorder

	// Capture state surfaced through the workspace UI.
	markStart int // -1 when no mark set
	markEnd   int
	scrubPos  int // -1 means "follow live"

	// Status line published from tool actions ("clip saved", "GIF
	// exported", etc.). Renders in the workspace footer.
	statusLine string
}

// NewWorkspace constructs a Capture workspace with a fresh recorder.
// budget is the recorder's ring size; pass editor.Settings().CaptureBudgetFrames
// (or DefaultBudgetFrames when bootstrapping outside an Editor).
func NewWorkspace(budget int) *Workspace {
	if budget <= 0 {
		budget = DefaultBudgetFrames
	}
	return &Workspace{
		recorder:  New(budget),
		markStart: -1,
		markEnd:   -1,
		scrubPos:  -1,
	}
}

// Recorder exposes the workspace's recorder so other capture-package
// units (timeline scrub, regression promotion, exports) can reach it.
func (w *Workspace) Recorder() *Recorder { return w.recorder }

// Name returns the stable workspace identifier.
func (w *Workspace) Name() string { return "capture" }

// DisplayName returns the dock window title.
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

// SetScrubPos updates the scrub position. -1 follows live; values
// inside the recorder range rehydrate the live screen via
// ApplyFrameToScreen.
func (w *Workspace) SetScrubPos(p int) {
	w.scrubPos = p
	if p >= 0 {
		ApplyFrameToScreen(w.recorder, p)
	}
}

// SetStatus surfaces a one-liner in the workspace footer. The next
// Update tick clears it once the caller's action completes.
func (w *Workspace) SetStatus(msg string) { w.statusLine = msg }

// Status returns the most recent status message for tests.
func (w *Workspace) Status() string { return w.statusLine }

// Render builds the Capture ImGui window each frame. The window
// hosts a frame-counter header, a timeline slider, mark/clear/clip/
// export buttons, and a status footer.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	flags := imgui.WindowFlagsNoCollapse
	if !imgui.BeginV(w.DisplayName(), nil, flags) {
		imgui.End()
		e.SetPanelRect(w.DisplayName(), widgets.Rect{})
		return
	}
	defer imgui.End()
	e.CaptureCurrentWindowRect(w.DisplayName())

	if e.Project() == nil {
		imgui.TextDisabled("(no project)")
		return
	}

	frameCount := w.recorder.FrameCount()
	budget := w.recorder.Budget()
	imgui.Text(fmt.Sprintf("CAPTURE — %d / %d frames", frameCount, budget))
	imgui.Separator()

	w.renderTimelineSlider(frameCount)
	w.renderToolButtons(e)

	if w.markStart >= 0 || w.markEnd >= 0 {
		imgui.Text(fmt.Sprintf("Mark: [%d, %d]", w.markStart, w.markEnd))
	}
	if w.statusLine != "" {
		imgui.Separator()
		imgui.TextDisabled(w.statusLine)
	}
}

// renderTimelineSlider emits the SliderInt over the captured frame
// indices. Moving the slider seeks the recorder via SetScrubPos.
func (w *Workspace) renderTimelineSlider(frameCount int) {
	if frameCount <= 0 {
		imgui.TextDisabled("(no captured frames)")
		return
	}
	max := int32(frameCount - 1)
	pos := int32(w.scrubPos)
	if pos < 0 || pos > max {
		pos = max
	}
	if imgui.SliderIntV("Frame", &pos, 0, max, "%d", 0) {
		w.SetScrubPos(int(pos))
	}
}

// renderToolButtons emits the mark / clear / clip / export tool row.
// Each button publishes a status line so the user gets immediate
// feedback even when the underlying action is asynchronous.
func (w *Workspace) renderToolButtons(e *editor.Editor) {
	if imgui.Button("Mark") {
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
	}
	imgui.SameLine()
	if imgui.Button("Clear") {
		w.ClearMark()
		w.statusLine = "mark cleared"
	}
	imgui.SameLine()
	if imgui.Button("Clip") {
		// PromoteRangeToClip lands alongside its full export flow in a
		// later feature unit; until then, surface a hint so the
		// workspace is observable.
		w.statusLine = "clip: select a sprite first"
	}
	imgui.SameLine()
	if imgui.Button("GIF") {
		w.statusLine = "GIF: use Ctrl+Shift+G"
	}
	imgui.SameLine()
	if imgui.Button("MP4") {
		w.statusLine = "MP4: ffmpeg required"
	}
	imgui.SameLine()
	if imgui.Button("Regression") {
		w.statusLine = "regression: pick a frame"
	}
	imgui.SameLine()
	if imgui.Button("Report") {
		w.statusLine = "bug-repro: use Ctrl+Shift+B"
	}
	_ = e
}

// Update keeps the recorder fed and routes capture-specific keymap
// actions. Called by the editor's focused-workspace input dispatch.
func (w *Workspace) Update(e *editor.Editor) {
	if e == nil || e.Project() == nil {
		return
	}
	if !w.recorder.Running() {
		w.recorder.Start()
	}
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
}

// RegisterWith installs the Capture workspace on the editor and starts
// its recorder. Idempotent: re-registering by name replaces any
// previous entry via the editor's RegisterWorkspace seam.
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
// features that need them.
func registerKeymap(e *editor.Editor) {
	km := e.KeyMap()
	if km == nil {
		return
	}
	km.Register("capture.toggle_mark", editor.Binding{Key: ebiten.KeyM})
	km.Register("capture.clear_mark", editor.Binding{Mods: editor.ModShift, Key: ebiten.KeyM})
	km.Register("capture.save_clip", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyC})
	km.Register("capture.export_gif", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyG})
	km.Register("capture.export_mp4", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyV})
	km.Register("capture.promote_regression", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyR})
	km.Register("capture.bug_report", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyB})
}
