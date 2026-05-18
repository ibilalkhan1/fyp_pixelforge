package editor

// tile_painter.go owns idea #1 v1's Paint tool implementation. The
// Paint tool is the only existing-but-stub Tool enum value (defined in
// tools.go but never dispatched in canvas.go's UpdateAt switch); this
// file wires it up with three sub-modes designers expect from any
// modern tile editor:
//
//   - Brush: paint one cell per click/drag step, dedup repeated cells
//     within a single stroke so a jittered mouse doesn't bloat the
//     undo diff list.
//   - Bucket: 4-connected flood-fill from the clicked cell, replacing
//     every contiguously-matching cell with the active tile. Tiled /
//     LDtk / Aseprite all converge on 4-connected as canonical.
//   - Rectangle: anchor-on-press, ghost-preview-on-drag, fill-on-
//     release. v1 commits on release; the preview is purely a UI
//     concern the caller renders.
//
// Every paint action commits as exactly one StrokeCommand on the
// editor's UndoStack — so Ctrl+Z reverts the whole stroke, not
// individual cells. The painter exposes its handlers as plain methods
// so tests can drive them directly without simulating ImGui events;
// the canvas UpdateAt switch routes mouse events into the same
// methods.

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// PaintSubMode picks which of the three painter behaviors the Paint
// tool dispatches into. Default is PaintBrush — the most common
// authoring operation by a wide margin.
type PaintSubMode int

const (
	PaintBrush PaintSubMode = iota
	PaintBucket
	PaintRectangle
)

// String returns a human-readable label for the toolbar's sub-mode
// picker and for status messages.
func (m PaintSubMode) String() string {
	switch m {
	case PaintBrush:
		return "brush"
	case PaintBucket:
		return "bucket"
	case PaintRectangle:
		return "rectangle"
	default:
		return "?"
	}
}

// TilePainter holds the editor-side state for the Paint tool. State
// is intentionally narrow:
//
//   - SubMode picks which paint algorithm runs.
//   - ActiveTileID is the value the painter writes into cells the
//     designer clicks. 0 = clear (the plan's documented "no tile
//     selected paints zero" behavior — designers use that on purpose
//     to erase).
//   - currentStroke buffers the cell-diff list a brush stroke
//     accumulates between BrushStartStroke and BrushEndStroke. Bucket
//     and rectangle fills don't use this; they collect diffs into a
//     local slice and commit a single StrokeCommand atomically.
//   - rectAnchor holds the rectangle sub-mode's anchor cell so a drag
//     across multiple frames computes the rect against a stable
//     starting point. rectAnchored gates whether an anchor has been
//     set this gesture.
//   - lastBrushCol/Row remember the most recently brushed cell so a
//     drag that re-enters the same cell skips re-painting (the
//     dedup). The "did we paint a cell already this stroke" set is
//     implemented as a small map so out-of-order drag paths still
//     dedup correctly.
type TilePainter struct {
	SubMode      PaintSubMode
	ActiveTileID int

	currentStroke []CellDiff
	strokeLayer   *pixelforge_project.TilemapLayer
	stroking      bool
	strokeCells   map[strokeKey]struct{}

	rectAnchored bool
	rectAnchor   tileCoord
}

// strokeKey is the per-cell hash the brush stroke uses to dedup
// repeated paints during one drag. (col, row) is enough — strokes are
// scoped to a single layer.
type strokeKey struct {
	col int
	row int
}

// tileCoord names a (Col, Row) pair locally; the painter never leaks
// this type outward — the public API speaks (col, row) as separate
// ints to keep the call sites readable and align with the rest of the
// canvas dispatch.
type tileCoord struct {
	col int
	row int
}

// NewTilePainter returns a painter in PaintBrush mode with no active
// tile selected (ActiveTileID = 0 = clear, per the plan's "paint zero
// is a feature" decision).
func NewTilePainter() *TilePainter {
	return &TilePainter{SubMode: PaintBrush}
}

// SetActiveTile updates the tile ID future paint actions will write.
// Out-of-range IDs are accepted; the caller (the palette panel) is
// responsible for offering only valid choices.
func (p *TilePainter) SetActiveTile(id int) {
	if p == nil {
		return
	}
	p.ActiveTileID = id
}

// SetSubMode switches the active sub-mode. Ends any in-progress
// stroke first — switching mid-stroke would leave the partial diff
// list orphaned on the painter.
func (p *TilePainter) SetSubMode(m PaintSubMode) {
	if p == nil {
		return
	}
	if p.stroking {
		// Drop the in-progress stroke silently; the canvas dispatch
		// never holds a stroke across mode switches, but a stray
		// programmatic mode change shouldn't crash.
		p.stroking = false
		p.currentStroke = nil
		p.strokeCells = nil
		p.strokeLayer = nil
	}
	p.rectAnchored = false
	p.SubMode = m
}

