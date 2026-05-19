package pixelforge_replay_test

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_replay"
)

func TestRecorder_Tick_AdvancesCounter(t *testing.T) {
	rec := pixelforge_replay.NewRecorder(pixelforge_replay.TraceMeta{
		Game: "recorder_advances", Width: 320, Height: 180, TPS: 60,
	})

	for i := 0; i < 5; i++ {
		rec.Tick(pixelforge_render.InputFrame{})
	}

	frames := rec.Frames()
	require.Len(t, frames, 5)
	for i, f := range frames {
		assert.Equal(t, uint64(i), f.Tick, "frame %d", i)
	}
}

func TestRecorder_Flush_Roundtrip(t *testing.T) {
	rec := pixelforge_replay.NewRecorder(pixelforge_replay.TraceMeta{
		Game: "recorder_roundtrip", Width: 320, Height: 180, TPS: 60, Seed: 7,
	})

	// 100 ticks of varied input — exercises every InputFrame
	// shape the recorder is expected to handle.
	for i := 0; i < 100; i++ {
		var f pixelforge_render.InputFrame
		switch i % 4 {
		case 0:
			// empty
		case 1:
			f.Keys = []ebiten.Key{ebiten.KeySpace}
		case 2:
			f.Keys = []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowUp}
		case 3:
			f.Pad = &pixelforge_render.GamepadState{
				Buttons: []ebiten.GamepadButton{ebiten.GamepadButton1},
				LeftX:   0.75,
			}
		}
		rec.Tick(f)
	}

	var buf bytes.Buffer
	require.NoError(t, rec.Flush(&buf))

	got, err := pixelforge_replay.LoadTrace(&buf)
	require.NoError(t, err)
	require.Len(t, got.Frames, 100)
	assert.Equal(t, uint64(100), got.Meta.DurationTicks, "Flush must rewrite DurationTicks to actual count")

	// Spot-check the four mod-4 buckets to confirm the round-trip
	// preserves each kind of input.
	assert.Empty(t, got.Frames[0].Keys)
	assert.Equal(t, []ebiten.Key{ebiten.KeySpace}, got.Frames[1].Keys)
	assert.Equal(t, []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowUp}, got.Frames[2].Keys)
	require.NotNil(t, got.Frames[3].Pad)
	assert.Equal(t, []ebiten.GamepadButton{ebiten.GamepadButton1}, got.Frames[3].Pad.Buttons)
	assert.InDelta(t, 0.75, got.Frames[3].Pad.LeftX, 1e-9)
}

func TestRecorder_Tick_DefensiveCopy(t *testing.T) {
	// Tick must deep-copy the InputFrame's Keys slice so a later
	// caller mutation does not retroactively edit the recorded
	// frame. Without the copy this test would see the mutated
	// key list in the recorded frame.
	rec := pixelforge_replay.NewRecorder(pixelforge_replay.TraceMeta{
		Game: "recorder_copy", Width: 320, Height: 180, TPS: 60,
	})

	keys := []ebiten.Key{ebiten.KeySpace}
	rec.Tick(pixelforge_render.InputFrame{Keys: keys})
	keys[0] = ebiten.KeyArrowDown

	frames := rec.Frames()
	require.Len(t, frames, 1)
	assert.Equal(t, []ebiten.Key{ebiten.KeySpace}, frames[0].Keys,
		"recorded Keys must not alias the caller's slice")
}

func TestRecorder_Trace_LiveSnapshot(t *testing.T) {
	rec := pixelforge_replay.NewRecorder(pixelforge_replay.TraceMeta{
		Game: "recorder_trace", Width: 320, Height: 180, TPS: 60,
	})
	rec.Tick(pixelforge_render.InputFrame{Keys: []ebiten.Key{ebiten.KeySpace}})
	rec.Tick(pixelforge_render.InputFrame{})

	tr := rec.Trace()
	require.NotNil(t, tr)
	assert.Equal(t, uint64(2), tr.Meta.DurationTicks)
	require.Len(t, tr.Frames, 2)
	assert.Equal(t, []ebiten.Key{ebiten.KeySpace}, tr.Frames[0].Keys)
	assert.Empty(t, tr.Frames[1].Keys)
}
