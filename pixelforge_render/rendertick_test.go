//go:build !js

package pixelforge_render_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// gameWithOneUpdate is the in-package mirror of
// pixelforge_ebiten/internal/ebitentesting.MainWithRunLoop. We can't
// import that helper because it lives under an internal/ path
// scoped to pixelforge_ebiten — duplicating the four-line shim here
// is cheaper than reorganising the engine's test-helper surface.
//
// Ebiten requires a live graphics context for NewImage + ReadPixels,
// which only exists inside RunGame; we therefore host the entire
// testing.M run inside one Update() callback and terminate after.
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

// TestMain hosts every test inside Ebitengine's game loop because
// ebiten.NewImage + (*Image).ReadPixels require a live graphics
// context. Mirrors the pattern used by pixelforge_ebiten's own test
// file (piebiten_test.go).
func TestMain(m *testing.M) {
	g := &gameWithOneUpdate{m: m, code: 1}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}

// newTestRuntime constructs a *capsuleruntime.Runtime with a fresh
// minimal project that is just big enough for advanceAndRender to
// produce a real frame. Test-only helper to keep each test inline +
// readable.
func newTestRuntime(t *testing.T) *capsuleruntime.Runtime {
	t.Helper()
	p := pixelforge_project.NewProject("rendertick-test")
	rt, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{
		SkipSubscribers: true,
	})
	require.NoError(t, err)
	require.NotNil(t, rt)
	return rt
}

