//go:build !js

package pixelforge_replay_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_replay"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// gameWithOneUpdate mirrors the pattern used by
// pixelforge_render/rendertick_test.go: Ebitengine requires a live
// graphics context to allocate *ebiten.Image and call ReadPixels,
// which only exists inside RunGame. Hosting the testing.M run
// inside one Update + Termination is the standard workaround.
type gameWithOneUpdate struct {
	m    *testing.M
	code int
}

func (g *gameWithOneUpdate) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*gameWithOneUpdate) Draw(*ebiten.Image) {}

func (*gameWithOneUpdate) Layout(int, int) (int, int) {
	return 1, 1
}

func TestMain(m *testing.M) {
	g := &gameWithOneUpdate{m: m, code: 1}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

// newReplayRuntime constructs a *capsuleruntime.Runtime suitable for
// replayer tests: subscribers skipped so the verbs.bus stays clean
// (the replayer's own subscribe is the only consumer), nil assets
// FS, minimal project.
func newReplayRuntime(t *testing.T) *capsuleruntime.Runtime {
	t.Helper()
	p := pixelforge_project.NewProject("replay-test")
	rt, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{
		SkipSubscribers: true,
	})
	require.NoError(t, err)
	require.NotNil(t, rt)
	return rt
}

// installStableDraw rigs the pixelforge globals with a checker-
// pattern Draw whose phase depends on pixelforge.Frame, so the
// replayer's per-tick output varies between ticks (and is
// reproducible across runs at the same tick). Mirrors the helper
// in rendertick_test.go but inlined here to avoid widening the
// render package's exported test surface.
func installStableDraw(t *testing.T) func() {
	t.Helper()
	prevInit := pixelforge.Init
	prevUpdate := pixelforge.Update
	prevDraw := pixelforge.Draw

	pixelforge.Init = func() {}
	pixelforge.Update = func() {}
	pixelforge.Draw = func() {
		scr := pixelforge.Screen()
		w, h := scr.W(), scr.H()
		phase := pixelforge.Frame & 1
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if ((x+y)+phase)&1 == 0 {
					scr.Set(x, y, 1)
				} else {
					scr.Set(x, y, 2)
				}
			}
		}
	}

	pixelforge_render.ResetForTest()

	return func() {
		pixelforge.Init = prevInit
		pixelforge.Update = prevUpdate
		pixelforge.Draw = prevDraw
		pixelforge_render.ResetForTest()
	}
}

// makeTraceTicks builds a Trace with `n` simple ticks (alternating
// empty and Space-held). Each ticks at position i so the replayer
// has a contiguous range to iterate.
func makeTraceTicks(n int) *pixelforge_replay.Trace {
	frames := make([]pixelforge_replay.TraceFrame, n)
	for i := 0; i < n; i++ {
		frames[i].Tick = uint64(i)
		if i%2 == 1 {
			frames[i].Keys = []ebiten.Key{ebiten.KeySpace}
		}
	}
	return &pixelforge_replay.Trace{
		Meta: pixelforge_replay.TraceMeta{
			Game:          "replayer_test",
			Width:         320,
			Height:        180,
			TPS:           60,
			DurationTicks: uint64(n),
		},
		Frames: frames,
	}
}

func TestReplayer_Determinism(t *testing.T) {
	defer installStableDraw(t)()
	rt := newReplayRuntime(t)

	// Use a fresh bus so the captured-event list is bounded to
	// what this test publishes (none, in this case — we only care
	// about pixel determinism here).
	piloop.ResetVerbsBusForTest()

	trace := makeTraceTicks(10)
	r := pixelforge_replay.NewReplayer()

	firstFrames, firstEvents, err := r.Run(rt, trace)
	require.NoError(t, err)
	require.Len(t, firstFrames, 10)

	// 9 more invocations; every pixel byte and every event must
	// match the first run exactly.
	for run := 1; run < 10; run++ {
		piloop.ResetVerbsBusForTest()
		gotFrames, gotEvents, err := r.Run(rt, trace)
		require.NoError(t, err)
		require.Len(t, gotFrames, 10)

		for i := range firstFrames {
			if !bytes.Equal(firstFrames[i].Pix, gotFrames[i].Pix) {
				t.Fatalf("run %d frame %d diverged from first run", run, i)
			}
		}
		assert.Equal(t, len(firstEvents), len(gotEvents), "run %d event count diverged", run)
	}
}

