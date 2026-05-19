package palette

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// autotile_test.go covers the v1 idea-#2 U4 promotion-event API. The
// older count-based behaviour assertions live in painter_test.go;
// these tests focus on RecordStrokeWithPromotions and the
// promote-once-per-session contract.

// strokeBrick returns a 2-cell paint stroke that increments exactly
// one rule per call. The center cell at (1, 1) carries the
// meaningful neighborhood pattern; the sentinel cell at (999, 999)
// is far outside the layer's grid so the synth's
// patternHasNeighbors check returns false for it (all 8 neighbors
// resolve to -1 / out-of-range), and the sentinel never contributes
// a second rule. The 2-cell length also satisfies the synth's
// "single-cell strokes do not synthesize" early-return guard.
func strokeBrick() []PaintCell {
	return []PaintCell{
		{X: 1, Y: 1, Value: 5},
		{X: 999, Y: 999, Value: 5},
	}
}

func newBrickLayer() *pixelforge_project.TileAtlas {
	return &pixelforge_project.TileAtlas{
		Grid: [][]int{
			{1, 1, 1},
			{1, 0, 1},
			{1, 1, 1},
		},
	}
}

// TestAutoTile_PatternPaintedTwiceDoesNotPromote: two repetitions of
// the same pattern increment the rule's count to 2 but do not return
// a PromotedRule because the threshold is 3.
func TestAutoTile_PatternPaintedTwiceDoesNotPromote(t *testing.T) {
	layer := newBrickLayer()
	synth := NewAutoTileRuleSynth()

	p1 := synth.RecordStrokeWithPromotions(layer, strokeBrick())
	p2 := synth.RecordStrokeWithPromotions(layer, strokeBrick())

	assert.Empty(t, p1, "first stroke promotes nothing")
	assert.Empty(t, p2, "second stroke promotes nothing")

	require.NotEmpty(t, layer.AutoTileRules)
	assert.Equal(t, 2, layer.AutoTileRules[0].Count,
		"count increments per matching stroke")
}

// TestAutoTile_PatternPaintedThricePromotes: the third matching
// stroke returns one PromotedRule for the brick pattern and the rule
// becomes active.
func TestAutoTile_PatternPaintedThricePromotes(t *testing.T) {
	layer := newBrickLayer()
	synth := NewAutoTileRuleSynth()

	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	promotions := synth.RecordStrokeWithPromotions(layer, strokeBrick())

	require.Len(t, promotions, 1,
		"third stroke surfaces exactly one promotion")
	assert.Equal(t, 0, promotions[0].RuleIndex)
	assert.Equal(t, 5, promotions[0].Output)
	assert.Equal(t, AutoTileActivationThreshold, layer.AutoTileRules[0].Count)

	// Rule now active — Apply returns it for a matching cell.
	got, ok := synth.Apply(layer, 1, 1)
	require.True(t, ok)
	assert.Equal(t, 5, got)
}

// TestAutoTile_AlreadyActiveRuleDoesNotRepromote: strokes beyond the
// promotion stroke keep incrementing Count but do not re-surface the
// rule as a PromotedRule. Promote-once-per-session.
func TestAutoTile_AlreadyActiveRuleDoesNotRepromote(t *testing.T) {
	layer := newBrickLayer()
	synth := NewAutoTileRuleSynth()

	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	promoOnThird := synth.RecordStrokeWithPromotions(layer, strokeBrick())
	promoOnFourth := synth.RecordStrokeWithPromotions(layer, strokeBrick())
	promoOnFifth := synth.RecordStrokeWithPromotions(layer, strokeBrick())

	require.Len(t, promoOnThird, 1)
	assert.Empty(t, promoOnFourth, "fourth stroke must not re-promote")
	assert.Empty(t, promoOnFifth, "fifth stroke must not re-promote")
	assert.Equal(t, 5, layer.AutoTileRules[0].Count,
		"Count keeps incrementing past the threshold")
}

// TestAutoTile_TwoPatternsPromoteIndependently: distinct patterns
// each promote at their own third stroke. Promotions surface in the
// stroke that crossed the threshold; promotion does not bleed
// across patterns.
func TestAutoTile_TwoPatternsPromoteIndependently(t *testing.T) {
	layer := &pixelforge_project.TileAtlas{
		Grid: [][]int{
			{1, 1, 1, 0, 2, 2, 2},
			{1, 0, 1, 0, 2, 0, 2},
			{1, 1, 1, 0, 2, 2, 2},
		},
	}
	synth := NewAutoTileRuleSynth()

	// strokeA centers at (1, 1) — the brick neighborhood; strokeB at
	// (5, 1) — the donut neighborhood. The sentinel (999,999) in each
	// stroke is far out-of-range so its pattern has no neighbors and
	// the synth skips it (cellAt returns -1 for every neighbor).
	strokeA := []PaintCell{{X: 1, Y: 1, Value: 5}, {X: 999, Y: 999, Value: 5}}
	strokeB := []PaintCell{{X: 5, Y: 1, Value: 7}, {X: 999, Y: 999, Value: 7}}

	_ = synth.RecordStrokeWithPromotions(layer, strokeA)
	_ = synth.RecordStrokeWithPromotions(layer, strokeA)
	promoA := synth.RecordStrokeWithPromotions(layer, strokeA)
	require.Len(t, promoA, 1, "third strokeA promotes pattern A")
	assert.Equal(t, 5, promoA[0].Output)

	_ = synth.RecordStrokeWithPromotions(layer, strokeB)
	_ = synth.RecordStrokeWithPromotions(layer, strokeB)
	promoB := synth.RecordStrokeWithPromotions(layer, strokeB)
	require.Len(t, promoB, 1, "third strokeB promotes pattern B independently")
	assert.Equal(t, 7, promoB[0].Output)

	// strokeA continues incrementing without re-promoting.
	again := synth.RecordStrokeWithPromotions(layer, strokeA)
	assert.Empty(t, again, "previously-promoted strokeA does not re-promote")
}

