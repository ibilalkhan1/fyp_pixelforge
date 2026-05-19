package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// marioProject builds a fixture project with the Mario physics preset
// and a single "hero" entity at (8, 0). Width/Height default to 16×16
// via Boot, so the body's footprint maps onto a single 16-px tile
// column. No TileAtlas is attached — tests that need a tilemap call
// withSolidFloor below to graft one on after Boot.
func marioProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-platformer-test")
	p.PhysicsPreset = "mario"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "hero", Name: "Hero", Position: pixelforge_project.EntityPosition{X: 8, Y: 0}},
			},
		},
	}
	return p
}

// withSolidFloor paints a single row of solid tiles (ID=1) across the
// scene at tile-row `row`. Tile dimensions are 16×16, the same cell
// size Mario's NES-class preset assumes. Designed to be called after
// bootDefault so the runtime's *Scene pointer already exists and the
// painted atlas is the one resolveTilemap reads.
func withSolidFloor(scene *pixelforge_project.Scene, row, cols int) {
	grid := make([][]int, row+1)
	for r := 0; r <= row; r++ {
		grid[r] = make([]int, cols)
	}
	for c := 0; c < cols; c++ {
		grid[row][c] = 1
	}
	scene.TileAtlases = []pixelforge_project.TileAtlas{
		{
			Name:  "ground",
			TileW: 16,
			TileH: 16,
			Grid:  grid,
		},
	}
}

func TestMotionSink_Jump_FromGroundedAppliesUpwardImpulse(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	require.NotNil(t, body)
	body.Grounded = true
	body.Velocity = pixelforge_physics.Vec2{}

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/jump",
		Args:  map[string]any{"entity": "hero", "strength": float64(5)},
	})

	// ApplyJump sets Velocity.Y = -strength (negative-Y = upward in
	// screen-space) and clears Grounded.
	assert.Equal(t, pixelforge_physics.FixedFromFloat(-5), body.Velocity.Y,
		"jump should set Velocity.Y to -strength")
	assert.False(t, body.Grounded, "jump should transition to airborne")
}

func TestMotionSink_Jump_AirborneIsNoOp(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	require.NotNil(t, body)
	body.Grounded = false
	body.Velocity = pixelforge_physics.Vec2{Y: pixelforge_physics.FixedFromFloat(3)}
	startVel := body.Velocity

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/jump",
		Args:  map[string]any{"entity": "hero", "strength": float64(5)},
	})

	// Airborne jump must not double-jump: velocity is preserved and
	// Grounded stays false.
	assert.Equal(t, startVel, body.Velocity, "airborne jump must not change velocity")
	assert.False(t, body.Grounded)
}

func TestMotionSink_Jump_NegativeStrengthRejected(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	body.Grounded = true
	body.Velocity = pixelforge_physics.Vec2{}

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/jump",
		Args:  map[string]any{"entity": "hero", "strength": float64(-5)},
	})

	// Negative-magnitude strength is rejected so a mis-authored
	// recipe can't yank the body downward.
	assert.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
		"negative strength must not mutate velocity")
	assert.True(t, body.Grounded, "Grounded must not be cleared when jump is rejected")
}

func TestMotionSink_Jump_MissingEntityNoMutation(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	body.Grounded = true
	startVel := body.Velocity

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/jump",
			Args:  map[string]any{"entity": "ghost", "strength": float64(5)},
		})
	})

	assert.Contains(t, out, "unknown entity")
	assert.Contains(t, out, "ghost")
	assert.Equal(t, startVel, body.Velocity, "unknown-entity jump must not mutate the real hero")
}

