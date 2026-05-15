package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeline_SetFramesClampsPosition(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 100})
	tl.SetPosition(80)
	tl.SetFrames(50)
	assert.Equal(t, 49, tl.Position(), "position clamps to new frames-1")
}

func TestTimeline_SetPositionClamps(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 100})
	tl.SetPosition(-5)
	assert.Equal(t, 0, tl.Position())
	tl.SetPosition(999)
	assert.Equal(t, 99, tl.Position())
}

func TestTimeline_ZeroFramesIgnoresPosition(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 0})
	tl.SetPosition(50)
	assert.Equal(t, 0, tl.Position())
}

func TestTimeline_MarkRangeOrders(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 100})
	tl.SetMarkRange(80, 20)
	s, e := tl.MarkRange()
	assert.Equal(t, 20, s)
	assert.Equal(t, 80, e)
}

func TestTimeline_ClearMark(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 100})
	tl.SetMarkRange(10, 20)
	tl.ClearMark()
	s, e := tl.MarkRange()
	assert.Equal(t, -1, s)
	assert.Equal(t, -1, e)
}

func TestTimeline_ScrubFiresCallback(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 200})
	got := -1
	tl.OnScrub = func(idx int) { got = idx }
	tl.Scrub(50)
	assert.Equal(t, 50, got)
}

func TestTimeline_XToFrame(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 200})
	// Halfway across the strip → frame ~100.
	assert.Equal(t, 100, tl.xToFrame(50))
	// Out-of-range x clamps.
	assert.Equal(t, 0, tl.xToFrame(-10))
	assert.Equal(t, 199, tl.xToFrame(500))
}

func TestTimeline_FrameToX(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 200})
	assert.Equal(t, 50, tl.frameToX(100))
	assert.Equal(t, 0, tl.frameToX(0))
}

func TestTimeline_SetMarkRangeClamps(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 30})
	tl.SetMarkRange(-5, 999)
	s, e := tl.MarkRange()
	assert.Equal(t, 0, s)
	assert.Equal(t, 29, e)
}

func TestTimeline_DefaultFramesPerTick(t *testing.T) {
	tl := NewTimeline(0, 0, 100, 20, TimelineOptions{Frames: 200})
	assert.Equal(t, 30, tl.framesPerTick)
}
