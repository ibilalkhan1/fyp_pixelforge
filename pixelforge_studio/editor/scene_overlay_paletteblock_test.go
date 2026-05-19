package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// scene_overlay_paletteblock_test.go covers U7's violation
// detection. The paint path is asserted via U9 e2e tests; here the
// focus is the pure-logic helpers.

// TestFindPaletteBlockViolations_PaintedUnassignedFlagged: an atlas
// with a painted cell at (0, 0) and no NESPaletteBlock assignment
// surfaces one violation at block (0, 0).
func TestFindPaletteBlockViolations_PaintedUnassignedFlagged(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{1, 0},
			{0, 0},
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	require.Len(t, violations, 1)
	assert.Equal(t, 0, violations[0].BlockCol)
	assert.Equal(t, 0, violations[0].BlockRow)
	assert.Equal(t, 0, violations[0].PixelX)
	assert.Equal(t, 0, violations[0].PixelY)
	assert.Equal(t, 16, violations[0].PixelW, "block width = 2 tiles * 8 px")
	assert.Equal(t, 16, violations[0].PixelH)
}

// TestFindPaletteBlockViolations_PaintedAssignedNotFlagged: block
// with painted cells AND an explicit NESPaletteBlock assignment
// (value in [0, 4)) does not violate.
func TestFindPaletteBlockViolations_PaintedAssignedNotFlagged(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{1, 0},
			{0, 0},
		},
		NESPaletteBlock: [][]int{
			{2},
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	assert.Empty(t, violations, "explicit bg_2 assignment suppresses the warning")
}

// TestFindPaletteBlockViolations_EmptyBlockNotFlagged: a 2x2 region
// with all-zero cells (no paint) does not violate even when no
// NESPaletteBlock assignment exists.
func TestFindPaletteBlockViolations_EmptyBlockNotFlagged(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{0, 0},
			{0, 0},
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	assert.Empty(t, violations, "empty blocks are not violations")
}

// TestFindPaletteBlockViolations_MultipleViolations: a 4x4 grid
// with two painted but unassigned blocks surfaces two violations.
func TestFindPaletteBlockViolations_MultipleViolations(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{1, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 5, 0},
			{0, 0, 0, 0},
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	require.Len(t, violations, 2)
	// First: block (0, 0) — painted (0, 0).
	// Second: block (1, 1) — painted (2, 2).
	assert.Equal(t, 0, violations[0].BlockCol)
	assert.Equal(t, 0, violations[0].BlockRow)
	assert.Equal(t, 1, violations[1].BlockCol)
	assert.Equal(t, 1, violations[1].BlockRow)
}

// TestFindPaletteBlockViolations_PartialAssignmentMatrixFlagsUnassignedOnly:
// a NESPaletteBlock matrix that has a row but doesn't cover every
// column — uncovered cells are treated as unassigned.
func TestFindPaletteBlockViolations_PartialAssignmentMatrixFlagsUnassignedOnly(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{1, 0, 0, 0},
			{0, 0, 0, 0},
			{0, 0, 5, 0},
			{0, 0, 0, 0},
		},
		NESPaletteBlock: [][]int{
			{2}, // block (0, 0) assigned to bg_2; block (0, 1) unassigned
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	require.Len(t, violations, 1)
	assert.Equal(t, 1, violations[0].BlockCol)
	assert.Equal(t, 1, violations[0].BlockRow,
		"only the painted-and-unassigned block surfaces")
}

// TestFindPaletteBlockViolations_SentinelExplicitlyUnassignedFlags:
// an NESPaletteBlock entry set to the unassigned sentinel (-1)
// explicitly is treated the same as "missing entry."
func TestFindPaletteBlockViolations_SentinelExplicitlyUnassignedFlags(t *testing.T) {
	atlas := &pixelforge_project.TileAtlas{
		TileW: 8, TileH: 8,
		Grid: [][]int{
			{1, 0},
			{0, 0},
		},
		NESPaletteBlock: [][]int{
			{UnassignedNESPaletteBlock},
		},
	}
	violations := FindPaletteBlockViolations(atlas)
	require.Len(t, violations, 1,
		"explicit sentinel still surfaces as unassigned")
}

// TestPaintPaletteBlockOverlay_DisabledNoOp: with the overlay flag
// off, the painter short-circuits without touching dst.
func TestPaintPaletteBlockOverlay_DisabledNoOp(t *testing.T) {
	scene := &pixelforge_project.Scene{
		TileAtlases: []pixelforge_project.TileAtlas{
			{Grid: [][]int{{1}}},
		},
	}
	overlays := pixelforge_project.EditorOverlays{
		ScanlineEnabled:     true,
		PaletteBlockEnabled: false,
	}
	assert.NotPanics(t, func() {
		PaintPaletteBlockOverlay(nil, scene, overlays)
	})
}

// TestPaintPaletteBlockOverlay_NoTileAtlases: a scene without any
// TileAtlas yields a no-op (even with the overlay enabled).
func TestPaintPaletteBlockOverlay_NoTileAtlases(t *testing.T) {
	scene := &pixelforge_project.Scene{}
	overlays := pixelforge_project.EditorOverlays{PaletteBlockEnabled: true}
	assert.NotPanics(t, func() {
		PaintPaletteBlockOverlay(nil, scene, overlays)
	})
}