func TestMotionSink_ApplyGravity_AccumulatesOverOneSecond(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	require.NotNil(t, body)
	body.Velocity = pixelforge_physics.Vec2{}

	// MarioConfig: Gravity.Y = 980 px/s². After 60 ticks at dt=1/60,
	// Velocity.Y ≈ 980 within Fixed32 rounding (matches the substrate
	// gravity_test invariant).
	for i := 0; i < 60; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_gravity",
			Args:  map[string]any{"entity": "hero"},
		})
	}

	got := body.Velocity.Y.Float()
	assert.InDelta(t, 980.0, got, 5.0, "vy after 1s should be ~980")
}

func TestMotionSink_ApplyGravity_SolidCollideStopsFall(t *testing.T) {
	rt := bootDefault(t, marioProject())
	withSolidFloor(rt.CurrentScene, 1, 4) // row=1 → tile-top at y=16
	body := rt.Bodies["hero"]
	// Shrink to 8×8 so the per-tile MTV picks the Y axis cleanly (the
	// 16×16 default body and 16×16 ground tile produce a degenerate
	// "both axes overlap equally" situation U14 will polish).
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Position the hero a few pixels above the floor with a downward
	// velocity that overshoots the floor by a few pixels per tick.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(4),
		Y: pixelforge_physics.FixedFromInt(4),
	}
	body.Velocity = pixelforge_physics.Vec2{Y: pixelforge_physics.FixedFromInt(8)}
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_gravity",
		Args:  map[string]any{"entity": "hero"},
	})

	// Body height = 8, floor tile top = y=16 → hero top snaps to y=8.
	// Velocity.Y zeroed on landing; Grounded set.
	assert.Equal(t, pixelforge_physics.FixedFromInt(8), body.Position.Y,
		"hero should snap so its bottom edge rests on the floor")
	assert.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
		"downward velocity should be zeroed on landing")
	assert.True(t, body.Grounded, "Grounded must be set after landing on a solid tile")
}

func TestMotionSink_ApplyGravity_StandingOnPlatformVelocityStaysBounded(t *testing.T) {
	rt := bootDefault(t, marioProject())
	withSolidFloor(rt.CurrentScene, 1, 4) // tile-top at y=16
	body := rt.Bodies["hero"]
	// Width/Height default to 16×16 in Boot — for the "resting on a
	// 16-px floor tile" scenario the substrate's per-tile MTV chooses
	// the smaller-overlap axis, so we shrink the hero to 8×8 here.
	// 8-px bodies are the substrate's canonical test footprint
	// (see pixelforge_physics/tilemap_test.go) and mirror the
	// classic SMB physics layout (hero collider narrower than the
	// 16-px ground tiles).
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Start the hero just above the floor so the very first
	// apply_gravity tick lands him; from then on, the integrator-
	// then-resolver loop is the canonical "standing on a platform"
	// scenario the AABB sweeper drives. Exercise it for many ticks to
	// make sure the body doesn't drift and Grounded stays set.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(4),
		Y: pixelforge_physics.FixedFromInt(7),
	}
	body.Velocity = pixelforge_physics.Vec2{}
	body.Sync()

	// First tick: drop onto the floor (Integrate then resolve).
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/apply_gravity",
		Args:  map[string]any{"entity": "hero"},
	})
	require.True(t, body.Grounded, "first gravity tick should land the hero on the floor")
	require.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
		"vy should be zeroed on landing")
	// Body height = 8, tile-top = 16 → body top should be at 8.
	require.Equal(t, 8, body.Position.Y.Int(),
		"hero should land with bottom edge flush on tile-top")

	// Subsequent ticks: each one re-applies gravity, the sweeper
	// catches the resulting overlap, snaps the body back to tile-top
	// and zeroes vy.
	for i := 0; i < 30; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_gravity",
			Args:  map[string]any{"entity": "hero"},
		})
		require.True(t, body.Grounded, "tick %d: Grounded must stay set while resting on platform", i)
		require.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
			"tick %d: vy must be zeroed every tick by ground resolution", i)
		require.Equal(t, 8, body.Position.Y.Int(),
			"tick %d: integer Y position must stay at 8 (bottom flush on tile-top) while resting", i)
	}
}

