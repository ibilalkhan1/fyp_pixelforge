package capture

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

// AttachTimeline wires a reusable widgets.Timeline to a Recorder. The
// returned Timeline is owned by the caller (typically the Capture
// workspace); its OnScrub reapplies the captured frame to the live
// screen and its OnMarkRange notifies a caller-supplied handler.
//
// The workspace re-syncs the timeline's Frames each Update so the
// scrubber tracks ring-buffer growth.
func AttachTimeline(tl *pguiwidgets.Timeline, rec *Recorder, onMark func(start, end int)) {
	tl.SetFrames(rec.FrameCount())
	tl.OnScrub = func(idx int) {
		ApplyFrameToScreen(rec, idx)
	}
	tl.OnMarkRange = func(start, end int) {
		if onMark != nil {
			onMark(start, end)
		}
	}
}

// ApplyFrameToScreen rehydrates the live screen with the captured
// frame at idx. Mirrors the piscope showCurrent pattern: SetData +
// restore palette + mapping is the cheapest "show frame N" path.
func ApplyFrameToScreen(rec *Recorder, idx int) {
	frame := rec.FrameAt(idx)
	if frame == nil {
		return
	}
	screen := pixelforge.Screen()
	if screen.W() == frame.Canvas.W() && screen.H() == frame.Canvas.H() {
		screen.SetData(frame.Canvas.Data())
	}
	pixelforge.Palette = frame.Palette
	pixelforge.PaletteMapping = frame.PaletteMapping
}

// SyncTimelineFrames updates the timeline's frame count to match the
// recorder's current ring size. Called by the Capture workspace from
// Update so the timeline tracks live capture growth.
func SyncTimelineFrames(tl *pguiwidgets.Timeline, rec *Recorder) {
	if tl.Frames() != rec.FrameCount() {
		tl.SetFrames(rec.FrameCount())
	}
}
