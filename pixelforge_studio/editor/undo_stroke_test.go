package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// undoStackTestFixture mints a layer + painter + stack for the
// undo/redo tests so each one starts from a known clean state.
func undoStackTestFixture(t *testing.T) (*TilePainter, *pixelforge_project.TileAtlas, *UndoStack) {
	t.Helper()
	layer := &pixelforge_project.TileAtlas{
		Name:  "undo-test",
		TileW: 8,
		TileH: 8,
	}
	ensurePainterCapacity(layer, 31, 31)
	return NewTilePainter(), layer, NewUndoStack()
}

// TestUndo_BrushStrokeReversed — a 5-cell brush stroke's Undo
// restores every cell to its pre-stroke value.
func TestUndo_BrushStrokeReversed(t *testing.T) {
	p, layer, stack := undoStackTestFixture(t)
	p.SetActiveTile(4)

	p.BrushStartStroke(layer)
	for i := 0; i < 5; i++ {
		p.BrushPaint(i, 0, 4)
	}
	p.BrushEndStroke(stack)

	for i := 0; i < 5; i++ {
		require.Equal(t, 4, layer.Grid[0][i])
	}

	cmd := stack.Undo()
	require.NotNil(t, cmd)
	for i := 0; i < 5; i++ {
		assert.Equal(t, 0, layer.Grid[0][i], "cell %d should revert to its pre-stroke 0", i)
	}
	assert.Equal(t, 0, stack.Depth())
	assert.Equal(t, 1, stack.RedoDepth(), "undone command moves to redo side")
}

// TestUndo_BucketFireReversed — a flood-fill that paints many cells
// reverts every cell on undo (single command, atomic revert).
func TestUndo_BucketFireReversed(t *testing.T) {
	p := NewTilePainter()
	stack := NewUndoStack()
	layer := &pixelforge_project.TileAtlas{
		Name:  "undo-bucket",
		TileW: 8,
		TileH: 8,
		Grid: [][]int{
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0},
		},
	}

	cmd := p.BucketFill(layer, 0, 0, 9, stack)
	require.NotNil(t, cmd)
	assert.Equal(t, 20, len(cmd.Diffs), "all 20 cells fill")
	for r := 0; r < 4; r++ {
		for c := 0; c < 5; c++ {
			require.Equal(t, 9, layer.Grid[r][c])
		}
	}

	stack.Undo()
	for r := 0; r < 4; r++ {
		for c := 0; c < 5; c++ {
			assert.Equal(t, 0, layer.Grid[r][c])
		}
	}
}

// TestUndo_RedoAfterUndo — Redo replays the most-recently-undone
// stroke, restoring post-stroke state. Standard undo/redo contract.
func TestUndo_RedoAfterUndo(t *testing.T) {
	p, layer, stack := undoStackTestFixture(t)
	p.SetActiveTile(6)

	p.BrushStartStroke(layer)
	p.BrushPaint(1, 1, 6)
	p.BrushPaint(2, 2, 6)
	p.BrushEndStroke(stack)

	stack.Undo()
	assert.Equal(t, 0, layer.Grid[1][1])
	assert.Equal(t, 0, layer.Grid[2][2])

	stack.Redo()
	assert.Equal(t, 6, layer.Grid[1][1])
	assert.Equal(t, 6, layer.Grid[2][2])
	assert.Equal(t, 1, stack.Depth())
	assert.Equal(t, 0, stack.RedoDepth())
}

// TestUndo_StackCapped — pushing 105 strokes leaves the stack at
// exactly 100 entries; the 5 oldest are evicted. Plan: cap is
// enforced so a long authoring session can't bloat memory.
func TestUndo_StackCapped(t *testing.T) {
	p := NewTilePainter()
	stack := NewUndoStack()
	layer := &pixelforge_project.TileAtlas{
		Name:  "cap-test",
		TileW: 8,
		TileH: 8,
	}
	ensurePainterCapacity(layer, 31, 31)
	p.SetActiveTile(1)

	const total = 105
	for i := 0; i < total; i++ {
		p.BrushStartStroke(layer)
		// Each stroke paints a fresh cell so OldID != NewID and the
		// diff list isn't empty (a no-op stroke would not push).
		p.BrushPaint(i%32, (i/32)%32, i%5+1)
		p.BrushEndStroke(stack)
	}

	assert.Equal(t, undoStackCapacity, stack.Depth(),
		"stack should cap at %d, got %d", undoStackCapacity, stack.Depth())
}

// TestUndo_NewPushClearsRedoBranch — after Undo, performing a new
// edit erases the redo side. Standard branching-history contract.
func TestUndo_NewPushClearsRedoBranch(t *testing.T) {
	p, layer, stack := undoStackTestFixture(t)
	p.SetActiveTile(2)

	p.BrushStartStroke(layer)
	p.BrushPaint(0, 0, 2)
	p.BrushEndStroke(stack)

	stack.Undo()
	require.Equal(t, 1, stack.RedoDepth())

	p.BrushStartStroke(layer)
	p.BrushPaint(1, 1, 3)
	p.BrushEndStroke(stack)

	assert.Equal(t, 0, stack.RedoDepth(), "new push must clear the redo branch")
}

// TestUndo_EmptyStackNoCrash — Undo/Redo on an empty stack are
// no-ops, not panics.
func TestUndo_EmptyStackNoCrash(t *testing.T) {
	stack := NewUndoStack()
	assert.Nil(t, stack.Undo())
	assert.Nil(t, stack.Redo())
}

// TestUndo_PushDropsEmptyCommand — a StrokeCommand with no diffs
// (e.g. a brush that hovered over a cell already showing the active
// tile) does not advance the stack. The painter relies on this.
func TestUndo_PushDropsEmptyCommand(t *testing.T) {
	stack := NewUndoStack()
	stack.Push(&StrokeCommand{Layer: &pixelforge_project.TileAtlas{}})
	assert.Equal(t, 0, stack.Depth())
}