func TestMotionSink_SolidCollide_SnapsToTileTopAndGrounds(t *testing.T) {
	rt := bootDefault(t, marioProject())
	withSolidFloor(rt.CurrentScene, 1, 4) // tile-top at y=16
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Place the hero overlapping the floor tile by 4px (mimics what
	// happens when a teleport or move_pattern verb drops the body
	// inside geometry — the explicit solid_collide verb is the recipe
	// author's escape hatch to "snap it out").
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(4),
		Y: pixelforge_physics.FixedFromInt(12),
	}
	body.Velocity = pixelforge_physics.Vec2{Y: pixelforge_physics.FixedFromInt(2)}
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "collision/solid_collide",
		Args:  map[string]any{"entity": "hero"},
	})

	// Body height = 8, tile-top = 16 → snapped to y=8 (bottom on
	// tile-top y=16).
	assert.Equal(t, pixelforge_physics.FixedFromInt(8), body.Position.Y,
		"solid_collide should snap the body so its bottom edge is on the tile-top")
	assert.True(t, body.Grounded, "solid_collide should set Grounded on upward resolution")
}

func TestMotionSink_SolidCollide_NoTilemapIsNoOp(t *testing.T) {
	rt := bootDefault(t, marioProject())
	// Intentionally no withSolidFloor — the scene's TileAtlases stays
	// empty. solid_collide must be a graceful no-op in that case.
	body := rt.Bodies["hero"]
	startPos := body.Position
	startVel := body.Velocity

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "collision/solid_collide",
		Args:  map[string]any{"entity": "hero"},
	})

	assert.Equal(t, startPos, body.Position, "no-tilemap solid_collide must not mutate position")
	assert.Equal(t, startVel, body.Velocity, "no-tilemap solid_collide must not mutate velocity")
}

func TestMotionSink_JumpFallLandCycle(t *testing.T) {
	rt := bootDefault(t, marioProject())
	withSolidFloor(rt.CurrentScene, 2, 8) // tile-top at y=32
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(8),
		Y: pixelforge_physics.FixedFromInt(24), // bottom edge at y=32 = on the floor
	}
	body.Velocity = pixelforge_physics.Vec2{}
	body.Grounded = true
	body.Sync()

	// 1) Jump from grounded.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/jump",
		Args:  map[string]any{"entity": "hero", "strength": float64(200)},
	})
	require.False(t, body.Grounded, "jump should clear grounded")
	require.Equal(t, pixelforge_physics.FixedFromFloat(-200), body.Velocity.Y,
		"jump should set vy to -strength")

	// 2) Tick gravity until the body lands again. Bound the loop so a
	//    regression in the integrator can't spin forever.
	landed := false
	for i := 0; i < 600; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_gravity",
			Args:  map[string]any{"entity": "hero"},
		})
		if body.Grounded {
			landed = true
			break
		}
	}
	require.True(t, landed, "hero should land back on the floor within the tick budget")

	// 3) After landing, vy is zeroed and the body is back at y=24
	//    (bottom on tile-top y=32).
	assert.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
		"vy should be zeroed on landing")
	assert.Equal(t, pixelforge_physics.FixedFromInt(24), body.Position.Y,
		"hero should land with bottom edge on tile-top")
}

func TestMotionSink_ApplyGravity_UnknownEntityLogsAndNoMutation(t *testing.T) {
	rt := bootDefault(t, marioProject())
	body := rt.Bodies["hero"]
	startVel := body.Velocity
	startPos := body.Position

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_gravity",
			Args:  map[string]any{"entity": "ghost"},
		})
	})

	assert.Contains(t, out, "unknown entity")
	assert.Contains(t, out, "ghost")
	assert.Equal(t, startVel, body.Velocity, "unknown-entity gravity must not touch the real hero")
	assert.Equal(t, startPos, body.Position)
}
