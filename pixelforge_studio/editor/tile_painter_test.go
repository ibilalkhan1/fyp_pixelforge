package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// newPainterFixture returns a fresh painter + layer + undo stack
// trio ready for direct API drive. Layer starts at the default
// 32×32 size (one screen) so most tests don't need to fiddle with
// grow semantics.
func newPainterFixture(t *testing.T) (*TilePainter, *pixelforge_project.TileAtlas, *UndoStack) {
	t.Helper()
	layer := &pixelforge_project.TileAtlas{
		Name:  "test",
		TileW: 8,
		TileH: 8,
	}
	ensurePainterCapacity(layer, 31, 31)
	return NewTilePainter(), layer, NewUndoStack()
}

// TestBrush_SingleClickPaintsOneCell — the canonical "click paints
// a cell" path. Plan's first painter test scenario.
func TestBrush_SingleClickPaintsOneCell(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	p.SetActiveTile(3)

	p.BrushStartStroke(layer)
	changed := p.BrushPaint(5, 10, 3)
	cmd := p.BrushEndStroke(stack)

	assert.True(t, changed, "single click should mutate the cell")
	assert.Equal(t, 3, layer.Grid[10][5])
	require.NotNil(t, cmd)
	assert.Len(t, cmd.Diffs, 1)
	assert.Equal(t, 1, stack.Depth())
}

// TestBrush_DragPaintsMultipleCells — held-LMB drag paints every
// cell the mouse passes through; all cells land in one stroke.
func TestBrush_DragPaintsMultipleCells(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	p.SetActiveTile(3)

	p.BrushStartStroke(layer)
	for _, cell := range []struct{ col, row int }{{5, 10}, {5, 11}, {5, 12}} {
		p.BrushPaint(cell.col, cell.row, 3)
	}
	cmd := p.BrushEndStroke(stack)

	require.NotNil(t, cmd)
	assert.Len(t, cmd.Diffs, 3)
	assert.Equal(t, 3, layer.Grid[10][5])
	assert.Equal(t, 3, layer.Grid[11][5])
	assert.Equal(t, 3, layer.Grid[12][5])
	assert.Equal(t, 1, stack.Depth(), "one stroke regardless of cell count")
}

// TestBrush_DedupesRepeatedCellsInStroke — a jittered drag visiting
// the same cell three times produces only one diff per unique cell.
// Plan: critical for keeping undo payloads small.
func TestBrush_DedupesRepeatedCellsInStroke(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	p.SetActiveTile(7)

	p.BrushStartStroke(layer)
	// Path: (5,10) -> (6,10) -> back to (5,10) (jitter).
	p.BrushPaint(5, 10, 7)
	p.BrushPaint(6, 10, 7)
	p.BrushPaint(5, 10, 7)
	cmd := p.BrushEndStroke(stack)

	require.NotNil(t, cmd)
	assert.Len(t, cmd.Diffs, 2, "the revisited cell should not produce a second diff")
}

// TestBucket_FourConnectedFill — bucket from (0,0) flood-fills the
// connected zero region but leaves the unconnected wall in place.
func TestBucket_FourConnectedFill(t *testing.T) {
	p := NewTilePainter()
	stack := NewUndoStack()
	layer := &pixelforge_project.TileAtlas{
		Name:  "test",
		TileW: 8,
		TileH: 8,
		Grid: [][]int{
			{0, 0, 1},
			{0, 0, 1},
			{1, 1, 1},
		},
	}

	cmd := p.BucketFill(layer, 0, 0, 5, stack)

	require.NotNil(t, cmd)
	// Cells (0,0), (0,1), (1,0), (1,1) — the connected zeros — fill.
	assert.Equal(t, 5, layer.Grid[0][0])
	assert.Equal(t, 5, layer.Grid[0][1])
	assert.Equal(t, 5, layer.Grid[1][0])
	assert.Equal(t, 5, layer.Grid[1][1])
	// Unchanged walls.
	assert.Equal(t, 1, layer.Grid[0][2])
	assert.Equal(t, 1, layer.Grid[1][2])
	assert.Equal(t, 1, layer.Grid[2][0])
	assert.Equal(t, 1, layer.Grid[2][1])
	assert.Equal(t, 1, layer.Grid[2][2])
	assert.Len(t, cmd.Diffs, 4, "only the four connected zero cells should diff")
}

// TestBucket_DoesNotFloodAcrossDifferentValues — a value-1 wall
// blocks 4-connected fill from reaching cells on its far side.
func TestBucket_DoesNotFloodAcrossDifferentValues(t *testing.T) {
	p := NewTilePainter()
	stack := NewUndoStack()
	layer := &pixelforge_project.TileAtlas{
		Name:  "test",
		TileW: 8,
		TileH: 8,
		Grid: [][]int{
			{0, 1, 0},
		},
	}

	cmd := p.BucketFill(layer, 0, 0, 5, stack)

	require.NotNil(t, cmd)
	assert.Equal(t, 5, layer.Grid[0][0])
	assert.Equal(t, 1, layer.Grid[0][1], "wall stays")
	assert.Equal(t, 0, layer.Grid[0][2], "cell beyond wall is unreachable")
	assert.Len(t, cmd.Diffs, 1)
}

