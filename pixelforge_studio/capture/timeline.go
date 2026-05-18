// Package capture's timeline.go was rewritten in U7 of the ImGui
// migration plan: the pguiwidgets.Timeline widget retired in favour
// of an ImGui SliderInt embedded in Workspace.renderTimelineSlider.
// The frame-apply helper (ApplyFrameToScreen) survives because the
// underlying screen-rehydration contract is the same regardless of
// which UI surface drives the scrub.
package capture

import "github.com/ibilalkhan1/fyp_pixelforge"

// ApplyFrameToScreen rehydrates the live screen with the captured
// frame at idx. Mirrors the piscope showCurrent pattern: SetData +
// restore palette + mapping is the cheapest "show frame N" path.
//
// The Capture workspace's SliderInt seeks via SetScrubPos, which
// calls this helper directly. Out-of-range idx is a no-op.
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