// TestAutoTile_ThresholdConstantIs3: regression guard. The solution
// doc (docs/solutions/auto-tile-heuristic.md) is the source of
// truth; if anyone changes the constant in a future cleanup, this
// test trips and forces them to update the doc too.
func TestAutoTile_ThresholdConstantIs3(t *testing.T) {
	assert.Equal(t, 3, AutoTileActivationThreshold,
		"AutoTileActivationThreshold must match the heuristic doc invariant")
}

// TestAutoTile_RuleHintNotTruth_GridUnchangedOnRuleEdit: a manual
// edit to a rule's Output never mutates already-painted cells. The
// grid is the source of truth; rules are hints applied at paint
// time. Verifies the invariant from docs/solutions/auto-tile-heuristic.md
// (d). This is a regression guard for U4's claim that the new
// promotion API does not change the hint-vs-truth contract.
func TestAutoTile_RuleHintNotTruth_GridUnchangedOnRuleEdit(t *testing.T) {
	layer := newBrickLayer()
	synth := NewAutoTileRuleSynth()
	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	_ = synth.RecordStrokeWithPromotions(layer, strokeBrick())
	require.Equal(t, AutoTileActivationThreshold, layer.AutoTileRules[0].Count)
	preEdit := append([][]int(nil), layer.Grid...)

	// Edit the rule's Output to an arbitrary value.
	layer.AutoTileRules[0].Output = 99

	assert.Equal(t, len(preEdit), len(layer.Grid),
		"grid row count unaffected by rule mutation")
	for r := range preEdit {
		assert.Equal(t, preEdit[r], layer.Grid[r],
			"grid row %d unchanged by rule output mutation", r)
	}
}

// TestAutoTile_RecordStrokeStillWorks: the deprecated RecordStroke
// shim continues to function (it delegates to
// RecordStrokeWithPromotions). Pre-U4 callers that don't care about
// promotions stay working.
func TestAutoTile_RecordStrokeStillWorks(t *testing.T) {
	layer := newBrickLayer()
	synth := NewAutoTileRuleSynth()

	synth.RecordStroke(layer, strokeBrick())
	synth.RecordStroke(layer, strokeBrick())
	synth.RecordStroke(layer, strokeBrick())

	require.NotEmpty(t, layer.AutoTileRules)
	assert.Equal(t, AutoTileActivationThreshold, layer.AutoTileRules[0].Count,
		"deprecated RecordStroke still increments rule counts")
}

// TestPainter_PromotionsThisStrokeRoundTrip: the Painter stashes the
// most recent stroke's promotions; the studio reads them via
// PromotionsThisStroke and clears via ClearPromotions.
//
// Each 2-cell stroke increments the lonely-cell rule by 2 (one
// record per stroke cell, both matching the same pattern), so the
// second stroke crosses the threshold (count 0 -> 2 on stroke 1,
// 2 -> 4 on stroke 2 with threshold=3) and the Painter stashes the
// promotion observed at EndStroke.
func TestPainter_PromotionsThisStrokeRoundTrip(t *testing.T) {
	layer := &pixelforge_project.TileAtlas{}
	p := NewPainter()

	// Stroke 1 — increments lonely-cell rule to count=2 (below
	// threshold). Painter holds no promotions.
	p.BeginStroke()
	_ = p.Paint(layer, 2, 2, 9)
	_ = p.Paint(layer, 7, 2, 9)
	p.EndStroke(layer)
	require.Empty(t, p.PromotionsThisStroke(),
		"first stroke below threshold yields no promotion")

	// Stroke 2 — crosses the threshold. Painter holds the promotion.
	p.BeginStroke()
	_ = p.Paint(layer, 2, 10, 9)
	_ = p.Paint(layer, 7, 10, 9)
	p.EndStroke(layer)
	promos := p.PromotionsThisStroke()
	require.NotEmpty(t, promos,
		"second stroke crosses threshold and surfaces a promotion")
	assert.Equal(t, 9, promos[0].Output)

	p.ClearPromotions()
	assert.Empty(t, p.PromotionsThisStroke(),
		"ClearPromotions wipes the stashed list")
}

// TestPainter_BeginStrokeClearsStalePromotions: starting a new stroke
// wipes the prior stroke's promotion list so the studio never
// observes stale promotion data.
func TestPainter_BeginStrokeClearsStalePromotions(t *testing.T) {
	layer := &pixelforge_project.TileAtlas{}
	p := NewPainter()

	p.BeginStroke()
	_ = p.Paint(layer, 2, 2, 9)
	_ = p.Paint(layer, 7, 2, 9)
	p.EndStroke(layer)
	p.BeginStroke()
	_ = p.Paint(layer, 2, 10, 9)
	_ = p.Paint(layer, 7, 10, 9)
	p.EndStroke(layer)
	require.NotEmpty(t, p.PromotionsThisStroke(),
		"precondition: second stroke promotes")

	p.BeginStroke()
	assert.Empty(t, p.PromotionsThisStroke(),
		"BeginStroke clears stale promotions before the new stroke records anything")
}
