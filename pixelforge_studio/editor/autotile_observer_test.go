package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// autotile_observer_test.go covers the U6 + U7 editor-side seams:
// observer wiring, promotion queueing, session suppression. The
// concrete observer (palette.AutoTileRuleSynth wrapper) is exercised
// via the palette package; here we test the contracts the editor
// promises to that observer + to the U7 toast subsystem.

// stubAutoTileObserver records every RecordStroke call so tests can
// inspect what the canvas dispatch handed in. The next promotion to
// return is configurable per-call via the queue.
type stubAutoTileObserver struct {
	calls       []stubObserverCall
	nextReturns [][]AutoTilePromotion
}

type stubObserverCall struct {
	Layer *pixelforge_project.TileAtlas
	Cells []AutoTileCell
}

func (s *stubAutoTileObserver) RecordStroke(
	layer *pixelforge_project.TileAtlas,
	cells []AutoTileCell,
) []AutoTilePromotion {
	s.calls = append(s.calls, stubObserverCall{Layer: layer, Cells: cells})
	if len(s.nextReturns) == 0 {
		return nil
	}
	out := s.nextReturns[0]
	s.nextReturns = s.nextReturns[1:]
	return out
}

func (s *stubAutoTileObserver) queue(p ...AutoTilePromotion) {
	s.nextReturns = append(s.nextReturns, p)
}

// TestEditor_SetAutoTileObserverWiresLookup: the observer set via
// SetAutoTileObserver is the one AutoTileObserver returns. Nil sets
// clear the wiring.
func TestEditor_SetAutoTileObserverWiresLookup(t *testing.T) {
	e := New()
	assert.Nil(t, e.AutoTileObserver())

	stub := &stubAutoTileObserver{}
	e.SetAutoTileObserver(stub)
	assert.Same(t, stub, e.AutoTileObserver().(*stubAutoTileObserver))

	e.SetAutoTileObserver(nil)
	assert.Nil(t, e.AutoTileObserver())
}

// TestEditor_RecordStrokeAndQueueDelegatesToObserver: the canvas
// dispatch's hook turns a StrokeCommand into AutoTileCell entries
// and calls the observer.
func TestEditor_RecordStrokeAndQueueDelegatesToObserver(t *testing.T) {
	e := New()
	stub := &stubAutoTileObserver{}
	e.SetAutoTileObserver(stub)

	layer := &pixelforge_project.TileAtlas{}
	cmd := &StrokeCommand{
		Layer: layer,
		Diffs: []CellDiff{
			{Col: 1, Row: 2, NewID: 5},
			{Col: 3, Row: 4, NewID: 7},
		},
	}
	e.recordStrokeAndQueuePromotions(layer, cmd)

	require.Len(t, stub.calls, 1)
	assert.Same(t, layer, stub.calls[0].Layer)
	require.Len(t, stub.calls[0].Cells, 2)
	assert.Equal(t, AutoTileCell{X: 1, Y: 2, Value: 5}, stub.calls[0].Cells[0])
	assert.Equal(t, AutoTileCell{X: 3, Y: 4, Value: 7}, stub.calls[0].Cells[1])
}

// TestEditor_RecordStrokeAndQueueSkipsWhenObserverUnwired: with no
// observer set the hook is a quiet no-op (so v0 builds that haven't
// wired palette can still paint).
func TestEditor_RecordStrokeAndQueueSkipsWhenObserverUnwired(t *testing.T) {
	e := New()
	layer := &pixelforge_project.TileAtlas{}
	cmd := &StrokeCommand{
		Layer: layer,
		Diffs: []CellDiff{{Col: 1, Row: 2, NewID: 5}},
	}
	assert.NotPanics(t, func() {
		e.recordStrokeAndQueuePromotions(layer, cmd)
	})
	assert.Nil(t, e.QueuedPromotion())
}

