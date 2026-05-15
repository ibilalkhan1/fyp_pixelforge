package palette

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// newByteReader is a shim that hides the bytes.NewReader return type.
func newByteReader(b []byte) io.Reader { return bytes.NewReader(b) }

// Pixel mode paints palette index onto the tilemap cell.
func TestPainter_PixelModePaintsCell(t *testing.T) {
	scene := &pixelforge_project.Scene{}
	layer := EnsureLayer(scene, 8, 8)
	p := NewPainter()
	p.BeginStroke()
	require.True(t, p.Paint(layer, 5, 5, 8))
	p.EndStroke(layer)
	assert.Equal(t, 8, layer.Grid[5][5])
}

// Paint into a fresh layer auto-grows the grid.
func TestPainter_AutoGrowsGrid(t *testing.T) {
	scene := &pixelforge_project.Scene{}
	layer := EnsureLayer(scene, 8, 8)
	p := NewPainter()
	p.Paint(layer, 50, 50, 4)
	assert.GreaterOrEqual(t, len(layer.Grid), 51)
	assert.Equal(t, 4, layer.Grid[50][50])
}

// Painting a single-tile stroke does not synthesise a rule.
func TestPainter_SingleTileStrokeNoRule(t *testing.T) {
	scene := &pixelforge_project.Scene{}
	layer := EnsureLayer(scene, 8, 8)
	p := NewPainter()
	p.BeginStroke()
	p.Paint(layer, 5, 5, 8)
	p.EndStroke(layer)
	assert.Len(t, layer.AutoTileRules, 0)
}

// Repeating the same (pattern → output) twice promotes a rule to the
// activation threshold. We drive the synth directly so the test
// doesn't drift with painter heuristics.
func TestAutoTile_PatternPaintedTwicePromotesRule(t *testing.T) {
	layer := &pixelforge_project.TilemapLayer{
		Grid: [][]int{
			{1, 1, 1},
			{1, 0, 1},
			{1, 1, 1},
		},
	}
	synth := NewAutoTileRuleSynth()

	// Two strokes that each paint (1,1) with value 5 against the same
	// neighborhood. Each stroke also paints a neighboring sentinel
	// cell so it's not a single-tile stroke (which the plan excludes).
	synth.RecordStroke(layer, []PaintCell{{X: 1, Y: 1, Value: 5}, {X: 0, Y: 0, Value: 5}})
	synth.RecordStroke(layer, []PaintCell{{X: 1, Y: 1, Value: 5}, {X: 0, Y: 0, Value: 5}})

	promoted := false
	for _, r := range layer.AutoTileRules {
		if r.Count >= AutoTileActivationThreshold {
			promoted = true
			break
		}
	}
	assert.True(t, promoted)
}

// Two distinct patterns with the same output remain separate entries.
func TestAutoTile_DistinctPatternsKeepSeparateRules(t *testing.T) {
	layer := &pixelforge_project.TilemapLayer{
		Grid: [][]int{
			{1, 2, 0},
			{0, 0, 0},
			{0, 0, 0},
		},
	}
	synth := NewAutoTileRuleSynth()
	// Two strokes with distinct neighborhoods both painting value 9.
	synth.RecordStroke(layer, []PaintCell{{X: 0, Y: 1, Value: 9}, {X: 1, Y: 1, Value: 9}})
	synth.RecordStroke(layer, []PaintCell{{X: 2, Y: 1, Value: 9}, {X: 2, Y: 2, Value: 9}})

	// Patterns must differ → at least two rule entries.
	assert.GreaterOrEqual(t, len(layer.AutoTileRules), 2)
}

// Out-of-range cellAt returns -1 (wildcard) — used by patternMatches
// when the painter examines cells near the grid edge.
func TestAutoTile_CellAtOutOfRange(t *testing.T) {
	layer := &pixelforge_project.TilemapLayer{Grid: [][]int{{1, 2}}}
	assert.Equal(t, -1, cellAt(layer, -1, 0))
	assert.Equal(t, -1, cellAt(layer, 0, -1))
	assert.Equal(t, -1, cellAt(layer, 99, 0))
	assert.Equal(t, -1, cellAt(layer, 0, 99))
}

// patternMatches treats -1 in the rule as a wildcard.
func TestAutoTile_PatternMatchesWildcards(t *testing.T) {
	rule := [9]int{-1, 1, -1, 1, 2, 1, -1, 1, -1}
	exact := [9]int{0, 1, 0, 1, 2, 1, 0, 1, 0}
	assert.True(t, patternMatches(rule, exact))
	mismatch := [9]int{0, 1, 0, 1, 3, 1, 0, 1, 0} // center differs
	assert.False(t, patternMatches(rule, mismatch))
}

// Painting the same value into the same cell is a no-op.
func TestPainter_SameValueIsNoOp(t *testing.T) {
	layer := &pixelforge_project.TilemapLayer{}
	p := NewPainter()
	assert.True(t, p.Paint(layer, 5, 5, 8))
	assert.False(t, p.Paint(layer, 5, 5, 8))
}

// EnsureLayer creates the first tilemap when none exist.
func TestEnsureLayer_CreatesWhenAbsent(t *testing.T) {
	scene := &pixelforge_project.Scene{}
	layer := EnsureLayer(scene, 16, 16)
	require.NotNil(t, layer)
	assert.Equal(t, 16, layer.TileW)
	assert.Equal(t, 16, layer.TileH)
	assert.Len(t, scene.Tilemaps, 1)
}

// Manual edit of a rule's Output mutates the synthesized output for
// subsequent matching strokes.
func TestAutoTile_ManualRuleEditOverrides(t *testing.T) {
	layer := &pixelforge_project.TilemapLayer{}
	synth := NewAutoTileRuleSynth()
	// Seed a single rule (count=2 promotes immediately).
	pattern := neighborhood(layer, 5, 5)
	layer.AutoTileRules = append(layer.AutoTileRules, pixelforge_project.AutoTileRule{
		Pattern: pattern,
		Output:  7,
		Count:   AutoTileActivationThreshold,
	})
	got, ok := synth.Apply(layer, 5, 5)
	require.True(t, ok)
	assert.Equal(t, 7, got)

	// User edits the rule output to 13.
	layer.AutoTileRules[0].Output = 13
	got, ok = synth.Apply(layer, 5, 5)
	require.True(t, ok)
	assert.Equal(t, 13, got)
}

// Round-tripping a project with tilemaps through save/load preserves
// the data and back-fills empty fields cleanly.
func TestTilemap_RoundTripsThroughProject(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	require.Len(t, p.Scenes, 1)
	layer := EnsureLayer(&p.Scenes[0], 8, 8)
	painter := NewPainter()
	painter.BeginStroke()
	painter.Paint(layer, 2, 3, 7)
	painter.EndStroke(layer)

	data, err := pixelforge_project.Encode(p)
	require.NoError(t, err)

	loaded, err := pixelforge_project.LoadReader(newByteReader(data))
	require.NoError(t, err)
	require.Len(t, loaded.Scenes[0].Tilemaps, 1)
	assert.Equal(t, 7, loaded.Scenes[0].Tilemaps[0].Grid[3][2])
}
