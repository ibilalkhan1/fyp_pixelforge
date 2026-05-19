package capsuleruntime_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// asteroidsProject builds a fixture project with the Asteroids physics
// preset and a single "ship" entity at (100, 100). The screen is the
// default 320×240 so screen-wrap math has stable expected values.
func asteroidsProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-test")
	p.PhysicsPreset = "asteroids"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "ship", Name: "Ship", Position: pixelforge_project.EntityPosition{X: 100, Y: 100}},
			},
		},
	}
	return p
}

// emptyPhysicsProject builds a fixture project with no physics preset
// — the resulting World has ScreenWrap=false so screen_wrap is a
// no-op. Used to exercise the "wrap-disabled" path.
func emptyPhysicsProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-test-empty")
	// PhysicsPreset left empty → zero-value PhysicsConfig.
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "ship", Name: "Ship", Position: pixelforge_project.EntityPosition{X: 321, Y: 100}},
			},
		},
	}
	return p
}

// captureLog redirects log output for the duration of fn and returns
// the captured bytes. The motion sink logs warnings via log.Printf
// when it can't find an entity; tests assert the warning fired
// without taking a dependency on a stub logger.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prevW := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevW)
		log.SetFlags(prevFlags)
	}()
	fn()
	return buf.String()
}

func TestMotionSink_ApplyThrust_DirectionZeroPushesPositiveX(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	require.NotNil(t, rt.Bodies["ship"])
	require.Equal(t, pixelforge_physics.FixedZero, rt.Bodies["ship"].Velocity.X)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(0), "force": float64(10)},
	})

	vel := rt.Bodies["ship"].Velocity
	want := pixelforge_physics.FixedFromFloat(10)
	// LUT-trig tolerance: ±2 raw Fixed32 units (~3e-5 in float).
	assert.InDelta(t, want.Float(), vel.X.Float(), 1e-3, "Velocity.X")
	assert.InDelta(t, 0.0, vel.Y.Float(), 1e-3, "Velocity.Y should be ~0 at direction=0")
}

func TestMotionSink_ApplyThrust_Direction90PushesPositiveY(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(90), "force": float64(10)},
	})

	vel := rt.Bodies["ship"].Velocity
	want := pixelforge_physics.FixedFromFloat(10)
	assert.InDelta(t, 0.0, vel.X.Float(), 1e-3, "Velocity.X should be ~0 at direction=90")
	assert.InDelta(t, want.Float(), vel.Y.Float(), 1e-3, "Velocity.Y")
}

func TestMotionSink_ApplyThrust_Accumulates(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	for i := 0; i < 3; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_thrust",
			Args:  map[string]any{"entity": "ship", "direction": float64(0), "force": float64(5)},
		})
	}

	vel := rt.Bodies["ship"].Velocity
	// 3 × 5 = 15 along X.
	assert.InDelta(t, 15.0, vel.X.Float(), 1e-3, "Velocity.X should accumulate")
	assert.InDelta(t, 0.0, vel.Y.Float(), 1e-3, "Velocity.Y should remain 0")
}

func TestMotionSink_ApplyThrust_DirectionWrapsMod360(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	// direction=400 should behave like direction=40.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(400), "force": float64(10)},
	})
	got := rt.Bodies["ship"].Velocity

	// Re-boot to clear state, then compare against direction=40.
	rt2 := bootDefault(t, asteroidsProject())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(40), "force": float64(10)},
	})
	want := rt2.Bodies["ship"].Velocity

	assert.Equal(t, want.X, got.X, "direction wrap-around mismatch on X")
	assert.Equal(t, want.Y, got.Y, "direction wrap-around mismatch on Y")
}

func TestMotionSink_ApplyThrust_NegativeDirectionFoldsIntoRange(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	// direction=-45 should behave like direction=315 (negative inputs
	// fold mod-360 into the positive range, matching the rotate path
	// so authored thrust angles can use either convention).
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(-45), "force": float64(10)},
	})
	got := rt.Bodies["ship"].Velocity

	rt2 := bootDefault(t, asteroidsProject())
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_thrust",
		Args:  map[string]any{"entity": "ship", "direction": float64(315), "force": float64(10)},
	})
	want := rt2.Bodies["ship"].Velocity

	assert.Equal(t, want.X, got.X, "negative direction X mismatch")
	assert.Equal(t, want.Y, got.Y, "negative direction Y mismatch")
}

func TestMotionSink_RotateEntity_AddsDelta(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	require.Equal(t, pixelforge_physics.FixedZero, rt.Bodies["ship"].Rotation)

	for i := 0; i < 3; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/rotate_entity",
			Args:  map[string]any{"entity": "ship", "delta": float64(90)},
		})
	}

	got := rt.Bodies["ship"].Rotation.Float()
	assert.InDelta(t, 270.0, got, 1e-6, "rotation should be 3×90=270 degrees")
}

func TestMotionSink_RotateEntity_WrapsModulo360(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	for i := 0; i < 2; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/rotate_entity",
			Args:  map[string]any{"entity": "ship", "delta": float64(200)},
		})
	}

	got := rt.Bodies["ship"].Rotation.Float()
	// 200 + 200 = 400 → 400 - 360 = 40.
	assert.InDelta(t, 40.0, got, 1e-6, "rotation should fold into [0, 360)")
}