// stableDrawScene installs deterministic Init/Update/Draw callbacks
// on the pixelforge globals. The Draw step paints a checker pattern
// whose phase depends on pixelforge.Frame, so different ticks produce
// visibly different output (used by TestRenderTickAt_TickAdvances).
// The Update step is intentionally a no-op so the only state moving
// between ticks is pixelforge.Frame itself — keeps the determinism
// test free of hidden state.
//
// Returns a cleanup that restores the previous Init/Update/Draw
// hooks; defer it from each test to keep tests independent.
func stableDrawScene(t *testing.T) func() {
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

// TestRenderTickAt_BitIdentical is the load-bearing determinism
// guarantee. Same rt + same tick + same inputs ⇒ byte-identical
// pixel output, asserted across 100 invocations to give a fighting
// chance of catching any hidden non-determinism.
func TestRenderTickAt_BitIdentical(t *testing.T) {
	defer stableDrawScene(t)()
	rt := newTestRuntime(t)

	inputs := pixelforge_render.InputFrame{Keys: []ebiten.Key{ebiten.KeyA}}

	first, err := pixelforge_render.RenderTickAtRGBA(rt, 42, inputs)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotEmpty(t, first.Pix)

	for i := 0; i < 99; i++ {
		next, err := pixelforge_render.RenderTickAtRGBA(rt, 42, inputs)
		require.NoError(t, err)
		require.NotNil(t, next)
		if !bytes.Equal(first.Pix, next.Pix) {
			t.Fatalf("iteration %d diverged: pixel bytes differ from first", i+1)
		}
	}
}

// TestRenderTickAt_TickAdvances ensures the tick counter is actually
// fed through to engine state — a frame at tick 0 differs from one at
// tick 1 because the test Draw paints a phase that depends on
// pixelforge.Frame.
func TestRenderTickAt_TickAdvances(t *testing.T) {
	defer stableDrawScene(t)()
	rt := newTestRuntime(t)

	frame0, err := pixelforge_render.RenderTickAtRGBA(rt, 0, pixelforge_render.InputFrame{})
	require.NoError(t, err)
	frame1, err := pixelforge_render.RenderTickAtRGBA(rt, 1, pixelforge_render.InputFrame{})
	require.NoError(t, err)

	assert.False(t, bytes.Equal(frame0.Pix, frame1.Pix),
		"frame at tick 0 and tick 1 must differ when Draw depends on Frame")
}

// TestRenderTickAt_EmptyInput exercises the zero-value InputFrame
// path: no panic, frame still renders.
func TestRenderTickAt_EmptyInput(t *testing.T) {
	defer stableDrawScene(t)()
	rt := newTestRuntime(t)

	frame, err := pixelforge_render.RenderTickAtRGBA(rt, 7, pixelforge_render.InputFrame{})
	require.NoError(t, err)
	require.NotNil(t, frame)
	assert.NotEmpty(t, frame.Pix, "empty-input frame must still produce pixels")
}

// TestRenderTickAt_NilRuntime returns the documented sentinel
// without panicking.
func TestRenderTickAt_NilRuntime(t *testing.T) {
	_, err := pixelforge_render.RenderTickAtRGBA(nil, 0, pixelforge_render.InputFrame{})
	assert.ErrorIs(t, err, pixelforge_render.ErrNilRuntime)

	err = pixelforge_render.RenderTickAtScreen(nil, ebiten.NewImage(8, 8), 0, pixelforge_render.InputFrame{})
	assert.ErrorIs(t, err, pixelforge_render.ErrNilRuntime)
}

// TestRenderTickAt_NilScreen guards the on-screen entry point's
// destination check.
func TestRenderTickAt_NilScreen(t *testing.T) {
	rt := newTestRuntime(t)
	err := pixelforge_render.RenderTickAtScreen(rt, nil, 0, pixelforge_render.InputFrame{})
	assert.ErrorIs(t, err, pixelforge_render.ErrNilScreen)
}

// TestRenderTickAt_ScreenPathMatchesRGBA confirms the two entry
// points produce equivalent output: drawing into an ebiten.Image via
// RenderTickAtScreen then reading it back yields the same bytes as
// RenderTickAtRGBA. This is the formal expression of "one render
// path, two thin public wrappers" — if the two ever diverged, the
// parity contract would be broken at the seam itself.
func TestRenderTickAt_ScreenPathMatchesRGBA(t *testing.T) {
	defer stableDrawScene(t)()
	rt := newTestRuntime(t)

	scr := pixelforge.Screen()
	w, h := scr.W(), scr.H()

	dst := ebiten.NewImage(w, h)
	err := pixelforge_render.RenderTickAtScreen(rt, dst, 99, pixelforge_render.InputFrame{})
	require.NoError(t, err)

	rgbaFromScreen := make([]byte, w*h*4)
	dst.ReadPixels(rgbaFromScreen)

	rgbaDirect, err := pixelforge_render.RenderTickAtRGBA(rt, 99, pixelforge_render.InputFrame{})
	require.NoError(t, err)

	assert.True(t, bytes.Equal(rgbaFromScreen, rgbaDirect.Pix),
		"on-screen and off-screen paths must produce byte-equal output")
}

// ---- U17: multi-screen camera offset tests -----------------------

// newDKTallRuntime constructs a runtime with a Player entity in a
// 4-screen-tall scene. Used by the camera-offset tests below.
func newDKTallRuntime(t *testing.T, heroY float64) *capsuleruntime.Runtime {
	t.Helper()
	p := pixelforge_project.NewProject("dk-camera-test")
	p.ScreenWidth = 256
	p.ScreenHeight = 240
	p.PhysicsPreset = "dk"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:                "main",
			Name:              "Main",
			GridHeightScreens: 4,
			GridWidthScreens:  1,
			Entities: []pixelforge_project.Entity{
				{
					ID:        "hero",
					Name:      "Hero",
					Archetype: pixelforge_project.ArchetypePlayer,
					Position:  pixelforge_project.EntityPosition{X: 100, Y: heroY},
				},
			},
		},
	}
	rt, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{
		SkipSubscribers: true,
	})
	require.NoError(t, err)
	return rt
}

// TestCameraOffset_MultiScreenFollowsHero asserts the canonical
// "4-screen tall scene with hero at Y=1000" case from U17's plan:
// the camera offset is heroY - screenH/2 (clamped to the valid
// range).
func TestCameraOffset_MultiScreenFollowsHero(t *testing.T) {
	rt := newDKTallRuntime(t, 500)

	pixelforge_render.UpdateCameraOffsetForTest(rt)

	// screenH = 240, heroY = 500 → offset = 500 - 120 = 380.
	// totalH = 4*240 = 960, maxOffset = 960 - 240 = 720, so 380 stays.
	assert.Equal(t, 380, rt.CameraOffsetY,
		"camera should center on hero in middle of level")
}

