package palette

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// PaintMode picks between palette-color pixel painting and existing-
// sprite tile painting. v1 ships pixel mode; tile mode reads from the
// asset browser's selected sprite.
type PaintMode int

const (
	PaintPixel PaintMode = iota
	PaintTile
)

// Painter implements the M2 paint tool. It records strokes as the user
// drags across the canvas and hands them to the autotile synth on
// stroke end.
type Painter struct {
	Mode  PaintMode
	Synth *AutoTileRuleSynth

	// stroke buffers paints made during the active mouse-button hold.
	stroke    []PaintCell
	stroking  bool
}

// NewPainter returns a fresh painter in pixel mode.
func NewPainter() *Painter {
	return &Painter{Mode: PaintPixel, Synth: NewAutoTileRuleSynth()}
}

// BeginStroke marks the start of a paint operation. Subsequent Paint
// calls accumulate into this stroke until EndStroke.
func (p *Painter) BeginStroke() {
	p.stroke = p.stroke[:0]
	p.stroking = true
}

// Stroking reports whether a stroke is in progress.
func (p *Painter) Stroking() bool { return p.stroking }

// Paint writes value at (tx, ty) in layer and records the cell in the
// active stroke. Out-of-range coordinates are clamped to the layer.
// Returns true when the layer was actually mutated (no-op when the cell
// already holds the same value).
func (p *Painter) Paint(layer *pixelforge_project.TilemapLayer, tx, ty, value int) bool {
	if layer == nil {
		return false
	}
	ensureLayerCapacity(layer, tx, ty)
	if ty < 0 || tx < 0 {
		return false
	}
	if ty >= len(layer.Grid) {
		return false
	}
	row := layer.Grid[ty]
	if tx >= len(row) {
		return false
	}
	if row[tx] == value {
		return false
	}
	// If a synthesized auto-tile rule applies, use its output instead.
	if p.Synth != nil {
		if synthesised, ok := p.Synth.Apply(layer, tx, ty); ok && synthesised != value {
			value = synthesised
		}
	}
	layer.Grid[ty][tx] = value
	if p.stroking {
		p.stroke = append(p.stroke, PaintCell{X: tx, Y: ty, Value: value})
	}
	return true
}

// EndStroke finishes the active stroke and feeds it to the autotile
// synth so future strokes can pick up patterns observed here.
func (p *Painter) EndStroke(layer *pixelforge_project.TilemapLayer) {
	if !p.stroking {
		return
	}
	p.stroking = false
	if p.Synth != nil {
		p.Synth.RecordStroke(layer, p.stroke)
	}
}

// Stroke returns a copy of the cells captured during the active
// stroke. Test helper.
func (p *Painter) Stroke() []PaintCell {
	out := make([]PaintCell, len(p.stroke))
	copy(out, p.stroke)
	return out
}

// EnsureLayer returns the first tilemap layer in scene, creating one
// with sensible defaults when none exist.
func EnsureLayer(scene *pixelforge_project.Scene, tileW, tileH int) *pixelforge_project.TilemapLayer {
	if scene == nil {
		return nil
	}
	if len(scene.Tilemaps) == 0 {
		scene.Tilemaps = append(scene.Tilemaps, pixelforge_project.TilemapLayer{
			Name:          "tilemap",
			TileW:         tileW,
			TileH:         tileH,
			Grid:          [][]int{},
			AutoTileRules: []pixelforge_project.AutoTileRule{},
		})
	}
	return &scene.Tilemaps[0]
}

// ensureLayerCapacity grows layer.Grid so that (tx, ty) is in range.
// Cells are zero-initialised; if the grid is empty, a 32×32 default is
// used as a starting size before extending further.
func ensureLayerCapacity(layer *pixelforge_project.TilemapLayer, tx, ty int) {
	if tx < 0 || ty < 0 {
		return
	}
	if len(layer.Grid) == 0 {
		layer.Grid = make([][]int, 32)
		for i := range layer.Grid {
			layer.Grid[i] = make([]int, 32)
		}
	}
	for ty >= len(layer.Grid) {
		layer.Grid = append(layer.Grid, make([]int, max(32, len(layer.Grid[0]))))
	}
	for y := range layer.Grid {
		for tx >= len(layer.Grid[y]) {
			layer.Grid[y] = append(layer.Grid[y], 0)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
