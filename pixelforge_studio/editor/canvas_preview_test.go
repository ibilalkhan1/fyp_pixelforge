// canvas_preview_test.go covers the U5 editor-preview integration of
// the three idea #1 engine packages: pixelforge_camera (camera
// follower), pixelforge_tilemap (tilemap renderer), and
// pixelforge_entity (entity sprite renderer).
//
// These tests target the seams sceneGame exposes for hermetic
// testability — tilemapRenderFn / entityRenderFn / sheetResolver —
// rather than booting the real engine. They prove that the editor
// preview wires the right data through to the right renderer at the
// right tick, which is the U5 contract.
package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// configureMarioScene mutates the editor's default scene into a
// minimal Mario-strip fixture: a single 16x1-screen level with the
// camera defaults populated. Tests that need a Player append one.
func configureMarioScene(t *testing.T, e *Editor) *pixelforge_project.Scene {
	t.Helper()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.GridWidthScreens = pixelforge_project.DefaultGridWidthScreens
	scene.GridHeightScreens = pixelforge_project.DefaultGridHeightScreens
	scene.Camera = pixelforge_project.SceneCameraConfig{
		DeadZoneW:  pixelforge_project.DefaultCameraDeadZoneW,
		DeadZoneH:  pixelforge_project.DefaultCameraDeadZoneH,
		LookAheadX: pixelforge_project.DefaultCameraLookAheadX,
		SmoothingX: pixelforge_project.DefaultCameraSmoothingX,
		SmoothingY: pixelforge_project.DefaultCameraSmoothingY,
	}
	return scene
}

// resetCameraGlobal nulls pixelforge.Camera so a test starts from a
// known baseline + the prior test's camera state cannot leak in.
func resetCameraGlobal(t *testing.T) {
	t.Helper()
	prev := pixelforge.Camera
	pixelforge.Camera = pixelforge.Position{}
	t.Cleanup(func() { pixelforge.Camera = prev })
}

// TestPreview_TilemapRenders — sceneGame.drawWorld dispatches to the
// tilemap renderer for each layer whose SpriteSheetRef resolves to a
// decoded sheet. Uses the sheetResolver seam to inject a fake
// decoded canvas + the tilemapRenderFn seam to spy on the call.
func TestPreview_TilemapRenders(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)

	// Register a sprite asset so the resolver lookup succeeds, and a
	// tilemap layer that binds to it.
	e.project.Sprites = append(e.project.Sprites, pixelforge_project.SpriteAsset{
		Name:   "overworld_tiles",
		Width:  32, Height: 8,
		FrameW: 8, FrameH: 8,
	})
	scene.Tilemaps = []pixelforge_project.TilemapLayer{{
		Name:           "ground",
		TileW:          8,
		TileH:          8,
		SpriteSheetRef: "overworld_tiles",
		Grid: [][]int{
			{1, 0, 2},
		},
	}}

	g := newSceneGame(e)

	// Install a fake sheet so the resolver returns a non-nil canvas.
	fakeSheet := pixelforge.NewCanvas(32, 8)
	g.sheetResolver = func(asset *pixelforge_project.SpriteAsset) (*pixelforge.Canvas, bool) {
		assert.Equal(t, "overworld_tiles", asset.Name)
		return &fakeSheet, true
	}

	// Spy on the tilemap renderer.
	var renderCalls []*pixelforge_project.TilemapLayer
	g.tilemapRenderFn = func(layer *pixelforge_project.TilemapLayer, sheet *pixelforge.Canvas) {
		renderCalls = append(renderCalls, layer)
		assert.Same(t, &fakeSheet, sheet)
	}
	g.entityRenderFn = func(_ []pixelforge_project.Entity, _ []pixelforge_project.SpriteAsset) {}

	g.drawWorld()

	require.Len(t, renderCalls, 1)
	assert.Equal(t, "ground", renderCalls[0].Name)
}