func TestReplayer_BusEventCapture(t *testing.T) {
	defer installStableDraw(t)()

	// Reset the bus BEFORE booting the runtime so our subscribe
	// (which Run does) is on the same bus as any publisher the
	// test fires.
	piloop.ResetVerbsBusForTest()
	rt := newReplayRuntime(t)

	// A trace where one tick triggers a fake catalog that
	// publishes a verb event. The "trigger" here is simulated by
	// wiring pixelforge.Update to publish at a known frame; the
	// replayer subscribes via Run and should observe the event
	// with the right tick stamp.
	publishedTick := uint64(7)
	prevUpdate := pixelforge.Update
	pixelforge.Update = func() {
		if uint64(pixelforge.Frame) == publishedTick {
			piloop.VerbsBus().Publish(&piloop.VerbEvent{
				Topic: "motion/jump",
				Args:  map[string]any{"entity": "hero"},
			})
		}
	}
	defer func() { pixelforge.Update = prevUpdate }()

	trace := makeTraceTicks(10)
	r := pixelforge_replay.NewReplayer()

	_, events, err := r.Run(rt, trace)
	require.NoError(t, err)
	require.Len(t, events, 1, "exactly one verb event must be captured")
	assert.Equal(t, publishedTick, events[0].Tick, "captured event must carry the publishing tick")
	assert.Equal(t, "motion/jump", events[0].Event.Topic)
	assert.Equal(t, "hero", events[0].Event.Args["entity"])
}

func TestReplayer_EmptyTrace(t *testing.T) {
	defer installStableDraw(t)()
	piloop.ResetVerbsBusForTest()
	rt := newReplayRuntime(t)

	empty := &pixelforge_replay.Trace{
		Meta: pixelforge_replay.TraceMeta{
			Game: "empty", Width: 320, Height: 180, TPS: 60,
		},
	}

	r := pixelforge_replay.NewReplayer()
	frames, events, err := r.Run(rt, empty)
	require.NoError(t, err)
	assert.Empty(t, frames)
	assert.Empty(t, events)
}

func TestReplayer_NilRuntime(t *testing.T) {
	r := pixelforge_replay.NewReplayer()
	_, _, err := r.Run(nil, &pixelforge_replay.Trace{})
	assert.ErrorIs(t, err, pixelforge_replay.ErrNilRuntime)
}

func TestReplayer_NilTrace(t *testing.T) {
	rt := newReplayRuntime(t)
	r := pixelforge_replay.NewReplayer()
	_, _, err := r.Run(rt, nil)
	assert.ErrorIs(t, err, pixelforge_replay.ErrNilTrace)
}

func TestReplayer_NonContiguousTicks(t *testing.T) {
	// A trace whose frames skip ticks (e.g. tick 0, 5, 10) must
	// still iterate — the Replayer does not synthesize missing
	// frames. The render path accepts any tick value.
	defer installStableDraw(t)()
	piloop.ResetVerbsBusForTest()
	rt := newReplayRuntime(t)

	trace := &pixelforge_replay.Trace{
		Meta: pixelforge_replay.TraceMeta{
			Game: "skip", Width: 320, Height: 180, TPS: 60,
		},
		Frames: []pixelforge_replay.TraceFrame{
			{Tick: 0},
			{Tick: 5},
			{Tick: 10},
		},
	}

	r := pixelforge_replay.NewReplayer()
	frames, _, err := r.Run(rt, trace)
	require.NoError(t, err)
	require.Len(t, frames, 3, "Replayer must emit one frame per trace entry, not per logical-tick gap")
}
