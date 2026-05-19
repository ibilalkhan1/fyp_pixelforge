package pixelforge_replay_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_replay"
)

// makeMixedTrace builds a Trace with `n` frames of varied input —
// some empty, some single-key, some with multi-key + gamepad state.
// Used by the round-trip + large-file tests to exercise the full
// encode/decode surface in one shot.
func makeMixedTrace(n int) *pixelforge_replay.Trace {
	frames := make([]pixelforge_replay.TraceFrame, 0, n)
	for i := 0; i < n; i++ {
		var f pixelforge_replay.TraceFrame
		f.Tick = uint64(i)
		switch i % 4 {
		case 0:
			// empty input
		case 1:
			f.Keys = []ebiten.Key{ebiten.KeySpace}
		case 2:
			f.Keys = []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowUp}
		case 3:
			f.Keys = []ebiten.Key{ebiten.KeyA}
			f.Pad = &pixelforge_render.GamepadState{
				Buttons: []ebiten.GamepadButton{ebiten.GamepadButton0},
				LeftX:   0.5,
				LeftY:   -0.25,
			}
		}
		frames = append(frames, f)
	}
	return &pixelforge_replay.Trace{
		Meta: pixelforge_replay.TraceMeta{
			Game:          "trace_test",
			Seed:          42,
			Width:         320,
			Height:        180,
			TPS:           60,
			DurationTicks: uint64(n),
		},
		Frames: frames,
	}
}

func TestTrace_RoundTrip(t *testing.T) {
	want := makeMixedTrace(100)

	var buf bytes.Buffer
	require.NoError(t, want.Encode(&buf))

	got, err := pixelforge_replay.LoadTrace(&buf)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, want.Meta, got.Meta)
	require.Equal(t, len(want.Frames), len(got.Frames), "frame count must round-trip")
	for i := range want.Frames {
		assert.Equal(t, want.Frames[i].Tick, got.Frames[i].Tick, "frame %d tick", i)
		assert.Equal(t, want.Frames[i].Keys, got.Frames[i].Keys, "frame %d keys", i)
		if want.Frames[i].Pad == nil {
			assert.Nil(t, got.Frames[i].Pad, "frame %d pad", i)
		} else {
			require.NotNil(t, got.Frames[i].Pad, "frame %d pad", i)
			assert.Equal(t, want.Frames[i].Pad.Buttons, got.Frames[i].Pad.Buttons)
			assert.InDelta(t, want.Frames[i].Pad.LeftX, got.Frames[i].Pad.LeftX, 1e-9)
			assert.InDelta(t, want.Frames[i].Pad.LeftY, got.Frames[i].Pad.LeftY, 1e-9)
		}
	}
}

func TestTrace_EmptyInputCompressionDecode(t *testing.T) {
	raw := strings.Join([]string{
		`{"v":1,"meta":{"game":"hold_test","width":320,"height":180,"tps":60,"duration_ticks":12}}`,
		`{"tick":47,"keys":[],"pad":null,"hold":12}`,
	}, "\n") + "\n"

	got, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, got.Frames, 12, "hold:12 must expand to 12 frames")
	for i, f := range got.Frames {
		assert.Equal(t, uint64(47+i), f.Tick, "frame %d tick", i)
		assert.Empty(t, f.Keys, "frame %d keys", i)
		assert.Nil(t, f.Pad, "frame %d pad", i)
	}
}

func TestTrace_MalformedLine(t *testing.T) {
	raw := strings.Join([]string{
		`{"v":1,"meta":{"game":"x","width":320,"height":180,"tps":60,"duration_ticks":2}}`,
		`{"tick":0,"keys":[],"pad":null}`,
		`this is not json`,
	}, "\n") + "\n"

	_, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.Error(t, err)
	assert.True(t, errors.Is(err, pixelforge_replay.ErrMalformedTrace), "want ErrMalformedTrace, got %v", err)
	assert.Contains(t, err.Error(), "line 3", "error must include 1-based line number")
}

func TestTrace_VersionMismatch(t *testing.T) {
	raw := `{"v":2,"meta":{"game":"vfuture","width":320,"height":180,"tps":60,"duration_ticks":0}}` + "\n"

	_, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.Error(t, err)
	assert.True(t, errors.Is(err, pixelforge_replay.ErrTraceVersion), "want ErrTraceVersion, got %v", err)
}

func TestTrace_LargeFile(t *testing.T) {
	// 5400 frames = 90 seconds at 60 TPS (the Asteroids-proof
	// recording length). Round-trip exercises the scanner buffer
	// behaviour on the full plan-target trace size.
	want := makeMixedTrace(5400)

	var buf bytes.Buffer
	require.NoError(t, want.Encode(&buf))

	got, err := pixelforge_replay.LoadTrace(&buf)
	require.NoError(t, err)
	require.Equal(t, len(want.Frames), len(got.Frames))
	// Spot-check the first / middle / last frame rather than
	// asserting every field on 5400 frames (the round-trip test
	// already covers per-field equality on 100 frames).
	for _, i := range []int{0, 2700, 5399} {
		assert.Equal(t, want.Frames[i].Tick, got.Frames[i].Tick)
		assert.Equal(t, want.Frames[i].Keys, got.Frames[i].Keys)
	}
}

func TestTrace_UnknownKeyNames(t *testing.T) {
	raw := strings.Join([]string{
		`{"v":1,"meta":{"game":"unk","width":320,"height":180,"tps":60,"duration_ticks":1}}`,
		`{"tick":0,"keys":["Joystick99","Space"],"pad":null}`,
	}, "\n") + "\n"

	got, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.NoError(t, err, "unknown key names must not error")
	require.Len(t, got.Frames, 1)
	// The unknown name is dropped; the recognised "Space" survives.
	assert.Equal(t, []ebiten.Key{ebiten.KeySpace}, got.Frames[0].Keys)
}

func TestTrace_MissingMeta(t *testing.T) {
	// A trace whose first line is a frame (no meta header at all)
	// must return ErrMissingMeta.
	raw := `{"tick":0,"keys":[],"pad":null}` + "\n"

	_, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.Error(t, err)
	assert.True(t, errors.Is(err, pixelforge_replay.ErrMissingMeta), "want ErrMissingMeta, got %v", err)
}

func TestTrace_KeyNameAliases(t *testing.T) {
	// Hand-authored traces may use "KeyA" instead of the canonical
	// "A". The decoder accepts both forms; the test confirms the
	// alias path lands on the same ebiten.KeyA constant.
	raw := strings.Join([]string{
		`{"v":1,"meta":{"game":"alias","width":320,"height":180,"tps":60,"duration_ticks":1}}`,
		`{"tick":0,"keys":["KeyA","KeyD"],"pad":null}`,
	}, "\n") + "\n"

	got, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, got.Frames, 1)
	assert.Equal(t, []ebiten.Key{ebiten.KeyA, ebiten.KeyD}, got.Frames[0].Keys)
}

func TestTrace_EmptyTraceFrames(t *testing.T) {
	// Meta-only file (no frame lines) decodes to a valid Trace
	// with zero frames. Used by the Replayer.Run empty-trace test.
	raw := `{"v":1,"meta":{"game":"empty","width":320,"height":180,"tps":60,"duration_ticks":0}}` + "\n"

	got, err := pixelforge_replay.LoadTrace(strings.NewReader(raw))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Empty(t, got.Frames)
}
