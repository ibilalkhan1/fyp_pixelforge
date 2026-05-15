package capture

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestAttachTimeline_ScrubInvokesApply(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(4)
	pixelforge.SetColor(5)
	pixelforge.RectFill(0, 0, 3, 3)
	rec.SaveFrame()
	pixelforge.SetColor(11)
	pixelforge.RectFill(0, 0, 3, 3)
	rec.SaveFrame()

	tl := pguiwidgets.NewTimeline(0, 0, 100, 20, pguiwidgets.TimelineOptions{})
	AttachTimeline(tl, rec, nil)
	require.Equal(t, 2, tl.Frames())

	// Scrub to frame 0 — screen should show color 5.
	tl.Scrub(0)
	assert.Equal(t, pixelforge.Color(5), pixelforge.GetPixel(0, 0))
}

func TestAttachTimeline_OnMarkPropagates(t *testing.T) {
	pixelforge.SetScreenSize(4, 4)
	rec := New(4)
	rec.SaveFrame()
	rec.SaveFrame()
	rec.SaveFrame()

	tl := pguiwidgets.NewTimeline(0, 0, 100, 20, pguiwidgets.TimelineOptions{})
	gotStart, gotEnd := -1, -1
	AttachTimeline(tl, rec, func(s, e int) {
		gotStart, gotEnd = s, e
	})
	tl.SetMarkRange(10, 0) // out-of-order is corrected
	// SetMarkRange is the imperative path; explicit OnMarkRange fires
	// via UpdateDrag's release. Simulate by directly calling the
	// widget's callback.
	require.NotNil(t, tl.OnMarkRange)
	tl.OnMarkRange(0, 2)
	assert.Equal(t, 0, gotStart)
	assert.Equal(t, 2, gotEnd)
}

func TestSyncTimelineFrames_TracksRecorder(t *testing.T) {
	pixelforge.SetScreenSize(2, 2)
	rec := New(8)
	tl := pguiwidgets.NewTimeline(0, 0, 100, 20, pguiwidgets.TimelineOptions{})
	SyncTimelineFrames(tl, rec)
	assert.Equal(t, 0, tl.Frames())

	rec.SaveFrame()
	rec.SaveFrame()
	rec.SaveFrame()
	SyncTimelineFrames(tl, rec)
	assert.Equal(t, 3, tl.Frames())
}
