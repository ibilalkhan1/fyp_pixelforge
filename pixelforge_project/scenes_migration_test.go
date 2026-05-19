package pixelforge_project

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scenes_migration_test.go covers the load-time migration shim that
// idea #2 v1 introduces. The shim reads pre-v1 .pforge files that
// carry the legacy `tilemaps` JSON key and re-binds them to the
// renamed TileAtlases field. Re-marshal drops the legacy key and
// emits only `tile_atlases`.

// TestScene_LegacyTilemapsLoadsAsTileAtlases: a pre-v1 scene with the
// legacy `tilemaps` key + painted cells migrates to TileAtlases. The
// content is preserved field-for-field; the on-disk re-marshal uses
// only `tile_atlases`.
func TestScene_LegacyTilemapsLoadsAsTileAtlases(t *testing.T) {
	const legacy = `{
		"id": "main",
		"name": "Main",
		"entities": [],
		"tilemaps": [
			{
				"name": "ground",
				"tile_w": 8,
				"tile_h": 8,
				"grid": [[1, 2, 3]],
				"auto_tile_rules": []
			}
		]
	}`

	var s Scene
	require.NoError(t, json.Unmarshal([]byte(legacy), &s))
	require.Len(t, s.TileAtlases, 1, "legacy tilemaps key migrates to TileAtlases")
	assert.Equal(t, "ground", s.TileAtlases[0].Name)
	assert.Equal(t, [][]int{{1, 2, 3}}, s.TileAtlases[0].Grid)

	out, err := json.Marshal(s)
	require.NoError(t, err)
	got := string(out)
	assert.Contains(t, got, `"tile_atlases":`,
		"re-marshal writes the v1+ key")
	assert.NotContains(t, got, `"tilemaps":`,
		"re-marshal drops the legacy key")
}

// TestScene_LegacyAndNewFieldsCoexistPrefersNew: a hand-edited file
// that carries both `tilemaps` and `tile_atlases` keys resolves
// predictably: `tile_atlases` wins. The legacy key is logged and
// dropped on re-marshal.
func TestScene_LegacyAndNewFieldsCoexistPrefersNew(t *testing.T) {
	const both = `{
		"id": "main",
		"name": "Main",
		"entities": [],
		"tilemaps": [{"name": "legacy", "tile_w": 8, "tile_h": 8, "grid": [], "auto_tile_rules": []}],
		"tile_atlases": [{"name": "new", "tile_w": 8, "tile_h": 8, "grid": [], "auto_tile_rules": []}]
	}`

	var s Scene
	require.NoError(t, json.Unmarshal([]byte(both), &s))
	require.Len(t, s.TileAtlases, 1)
	assert.Equal(t, "new", s.TileAtlases[0].Name,
		"when both keys are present, tile_atlases wins")
}

// TestScene_LegacyEmptyTilemapsMigratesToEmptyTileAtlases: the only
// real fixture (editor.pforge prior to the bump) had `"tilemaps":[]`.
// Loading it must produce an empty TileAtlases slice with no errors
// and no spurious migration log churn beyond the migration message.
func TestScene_LegacyEmptyTilemapsMigratesToEmptyTileAtlases(t *testing.T) {
	const legacyEmpty = `{
		"id": "main",
		"name": "Main",
		"entities": [],
		"tilemaps": []
	}`
	var s Scene
	require.NoError(t, json.Unmarshal([]byte(legacyEmpty), &s))
	assert.Empty(t, s.TileAtlases,
		"empty legacy tilemaps migrates to empty TileAtlases")

	out, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(out), `"tilemaps":`,
		"re-marshal does not re-emit the legacy key")
}

// TestScene_ReservedFieldsOmitempty: a TileAtlas with zero-valued
// reserved fields (AnimationFps, ParallaxFactor, SlopeFlags,
// NESPaletteBlock) marshals without those keys so pre-v1 files and
// untouched-default v1 files stay byte-stable.
func TestScene_ReservedFieldsOmitempty(t *testing.T) {
	a := TileAtlas{
		Name:          "ground",
		TileW:         8,
		TileH:         8,
		Grid:          [][]int{{0}},
		AutoTileRules: []AutoTileRule{},
	}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	s := string(data)
	for _, k := range []string{`"animation_fps"`, `"parallax_factor"`,
		`"slope_flags"`, `"nes_palette_block"`} {
		assert.NotContains(t, s, k,
			"zero-valued reserved field %s must not emit", k)
	}
}

// TestScene_ReservedFieldsRoundTrip: a TileAtlas with each reserved
// field populated emits the keys and round-trips the values cleanly.
func TestScene_ReservedFieldsRoundTrip(t *testing.T) {
	a := TileAtlas{
		Name:            "fancy",
		TileW:           8,
		TileH:           8,
		Grid:            [][]int{{0}},
		AutoTileRules:   []AutoTileRule{},
		AnimationFps:    12,
		ParallaxFactor:  0.5,
		SlopeFlags:      []int{0, 1, 2},
		NESPaletteBlock: [][]int{{0, 1}, {2, 3}},
	}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, `"animation_fps":12`)
	assert.Contains(t, s, `"parallax_factor":0.5`)
	assert.Contains(t, s, `"slope_flags":[0,1,2]`)
	assert.Contains(t, s, `"nes_palette_block":[[0,1],[2,3]]`)

	var loaded TileAtlas
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, 12, loaded.AnimationFps)
	assert.InDelta(t, 0.5, loaded.ParallaxFactor, 0.0001)
	assert.Equal(t, []int{0, 1, 2}, loaded.SlopeFlags)
	assert.Equal(t, [][]int{{0, 1}, {2, 3}}, loaded.NESPaletteBlock)
}

