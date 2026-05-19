package pixelforge_project

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// palette_defaults_test.go covers idea #3 v1 U1's load-time backfill
// for sub-palette overlays + the SpriteAsset.SubPalette defaulting +
// the EditorOverlays defaulting. Each test exercises one path:
// default-when-empty, preserve-when-set, clamp-out-of-range,
// round-trip-faithfully.

// TestPaletteData_DefaultsBackfillSubPalettes: a project loaded from
// a JSON blob with no sub-palette overlays receives the canonical
// bg_0..bg_3 + sprite_0..sprite_3 assignment after applyDefaults.
func TestPaletteData_DefaultsBackfillSubPalettes(t *testing.T) {
	const pre = `{"schema_version": 1, "name": "pre", "scenes": []}`
	p := loadFromBytes(t, []byte(pre))
	assert.Equal(t, "bg_0", p.Palette.BGSubPalettes[0].Name)
	assert.Equal(t, [4]int{1, 2, 3, 4}, p.Palette.BGSubPalettes[0].Slots)
	assert.Equal(t, "bg_3", p.Palette.BGSubPalettes[3].Name)
	assert.Equal(t, "sprite_0", p.Palette.SpriteSubPalettes[0].Name)
	assert.Equal(t, [4]int{17, 18, 19, 20}, p.Palette.SpriteSubPalettes[0].Slots)
	assert.Equal(t, "sprite_3", p.Palette.SpriteSubPalettes[3].Name)
}

// TestPaletteData_RoundTripWithCustomSubPalettes: a palette with
// designer-edited sub-palette names + slot assignments round-trips
// losslessly through marshal/unmarshal.
func TestPaletteData_RoundTripWithCustomSubPalettes(t *testing.T) {
	pd := DefaultPalette()
	pd.BGSubPalettes = [4]SubPalette{
		{Name: "ground", Slots: [4]int{1, 5, 9, 13}},
		{Name: "sky", Slots: [4]int{2, 6, 10, 14}},
		{Name: "water", Slots: [4]int{3, 7, 11, 15}},
		{Name: "fire", Slots: [4]int{4, 8, 12, 16}},
	}
	data, err := json.Marshal(pd)
	require.NoError(t, err)

	var loaded PaletteData
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, "ground", loaded.BGSubPalettes[0].Name)
	assert.Equal(t, [4]int{1, 5, 9, 13}, loaded.BGSubPalettes[0].Slots)
	assert.Equal(t, "fire", loaded.BGSubPalettes[3].Name)
}

// TestPaletteData_SanitizeClampsOutOfRangeSlots: applyDefaults
// clamps slot indices outside [0, 64) into the legal range so the
// inspector and renderer never see a panic-inducing value.
func TestPaletteData_SanitizeClampsOutOfRangeSlots(t *testing.T) {
	pd := DefaultPalette()
	pd.BGSubPalettes[0] = SubPalette{Name: "bg_0", Slots: [4]int{70, -3, 64, 100}}
	pd.SpriteSubPalettes[1] = SubPalette{Name: "sprite_1", Slots: [4]int{-1, 99, 5, MaxColors - 1}}

	pd.ApplyDefaults()

	assert.Equal(t, [4]int{MaxColors - 1, 0, MaxColors - 1, MaxColors - 1},
		pd.BGSubPalettes[0].Slots, "BG slots clamp")
	assert.Equal(t, [4]int{0, MaxColors - 1, 5, MaxColors - 1},
		pd.SpriteSubPalettes[1].Slots, "Sprite slots clamp")
}

// TestPaletteData_IdempotentOnPopulated: re-running applyDefaults on
// an already-populated palette preserves designer overrides — the
// populated-check goes by Name string, not by per-slot diff.
func TestPaletteData_IdempotentOnPopulated(t *testing.T) {
	pd := DefaultPalette()
	pd.ApplyDefaults()
	customised := pd.BGSubPalettes[2]
	customised.Name = "custom_bg_2"
	pd.BGSubPalettes[2] = customised

	pd.ApplyDefaults() // re-run

	assert.Equal(t, "custom_bg_2", pd.BGSubPalettes[2].Name,
		"second applyDefaults must not overwrite designer-edited names")
}