// TestEditor_RecordStrokeAndQueueSkipsEmptyDiffs: a StrokeCommand
// with no diffs (no-op stroke) bypasses the observer entirely.
func TestEditor_RecordStrokeAndQueueSkipsEmptyDiffs(t *testing.T) {
	e := New()
	stub := &stubAutoTileObserver{}
	e.SetAutoTileObserver(stub)

	layer := &pixelforge_project.TileAtlas{}
	cmd := &StrokeCommand{Layer: layer, Diffs: nil}
	e.recordStrokeAndQueuePromotions(layer, cmd)

	assert.Empty(t, stub.calls,
		"empty-diff strokes do not call into the observer")
}

// TestEditor_HandlePromotionsQueuesFirstAndSuppressesRest: when the
// observer returns multiple promotions for one stroke, the first
// surfaces via QueuedPromotion and the rest go straight into the
// session-suppression map.
func TestEditor_HandlePromotionsQueuesFirstAndSuppressesRest(t *testing.T) {
	e := New()
	promo1 := AutoTilePromotion{RuleIndex: 0, Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	promo2 := AutoTilePromotion{RuleIndex: 1, Pattern: [9]int{2, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 7}
	promo3 := AutoTilePromotion{RuleIndex: 2, Pattern: [9]int{3, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 9}

	e.handleAutoTilePromotions([]AutoTilePromotion{promo1, promo2, promo3})

	require.NotNil(t, e.QueuedPromotion())
	assert.Equal(t, promo1, *e.QueuedPromotion(),
		"first promotion surfaces as the visible toast")
	assert.Equal(t, 2, e.SuppressedRuleCount(),
		"second and third promotions auto-suppress for the session")
	assert.True(t, e.isSuppressed(promo2))
	assert.True(t, e.isSuppressed(promo3))
}

// TestEditor_HandlePromotionsRespectsSuppression: a promotion whose
// signature is already in the suppression map does not re-surface.
// Matches R5 (No is sticky for the session).
func TestEditor_HandlePromotionsRespectsSuppression(t *testing.T) {
	e := New()
	promo := AutoTilePromotion{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	e.suppressRule(promo)

	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	assert.Nil(t, e.QueuedPromotion(),
		"a suppressed promotion never becomes a queued toast")
}

// TestEditor_ClearQueuedPromotion: the toast subsystem calls
// ClearQueuedPromotion after the designer dismisses (Yes / No /
// Esc); the next promotion gets a clean slot.
func TestEditor_ClearQueuedPromotion(t *testing.T) {
	e := New()
	promo := AutoTilePromotion{Output: 5}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})
	require.NotNil(t, e.QueuedPromotion())

	e.ClearQueuedPromotion()
	assert.Nil(t, e.QueuedPromotion())
}

// TestEditor_SessionSuppressionAddsAndChecks: suppressRule +
// isSuppressed form the basic set/get pair. (Pattern, Output) is
// the signature; same shape on either field means same signature.
func TestEditor_SessionSuppressionAddsAndChecks(t *testing.T) {
	e := New()
	a := AutoTilePromotion{Pattern: [9]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, Output: 5}
	b := AutoTilePromotion{Pattern: [9]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, Output: 6}
	c := AutoTilePromotion{Pattern: [9]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, Output: 5}

	e.suppressRule(a)
	assert.True(t, e.isSuppressed(c),
		"same (Pattern, Output) signature matches even on different promotion instance")
	assert.False(t, e.isSuppressed(b),
		"different Output produces a different signature")
}

// TestEditor_ResetSessionRuleSuppressionClears: ResetSession-
// RuleSuppression wipes the map (called when a new project loads).
func TestEditor_ResetSessionRuleSuppressionClears(t *testing.T) {
	e := New()
	e.suppressRule(AutoTilePromotion{Output: 5})
	require.Equal(t, 1, e.SuppressedRuleCount())

	e.ResetSessionRuleSuppression()
	assert.Equal(t, 0, e.SuppressedRuleCount())
}
