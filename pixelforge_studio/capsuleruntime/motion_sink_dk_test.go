package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// dkProject builds a fixture project with the DK physics preset and a
// single "hero" entity at (8, 16). The hero is sized 8x8 in tests
// (smaller than the canonical 16x16 from Boot) so the per-tile MTV
// for ladder/platform overlaps picks the correct axis — this is the
// same shrink the platformer tests use (see motion_sink_platformer
// _test.go for the rationale).
func dkProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-dk-test")
	p.PhysicsPreset = "dk"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{
					ID: "hero", Name: "Hero",
					Archetype: pixelforge_project.ArchetypePlayer,
					Position:  pixelforge_project.EntityPosition{X: 8, Y: 16},
				},
			},
		},
	}
	return p
}

// withDKLayout paints a 16x16 atlas with solids (id=1) and ladders
// (id=2). The caller passes a 2D grid where '#' = solid, 'L' =
// ladder, '.' = empty.
//
// Example (4 rows tall):
//
//	....
//	..L.
//	..L.
//	####
func withDKLayout(scene *pixelforge_project.Scene, rows []string) {
	grid := make([][]int, len(rows))
	for r, row := range rows {
		grid[r] = make([]int, len(row))
		for c, ch := range row {
			switch ch {
			case '#':
				grid[r][c] = 1
			case 'L':
				grid[r][c] = 2 // pixelforge_physics.DefaultLadderTileID
			default:
				grid[r][c] = 0
			}
		}
	}
	scene.TileAtlases = []pixelforge_project.TileAtlas{
		{
			Name:  "dk",
			TileW: 16,
			TileH: 16,
			Grid:  grid,
		},
	}
}

func TestMotionSink_LadderClimb_UpAscendsAndDisablesGravity(t *testing.T) {
	rt := bootDefault(t, dkProject())
	// 4-tall scene: floor at row 3, ladder cells at rows 1-2 col 0.
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"####",
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Position the hero on the ladder column (col 0, row 2 → pixel
	// (0, 32)). Body footprint 8x8 with top at Y=32.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(32),
	}
	body.Velocity = pixelforge_physics.Vec2{}
	body.OnLadder = true
	body.Sync()

	startY := body.Position.Y
	for i := 0; i < 5; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/ladder_climb",
			Args:  map[string]any{"entity": "hero", "direction": "up"},
		})
	}

	// Body should have ASCENDED (negative-Y is up).
	assert.True(t, body.Position.Y < startY, "hero should ascend while climbing up; startY=%v, endY=%v", startY, body.Position.Y)
	assert.True(t, body.LadderClimbing, "LadderClimbing should be set during climb")
	// Gravity must be suppressed — velocity.Y stays at 0 (the ladder
	// movement helper zeroes it each tick).
	assert.Equal(t, pixelforge_physics.FixedZero, body.Velocity.Y,
		"gravity should be suppressed while climbing")
}

func TestMotionSink_LadderClimb_DownDescends(t *testing.T) {
	rt := bootDefault(t, dkProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"####",
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(16), // on ladder row 1
	}
	body.Velocity = pixelforge_physics.Vec2{}
	body.Sync()

	startY := body.Position.Y
	for i := 0; i < 5; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/ladder_climb",
			Args:  map[string]any{"entity": "hero", "direction": "down"},
		})
	}

	assert.True(t, body.Position.Y > startY, "hero should descend while climbing down")
	assert.True(t, body.LadderClimbing)
}

func TestMotionSink_LadderClimb_OffReleasesClimb(t *testing.T) {
	rt := bootDefault(t, dkProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"####",
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(16),
	}
	body.LadderClimbing = true
	body.OnLadder = true
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/ladder_climb",
		Args:  map[string]any{"entity": "hero", "direction": "off"},
	})

	assert.False(t, body.LadderClimbing, "off should clear LadderClimbing")
}

func TestMotionSink_LadderClimb_ReleaseRestoresGravity(t *testing.T) {
	rt := bootDefault(t, dkProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"....", // no floor — body should fall after release
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(16),
	}
	body.LadderClimbing = true
	body.OnLadder = true
	body.Sync()

	// Release.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/ladder_climb",
		Args:  map[string]any{"entity": "hero", "direction": "off"},
	})
	require.False(t, body.LadderClimbing)

	// Now fire apply_gravity — body should accumulate downward velocity.
	for i := 0; i < 10; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/apply_gravity",
			Args:  map[string]any{"entity": "hero"},
		})
	}
	assert.Greater(t, body.Velocity.Y.Float(), 0.0,
		"after release, gravity should re-accumulate downward velocity")
}

func TestMotionSink_LadderClimb_AutoReleaseAtTop(t *testing.T) {
	rt := bootDefault(t, dkProject())
	// Ladder in col 1 spanning row 2 only (one tile tall). Body
	// climbing up past row 2 reaches the "top" condition.
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"....",
		".L..",
		"####",
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Start the hero ABOVE the ladder's topmost tile (row 2 → top at
	// y=32) so the first climb-up tick triggers the auto-release path.
	// Body top edge at y=32, ladder top at y=32 → top-reached.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(16),
		Y: pixelforge_physics.FixedFromInt(32),
	}
	body.OnLadder = true
	body.Sync()

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/ladder_climb",
		Args:  map[string]any{"entity": "hero", "direction": "up"},
	})

	assert.False(t, body.LadderClimbing, "auto-top reach should clear LadderClimbing")
	assert.True(t, body.Grounded, "auto-top reach should ground the hero on the platform above")
}

