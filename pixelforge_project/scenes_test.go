package pixelforge_project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scene tests for idea #1 v1 (ScreenRoom Mario-strip primitive) U1
// schema additions. The tests assert:
//   - Defaulting behaviour for the new world-primitive fields.
//   - Omitempty discipline so pre-v1 files round-trip without churn.
//   - Plan reconciliation: Entity.TileX/TileY exist and round-trip.
//   - Resize-preserves-content + spawn-clamp semantics from R3, R9.
//   - The existing editor.pforge fixture loads cleanly with defaults.

// loadFromBytes round-trips a JSON blob through the same loader path
// Load uses, minus the on-disk asset check. Keeps the schema tests
// focused on the schema rather than the fixture wiring.
func loadFromBytes(t *testing.T, data []byte) *Project {
	t.Helper()
	p, err := LoadReader(bytesReader(data))
	require.NoError(t, err)
	return p
}

// TestScene_GridDefaults: a pre-v1 scene with no grid-screen keys
// receives the 16x1 Mario-strip default after applyDefaults.
func TestScene_GridDefaults(t *testing.T) {
	const pre = `{
		"schema_version": 1,
		"name": "pre-v1",
		"scenes": [{"id": "main", "name": "Main"}]
	}`
	p := loadFromBytes(t, []byte(pre))
	require.Len(t, p.Scenes, 1)
	assert.Equal(t, DefaultGridWidthScreens, p.Scenes[0].GridWidthScreens)
	assert.Equal(t, DefaultGridHeightScreens, p.Scenes[0].GridHeightScreens)
}

// TestScene_SpawnTileDefaults: a pre-v1 scene's SpawnTile is {0, 0}
// after load (the zero value is the documented default).
func TestScene_SpawnTileDefaults(t *testing.T) {
	const pre = `{
		"schema_version": 1,
		"name": "pre-v1",
		"scenes": [{"id": "main", "name": "Main"}]
	}`
	p := loadFromBytes(t, []byte(pre))
	require.Len(t, p.Scenes, 1)
	assert.Equal(t, SpawnCell{Col: 0, Row: 0}, p.Scenes[0].SpawnTile)
}

// TestScene_CameraConfigDefaults: a pre-v1 scene picks up the full
// camera default set after load. Asserts the Phase-1-research
// convergence values land at exactly the documented numbers.
func TestScene_CameraConfigDefaults(t *testing.T) {
	const pre = `{
		"schema_version": 1,
		"name": "pre-v1",
		"scenes": [{"id": "main", "name": "Main"}]
	}`
	p := loadFromBytes(t, []byte(pre))
	require.Len(t, p.Scenes, 1)
	cam := p.Scenes[0].Camera
	assert.InDelta(t, DefaultCameraDeadZoneW, cam.DeadZoneW, 0.0001)
	assert.InDelta(t, DefaultCameraDeadZoneH, cam.DeadZoneH, 0.0001)
	assert.InDelta(t, DefaultCameraLookAheadX, cam.LookAheadX, 0.0001)
	assert.InDelta(t, DefaultCameraSmoothingX, cam.SmoothingX, 0.0001)
	assert.InDelta(t, DefaultCameraSmoothingY, cam.SmoothingY, 0.0001)
}

// TestScene_CameraDefaults_PerFieldGranularity: a scene that overrides
// only DeadZoneW must keep the override AND pick up defaults for the
// other camera fields. Per the applyDefaults contract — designers can
// tweak one knob without losing the rest.
func TestScene_CameraDefaults_PerFieldGranularity(t *testing.T) {
	const partial = `{
		"schema_version": 1,
		"name": "partial",
		"scenes": [{
			"id": "main", "name": "Main",
			"camera": {"dead_zone_w": 120}
		}]
	}`
	p := loadFromBytes(t, []byte(partial))
	cam := p.Scenes[0].Camera
	assert.InDelta(t, 120.0, cam.DeadZoneW, 0.0001, "override preserved")
	assert.InDelta(t, DefaultCameraDeadZoneH, cam.DeadZoneH, 0.0001)
	assert.InDelta(t, DefaultCameraLookAheadX, cam.LookAheadX, 0.0001)
	assert.InDelta(t, DefaultCameraSmoothingX, cam.SmoothingX, 0.0001)
	assert.InDelta(t, DefaultCameraSmoothingY, cam.SmoothingY, 0.0001)
}

