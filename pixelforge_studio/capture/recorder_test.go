package capture

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ensureScreen(t *testing.T) {
	t.Helper()
	if pixelforge.Screen().W() == 0 || pixelforge.Screen().H() == 0 {
		pixelforge.SetScreenSize(8, 6)
	}
}

func TestRecorder_AcceptsAndOrdersFrames(t *testing.T) {
	ensureScreen(t)
	r := New(5)
	for i := 0; i < 3; i++ {
		r.SaveFrame()
	}
	assert.Equal(t, 3, r.FrameCount())
	assert.NotNil(t, r.MostRecent())
	assert.Equal(t, 2, r.MostRecent().FrameNumber)
}

func TestRecorder_EvictsOldFramesAtBudget(t *testing.T) {
	ensureScreen(t)
	r := New(5)
	for i := 0; i < 8; i++ {
		r.SaveFrame()
	}
	frames := r.Frames()
	require.Len(t, frames, 5)
	assert.Equal(t, 3, frames[0].FrameNumber, "oldest 3 should have evicted")
	assert.Equal(t, 7, frames[len(frames)-1].FrameNumber)
}

func TestRecorder_NewZeroBudgetPanics(t *testing.T) {
	assert.Panics(t, func() { New(0) })
	assert.Panics(t, func() { New(-1) })
}

func TestRecorder_RecordedEventBelongsToCurrentFrame(t *testing.T) {
	ensureScreen(t)
	r := New(4)
	r.RecordEvent("loop", "EventInit")
	r.SaveFrame() // captures the event into frame 0
	r.SaveFrame() // frame 1 with no events
	frames := r.Frames()
	require.Len(t, frames, 2)
	require.Len(t, frames[0].Events, 1)
	assert.Equal(t, "loop", frames[0].Events[0].Target)
	assert.Len(t, frames[1].Events, 0)
}

func TestRecorder_RecordedInputBelongsToCurrentFrame(t *testing.T) {
	ensureScreen(t)
	r := New(4)
	r.RecordInput("key", "down:32")
	r.SaveFrame()
	r.SaveFrame()
	frames := r.Frames()
	require.Len(t, frames, 2)
	require.Len(t, frames[0].Inputs, 1)
	assert.Equal(t, "key", frames[0].Inputs[0].Target)
	assert.Empty(t, frames[1].Inputs)
}

func TestRecorder_ResetEmptiesRingKeepsRunningState(t *testing.T) {
	ensureScreen(t)
	r := New(3)
	for i := 0; i < 3; i++ {
		r.SaveFrame()
	}
	r.Reset()
	assert.Equal(t, 0, r.FrameCount())
	r.SaveFrame()
	frames := r.Frames()
	require.Len(t, frames, 1)
	assert.Equal(t, 0, frames[0].FrameNumber, "frame numbers should restart at 0")
}

func TestRecorder_SetBudgetDropsExistingFrames(t *testing.T) {
	ensureScreen(t)
	r := New(3)
	for i := 0; i < 3; i++ {
		r.SaveFrame()
	}
	r.SetBudget(8)
	assert.Equal(t, 0, r.FrameCount(), "resizing drops existing history")
	assert.Equal(t, 8, r.Budget())
}

func TestRecorder_StartIsIdempotent(t *testing.T) {
	ensureScreen(t)
	r := New(4)
	r.Start()
	assert.True(t, r.Running())
	r.Start() // second call should be a no-op
	assert.True(t, r.Running())
	r.Stop()
	assert.False(t, r.Running())
	r.Stop() // also idempotent
	assert.False(t, r.Running())
}

func TestRecorder_FrameAtBounds(t *testing.T) {
	ensureScreen(t)
	r := New(4)
	r.SaveFrame()
	r.SaveFrame()
	assert.Nil(t, r.FrameAt(-1))
	assert.Nil(t, r.FrameAt(99))
	assert.NotNil(t, r.FrameAt(0))
	assert.NotNil(t, r.FrameAt(1))
}

func TestRecorder_InputCountAcrossManyFrames(t *testing.T) {
	ensureScreen(t)
	r := New(200)
	for tick := 0; tick < 100; tick++ {
		r.RecordInput("key", "tick")
		r.RecordInput("mouse.move", "0,0")
		r.RecordInput("pad.button", "down")
		r.SaveFrame()
	}
	total := 0
	for _, f := range r.Frames() {
		total += len(f.Inputs)
	}
	assert.Equal(t, 300, total)
}