// BrushStartStroke opens a new brush stroke against layer. Subsequent
// BrushPaint calls accumulate diffs into this stroke until
// BrushEndStroke commits or discards them. Calling BrushStartStroke
// while a stroke is already open is a no-op (the canvas dispatch
// guarantees one stroke per mouse-down/up, but the guard makes the
// API safe for tests).
func (p *TilePainter) BrushStartStroke(layer *pixelforge_project.TilemapLayer) {
	if p == nil || layer == nil {
		return
	}
	if p.stroking {
		return
	}
	p.stroking = true
	p.strokeLayer = layer
	p.currentStroke = nil
	p.strokeCells = make(map[strokeKey]struct{})
}

// BrushPaint writes tileID at (col, row) on the active stroke's layer.
// Duplicates within the same stroke are dropped (the dedup gate
// the plan calls out). Returns true when the cell was actually changed
// — the canvas dispatch uses the return to decide whether to mark
// dirty.
//
// The painter requires an open stroke; callers that paint without a
// surrounding Start/End pair (tests or future programmatic paths)
// should use BrushPaintAtomic instead.
func (p *TilePainter) BrushPaint(col, row, tileID int) bool {
	if p == nil || !p.stroking || p.strokeLayer == nil {
		return false
	}
	return p.paintCell(p.strokeLayer, col, row, tileID, true)
}

// BrushEndStroke commits the active stroke onto stack and returns the
// committed command (nil when the stroke was empty — that path
// happens when the designer clicks but never actually changed a cell,
// e.g. the brush hovered over a cell already showing the active
// tile). The stack handles empty pushes as no-ops; the return value
// is exposed for tests.
func (p *TilePainter) BrushEndStroke(stack *UndoStack) *StrokeCommand {
	if p == nil || !p.stroking {
		return nil
	}
	layer := p.strokeLayer
	diffs := p.currentStroke
	p.stroking = false
	p.currentStroke = nil
	p.strokeCells = nil
	p.strokeLayer = nil
	if layer == nil || len(diffs) == 0 {
		return nil
	}
	cmd := &StrokeCommand{Layer: layer, Diffs: diffs}
	if stack != nil {
		stack.Push(cmd)
	}
	return cmd
}

// BucketFill performs a 4-connected flood-fill from (col, row) on
// layer, replacing every contiguously-matching cell with tileID.
// Commits as exactly one StrokeCommand on stack and returns it (or
// nil when the fill was a no-op — clicking on a cell already showing
// tileID). 4-connected (not 8-connected) is the canonical Tiled / LDtk
// / Aseprite behavior.
func (p *TilePainter) BucketFill(layer *pixelforge_project.TilemapLayer, col, row, tileID int, stack *UndoStack) *StrokeCommand {
	if p == nil || layer == nil {
		return nil
	}
	ensurePainterCapacity(layer, col, row)
	if row < 0 || col < 0 || row >= len(layer.Grid) || col >= len(layer.Grid[row]) {
		return nil
	}
	target := layer.Grid[row][col]
	if target == tileID {
		// Clicking on a cell already showing the target is a no-op —
		// no diffs, no command, no dirty flip.
		return nil
	}

	diffs := make([]CellDiff, 0, 16)
	// Iterative BFS keeps the call stack bounded for very large fills
	// (a full 16-screen ground row is 512 cells — recursion would
	// blow the stack on Windows's default 1 MB).
	queue := []tileCoord{{col: col, row: row}}
	visited := map[strokeKey]struct{}{{col: col, row: row}: {}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.row < 0 || cur.row >= len(layer.Grid) {
			continue
		}
		if cur.col < 0 || cur.col >= len(layer.Grid[cur.row]) {
			continue
		}
		if layer.Grid[cur.row][cur.col] != target {
			continue
		}
		// Record the diff before mutating so OldID is captured.
		diffs = append(diffs, CellDiff{
			Col: cur.col, Row: cur.row,
			OldID: target, NewID: tileID,
		})
		layer.Grid[cur.row][cur.col] = tileID

		neighbors := [4]tileCoord{
			{col: cur.col - 1, row: cur.row},
			{col: cur.col + 1, row: cur.row},
			{col: cur.col, row: cur.row - 1},
			{col: cur.col, row: cur.row + 1},
		}
		for _, n := range neighbors {
			key := strokeKey{col: n.col, row: n.row}
			if _, seen := visited[key]; seen {
				continue
			}
			visited[key] = struct{}{}
			queue = append(queue, n)
		}
	}

	if len(diffs) == 0 {
		return nil
	}
	cmd := &StrokeCommand{Layer: layer, Diffs: diffs}
	if stack != nil {
		stack.Push(cmd)
	}
	return cmd
}

// RectangleAnchor marks the start of a rectangle gesture at (col,
// row). Subsequent RectangleFill calls compute the rect against this
// anchor. The canvas dispatch invokes RectangleAnchor on LMB-down and
// RectangleFill on LMB-up; the in-between drag is purely a ghost-
// preview concern the workspace renders without mutating state.
func (p *TilePainter) RectangleAnchor(col, row int) {
	if p == nil {
		return
	}
	p.rectAnchored = true
	p.rectAnchor = tileCoord{col: col, row: row}
}