// TestSpriteAsset_SubPaletteOmitEmpty: marshal a sprite with no
// SubPalette set — no `sub_palette` key surfaces in JSON.
func TestSpriteAsset_SubPaletteOmitEmpty(t *testing.T) {
	s := SpriteAsset{Name: "hero", Width: 8, Height: 8}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "sub_palette",
		"empty SubPalette must not emit a sub_palette key")
}

// TestSpriteAsset_SubPaletteDefaultsToSprite0: a project loaded with
// a sprite that has no SubPalette assignment receives the
// DefaultSubPaletteName via applyDefaults.
func TestSpriteAsset_SubPaletteDefaultsToSprite0(t *testing.T) {
	const pre = `{
		"schema_version": 1,
		"name": "pre",
		"sprites": [{"name": "hero", "width": 8, "height": 8, "frame_w": 8, "frame_h": 8}],
		"scenes": []
	}`
	p := loadFromBytes(t, []byte(pre))
	require.Len(t, p.Sprites, 1)
	assert.Equal(t, DefaultSubPaletteName, p.Sprites[0].SubPalette,
		"sprite with no SubPalette gets the default sprite_0")
}

// TestSpriteAsset_SubPaletteRoundTrip: a sprite with an explicit
// SubPalette ("sprite_2") round-trips through marshal/unmarshal.
func TestSpriteAsset_SubPaletteRoundTrip(t *testing.T) {
	s := SpriteAsset{Name: "hero", SubPalette: "sprite_2"}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"sub_palette":"sprite_2"`)
	var loaded SpriteAsset
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, "sprite_2", loaded.SubPalette)
}

// TestEditorOverlays_DefaultsToBothEnabled: a project loaded with no
// editor_overlays key receives ScanlineEnabled=true and
// PaletteBlockEnabled=true via applyDefaults.
func TestEditorOverlays_DefaultsToBothEnabled(t *testing.T) {
	const pre = `{"schema_version": 1, "name": "pre", "scenes": []}`
	p := loadFromBytes(t, []byte(pre))
	assert.True(t, p.EditorOverlays.ScanlineEnabled)
	assert.True(t, p.EditorOverlays.PaletteBlockEnabled)
	assert.True(t, p.EditorOverlays.Set,
		"Set is the marker that distinguishes 'designer authored' from 'never touched'")
}

// TestEditorOverlays_RoundTripFalseFalse: a project with both
// overlays explicitly turned off round-trips through save/load
// without flipping back on. The Set field carries the intent across
// the omitempty boundary.
func TestEditorOverlays_RoundTripFalseFalse(t *testing.T) {
	p := NewProject("t")
	p.applyDefaults()
	p.EditorOverlays.ScanlineEnabled = false
	p.EditorOverlays.PaletteBlockEnabled = false
	// Set stays true (applyDefaults flipped it on construction).

	data, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"editor_overlays"`,
		"explicit {false,false,set:true} state survives omitempty")

	var loaded Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	loaded.applyDefaults()
	assert.False(t, loaded.EditorOverlays.ScanlineEnabled,
		"explicit-off survives reload")
	assert.False(t, loaded.EditorOverlays.PaletteBlockEnabled,
		"explicit-off survives reload")
}

// TestEditorOverlays_NewProjectDefaultsBothTrue: NewProject doesn't
// call applyDefaults itself (the loader does); but after applyDefaults
// the freshly-constructed project shows both overlays enabled.
func TestEditorOverlays_NewProjectDefaultsBothTrue(t *testing.T) {
	p := NewProject("t")
	p.applyDefaults()
	assert.True(t, p.EditorOverlays.ScanlineEnabled)
	assert.True(t, p.EditorOverlays.PaletteBlockEnabled)
}

// TestEditorOverlays_LegacyProjectLoadsWithDefaults: the canonical
// editor.pforge fixture (no editor_overlays key) loads with both
// overlays enabled — pre-v1 projects pick up the new authoring hint.
func TestEditorOverlays_LegacyProjectLoadsWithDefaults(t *testing.T) {
	const legacy = `{"schema_version": 1, "name": "legacy", "scenes": [{"id": "main", "name": "Main"}]}`
	p := loadFromBytes(t, []byte(legacy))
	assert.True(t, p.EditorOverlays.ScanlineEnabled)
	assert.True(t, p.EditorOverlays.PaletteBlockEnabled)
}