// TestPreview_TilemapSkippedWhenSheetUnresolved — when the
// sheetResolver returns (nil, false) the renderer is NOT called.
// This is the documented behavior for layers whose sheet has not
// been decoded yet (which is the v1 default since no decoder
// pipeline exists in U5's scope).
func TestPreview_TilemapSkippedWhenSheetUnresolved(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	e.project.Sprites = append(e.project.Sprites, pixelforge_project.SpriteAsset{Name: "tiles"})
	scene.Tilemaps = []pixelforge_project.TilemapLayer{{
		Name: "ground", TileW: 8, TileH: 8, SpriteSheetRef: "tiles",
		Grid: [][]int{{1}},
	}}

	g := newSceneGame(e)
	// Default sheetResolver is nil → resolveSheet returns (nil, false).
	called := 0
	g.tilemapRenderFn = func(_ *pixelforge_project.TilemapLayer, _ *pixelforge.Canvas) { called++ }
	g.entityRenderFn = func(_ []pixelforge_project.Entity, _ []pixelforge_project.SpriteAsset) {}

	g.drawWorld()

	assert.Equal(t, 0, called, "tilemap render should be skipped when sheet does not resolve")
}

// TestPreview_EntitiesRender — sceneGame.drawWorld always dispatches
// to the entity renderer with the scene's entities + the project's
// sprites, regardless of how many tilemaps are present.
func TestPreview_EntitiesRender(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "p", Archetype: pixelforge_project.ArchetypePlayer, TileX: 4, TileY: 24},
		{ID: "e1", Archetype: pixelforge_project.ArchetypeEnemy, TileX: 10, TileY: 24},
		{ID: "e2", Archetype: pixelforge_project.ArchetypeEnemy, TileX: 20, TileY: 24},
	}

	g := newSceneGame(e)
	var passedEntities []pixelforge_project.Entity
	g.entityRenderFn = func(ents []pixelforge_project.Entity, sprites []pixelforge_project.SpriteAsset) {
		passedEntities = ents
		assert.NotNil(t, sprites)
	}
	g.tilemapRenderFn = func(_ *pixelforge_project.TilemapLayer, _ *pixelforge.Canvas) {}

	g.drawWorld()

	require.Len(t, passedEntities, 3)
	assert.Equal(t, "p", passedEntities[0].ID)
}

// TestPreview_CameraTracksPlayer — when the Player moves outside the
// dead-zone, several preview ticks advance pixelforge.Camera.X
// toward the player. The exact converged value depends on the
// smoothing constants; the assertion is the strictly-monotonic
// direction of travel.
func TestPreview_CameraTracksPlayer(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)

	player := pixelforge_project.Entity{
		ID: "mario", Archetype: pixelforge_project.ArchetypePlayer,
		TileX: 4, TileY: 24,
	}
	scene.Entities = []pixelforge_project.Entity{player}

	g := newSceneGame(e)
	// Establish baseline velocity on the first tick.
	g.updateCamera(1.0 / 60.0)
	startCam := pixelforge.Camera.X

	// Move the player far to the right of the dead-zone, then tick
	// several frames. Camera should track right.
	scene.Entities[0].TileX = 20 // 160 px right of original (4*8 -> 20*8 = 160)
	for i := 0; i < 30; i++ {
		g.updateCamera(1.0 / 60.0)
	}
	assert.Greater(t, pixelforge.Camera.X, startCam, "camera should advance after player exits dead-zone")
}

// TestPreview_CameraStopsAtLevelEdge — when the player walks toward
// the right edge of the level, the camera clamps at (BoundsW -
// viewport) and never overshoots. Drives the player past the edge
// and ticks long enough for the smoothing to fully settle.
func TestPreview_CameraStopsAtLevelEdge(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)

	// Player far past the right edge of a 16x1 level.
	// Level pixel width = 16 * 256 = 4096; viewport = 256 default.
	// Max camera = 4096 - 256 = 3840.
	scene.Entities = []pixelforge_project.Entity{{
		ID: "mario", Archetype: pixelforge_project.ArchetypePlayer,
		TileX: 510, TileY: 24,
	}}

	g := newSceneGame(e)
	for i := 0; i < 600; i++ { // 10 s of preview at 60 fps — well past settle.
		g.updateCamera(1.0 / 60.0)
	}
	assert.LessOrEqual(t, pixelforge.Camera.X, 3840, "camera must clamp at level right edge")
	assert.Greater(t, pixelforge.Camera.X, 3800, "camera should have settled at (or near) the clamp")
}

// TestPreview_NoPlayer_CameraStill — when the active scene has no
// Player entity, sceneGame.Update is a camera no-op: pixelforge.
// Camera stays at its prior value.
func TestPreview_NoPlayer_CameraStill(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	// Deliberately tag every entity as non-Player.
	scene.Entities = []pixelforge_project.Entity{
		{ID: "e1", Archetype: pixelforge_project.ArchetypeEnemy, TileX: 8, TileY: 24},
	}
	pixelforge.Camera = pixelforge.Position{X: 42, Y: 0}

	g := newSceneGame(e)
	for i := 0; i < 60; i++ {
		require.NoError(t, g.Update())
	}
	assert.Equal(t, 42, pixelforge.Camera.X, "no Player → camera state preserved")
}