// RectangleAnchored reports whether RectangleAnchor has set an anchor
// for the in-progress gesture. The workspace's ghost-preview renderer
// reads this to decide whether to draw the rect overlay.
func (p *TilePainter) RectangleAnchored() bool {
	if p == nil {
		return false
	}
	return p.rectAnchored
}

// RectangleAnchorCell returns the active anchor (col, row). Only
// meaningful when RectangleAnchored() == true.
func (p *TilePainter) RectangleAnchorCell() (int, int) {
	if p == nil {
		return 0, 0
	}
	return p.rectAnchor.col, p.rectAnchor.row
}

// RectangleFill commits the rect spanning the existing anchor and
// (endCol, endRow) (inclusive on both corners) into one StrokeCommand
// and returns it. Min/max math handles drags in any direction —
// designers expect to anchor-bottom-right, drag-top-left and have the
// rect fill the same cells.
//
// Pre-condition: RectangleAnchor must have been called. The canvas
// dispatch guarantees this; programmatic callers (tests) must respect
// the contract.
func (p *TilePainter) RectangleFill(layer *pixelforge_project.TilemapLayer, endCol, endRow, tileID int, stack *UndoStack) *StrokeCommand {
	if p == nil || layer == nil || !p.rectAnchored {
		return nil
	}
	startCol, startRow := p.rectAnchor.col, p.rectAnchor.row
	p.rectAnchored = false

	minCol, maxCol := startCol, endCol
	if minCol > maxCol {
		minCol, maxCol = maxCol, minCol
	}
	minRow, maxRow := startRow, endRow
	if minRow > maxRow {
		minRow, maxRow = maxRow, minRow
	}
	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	ensurePainterCapacity(layer, maxCol, maxRow)

	diffs := make([]CellDiff, 0, (maxCol-minCol+1)*(maxRow-minRow+1))
	for r := minRow; r <= maxRow; r++ {
		if r >= len(layer.Grid) {
			break
		}
		for c := minCol; c <= maxCol; c++ {
			if c >= len(layer.Grid[r]) {
				break
			}
			old := layer.Grid[r][c]
			if old == tileID {
				continue
			}
			diffs = append(diffs, CellDiff{
				Col: c, Row: r,
				OldID: old, NewID: tileID,
			})
			layer.Grid[r][c] = tileID
		}
	}

	if len(diffs) == 0 {
		return nil
	}
	cmd := &StrokeCommand{Layer: layer, Diffs: diffs}
	if stack != nil {
		stack.Push(cmd)
	}
	return cmd
}

// CancelRectangle clears the in-progress rectangle anchor without
// committing. The canvas dispatch uses this when a sub-mode switch
// (or scene change) happens mid-gesture.
func (p *TilePainter) CancelRectangle() {
	if p == nil {
		return
	}
	p.rectAnchored = false
}

// paintCell is the shared single-cell write path. It grows the layer
// to fit (col, row), records the OldID/NewID diff onto the active
// stroke (if dedup is true and the cell hasn't been touched in this
// stroke yet), and writes the new value. Returns true when the cell
// changed — both "would change" and "was actually changed" collapse
// here because the dedup gate already drops repeat visits.
func (p *TilePainter) paintCell(layer *pixelforge_project.TilemapLayer, col, row, tileID int, dedup bool) bool {
	if layer == nil || col < 0 || row < 0 {
		return false
	}
	ensurePainterCapacity(layer, col, row)
	if row >= len(layer.Grid) || col >= len(layer.Grid[row]) {
		return false
	}
	if dedup {
		key := strokeKey{col: col, row: row}
		if _, seen := p.strokeCells[key]; seen {
			return false
		}
		p.strokeCells[key] = struct{}{}
	}
	old := layer.Grid[row][col]
	if old == tileID {
		// No mutation needed but the cell still counts as "seen this
		// stroke" so the dedup gate above prevents a future revisit
		// from minting a phantom diff.
		return false
	}
	p.currentStroke = append(p.currentStroke, CellDiff{
		Col: col, Row: row,
		OldID: old, NewID: tileID,
	})
	layer.Grid[row][col] = tileID
	return true
}

// ensurePainterCapacity grows layer.Grid to fit (col, row). Mirrors
// the palette package's ensureLayerCapacity helper but is package-
// local here so editor doesn't import palette. The default starting
// size of 32×32 matches one 256×240 screen at 8×8 tiles — the
// smallest meaningful grid in the Mario-strip primitive.
func ensurePainterCapacity(layer *pixelforge_project.TilemapLayer, col, row int) {
	if layer == nil || col < 0 || row < 0 {
		return
	}
	if len(layer.Grid) == 0 {
		const defaultSize = 32
		layer.Grid = make([][]int, defaultSize)
		for i := range layer.Grid {
			layer.Grid[i] = make([]int, defaultSize)
		}
	}
	for row >= len(layer.Grid) {
		// Match the existing row width so the grid stays rectangular.
		width := len(layer.Grid[0])
		if width <= 0 {
			width = 32
		}
		layer.Grid = append(layer.Grid, make([]int, width))
	}
	for y := range layer.Grid {
		for col >= len(layer.Grid[y]) {
			layer.Grid[y] = append(layer.Grid[y], 0)
		}
	}
}