func TestMotionSink_LadderClimb_NotOnLadderIsLogAndNoMove(t *testing.T) {
	rt := bootDefault(t, dkProject())
	// No ladder painted — hero is mid-air.
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(8),
		Y: pixelforge_physics.FixedFromInt(16),
	}
	startPos := body.Position

	out := captureLog(t, func() {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/ladder_climb",
			Args:  map[string]any{"entity": "hero", "direction": "up"},
		})
	})

	assert.Contains(t, out, "not on a ladder",
		"climb without ladder should log a warning")
	assert.Equal(t, startPos, body.Position,
		"climb without ladder must not move the body")
}

func TestMotionSink_LadderClimb_PublishesEnteredOnFirstTick(t *testing.T) {
	rt := bootDefault(t, dkProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"####",
	})
	body := rt.Bodies["hero"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(16),
	}
	body.Sync()

	rec := newBusRecorder()
	rec.attach()
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/ladder_climb",
		Args:  map[string]any{"entity": "hero", "direction": "up"},
	})

	entered := rec.filterTopic("motion/ladder_entered")
	require.NotEmpty(t, entered, "first ladder_climb tick should publish motion/ladder_entered")
	assert.Equal(t, "hero", entered[0].Args["entity"])
}

func TestMotionSink_LadderClimb_PublishesExitedOnOff(t *testing.T) {
	rt := bootDefault(t, dkProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"L...",
		"L...",
		"####",
	})
	body := rt.Bodies["hero"]
	body.LadderClimbing = true // simulate already on ladder
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(16),
	}
	body.Sync()

	rec := newBusRecorder()
	rec.attach()
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "motion/ladder_climb",
		Args:  map[string]any{"entity": "hero", "direction": "off"},
	})

	exited := rec.filterTopic("motion/ladder_exited")
	require.NotEmpty(t, exited, "off should publish motion/ladder_exited")
	assert.Equal(t, "hero", exited[0].Args["entity"])
}

// ---- barrel_roll tests --------------------------------------------

func dkBarrelProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("motion-sink-dk-barrel-test")
	p.PhysicsPreset = "dk"
	p.Scenes = []pixelforge_project.Scene{
		{
			ID:   "main",
			Name: "Main",
			Entities: []pixelforge_project.Entity{
				{ID: "barrel", Name: "Barrel",
					Position: pixelforge_project.EntityPosition{X: 8, Y: 8}},
			},
		},
	}
	return p
}

func TestMotionSink_BarrelRoll_MovesRightwardAcrossPlatform(t *testing.T) {
	rt := bootDefault(t, dkBarrelProject())
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"####",
	})
	body := rt.Bodies["barrel"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(8),
	}
	body.Sync()

	startX := body.Position.X
	for i := 0; i < 5; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/barrel_roll",
			Args:  map[string]any{"entity": "barrel"},
		})
	}

	assert.Greater(t, body.Position.X, startX, "barrel should have moved rightward across the platform")
}

func TestMotionSink_BarrelRoll_FallsOffPlatformEdge(t *testing.T) {
	rt := bootDefault(t, dkBarrelProject())
	// Platform ends at col 2; col 3 is air.
	withDKLayout(rt.CurrentScene, []string{
		"....",
		"###.",
		"....",
	})
	body := rt.Bodies["barrel"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Position the barrel right at the platform edge so it walks off
	// the cliff within a handful of ticks.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(40), // col 2.5
		Y: pixelforge_physics.FixedFromInt(8),
	}
	body.Sync()

	startY := body.Position.Y
	for i := 0; i < 20; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/barrel_roll",
			Args:  map[string]any{"entity": "barrel"},
		})
	}

	assert.Greater(t, body.Position.Y, startY, "barrel should have fallen off the platform edge")
}

func TestMotionSink_BarrelRoll_ReversesAtWall(t *testing.T) {
	rt := bootDefault(t, dkBarrelProject())
	// Place a tall wall at col 3 so the barrel rolls along the floor
	// and bounces off it. Floor is row 2; the wall sits at col 3
	// rows 0-2 (so the wall rises above the floor by 2 tiles).
	withDKLayout(rt.CurrentScene, []string{
		"...#",
		"...#",
		"...#",
		"########",
	})
	body := rt.Bodies["barrel"]
	body.Width = pixelforge_physics.FixedFromInt(8)
	body.Height = pixelforge_physics.FixedFromInt(8)
	// Position the barrel resting on top of the floor row (row 3 →
	// tile-top at Y=48) with bottom flush on tile-top.
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(0),
		Y: pixelforge_physics.FixedFromInt(40),
	}
	body.Grounded = true
	body.Sync()

	// Roll for enough ticks to hit the wall and bounce.
	for i := 0; i < 60; i++ {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "motion/barrel_roll",
			Args:  map[string]any{"entity": "barrel"},
		})
	}

	state, ok := rt.BarrelStates["barrel"]
	require.True(t, ok, "barrel state should be tracked")
	// After hitting the right wall, the barrel must have flipped to
	// rolling leftward.
	assert.Equal(t, pixelforge_physics.BarrelLeft, state.Direction,
		"barrel should have reversed direction after hitting the wall")
}

// ---- multi-screen camera scroll tests -----------------------------

// dkTallProject builds a 4-screen-tall scene with a Player entity.
// Used by the camera-offset tests; the renderer (not exercised in
// this _test.go) reads rt.CameraOffsetY to scroll the level.
func dkTallProject() *pixelforge_project.Project {
	p := pixelforge_project.NewProject("dk-tall-scene")
	p.PhysicsPreset = "dk"
	p.ScreenWidth = 256
	p.ScreenHeight = 240
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
					Position:  pixelforge_project.EntityPosition{X: 100, Y: 1000},
				},
			},
		},
	}
	return p
}
