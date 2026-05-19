package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// tilepainter_widget_test.go covers the inspector widget logic that
// doesn't require a live ImGui frame: editor-state proxying, palette
// header text, active-rules counting, threshold alignment with the
// palette package's source of truth.

// TestEditor_SelectedTileAccessor: SetSelectedTile/SelectedTile
// proxy to the TilePainter's ActiveTileID so the inspector widget
// and the toolbar palette share one source of truth.
func TestEditor_SelectedTileAccessor(t *testing.T) {
	e := New()
	require.NotNil(t, e.Painter(),
		"NewWithSettings constructs the painter; bare New also wires it")
	assert.Equal(t, 0, e.SelectedTile(), "default selected tile is 0")
	e.SetSelectedTile(5)
	assert.Equal(t, 5, e.SelectedTile())
	assert.Equal(t, 5, e.Painter().ActiveTileID,
		"setter writes through to the painter")

	// Writing through the painter directly is also visible.
	e.Painter().SetActiveTile(9)
	assert.Equal(t, 9, e.SelectedTile())
}

// TestEditor_PaintSubModeAccessor: PaintSubMode/SetPaintSubMode
// proxy to the TilePainter's SubMode.
func TestEditor_PaintSubModeAccessor(t *testing.T) {
	e := New()
	assert.Equal(t, PaintBrush, e.PaintSubMode(), "default sub-mode is Brush")
	e.SetPaintSubMode(PaintBucket)
	assert.Equal(t, PaintBucket, e.PaintSubMode())
	assert.Equal(t, PaintBucket, e.Painter().SubMode)
}

// TestEditor_AccessorsNilSafe: a zero-value Editor without a wired
// painter doesn't crash on the accessors. The accessors silently
// return defaults — matches the existing nil-safe accessor pattern.
func TestEditor_AccessorsNilSafe(t *testing.T) {
	e := &Editor{}
	assert.Equal(t, 0, e.SelectedTile())
	assert.Equal(t, PaintBrush, e.PaintSubMode())
	// Setters must not panic.
	assert.NotPanics(t, func() {
		e.SetSelectedTile(7)
		e.SetPaintSubMode(PaintRectangle)
	})
}

// TestTilepainterDraw_RegistersOnInit: the U3 init() block
// registered the tilepainter drawer under the canonical name.
// Locked-down: tests in this package should never have to call
// RegisterWidget("tilepainter", ...) again.
func TestTilepainterDraw_RegistersOnInit(t *testing.T) {
	drawer, ok := pfcomponent.LookupWidget("tilepainter")
	require.True(t, ok, "init() must register tilepainter")
	assert.NotNil(t, drawer)
}

// TestTilepainterActiveRulesCount_NoneActive: an atlas with rules
// below the threshold reports 0 active rules.
func TestTilepainterActiveRulesCount_NoneActive(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		AutoTileRules: []pixelforge_project.AutoTileRule{
			{Count: 0}, {Count: 1}, {Count: 2},
		},
	}
	assert.Equal(t, 0, tilepainterCountActiveRules(atlas))
}

// TestTilepainterActiveRulesCount_SomeActive: only rules with Count
// >= threshold count toward the active total.
func TestTilepainterActiveRulesCount_SomeActive(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		AutoTileRules: []pixelforge_project.AutoTileRule{
			{Count: 2}, {Count: 3}, {Count: 5}, {Count: 1}, {Count: 10},
		},
	}
	assert.Equal(t, 3, tilepainterCountActiveRules(atlas),
		"three rules cross threshold=3")
}

// TestTilepainterActiveThresholdAlignment: the local
// activeRuleThreshold helper must report the same value the
// palette package's AutoTileActivationThreshold ships (= 3). The
// constant lives in the palette package; importing it here would
// introduce an editor → palette → editor cycle, so the test pins
// the literal value instead. If the palette constant bumps,
// editor/tilepainter_widget.go and this test update in lockstep.
func TestTilepainterActiveThresholdAlignment(t *testing.T) {
	assert.Equal(t, 3, activeRuleThreshold(),
		"the inspector widget's threshold must stay aligned with palette.AutoTileActivationThreshold")
}

// TestTilepainterPaletteHeader_NoSheet: an atlas with no
// SpriteSheetRef gets the "(no sheet — IDs only)" hint so designers
// understand the cell IDs aren't visually rendered until a sheet
// binds.
func TestTilepainterPaletteHeader_NoSheet(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	assert.Equal(t, "Palette (no sheet — IDs only)",
		tilepainterPaletteHeader(atlas))
}