// TestRectangle_DragPreviewThenCommit — anchor on press, fill on
// release. Plan calls for inclusive bounds: anchor (5,10) + end
// (8,12) = 4 cols × 3 rows = 12 cells.
func TestRectangle_DragPreviewThenCommit(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	p.SetActiveTile(9)

	p.RectangleAnchor(5, 10)
	assert.True(t, p.RectangleAnchored())
	cmd := p.RectangleFill(layer, 8, 12, 9, stack)
	assert.False(t, p.RectangleAnchored(), "anchor clears after fill")

	require.NotNil(t, cmd)
	assert.Len(t, cmd.Diffs, 12)
	for row := 10; row <= 12; row++ {
		for col := 5; col <= 8; col++ {
			assert.Equal(t, 9, layer.Grid[row][col])
		}
	}
}

// TestPainter_NoActiveTileSelected_PaintsZero — the plan's
// "paint zero is a feature, not a no-op" decision: a brush stroke
// with ActiveTileID = 0 clears the cells it touches.
func TestPainter_NoActiveTileSelected_PaintsZero(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	// Pre-fill the target cell with a non-zero so the "clear" path
	// is observable.
	layer.Grid[5][3] = 7

	p.SetActiveTile(0)
	p.BrushStartStroke(layer)
	changed := p.BrushPaint(3, 5, 0)
	cmd := p.BrushEndStroke(stack)

	assert.True(t, changed)
	assert.Equal(t, 0, layer.Grid[5][3], "tile 0 = clear, not no-op")
	require.NotNil(t, cmd)
	assert.Equal(t, 7, cmd.Diffs[0].OldID, "diff preserves the pre-clear value")
	assert.Equal(t, 0, cmd.Diffs[0].NewID)
}

// TestRightClick_SetSpawnHere — the right-click "Set spawn here"
// flow mutates Scene.SpawnTile and marks the editor dirty.
func TestRightClick_SetSpawnHere(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	e.ClearDirty()

	handled := HandleRightClick(e, RightClickContext{
		Scene: scene,
		Cell:  &RightClickCell{Col: 8, Row: 24},
	})

	assert.True(t, handled)
	assert.Equal(t, 8, scene.SpawnTile.Col)
	assert.Equal(t, 24, scene.SpawnTile.Row)
	assert.True(t, e.IsDirty())
}

// TestRightClick_EntityClickReturnsUnhandledV1 — v1 leaves the
// entity branch wired but with no actions; U7 will extend it. Pin
// the current contract so the U7 work surfaces as a deliberate
// behavior change.
func TestRightClick_EntityClickReturnsUnhandledV1(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "ent-1",
	})
	assert.False(t, handled, "v1 has no entity-click actions; U7 will add them")
}

// TestTilePalette_RendersTilesFromBoundSheet — the palette's
// selected-tile state contract is testable without a live ImGui
// context: clicking a tile button is equivalent to calling
// SetActiveTile, and the painter is the source of truth for the
// active selection. The fallback path (no sheet bound) still
// presents tilePaletteFallbackCount choices.
func TestTilePalette_RendersTilesFromBoundSheet(t *testing.T) {
	painter := NewTilePainter()
	palette := NewTilePalette(painter)

	layer := &pixelforge_project.TileAtlas{
		Name:           "ground",
		TileW:          8,
		TileH:          8,
		SpriteSheetRef: "tiles_overworld",
	}

	// Header reflects the bound sheet — designers can see at a
	// glance what the integer cell IDs are addressing.
	assert.Contains(t, paletteHeader(layer), "tiles_overworld")
	// Fallback count is what the v1 panel renders.
	assert.Equal(t, tilePaletteFallbackCount, tilePaletteCount(layer))

	// Programmatic selection (the equivalent of a button click)
	// updates the painter's ActiveTileID.
	palette.Painter.SetActiveTile(7)
	assert.Equal(t, 7, painter.ActiveTileID)

	// An unbound layer renders the same fallback count but with a
	// distinct header — useful for the designer to notice the
	// missing binding.
	noSheet := &pixelforge_project.TileAtlas{Name: "scratch"}
	assert.Equal(t, tilePaletteFallbackCount, tilePaletteCount(noSheet))
	assert.Contains(t, paletteHeader(noSheet), "no sheet")
}

// TestPainter_SubModeSwitchDropsInProgressStroke — switching modes
// mid-stroke must not leak partial state into the next gesture.
func TestPainter_SubModeSwitchDropsInProgressStroke(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	p.SetActiveTile(2)

	p.BrushStartStroke(layer)
	p.BrushPaint(1, 1, 2)
	p.SetSubMode(PaintBucket) // mid-stroke switch

	// Subsequent BrushEndStroke with no open stroke is a no-op.
	cmd := p.BrushEndStroke(stack)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, stack.Depth())
}

// TestPainter_BucketNoOpOnSameValue — bucket-clicking a cell that
// already shows the active tile commits no command and does not
// mark dirty.
func TestPainter_BucketNoOpOnSameValue(t *testing.T) {
	p, layer, stack := newPainterFixture(t)
	layer.Grid[2][2] = 4

	cmd := p.BucketFill(layer, 2, 2, 4, stack)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, stack.Depth())
}
