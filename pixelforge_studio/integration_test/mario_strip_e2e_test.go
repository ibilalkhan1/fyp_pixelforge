// mario_strip_e2e_test.go exercises the idea #1 v1 stack (ScreenRoom
// Mario-strip primitive) end-to-end:
//   - U1's Scene grid/spawn/camera schema + Entity TileX/TileY
//   - U2's pixelforge_tilemap renderer is exercised structurally via
//     the scene's TilemapLayer.Grid (the renderer itself has its own
//     unit tests; here we assert grid state, not rendered pixels)
//   - U3's pixelforge_camera.Follower drives the camera (AE4)
//   - U6's TilePainter brush + bucket fill (AE1)
//   - U7's right-click "Set as Player" mutual exclusivity + spawn
//     teleport (AE2, AE3)
//   - U8's ResizeGrid preserves painted content + entity coords (AE5)
//
// The fixture mario_strip_scene.pforge is hand-authored so the load
// path also covers AE6 (pre-v1 schema additions defaulted at load) on
// a second fixture path (the editor's own cart_assets/editor.pforge,
// which never carried world-primitive fields).
//
// Tests drive the public APIs directly; no ImGui events are simulated.
// Sprite assets carry empty RelativePath strings so the loader skips
// asset validation, and pixelforge_entity.RenderAll / pixelforge_
// tilemap.Render are not invoked here (the tests assert on schema
// state, which is the AE contract).
package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_camera"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

const marioFixturePath = "fixtures/mario_strip_scene.pforge"

// loadMario is the standard fixture loader for the Mario-strip AE
// tests. Returns the canonicalised project (defaults applied) ready
// for direct API drive. Mirrors loadGoomba (verb_sheet_e2e_test.go)
// so both fixtures live in the same test package without contention.
func loadMario(t *testing.T) *pixelforge_project.Project {
	t.Helper()
	abs, err := filepath.Abs(marioFixturePath)
	require.NoError(t, err)
	p, err := pixelforge_project.Load(abs)
	require.NoError(t, err, "fixture %s must load cleanly", abs)
	return p
}