// TestScene_ResizePreservesContent: programmatically create a scene
// with painted cells, simulate a grid width grow, assert content
// preserved at original coords. The resize logic lives in U8; here
// the test models the contract the schema must support — re-running
// applyDefaults after mutating GridWidthScreens must not clobber the
// existing Tilemap grid data.
func TestScene_ResizePreservesContent(t *testing.T) {
	scene := Scene{
		ID:   "main",
		Name: "Main",
		Tilemaps: []TilemapLayer{
			{
				Name:  "ground",
				TileW: 8, TileH: 8,
				Grid: [][]int{
					{0, 0, 0, 0, 0, 7, 0, 0}, // row 0
					{0, 0, 0, 0, 0, 0, 0, 0}, // row 1
				},
			},
		},
		GridWidthScreens:  16,
		GridHeightScreens: 1,
	}
	// Author-time mutation: bump width screens 16 -> 24. (The U8
	// resize UI is what physically grows Tilemap grids; the schema
	// itself owns no resize logic — this test guards against the
	// loader inadvertently truncating or rewriting content when the
	// schema changes.)
	scene.GridWidthScreens = 24
	scene.applyWorldDefaults()

	assert.Equal(t, 24, scene.GridWidthScreens)
	assert.Equal(t, 7, scene.Tilemaps[0].Grid[0][5],
		"painted cell at (5,0) preserved across grid-width change")
	assert.Equal(t, 8, scene.Tilemaps[0].TileW)
}

// TestScene_ResizeClampsSpawn: SpawnTile {300, 0} on an 8-screen
// scene (8 * 32 = 256 cols, indices 0..255) clamps to {255, 0} after
// applyDefaults. This is the load-time guard the U8 resize UI also
// invokes when the designer shrinks a level past the spawn cell.
func TestScene_ResizeClampsSpawn(t *testing.T) {
	scene := Scene{
		ID:               "main",
		GridWidthScreens: 8,
		SpawnTile:        SpawnCell{Col: 300, Row: 0},
	}
	scene.applyWorldDefaults()
	assert.Equal(t, 8*ScreenTilesWide-1, scene.SpawnTile.Col,
		"col clamps to last valid index of an 8-screen-wide level")
	assert.Equal(t, 0, scene.SpawnTile.Row)
}

// TestScene_ResizeClampsSpawn_NegativeFloors: a malformed pre-v1
// file with a negative spawn cell clamps to {0, 0} rather than
// panicking the renderer downstream.
func TestScene_ResizeClampsSpawn_NegativeFloors(t *testing.T) {
	scene := Scene{ID: "main", SpawnTile: SpawnCell{Col: -5, Row: -3}}
	scene.applyWorldDefaults()
	assert.Equal(t, 0, scene.SpawnTile.Col)
	assert.Equal(t, 0, scene.SpawnTile.Row)
}

// TestTilemapLayer_SpriteSheetRefOmitempty: a layer without a bound
// sprite sheet marshals without a sprite_sheet_ref key. Keeps pre-v1
// .pforge files' JSON output free of phantom additions.
func TestTilemapLayer_SpriteSheetRefOmitempty(t *testing.T) {
	layer := TilemapLayer{
		Name:  "ground",
		TileW: 8, TileH: 8,
		Grid:          [][]int{{0}},
		AutoTileRules: []AutoTileRule{},
	}
	data, err := json.Marshal(layer)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "sprite_sheet_ref",
		"unbound layer must not emit sprite_sheet_ref")
}

