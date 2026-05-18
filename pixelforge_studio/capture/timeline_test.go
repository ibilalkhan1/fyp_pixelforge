package capture

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

// U7 rewrote the capture timeline on ImGui — the pguiwidgets.Timeline
// widget and the AttachTimeline / SyncTimelineFrames helpers retired.
// The surviving timeline API is ApplyFrameToScreen, which both the
// ImGui SliderInt (via Workspace.SetScrubPos) and any future tooling
// call to rehydrate a recorded frame.

func TestApplyFrameToScreen_RestoresCanvas(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(4)
	// Frame 0: solid color 7.
	pixelforge.SetColor(7)
	pixelforge.RectFill(0, 0, 3, 3)
	rec.SaveFrame()
	// Frame 1: solid color 3.
	pixelforge.SetColor(3)
	pixelforge.RectFill(0, 0, 3, 3)
	rec.SaveFrame()

	// Clear to color 0; then apply frame 0; verify it's all 7s.
	pixelforge.SetColor(0)
	pixelforge.RectFill(0, 0, 3, 3)
	ApplyFrameToScreen(rec, 0)
	assert.Equal(t, pixelforge.Color(7), pixelforge.GetPixel(2, 2))

	// Apply frame 1 → all 3s.
	ApplyFrameToScreen(rec, 1)
	assert.Equal(t, pixelforge.Color(3), pixelforge.GetPixel(1, 1))
}

func TestApplyFrameToScreen_OutOfRangeNoOp(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(4)
	rec.SaveFrame()
	// Should not panic; should not change anything.
	ApplyFrameToScreen(rec, -1)
	ApplyFrameToScreen(rec, 99)
}