// TestPreview_ViewBoxAccountsForCameraOffset — with the camera
// scrolled by 100 scene-px to the right, the viewBox shifts left by
// 100 (at native 1:1 scale) so a click at screen-coord (50, 0) maps
// to scene-coord (150, 0) through windowToScene.
func TestPreview_ViewBoxAccountsForCameraOffset(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	p := e.project
	p.ScreenWidth = 320
	p.ScreenHeight = 180

	// Use an area exactly matching the project's screen size so
	// scaleX = scaleY = 1 and the math is trivially observable.
	area := widgets.Rect{X: 0, Y: 0, W: 320, H: 180}

	// With camera at zero, viewBox starts at (0, 0).
	vb0 := e.canvas.viewBox(area, p)
	assert.Equal(t, 0, vb0.X)
	x0, y0 := e.canvas.windowToScene(vb0, p, 50, 0)
	assert.Equal(t, 50, x0)
	assert.Equal(t, 0, y0)

	// Scroll the camera right by 100 scene-pixels.
	pixelforge.Camera = pixelforge.Position{X: 100, Y: 0}
	vb1 := e.canvas.viewBox(area, p)
	assert.Equal(t, -100, vb1.X, "viewBox shifts left by camera offset")

	x1, y1 := e.canvas.windowToScene(vb1, p, 50, 0)
	// Click at screen-coord (50, 0) → scene-coord (150, 0).
	assert.Equal(t, 150, x1)
	assert.Equal(t, 0, y1)
}

// TestPreview_DragClampWidensToLevelBounds — sceneDragBounds returns
// GridWidthScreens * ScreenWidth (256) for a Mario-strip scene, not
// the project's pre-v1 single-screen ScreenWidth.
func TestPreview_DragClampWidensToLevelBounds(t *testing.T) {
	e := New()
	scene := configureMarioScene(t, e)

	maxX, maxY := sceneDragBounds(scene, e.project)
	// 16 screens × 256 = 4096; 1 screen × 240 = 240.
	assert.Equal(t, float64(16*256), maxX)
	assert.Equal(t, float64(1*240), maxY)
}

// TestPreview_DragClampFallsBackForPreV1 — when a scene has no
// grid-screens fields populated (legacy fresh-from-NewProject), the
// clamp falls back to the project's ScreenWidth/Height so existing
// single-screen examples still pin entities inside the viewport.
func TestPreview_DragClampFallsBackForPreV1(t *testing.T) {
	e := New()
	scene := e.activeScene()
	// Explicitly leave GridWidthScreens / GridHeightScreens at zero
	// to simulate a pre-v1 scene that hasn't been through
	// applyDefaults.
	scene.GridWidthScreens = 0
	scene.GridHeightScreens = 0

	e.project.ScreenWidth = 320
	e.project.ScreenHeight = 180
	maxX, maxY := sceneDragBounds(scene, e.project)
	assert.Equal(t, float64(320), maxX)
	assert.Equal(t, float64(180), maxY)
}

// TestPreview_FollowerCacheReusedAcrossTicks — the cached Follower
// pointer is identical across consecutive updateCamera calls when
// the scene + config + bounds don't change. Proves the lookahead /
// smoothing accumulators persist across frames.
func TestPreview_FollowerCacheReusedAcrossTicks(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	scene.Entities = []pixelforge_project.Entity{{
		ID: "p", Archetype: pixelforge_project.ArchetypePlayer, TileX: 4, TileY: 24,
	}}

	g := newSceneGame(e)
	g.updateCamera(1.0 / 60.0)
	first := g.follower
	g.updateCamera(1.0 / 60.0)
	second := g.follower
	assert.Same(t, first, second, "Follower should be cached across ticks for the same scene config")
}