// findMarioEntity is the Mario-strip variant of findEntity in
// verb_sheet_e2e_test.go (package-scoped helper exists there but this
// test reads its own fixture so a local copy keeps the call sites
// obvious).
func findMarioEntity(t *testing.T, p *pixelforge_project.Project, id string) *pixelforge_project.Entity {
	t.Helper()
	require.NotEmpty(t, p.Scenes, "project has no scenes")
	for i := range p.Scenes[0].Entities {
		e := &p.Scenes[0].Entities[i]
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("entity %q not found in scene[0]", id)
	return nil
}

// paintGroundRow paints the canonical "ground" strip at row 25 across
// every column of the active grid (cols 0..511 for the default
// 16-screen layout). Uses BucketFill since the layer's row 25 starts
// uniformly at tile 0 (the painter dispatch grows the grid and fills
// contiguous matching cells in one StrokeCommand). Returns the layer
// pointer for the caller to keep referencing.
//
// Hoisted into a helper so AE5 (resize-preserves-content) can set up
// the same painted state AE1 produces without duplicating the
// painter-drive sequence.
func paintGroundRow(t *testing.T, layer *pixelforge_project.TilemapLayer) {
	t.Helper()
	require.NotNil(t, layer)
	painter := editor.NewTilePainter()
	stack := editor.NewUndoStack()
	painter.SetActiveTile(1)
	// Walk the full row in a single brush stroke. BucketFill would
	// also work but a brush stroke keeps the painted cells exactly to
	// row 25 (bucket would flood vertically across any contiguous
	// zeroes too — we want only one row).
	painter.BrushStartStroke(layer)
	for col := 0; col < 512; col++ {
		painter.BrushPaint(col, 25, 1)
	}
	cmd := painter.BrushEndStroke(stack)
	require.NotNil(t, cmd, "ground-row stroke must commit a StrokeCommand")
}

// resetCamera zeroes the global pixelforge.Camera so each test starts
// from a clean state. The camera is process-global; without this
// reset, test order would couple AE4 (camera-position assertions) to
// whichever test ran before it.
func resetCamera(t *testing.T) {
	t.Helper()
	pixelforge.Camera.X = 0
	pixelforge.Camera.Y = 0
}

// -----------------------------------------------------------------------------
// AE1 — Brush + bucket painting end-to-end
// -----------------------------------------------------------------------------

// TestE2E_AE1_BrushPaintsBucketFills exercises U6's TilePainter via the
// public API: brush stroke writes individual cells, bucket fill flood-
// fills a contiguous region, and every stroke lands as one command on
// the UndoStack. Both Grid mutation and StrokeCommand discipline are
// asserted — the dual proves the "Ctrl+Z reverts the whole stroke"
// contract holds at the API layer.
func TestE2E_AE1_BrushPaintsBucketFills(t *testing.T) {
	p := loadMario(t)
	require.Len(t, p.Scenes[0].Tilemaps, 1, "fixture must carry one tilemap layer")
	layer := &p.Scenes[0].Tilemaps[0]

	painter := editor.NewTilePainter()
	stack := editor.NewUndoStack()
	painter.SetActiveTile(2)

	// Brush stroke: paint a 3-cell horizontal line at row 10.
	painter.BrushStartStroke(layer)
	painter.BrushPaint(5, 10, 2)
	painter.BrushPaint(6, 10, 2)
	painter.BrushPaint(7, 10, 2)
	brushCmd := painter.BrushEndStroke(stack)
	require.NotNil(t, brushCmd, "brush stroke must commit")
	assert.Len(t, brushCmd.Diffs, 3, "stroke records one diff per painted cell")
	assert.Equal(t, 2, layer.Grid[10][5])
	assert.Equal(t, 2, layer.Grid[10][6])
	assert.Equal(t, 2, layer.Grid[10][7])
	assert.Equal(t, 1, stack.Depth(), "brush stroke commits as one undo entry")

	// Bucket fill: flood-fill the empty cell (0, 0) with tile 9. The
	// brush stroke at row 10 cols 5-7 is a barrier of "2" cells — the
	// 4-connected BFS routes around them but still fills every other
	// cell of the (default-painter-allocated) 32x32 region.
	painter.SetActiveTile(9)
	bucketCmd := painter.BucketFill(layer, 0, 0, 9, stack)
	require.NotNil(t, bucketCmd, "bucket fill must commit")
	assert.NotEmpty(t, bucketCmd.Diffs, "bucket fill produces at least one diff")
	assert.Equal(t, 9, layer.Grid[0][0], "bucket flood-fills the source cell")
	// The brush-painted cells must remain at their painted value — the
	// flood-fill respects the 4-connected matching-value contract.
	assert.Equal(t, 2, layer.Grid[10][5], "brush-painted cell preserved across bucket fill")
	assert.Equal(t, 2, layer.Grid[10][6])
	assert.Equal(t, 2, layer.Grid[10][7])
	assert.Equal(t, 2, stack.Depth(), "stack now holds brush + bucket commands")
}

// -----------------------------------------------------------------------------
// AE2 — Player tag mutual exclusivity via right-click handler
// -----------------------------------------------------------------------------

// TestE2E_AE2_PlayerTagMutualExclusivity drives U7's right-click
// "Set as Player" through editor.HandleRightClick. The fixture starts
// with exactly one Player (Mario); reassigning the tag to goomba1 must
// demote Mario to ArchetypeWorld in the same call. The invariant is
// the schema-level "exactly one Player per scene" contract idea #5's
// AE2 deferred to idea #1.
func TestE2E_AE2_PlayerTagMutualExclusivity(t *testing.T) {
	p := loadMario(t)
	scene := &p.Scenes[0]

	// Baseline: fixture carries exactly one Player.
	var players []string
	for _, e := range scene.Entities {
		if e.Archetype == pixelforge_project.ArchetypePlayer {
			players = append(players, e.ID)
		}
	}
	require.Equal(t, []string{"mario"}, players,
		"fixture must carry exactly one Player (Mario) at load")

	// Reassign via the public right-click dispatch. NewEditor wires the
	// painter / undo stack / confirm dialog already; SetProject hands
	// the fixture project to the editor's mutation surface.
	e := editor.New()
	e.SetProject(p)

	ok := editor.HandleRightClick(e, editor.RightClickContext{
		Scene:    scene,
		EntityID: "goomba1",
	})
	require.True(t, ok, "Set as Player must report handled=true")

	mario := findMarioEntity(t, p, "mario")
	goomba := findMarioEntity(t, p, "goomba1")
	assert.Equal(t, pixelforge_project.ArchetypeWorld, mario.Archetype,
		"previous Player must be demoted to World")
	assert.Equal(t, pixelforge_project.ArchetypePlayer, goomba.Archetype,
		"clicked entity must become the new Player")

	// Exactly one Player after the swap — the mutual-exclusivity
	// invariant the inspector copy promises.
	count := 0
	for _, ent := range scene.Entities {
		if ent.Archetype == pixelforge_project.ArchetypePlayer {
			count++
		}
	}
	assert.Equal(t, 1, count, "exactly one Player must remain after Set as Player")
	assert.True(t, e.IsDirty(), "mutating the Player tag must flip the dirty flag")
}

// -----------------------------------------------------------------------------
// AE3 — SpawnTile honored on preview start
// -----------------------------------------------------------------------------

// TestE2E_AE3_SpawnCellRespected verifies U7's TeleportPlayerToSpawn:
// when the Player's authored TileX/TileY differs from the scene's
// SpawnTile, the teleport moves the Player to SpawnTile on the first
// preview tick. Both TileX/TileY and the legacy float Position are
// updated (the editor's hit-test reads Position; the renderer reads
// TileX/TileY — they must agree post-teleport).
func TestE2E_AE3_SpawnCellRespected(t *testing.T) {
	p := loadMario(t)
	scene := &p.Scenes[0]

	mario := findMarioEntity(t, p, "mario")
	// Move Mario somewhere far from the SpawnTile so the teleport's
	// effect is unambiguous. The fixture's SpawnTile is (4, 24); we
	// place Mario at (100, 24) before invoking the teleport helper.
	mario.TileX = 100
	mario.TileY = 24
	mario.Position.X = 800
	mario.Position.Y = 192

	require.Equal(t, 4, scene.SpawnTile.Col, "fixture SpawnTile must remain {4, 24}")
	require.Equal(t, 24, scene.SpawnTile.Row)

	editor.TeleportPlayerToSpawn(scene)

	assert.Equal(t, 4, mario.TileX, "post-teleport TileX must equal SpawnTile.Col")
	assert.Equal(t, 24, mario.TileY, "post-teleport TileY must equal SpawnTile.Row")
	// Position (legacy float coords) must agree with the new tile cell
	// so canvas hit-test + renderer paint the same world location.
	assert.Equal(t, float64(4*8), mario.Position.X, "Position.X follows TileX after teleport")
	assert.Equal(t, float64(24*8), mario.Position.Y, "Position.Y follows TileY after teleport")
}

// -----------------------------------------------------------------------------
// AE4 — Camera follows player and clamps at level edge
// -----------------------------------------------------------------------------

// TestE2E_AE4_CameraFollowsAndStops drives a pixelforge_camera.
// Follower with the fixture's Scene.Camera config + level bounds
// (16 screens × 256 wide = 4096 px). Walks Mario right one tile per
// tick across the full level; asserts (a) camera.X advances after the
// Player exits the dead-zone and (b) camera.X clamps at the right
// edge (4096 - 256 = 3840) when the Player walks past the clamp
// boundary.
func TestE2E_AE4_CameraFollowsAndStops(t *testing.T) {
	resetCamera(t)
	p := loadMario(t)
	scene := &p.Scenes[0]

	boundsW := scene.GridWidthScreens * 256
	boundsH := scene.GridHeightScreens * 240
	require.Equal(t, 4096, boundsW, "16-screen-wide level must be 4096 px")
	require.Equal(t, 240, boundsH)

	cfg := pixelforge_camera.FollowerConfig{
		DeadZoneW:  scene.Camera.DeadZoneW,
		DeadZoneH:  scene.Camera.DeadZoneH,
		LookAheadX: scene.Camera.LookAheadX,
		SmoothingX: scene.Camera.SmoothingX,
		SmoothingY: scene.Camera.SmoothingY,
		BoundsW:    boundsW,
		BoundsH:    boundsH,
	}
	follower := pixelforge_camera.NewFollower(cfg)

	// Snapshot the camera before any walk — should be 0 at construction
	// (the Follower writes to pixelforge.Camera on every tick).
	follower.Update(float64(4*8), float64(24*8), 1.0/60.0)
	startCamX := pixelforge.Camera.X

	// Walk Mario right from TileX=4 to TileX=500 over 496 ticks.
	// Each tick the camera follower runs Update once with the new
	// player position. dt=1/60s matches the editor preview's fixed
	// step.
	for tx := 5; tx <= 500; tx++ {
		follower.Update(float64(tx*8), float64(24*8), 1.0/60.0)
	}
	midCamX := pixelforge.Camera.X
	assert.Greater(t, midCamX, startCamX,
		"camera.X must advance as Player walks right past the dead-zone")

	// Walk Mario all the way to the right edge of the level — the
	// camera should clamp at boundsW - viewportW (= 4096 - 256 = 3840)
	// when the Player approaches the right wall.
	for tx := 501; tx <= 511; tx++ {
		// Run many ticks per cell so the smoothed camera settles at
		// the clamp before we assert.
		for k := 0; k < 60; k++ {
			follower.Update(float64(tx*8), float64(24*8), 1.0/60.0)
		}
	}
	finalCamX := pixelforge.Camera.X
	assert.Equal(t, 3840, finalCamX,
		"camera.X must clamp at boundsW - viewportW (3840) at the right edge")
}

// -----------------------------------------------------------------------------
// AE5 — ResizeGrid preserves painted content + entity coords
// -----------------------------------------------------------------------------

// TestE2E_AE5_ResizePreservesCoords paints the ground row, resizes the
// scene from 16 to 24 screens, and asserts (a) the painted row stays
// at row 25 cols 0..511, (b) no entity falls outside the new bounds
// (all entities at tile_x <= 20 are inside the new 24*32 = 768 col
// range), (c) the entity coordinates are unchanged.
func TestE2E_AE5_ResizePreservesCoords(t *testing.T) {
	p := loadMario(t)
	scene := &p.Scenes[0]
	layer := &scene.Tilemaps[0]

	paintGroundRow(t, layer)

	// Pre-resize sanity: row 25 cols 0..511 are non-zero.
	require.GreaterOrEqual(t, len(layer.Grid), 26,
		"painted layer must extend to row 25")
	require.GreaterOrEqual(t, len(layer.Grid[25]), 512,
		"painted layer must extend to col 511")
	for col := 0; col < 512; col++ {
		require.Equal(t, 1, layer.Grid[25][col],
			"ground row must be painted with tile 1 at col %d before resize", col)
	}

	// Capture entity coords pre-resize so we can assert no movement.
	pre := map[string]struct{ x, y int }{}
	for _, e := range scene.Entities {
		pre[e.ID] = struct{ x, y int }{e.TileX, e.TileY}
	}

	// Grow to 24 screens. force=false is safe — all entities fit
	// inside 24*32 = 768 cols.
	flagged, err := editor.ResizeGrid(scene, 24, 1, false)
	require.NoError(t, err)
	assert.Empty(t, flagged,
		"grow path with all entities inside the new bounds must flag none")

	// Painted row preserved at original cells.
	assert.GreaterOrEqual(t, len(layer.Grid[25]), 512,
		"row 25 must still cover the originally painted columns after grow")
	for col := 0; col < 512; col++ {
		assert.Equal(t, 1, layer.Grid[25][col],
			"painted col %d must survive resize", col)
	}

	// Entity coords unchanged.
	for _, e := range scene.Entities {
		want := pre[e.ID]
		assert.Equal(t, want.x, e.TileX,
			"entity %q TileX must be unchanged across resize", e.ID)
		assert.Equal(t, want.y, e.TileY,
			"entity %q TileY must be unchanged across resize", e.ID)
	}
	assert.Equal(t, 24, scene.GridWidthScreens, "scene reflects the new width")
}

// -----------------------------------------------------------------------------
// AE6 — Pre-v1 file loads cleanly with defaulted world-primitive fields
// -----------------------------------------------------------------------------

// TestE2E_PreV1FileLoadsCleanly loads the studio's own editor.pforge
// (which predates idea #1's world-primitive schema additions) and
// asserts the loader applies the defaults from project.go's
// applyWorldDefaults: GridWidthScreens=16, GridHeightScreens=1, the
// full Camera config equal to the documented Mario-class constants.
func TestE2E_PreV1FileLoadsCleanly(t *testing.T) {
	abs, err := filepath.Abs(editorFixturePath)
	require.NoError(t, err)
	p, err := pixelforge_project.Load(abs)
	require.NoError(t, err, "pre-v1 editor.pforge must load without error")

	require.NotEmpty(t, p.Scenes, "fixture must have at least one scene")
	for i, scene := range p.Scenes {
		assert.Equal(t, pixelforge_project.DefaultGridWidthScreens, scene.GridWidthScreens,
			"scene[%d] GridWidthScreens must default to %d",
			i, pixelforge_project.DefaultGridWidthScreens)
		assert.Equal(t, pixelforge_project.DefaultGridHeightScreens, scene.GridHeightScreens,
			"scene[%d] GridHeightScreens must default to %d",
			i, pixelforge_project.DefaultGridHeightScreens)

		assert.InDelta(t, pixelforge_project.DefaultCameraDeadZoneW, scene.Camera.DeadZoneW, 0.001,
			"scene[%d] Camera.DeadZoneW must default", i)
		assert.InDelta(t, pixelforge_project.DefaultCameraDeadZoneH, scene.Camera.DeadZoneH, 0.001,
			"scene[%d] Camera.DeadZoneH must default", i)
		assert.InDelta(t, pixelforge_project.DefaultCameraLookAheadX, scene.Camera.LookAheadX, 0.001,
			"scene[%d] Camera.LookAheadX must default", i)
		assert.InDelta(t, pixelforge_project.DefaultCameraSmoothingX, scene.Camera.SmoothingX, 0.001,
			"scene[%d] Camera.SmoothingX must default", i)
		assert.InDelta(t, pixelforge_project.DefaultCameraSmoothingY, scene.Camera.SmoothingY, 0.001,
			"scene[%d] Camera.SmoothingY must default", i)
	}
}

// -----------------------------------------------------------------------------
// Full preview loop — 600 ticks no panic, deterministic result
// -----------------------------------------------------------------------------

// TestE2E_FullPreviewLoop_NoPanic simulates 600 Follower.Update calls
// (10 seconds at 60fps) with a varying player trajectory and asserts
// (a) no panic, (b) the final camera position is deterministic given
// the same input sequence — a regression guard against the camera
// follower acquiring hidden non-deterministic state.
func TestE2E_FullPreviewLoop_NoPanic(t *testing.T) {
	resetCamera(t)
	p := loadMario(t)
	scene := &p.Scenes[0]

	cfg := pixelforge_camera.FollowerConfig{
		DeadZoneW:  scene.Camera.DeadZoneW,
		DeadZoneH:  scene.Camera.DeadZoneH,
		LookAheadX: scene.Camera.LookAheadX,
		SmoothingX: scene.Camera.SmoothingX,
		SmoothingY: scene.Camera.SmoothingY,
		BoundsW:    scene.GridWidthScreens * 256,
		BoundsH:    scene.GridHeightScreens * 240,
	}

	// Run #1: walk a deterministic trajectory, record final camera.
	follower1 := pixelforge_camera.NewFollower(cfg)
	for tick := 0; tick < 600; tick++ {
		// Player walks right, then oscillates around tile 200 to
		// exercise look-ahead direction flips without leaving the
		// level bounds. The trajectory is bounded so the camera
		// clamping path is also exercised mid-loop.
		var tileX int
		switch {
		case tick < 200:
			tileX = 4 + tick
		case tick < 400:
			tileX = 204 + (tick%20 - 10)
		default:
			tileX = 200 + (tick - 400)
		}
		assert.NotPanics(t, func() {
			follower1.Update(float64(tileX*8), float64(24*8), 1.0/60.0)
		})
	}
	final1X := pixelforge.Camera.X
	final1Y := pixelforge.Camera.Y

	// Run #2: identical trajectory must reach the identical camera
	// position. The Follower is constructed fresh + the camera is
	// reset between runs so leftover state from run #1 cannot bias
	// run #2.
	resetCamera(t)
	follower2 := pixelforge_camera.NewFollower(cfg)
	for tick := 0; tick < 600; tick++ {
		var tileX int
		switch {
		case tick < 200:
			tileX = 4 + tick
		case tick < 400:
			tileX = 204 + (tick%20 - 10)
		default:
			tileX = 200 + (tick - 400)
		}
		follower2.Update(float64(tileX*8), float64(24*8), 1.0/60.0)
	}
	final2X := pixelforge.Camera.X
	final2Y := pixelforge.Camera.Y

	assert.Equal(t, final1X, final2X,
		"identical input must yield identical final camera.X")
	assert.Equal(t, final1Y, final2Y,
		"identical input must yield identical final camera.Y")
}