// TestTilepainterPaletteHeader_SheetBound: a bound atlas shows the
// plain "Palette" header.
func TestTilepainterPaletteHeader_SheetBound(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{SpriteSheetRef: "ground_tiles"}
	assert.Equal(t, "Palette", tilepainterPaletteHeader(atlas))
}


// TestSetTileAtlasBlockPalette_WritesBlockCoord: a click at tile
// (5, 7) writes block (3, 2) — the integer division semantics the
// 2x2 mapping requires.
func TestSetTileAtlasBlockPalette_WritesBlockCoord(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	changed := SetTileAtlasBlockPalette(atlas, 5, 7, 2)
	assert.True(t, changed)
	require.Len(t, atlas.NESPaletteBlock, 4)
	require.Len(t, atlas.NESPaletteBlock[3], 3)
	assert.Equal(t, 2, atlas.NESPaletteBlock[3][2])
	// Padding entries default to the unassigned sentinel.
	assert.Equal(t, UnassignedNESPaletteBlock, atlas.NESPaletteBlock[3][0])
	assert.Equal(t, UnassignedNESPaletteBlock, atlas.NESPaletteBlock[3][1])
}

// TestSetTileAtlasBlockPalette_RejectsOutOfRangeIndex: indices
// outside 0..3 (the four BG sub-palettes) reject without mutating
// the atlas.
func TestSetTileAtlasBlockPalette_RejectsOutOfRangeIndex(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	assert.False(t, SetTileAtlasBlockPalette(atlas, 0, 0, 99))
	assert.False(t, SetTileAtlasBlockPalette(atlas, 0, 0, -5))
	assert.Empty(t, atlas.NESPaletteBlock,
		"rejected indices do not grow the matrix")
}

// TestSetTileAtlasBlockPalette_AcceptsUnassignedSentinel: writing
// the sentinel (-1) explicitly is allowed so designers can clear an
// assignment via the picker.
func TestSetTileAtlasBlockPalette_AcceptsUnassignedSentinel(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	require.True(t, SetTileAtlasBlockPalette(atlas, 0, 0, 2))
	changed := SetTileAtlasBlockPalette(atlas, 0, 0, UnassignedNESPaletteBlock)
	assert.True(t, changed)
	assert.Equal(t, UnassignedNESPaletteBlock, atlas.NESPaletteBlock[0][0])
}

// TestSetTileAtlasBlockPalette_NoOpWhenAlreadySet: writing the
// same value returns false so the caller knows nothing changed
// (avoids spurious MarkDirty calls).
func TestSetTileAtlasBlockPalette_NoOpWhenAlreadySet(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	require.True(t, SetTileAtlasBlockPalette(atlas, 0, 0, 2))
	assert.False(t, SetTileAtlasBlockPalette(atlas, 0, 0, 2))
}

// TestLookupTileAtlasBlockPalette_DefaultsToUnassigned: querying a
// block that's never been set returns the unassigned sentinel.
func TestLookupTileAtlasBlockPalette_DefaultsToUnassigned(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	assert.Equal(t, UnassignedNESPaletteBlock,
		LookupTileAtlasBlockPalette(atlas, 0, 0))
	assert.Equal(t, UnassignedNESPaletteBlock,
		LookupTileAtlasBlockPalette(atlas, 100, 100))
}

// TestLookupTileAtlasBlockPalette_ReturnsAssignedValue: after
// SetTileAtlasBlockPalette, the lookup returns the value.
func TestLookupTileAtlasBlockPalette_ReturnsAssignedValue(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{}
	SetTileAtlasBlockPalette(atlas, 4, 6, 3)
	assert.Equal(t, 3, LookupTileAtlasBlockPalette(atlas, 4, 6),
		"adjacent cells in the same block resolve to the same assignment")
	assert.Equal(t, 3, LookupTileAtlasBlockPalette(atlas, 5, 7))
}

// TestTilepainterActiveRulesTooltip_FormatsOutputs: the tooltip
// composes a comma-separated list of active rules' Output values.
// Designers see "Active outputs: →tile 5, →tile 7" instead of
// 3x3 pattern arrays they can't read.
func TestTilepainterActiveRulesTooltip_FormatsOutputs(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		AutoTileRules: []pixelforge_project.AutoTileRule{
			{Count: 5, Output: 5},
			{Count: 1, Output: 99}, // below threshold; ignored
			{Count: 3, Output: 7},
		},
	}
	got := tilepainterActiveRulesTooltip(atlas)
	assert.Contains(t, got, "→tile 5")
	assert.Contains(t, got, "→tile 7")
	assert.NotContains(t, got, "→tile 99",
		"sub-threshold rules don't appear in the active-outputs list")
}