// TestCameraOffset_SingleScreenIsZero confirms the existing single-
// screen behaviour: GridHeightScreens=1 (or unset) ⇒ no scroll.
func TestCameraOffset_SingleScreenIsZero(t *testing.T) {
	p := pixelforge_project.NewProject("dk-single-screen")
	p.ScreenWidth = 256
	p.ScreenHeight = 240
	p.PhysicsPreset = "dk"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:                "main",
			GridHeightScreens: 1,
			Entities: []pixelforge_project.Entity{
				{
					ID:        "hero",
					Archetype: pixelforge_project.ArchetypePlayer,
					Position:  pixelforge_project.EntityPosition{X: 100, Y: 100},
				},
			},
		},
	}
	rt, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)

	// Pre-pollute to verify the helper writes zero, not just leaves
	// the previous value alone.
	rt.CameraOffsetY = 999
	pixelforge_render.UpdateCameraOffsetForTest(rt)
	assert.Equal(t, 0, rt.CameraOffsetY,
		"single-screen scenes should have zero camera offset")
}

// TestCameraOffset_ClampsAtTop guards the "hero near top of level"
// case: even though heroY - screenH/2 is negative, the offset is
// clamped to 0 (don't reveal off-world pixels above).
func TestCameraOffset_ClampsAtTop(t *testing.T) {
	rt := newDKTallRuntime(t, 50)
	pixelforge_render.UpdateCameraOffsetForTest(rt)
	assert.Equal(t, 0, rt.CameraOffsetY,
		"camera should clamp at the top of the level")
}

// TestCameraOffset_ClampsAtBottom guards the "hero near bottom of
// level" case: even though heroY - screenH/2 exceeds maxOffset, the
// offset is clamped to totalH - screenH (don't reveal off-world
// pixels below).
func TestCameraOffset_ClampsAtBottom(t *testing.T) {
	rt := newDKTallRuntime(t, 900)
	pixelforge_render.UpdateCameraOffsetForTest(rt)
	// totalH = 960, screenH = 240, maxOffset = 720. heroY=900 → raw
	// offset = 780; clamped to 720.
	assert.Equal(t, 720, rt.CameraOffsetY,
		"camera should clamp at the bottom of the level")
}

// TestCameraOffset_NoPlayerEntityIsZero covers the degenerate case
// where the scene has no Player archetype — offset stays zero
// rather than panicking.
func TestCameraOffset_NoPlayerEntityIsZero(t *testing.T) {
	p := pixelforge_project.NewProject("dk-no-player")
	p.ScreenWidth = 256
	p.ScreenHeight = 240
	p.PhysicsPreset = "dk"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:                "main",
			GridHeightScreens: 4,
			Entities: []pixelforge_project.Entity{
				{
					ID:        "rock",
					Archetype: pixelforge_project.ArchetypeHazard,
					Position:  pixelforge_project.EntityPosition{X: 100, Y: 500},
				},
			},
		},
	}
	rt, err := capsuleruntime.Boot(p, nil, capsuleruntime.Options{SkipSubscribers: true})
	require.NoError(t, err)
	pixelforge_render.UpdateCameraOffsetForTest(rt)
	assert.Equal(t, 0, rt.CameraOffsetY,
		"no-player scene should have zero camera offset")
}

// TestActiveInputFrame_RoundTrip checks that the input-frame stash
// the render path uses (the seam future U5 will read) actually
// captures the most recent inputs. Guards against the seam silently
// dropping the call.
func TestActiveInputFrame_RoundTrip(t *testing.T) {
	defer stableDrawScene(t)()
	rt := newTestRuntime(t)

	want := pixelforge_render.InputFrame{
		Keys: []ebiten.Key{ebiten.KeySpace, ebiten.KeyUp},
		Pad: &pixelforge_render.GamepadState{
			Buttons: []ebiten.GamepadButton{ebiten.GamepadButton0},
			LeftX:   0.5,
		},
	}
	_, err := pixelforge_render.RenderTickAtRGBA(rt, 1, want)
	require.NoError(t, err)

	got := pixelforge_render.ActiveInputFrame()
	assert.Equal(t, want.Keys, got.Keys)
	require.NotNil(t, got.Pad)
	assert.Equal(t, want.Pad.Buttons, got.Pad.Buttons)
	assert.InDelta(t, want.Pad.LeftX, got.Pad.LeftX, 1e-9)
}