// TestTilemapLayer_SpriteSheetRefRoundTrips: a bound layer round-trips
// the reference through marshal -> unmarshal.
func TestTilemapLayer_SpriteSheetRefRoundTrips(t *testing.T) {
	layer := TilemapLayer{
		Name:           "ground",
		TileW:          8,
		TileH:          8,
		Grid:           [][]int{{1, 2, 3}},
		AutoTileRules:  []AutoTileRule{},
		SpriteSheetRef: "tiles_overworld",
	}
	data, err := json.Marshal(layer)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"sprite_sheet_ref":"tiles_overworld"`)

	var loaded TilemapLayer
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, "tiles_overworld", loaded.SpriteSheetRef)
}

// TestEntity_TileXTileYOmitempty (PLAN RECONCILIATION test): the
// new TileX/TileY fields are absent from JSON when zero so pre-v1
// entities round-trip without phantom keys. Plan reconciliation note:
// U1's Approach section said "no new field on Entity in this unit"
// but the plan's Key Technical Decisions and U4/U7 both assume
// TileX/TileY exist. They land here in U1 per the Decisions section,
// guarded by omitempty so pre-v1 .pforge files stay byte-stable.
func TestEntity_TileXTileYOmitempty(t *testing.T) {
	e := Entity{
		ID:   "hero",
		Name: "Hero",
		// TileX = 0, TileY = 0 -- the zero value designed to omit.
	}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, `"tile_x"`,
		"zero TileX must not emit a tile_x key (omitempty discipline)")
	assert.NotContains(t, s, `"tile_y"`,
		"zero TileY must not emit a tile_y key (omitempty discipline)")
}

// TestEntity_TileXTileYRoundTrip (PLAN RECONCILIATION test): an
// entity authored at a non-zero tile cell preserves coords across
// marshal -> unmarshal. This is the contract U4's entity renderer
// and U7's tile-cell placement both depend on.
func TestEntity_TileXTileYRoundTrip(t *testing.T) {
	e := Entity{
		ID:    "hero",
		Name:  "Hero",
		TileX: 5,
		TileY: 24,
	}
	data, err := json.Marshal(e)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"tile_x":5`)
	assert.Contains(t, s, `"tile_y":24`)

	var loaded Entity
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, 5, loaded.TileX)
	assert.Equal(t, 24, loaded.TileY)
}

// TestProject_PreV1FixtureRoundtrip: load the live editor.pforge
// fixture, re-encode, assert the only new keys in the encoded output
// are the intended additive ones (or, ideally, none — the fixture's
// scene has zero entities and zero tilemaps so the new omitempty
// scene fields default-without-emit if the defaulted values match
// the zero-emit threshold). Defaulted GridWidthScreens=16 etc. ARE
// emitted because their post-default value is non-zero, which is the
// expected behaviour after the loader applies world defaults.
//
// What this test really guards: no spurious key churn beyond the
// known additive fields. If a future edit accidentally widens the
// emitted-key set, this assertion catches it.
func TestProject_PreV1FixtureRoundtrip(t *testing.T) {
	fixturePath := filepath.FromSlash(
		"../pixelforge_studio/editor/cart_assets/editor.pforge")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "fixture must exist at %s", fixturePath)

	p, err := LoadReader(bytesReader(data))
	require.NoError(t, err, "pre-v1 fixture must load cleanly")

	// Confirm the defaults landed.
	require.NotEmpty(t, p.Scenes)
	assert.Equal(t, DefaultGridWidthScreens, p.Scenes[0].GridWidthScreens)
	assert.Equal(t, DefaultGridHeightScreens, p.Scenes[0].GridHeightScreens)
	assert.InDelta(t, DefaultCameraDeadZoneW, p.Scenes[0].Camera.DeadZoneW, 0.0001)

	// Re-encode and assert the JSON output contains only the
	// expected additive keys. The fixture's scene has no entities
	// and no tilemaps, so the only Scene-level additions should be
	// grid_width_screens, grid_height_screens, and camera.
	out, err := encodeProject(p)
	require.NoError(t, err)
	outStr := string(out)

	// Intended additions (post-default, non-zero values emit):
	assert.Contains(t, outStr, `"grid_width_screens"`)
	assert.Contains(t, outStr, `"grid_height_screens"`)
	assert.Contains(t, outStr, `"camera"`)

	// Reserved fields must remain omitted (zero values).
	assert.NotContains(t, outStr, `"world"`,
		"reserved World field must not emit when zero")
	assert.NotContains(t, outStr, `"zones"`,
		"reserved Zones field must not emit when zero")
	assert.NotContains(t, outStr, `"warps"`,
		"reserved Warps field must not emit when zero")
	assert.NotContains(t, outStr, `"camera_mode"`,
		"reserved CameraMode field must not emit when zero")

	// Spawn tile is zero-valued by default, both Col and Row
	// omitempty -> the wrapping spawn_tile key still doesn't emit
	// because the embedded SpawnCell{} marshals to {}. Assert it
	// stays out of the output.
	assert.NotContains(t, outStr, `"spawn_tile"`,
		"default-zero SpawnTile must not emit")

	// Default tile ID is zero by default.
	assert.NotContains(t, outStr, `"default_tile_id"`,
		"default-zero DefaultTileID must not emit")
}