// TestSpawn_PlayerTeleportsOnStart — when SpawnTile is non-default,
// the first preview frame teleports the Player from its authored
// TileX / TileY to the spawn cell. Verified via TeleportPlayerToSpawn
// directly (the same helper sceneGame.ensureFollower invokes on the
// scene-change frame) so the test is deterministic regardless of any
// follower caching state.
func TestSpawn_PlayerTeleportsOnStart(t *testing.T) {
	scene := &pixelforge_project.Scene{
		ID:                "sc-1",
		GridWidthScreens:  16,
		GridHeightScreens: 1,
		SpawnTile:         pixelforge_project.SpawnCell{Col: 8, Row: 24},
		Entities: []pixelforge_project.Entity{{
			ID:        "mario",
			Archetype: pixelforge_project.ArchetypePlayer,
			TileX:     2, TileY: 24,
		}},
	}

	TeleportPlayerToSpawn(scene)

	assert.Equal(t, 8, scene.Entities[0].TileX, "Player teleported to SpawnTile.Col")
	assert.Equal(t, 24, scene.Entities[0].TileY, "Player teleported to SpawnTile.Row")
	// Position mirrors the spawn cell's pixel coords (TileW=TileH=8).
	assert.Equal(t, float64(64), scene.Entities[0].Position.X)
	assert.Equal(t, float64(192), scene.Entities[0].Position.Y)
}

// TestSpawn_DefaultSpawnTileNoTeleport — SpawnTile == {0, 0} is the
// "no override" sentinel. Player stays at its authored position so
// designers can preview a scene without first setting a spawn cell.
func TestSpawn_DefaultSpawnTileNoTeleport(t *testing.T) {
	scene := &pixelforge_project.Scene{
		ID: "sc-1",
		// SpawnTile left at zero-value (Col=0, Row=0).
		Entities: []pixelforge_project.Entity{{
			ID:        "mario",
			Archetype: pixelforge_project.ArchetypePlayer,
			TileX:     5, TileY: 24,
		}},
	}

	TeleportPlayerToSpawn(scene)

	assert.Equal(t, 5, scene.Entities[0].TileX, "Player position preserved when SpawnTile is default")
	assert.Equal(t, 24, scene.Entities[0].TileY)
}

// TestSpawn_NoPlayerEntityIsNoOp — a scene with no Player entity is a
// silent no-op (matches the camera-follower's "no Player → camera
// still" contract).
func TestSpawn_NoPlayerEntityIsNoOp(t *testing.T) {
	scene := &pixelforge_project.Scene{
		ID:        "sc-1",
		SpawnTile: pixelforge_project.SpawnCell{Col: 8, Row: 24},
		Entities: []pixelforge_project.Entity{{
			ID:        "e1",
			Archetype: pixelforge_project.ArchetypeEnemy,
			TileX:     3, TileY: 24,
		}},
	}

	// Does not panic, does not mutate the enemy.
	TeleportPlayerToSpawn(scene)
	assert.Equal(t, 3, scene.Entities[0].TileX)
}

// TestSpawn_FiresOnFirstFrameOfScene — when sceneGame.ensureFollower
// detects a scene-ID change (first frame of a scene === preview
// restart in v1), it invokes TeleportPlayerToSpawn so the Player
// lands at the spawn cell before the follower's first Update reads
// the player position. Validates the integration through updateCamera.
func TestSpawn_FiresOnFirstFrameOfScene(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	scene.ID = "sc-1"
	scene.SpawnTile = pixelforge_project.SpawnCell{Col: 12, Row: 24}
	scene.Entities = []pixelforge_project.Entity{{
		ID:        "mario",
		Archetype: pixelforge_project.ArchetypePlayer,
		TileX:     2, TileY: 24,
	}}

	g := newSceneGame(e)
	g.updateCamera(1.0 / 60.0)

	assert.Equal(t, 12, scene.Entities[0].TileX, "first frame should teleport Player to SpawnTile")
	assert.Equal(t, 24, scene.Entities[0].TileY)
}

// TestPreview_FollowerCacheRebuildsOnSceneChange — changing the
// scene's ID invalidates the cache and the next updateCamera
// reconstructs the Follower. Belt-and-braces for multi-scene
// projects (v1 only has one scene but the cache key contract should
// hold).
func TestPreview_FollowerCacheRebuildsOnSceneChange(t *testing.T) {
	resetCameraGlobal(t)
	e := New()
	scene := configureMarioScene(t, e)
	scene.ID = "scene-a"
	scene.Entities = []pixelforge_project.Entity{{
		ID: "p", Archetype: pixelforge_project.ArchetypePlayer, TileX: 4, TileY: 24,
	}}

	g := newSceneGame(e)
	g.updateCamera(1.0 / 60.0)
	first := g.follower
	require.NotNil(t, first)

	scene.ID = "scene-b"
	g.updateCamera(1.0 / 60.0)
	second := g.follower
	assert.NotSame(t, first, second, "scene-ID change should rebuild the Follower")
}