func TestMotionSink_RotateEntity_NegativeDeltaFoldsIntoRange(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/rotate_entity",
		Args:  map[string]any{"entity": "ship", "delta": float64(-30)},
	})

	got := rt.Bodies["ship"].Rotation.Float()
	// -30 → 330.
	assert.InDelta(t, 330.0, got, 1e-6, "negative rotation should fold to positive range")
}

func TestMotionSink_ScreenWrap_RightEdgeWrapsToLeft(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	body := rt.Bodies["ship"]
	// Place ship 1px past the right edge on a 320-wide screen.
	body.Position.X = pixelforge_physics.FixedFromFloat(321)
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/screen_wrap",
		Args:  map[string]any{"entity": "ship"},
	})

	// 321 - 320 = 1.
	assert.InDelta(t, 1.0, body.Position.X.Float(), 1e-6, "X should wrap from 321 to 1")
	assert.InDelta(t, 100.0, body.Position.Y.Float(), 1e-6, "Y should be unchanged")
}

func TestMotionSink_ScreenWrap_LeftEdgeWrapsToRight(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	body := rt.Bodies["ship"]
	body.Position.X = pixelforge_physics.FixedFromFloat(-1)
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/screen_wrap",
		Args:  map[string]any{"entity": "ship"},
	})

	// -1 + 320 = 319.
	assert.InDelta(t, 319.0, body.Position.X.Float(), 1e-6, "X should wrap from -1 to 319")
}

func TestMotionSink_ScreenWrap_DisabledIsNoOp(t *testing.T) {
	rt := bootDefault(t, emptyPhysicsProject())
	body := rt.Bodies["ship"]
	require.NotNil(t, body)
	// Confirm PhysicsConfig.ScreenWrap is off in the empty-preset world.
	require.False(t, rt.Physics.Config().ScreenWrap)

	startX := body.Position.X
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/screen_wrap",
		Args:  map[string]any{"entity": "ship"},
	})

	assert.Equal(t, startX, body.Position.X, "position must not change when ScreenWrap is disabled")
}

func TestMotionSink_UnknownEntityLogsWarningAndNoMutation(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	startVel := rt.Bodies["ship"].Velocity

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_thrust",
			Args:  map[string]any{"entity": "ghost", "direction": float64(0), "force": float64(10)},
		})
	})

	assert.Contains(t, out, "unknown entity")
	assert.Contains(t, out, "ghost")
	// Ship velocity untouched by the misdirected thrust.
	assert.Equal(t, startVel, rt.Bodies["ship"].Velocity)
}

func TestMotionSink_UnknownTopicSilentlyDropped(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	startVel := rt.Bodies["ship"].Velocity
	startPos := rt.Bodies["ship"].Position

	// motion/move_pattern is the next deferred-to-future topic — it's
	// explicitly out of v2 scope per plan-009's Scope Boundaries.
	// Route through the motion sink and verify it doesn't crash and
	// doesn't mutate. (Pre-U13 this test exercised motion/jump; pre-
	// U15 motion/place_on_grid; pre-U17 motion/ladder_climb; each
	// landed as a concrete handler in its respective unit, so the
	// unhandled-topic guard rotated to the next still-deferred verb.)
	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/move_pattern",
			Args:  map[string]any{"entity": "ship", "pattern": "loop"},
		})
	})

	// The handler logs a debugDrop for "not yet implemented" topics.
	assert.True(t, strings.Contains(out, "motion/move_pattern"), "expected debug-drop log for motion/move_pattern, got: %s", out)
	assert.Equal(t, startVel, rt.Bodies["ship"].Velocity, "velocity must not mutate on unhandled topic")
	assert.Equal(t, startPos, rt.Bodies["ship"].Position, "position must not mutate on unhandled topic")
}

func TestMotionSink_ApplyThrust_MissingEntityArgNoOp(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())
	startVel := rt.Bodies["ship"].Velocity

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_thrust",
			Args:  map[string]any{"direction": float64(0), "force": float64(10)},
		})
	})

	assert.Contains(t, out, "motion/apply_thrust")
	assert.Equal(t, startVel, rt.Bodies["ship"].Velocity)
}

func TestRuntime_BootInitializesPhysicsAndBodies(t *testing.T) {
	rt := bootDefault(t, asteroidsProject())

	require.NotNil(t, rt.Physics, "Boot should initialise rt.Physics")
	require.NotNil(t, rt.Bodies, "Boot should initialise rt.Bodies")
	require.Contains(t, rt.Bodies, "ship")

	body := rt.Bodies["ship"]
	assert.InDelta(t, 100.0, body.Position.X.Float(), 1e-6)
	assert.InDelta(t, 100.0, body.Position.Y.Float(), 1e-6)

	cfg := rt.Physics.Config()
	assert.True(t, cfg.ScreenWrap, "asteroids preset enables ScreenWrap")
	assert.Equal(t, 320, cfg.ScreenWidth)
	assert.Equal(t, 240, cfg.ScreenHeight)
}

func TestRuntime_BootUnknownPresetFallsBackToEmpty(t *testing.T) {
	p := pixelforge_project.NewProject("motion-sink-test-unknown")
	p.PhysicsPreset = "platformer-9000"
	p.Scenes = []pixelforge_project.Scene{
		{ID: "main", Name: "Main", Entities: []pixelforge_project.Entity{}},
	}

	out := captureLog(t, func() {
		_ = bootDefault(t, p)
	})

	assert.Contains(t, out, "unknown physics_preset")
}