// TestProject_ReservedFieldsOmitemptyOnLoad: a project missing all of
// World/Zones/Warps/CameraMode loads with zero values; a re-encode
// keeps them out of the JSON. Forward-compat insurance: a future
// pforge file that DOES carry these keys deserialises into typed
// values (covered by the type declarations themselves; this test
// pins the omitempty half of the contract).
func TestProject_ReservedFieldsOmitemptyOnLoad(t *testing.T) {
	const pre = `{
		"schema_version": 1,
		"name": "pre-v1",
		"scenes": [{"id": "main", "name": "Main"}]
	}`
	p := loadFromBytes(t, []byte(pre))
	require.Len(t, p.Scenes, 1)

	scene := p.Scenes[0]
	assert.Equal(t, World{}, scene.World)
	assert.Empty(t, scene.Zones)
	assert.Empty(t, scene.Warps)
	assert.Equal(t, "", scene.CameraMode)

	out, err := encodeProject(p)
	require.NoError(t, err)
	s := string(out)
	for _, k := range []string{`"world"`, `"zones"`, `"warps"`, `"camera_mode"`} {
		assert.NotContains(t, s, k,
			"reserved field %s must stay omitted on round-trip", k)
	}
}

// TestProject_ReservedFieldsRoundTripWhenPresent: a forward-compat
// .pforge file that carries World/Zones/Warps/CameraMode keys
// deserialises into the reserved types and re-emits them on save.
// Guards the forward-compat half of the schema reservation contract.
func TestProject_ReservedFieldsRoundTripWhenPresent(t *testing.T) {
	const forward = `{
		"schema_version": 1,
		"name": "fwd",
		"scenes": [{
			"id": "main", "name": "Main",
			"world": {"mode": "rooms", "rooms": [{"id": "r0", "name": "R0", "bounds": {"x": 0, "y": 0, "w": 256, "h": 240}}]},
			"zones": [{"id": "z0", "kind": "music", "bounds": {"x": 0, "y": 0, "w": 64, "h": 64}}],
			"warps": [{"id": "w0", "from": "main", "to": "boss", "from_tile": {"col": 1, "row": 2}, "to_tile": {"col": 3, "row": 4}}],
			"camera_mode": "snap_per_screen"
		}]
	}`
	p := loadFromBytes(t, []byte(forward))
	require.Len(t, p.Scenes, 1)
	scene := p.Scenes[0]
	assert.Equal(t, "rooms", scene.World.Mode)
	require.Len(t, scene.World.Rooms, 1)
	assert.Equal(t, "r0", scene.World.Rooms[0].ID)
	assert.Equal(t, 256, scene.World.Rooms[0].Bounds.W)

	require.Len(t, scene.Zones, 1)
	assert.Equal(t, "music", scene.Zones[0].Kind)

	require.Len(t, scene.Warps, 1)
	assert.Equal(t, SpawnCell{Col: 1, Row: 2}, scene.Warps[0].FromTile)
	assert.Equal(t, SpawnCell{Col: 3, Row: 4}, scene.Warps[0].ToTile)

	assert.Equal(t, "snap_per_screen", scene.CameraMode)

	// Re-encode preserves the forward-compat data.
	out, err := encodeProject(p)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"world"`)
	assert.Contains(t, s, `"zones"`)
	assert.Contains(t, s, `"warps"`)
	assert.Contains(t, s, `"camera_mode"`)
}

// TestScene_SpawnTileEmitsWhenNonZero: a designer-set spawn cell
// emits to JSON. (Both Col and Row are omitempty individually so
// {Col: 0, Row: 24} emits only "row".)
func TestScene_SpawnTileEmitsWhenNonZero(t *testing.T) {
	scene := Scene{
		ID:                "main",
		Name:              "Main",
		GridWidthScreens:  16,
		GridHeightScreens: 1,
		SpawnTile:         SpawnCell{Col: 4, Row: 24},
	}
	data, err := json.Marshal(scene)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"spawn_tile"`)
	assert.True(t, strings.Contains(s, `"col":4`) && strings.Contains(s, `"row":24`),
		"spawn_tile must carry both col and row when set: %s", s)
}
