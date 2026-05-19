package editor

// undo_stroke.go owns idea #1 v1's per-stroke undo discipline for the
// tile painter. Each user-visible paint action (one brush stroke, one
// bucket fire, one rectangle fill) lands on the stack as a single
// StrokeCommand carrying the minimal cell-diff payload the layer would
// need to revert.
//
// The stack is bounded (default 100 entries per the plan) so a long
// authoring session can't bloat memory; the oldest entry is evicted
// when a new push overflows. Redo follows the standard command-pattern:
// a successful Undo moves the command to the redo side, a fresh Push
// clears the redo side (the user "branched" off the undo history).
//
// Why command + cell-diff rather than full-grid snapshots: the plan's
// Key Technical Decisions section calls full-grid snapshots wasteful at
// 16-screen scale (16*32*30 = 15,360 cells). Cell-diff lists carry only
// the cells the stroke actually changed — a 5-cell brush stroke is 5
// diffs regardless of grid size.

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// undoStackCapacity caps the per-editor undo stack. The plan calls for
// 100; exported as a constant so tests can reference the same source
// of truth that the runtime enforces.
const undoStackCapacity = 100

// CellDiff is the unit of undo payload — the change to one tile cell.
// Storing both OldID and NewID lets Apply (redo) and Revert (undo)
// both work without re-reading the grid.
type CellDiff struct {
	Col   int
	Row   int
	OldID int
	NewID int
}

// StrokeCommand bundles one user-visible paint action: every cell the
// brush/bucket/rectangle changed, paired with the layer those changes
// belong to. The layer pointer is held directly (not by index) because
// scenes can reorder Tilemaps and the undo stack must not silently
// re-target a different layer after a re-order.
type StrokeCommand struct {
	Layer *pixelforge_project.TileAtlas
	Diffs []CellDiff
}

// Apply re-applies the stroke's NewIDs to the layer. Used by Redo and
// by the painter's own commit path (so a single code path mutates the
// grid in both directions).
func (c *StrokeCommand) Apply() {
	if c == nil || c.Layer == nil {
		return
	}
	for _, d := range c.Diffs {
		writeCell(c.Layer, d.Col, d.Row, d.NewID)
	}
}

// Revert restores the stroke's OldIDs. Used by Undo.
func (c *StrokeCommand) Revert() {
	if c == nil || c.Layer == nil {
		return
	}
	for _, d := range c.Diffs {
		writeCell(c.Layer, d.Col, d.Row, d.OldID)
	}
}

// UndoStack holds the bounded history of StrokeCommands plus the
// matching redo side. A fresh Push clears the redo side — that's the
// standard "undo, then make a new edit → forget the redo branch"
// semantics every editor user already knows.
type UndoStack struct {
	commands []*StrokeCommand
	redo     []*StrokeCommand
	capacity int
}

// NewUndoStack returns a stack capped at undoStackCapacity. Tests that
// want a smaller cap can construct UndoStack directly.
func NewUndoStack() *UndoStack {
	return &UndoStack{capacity: undoStackCapacity}
}

// Push adds cmd to the undo side and clears the redo side. A nil or
// no-op (empty Diffs) command is dropped — keeps the stack from
// filling with degenerate entries when the painter detects a no-op
// stroke (e.g. brush hovering over a cell already showing the target
// tile).
func (s *UndoStack) Push(cmd *StrokeCommand) {
	if s == nil || cmd == nil || len(cmd.Diffs) == 0 {
		return
	}
	cap := s.capacity
	if cap <= 0 {
		cap = undoStackCapacity
	}
	s.commands = append(s.commands, cmd)
	if len(s.commands) > cap {
		// Evict oldest. Slice trim (not copy) is safe because the
		// evicted pointer becomes unreachable and GCs naturally.
		s.commands = s.commands[len(s.commands)-cap:]
	}
	// Any new edit invalidates the redo branch.
	s.redo = s.redo[:0]
}

// Undo reverts the most recent command (moving it to the redo side)
// and returns the command that was reverted, or nil when the stack
// is empty.
func (s *UndoStack) Undo() *StrokeCommand {
	if s == nil || len(s.commands) == 0 {
		return nil
	}
	last := s.commands[len(s.commands)-1]
	s.commands = s.commands[:len(s.commands)-1]
	last.Revert()
	s.redo = append(s.redo, last)
	return last
}

// Redo re-applies the most recently undone command (moving it back to
// the undo side) and returns the command that was re-applied, or nil
// when there is nothing to redo.
func (s *UndoStack) Redo() *StrokeCommand {
	if s == nil || len(s.redo) == 0 {
		return nil
	}
	last := s.redo[len(s.redo)-1]
	s.redo = s.redo[:len(s.redo)-1]
	last.Apply()
	s.commands = append(s.commands, last)
	return last
}

// Depth returns the number of commands currently on the undo side.
// Used by tests to assert the capacity cap.
func (s *UndoStack) Depth() int {
	if s == nil {
		return 0
	}
	return len(s.commands)
}

// RedoDepth returns the number of commands currently on the redo side.
func (s *UndoStack) RedoDepth() int {
	if s == nil {
		return 0
	}
	return len(s.redo)
}

// writeCell sets layer.Grid[row][col] = id, growing the grid if
// necessary. The painter's BrushPaint / BucketFill / RectangleFill
// already grow the grid before recording diffs, so by the time
// Apply/Revert run the cells are in-range; this helper is defensive
// against future reorderings.
func writeCell(layer *pixelforge_project.TileAtlas, col, row, id int) {
	if layer == nil || col < 0 || row < 0 {
		return
	}
	ensurePainterCapacity(layer, col, row)
	if row >= len(layer.Grid) {
		return
	}
	if col >= len(layer.Grid[row]) {
		return
	}
	layer.Grid[row][col] = id
}