// TestScene_SanitizeClampsOutOfRange: a hand-edited file with an
// AnimationFps above the pf-tag declared max clamps to the max on
// load. Same for ParallaxFactor. The pf-tag and the sanitize clamp
// share a single source of truth (AnimationFpsMax /
// ParallaxFactorMax) so the inspector slider and the schema never
// disagree.
func TestScene_SanitizeClampsOutOfRange(t *testing.T) {
	const bad = `{
		"id": "main",
		"name": "Main",
		"entities": [],
		"tile_atlases": [
			{
				"name": "bad",
				"tile_w": 8,
				"tile_h": 8,
				"grid": [],
				"auto_tile_rules": [],
				"animation_fps": 99,
				"parallax_factor": 7.5
			}
		]
	}`
	var s Scene
	require.NoError(t, json.Unmarshal([]byte(bad), &s))
	require.Len(t, s.TileAtlases, 1)
	assert.Equal(t, AnimationFpsMax, s.TileAtlases[0].AnimationFps,
		"out-of-range animation_fps clamps to the pf-tag max")
	assert.InDelta(t, ParallaxFactorMax, s.TileAtlases[0].ParallaxFactor, 0.0001,
		"out-of-range parallax_factor clamps to the pf-tag max")
}

// TestScene_SanitizeNaNFallsBackToZero: NaN in a hand-edited
// parallax_factor falls back to 0 rather than poisoning the
// downstream camera math.
func TestScene_SanitizeNaNFallsBackToZero(t *testing.T) {
	a := TileAtlas{
		Name:           "nan",
		TileW:          8,
		TileH:          8,
		Grid:           [][]int{{0}},
		AutoTileRules:  []AutoTileRule{},
		ParallaxFactor: math.NaN(),
	}
	s := &Scene{TileAtlases: []TileAtlas{a}}
	sanitizeReservedFields(s)
	assert.False(t, math.IsNaN(s.TileAtlases[0].ParallaxFactor),
		"NaN parallax_factor is replaced")
	assert.InDelta(t, 0.0, s.TileAtlases[0].ParallaxFactor, 0.0001)
}

// TestScene_SanitizeNegativeClampsToZero: negative reserved-field
// values clamp to 0 (the floor of the declared range).
func TestScene_SanitizeNegativeClampsToZero(t *testing.T) {
	const bad = `{
		"id": "main",
		"name": "Main",
		"entities": [],
		"tile_atlases": [
			{
				"name": "bad",
				"tile_w": 8,
				"tile_h": 8,
				"grid": [],
				"auto_tile_rules": [],
				"animation_fps": -5,
				"parallax_factor": -1.0
			}
		]
	}`
	var s Scene
	require.NoError(t, json.Unmarshal([]byte(bad), &s))
	assert.Equal(t, 0, s.TileAtlases[0].AnimationFps)
	assert.InDelta(t, 0.0, s.TileAtlases[0].ParallaxFactor, 0.0001)
}

// TestScene_NormalizeSlices_TileAtlasesNilBackfill: a Project whose
// scene has nil TileAtlases is normalized to an empty slice on load,
// matching the existing slice-nil-backfill discipline that the saver
// relies on for git-diff-friendly output.
func TestScene_NormalizeSlices_TileAtlasesNilBackfill(t *testing.T) {
	p := &Project{
		Scenes: []Scene{{ID: "main", Name: "Main"}},
	}
	require.Nil(t, p.Scenes[0].TileAtlases,
		"precondition: TileAtlases is nil")
	p.normalizeSlices()
	require.NotNil(t, p.Scenes[0].TileAtlases,
		"normalizeSlices replaces nil with empty slice")
	assert.Empty(t, p.Scenes[0].TileAtlases)
}

// TestScene_TileAtlasesRoundTrip: a scene with two populated
// TileAtlases (one with painted cells, one empty) marshals + un-
// marshals losslessly. Field identity is preserved across the trip.
func TestScene_TileAtlasesRoundTrip(t *testing.T) {
	s := Scene{
		ID:       "main",
		Name:     "Main",
		Entities: []Entity{},
		TileAtlases: []TileAtlas{
			{
				Name:          "ground",
				TileW:         8,
				TileH:         8,
				Grid:          [][]int{{0, 0, 1}, {1, 0, 0}},
				AutoTileRules: []AutoTileRule{},
			},
			{
				Name:          "decoration",
				TileW:         8,
				TileH:         8,
				Grid:          [][]int{{}},
				AutoTileRules: []AutoTileRule{},
			},
		},
	}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), `"tile_atlases":`),
		"core key emitted")

	var loaded Scene
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded.TileAtlases, 2)
	assert.Equal(t, "ground", loaded.TileAtlases[0].Name)
	assert.Equal(t, [][]int{{0, 0, 1}, {1, 0, 0}}, loaded.TileAtlases[0].Grid)
	assert.Equal(t, "decoration", loaded.TileAtlases[1].Name)
}
