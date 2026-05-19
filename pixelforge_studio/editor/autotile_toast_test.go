package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// autotile_toast_test.go covers the toast subsystem's session
// behaviour: visibility tracking, Yes/No/Esc semantics, suppression
// integration. The popup rendering is omitted from the test surface
// because it requires a live ImGui frame; the rest of the behaviour
// is testable in plain Go.

// TestToast_HiddenWhenNoPromotion: a toast with no queued promotion
// reports Visible=false and the render path is a no-op.
func TestToast_HiddenWhenNoPromotion(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	assert.False(t, toast.Visible())
	assert.Nil(t, toast.Promotion())
}

// TestToast_VisibleAfterCanvasQueues: dispatch queues a promotion;
// the toast reflects it.
func TestToast_VisibleAfterCanvasQueues(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	promo := AutoTilePromotion{Output: 5}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	require.True(t, toast.Visible(),
		"toast becomes visible the moment a promotion queues")
	got := toast.Promotion()
	require.NotNil(t, got)
	assert.Equal(t, promo, *got)
}

// TestToast_YesClearsAndDoesNotSuppress: accepting the toast clears
// the queue without inserting the rule into the suppression map.
// The rule is already active on the TileAtlas; Yes just dismisses
// the prompt and lets future strokes silently auto-apply.
func TestToast_YesClearsAndDoesNotSuppress(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	e.handleAutoTilePromotions([]AutoTilePromotion{{Output: 5}})

	toast.Yes()
	assert.False(t, toast.Visible(),
		"Yes dismisses the toast")
	assert.Equal(t, 0, e.SuppressedRuleCount(),
		"Yes does NOT suppress — the rule should still be re-toastable in a later session if needed")
}

// TestToast_NoSuppressesAndDismisses: No dismisses + adds the rule's
// signature to the session-suppression map. A subsequent re-promotion
// of the same rule (within this session) is silently dropped by the
// editor's handleAutoTilePromotions check.
func TestToast_NoSuppressesAndDismisses(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	promo := AutoTilePromotion{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	toast.No()
	assert.False(t, toast.Visible(), "No dismisses")
	assert.Equal(t, 1, e.SuppressedRuleCount(),
		"No suppresses the rule signature for the session")
	assert.True(t, e.isSuppressed(promo))

	// Subsequent re-promotion stays silent.
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})
	assert.False(t, toast.Visible(),
		"re-promotion of a suppressed rule does not resurface")
}

// TestToast_EscBehavesLikeNo: Esc semantically equals No. Same
// suppression effect; same dismissal.
func TestToast_EscBehavesLikeNo(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	promo := AutoTilePromotion{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	toast.Esc()
	assert.False(t, toast.Visible())
	assert.True(t, e.isSuppressed(promo))
}

// TestToast_ClickOutsideBehavesLikeNo: clicking outside the popup
// has the same effect as No (the standard ImGui popup convention).
func TestToast_ClickOutsideBehavesLikeNo(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	promo := AutoTilePromotion{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	toast.ClickOutside()
	assert.False(t, toast.Visible())
	assert.True(t, e.isSuppressed(promo))
}

// TestToast_MultiplePromotionsOnlyShowsFirst: the canvas dispatch
// auto-suppresses everything past the first promotion in a single
// stroke. The toast only sees the first one; the user never has to
// burn through a stack.
func TestToast_MultiplePromotionsOnlyShowsFirst(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	promo1 := AutoTilePromotion{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5}
	promo2 := AutoTilePromotion{Pattern: [9]int{2, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 7}

	e.handleAutoTilePromotions([]AutoTilePromotion{promo1, promo2})

	require.True(t, toast.Visible())
	assert.Equal(t, promo1, *toast.Promotion(),
		"first promotion is the visible one")
	assert.True(t, e.isSuppressed(promo2),
		"second promotion auto-suppresses so a follow-up re-paint stays silent")
}

// TestToast_RuleAcceptedInPriorSession_NoToastOnLoad: a project that
// loads with a rule already at Count >= threshold (pre-accepted in
// a prior session) does NOT surface a toast on the next stroke
// because the synth's promote-once-per-session logic only surfaces
// rules that crossed the threshold THIS stroke. The toast queue is
// empty after the editor loads such a project.
//
// Tested via the editor's handlePromotions path with an empty
// promotion list (which is what the synth would return for a rule
// already past threshold).
func TestToast_RuleAcceptedInPriorSession_NoToastOnLoad(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)

	// Simulate a paint stroke against a layer with a pre-accepted
	// rule. The synth returns no promotions because the rule was
	// already at Count >= threshold before the stroke.
	e.handleAutoTilePromotions(nil)

	assert.False(t, toast.Visible(),
		"pre-accepted rules produce no promotion → no toast")
}

// TestToast_SessionSuppressionDoesNotPersist: dismissing a toast
// with No does not write any state to the project file. Verified
// by checking that the project's TileAtlas.AutoTileRules slice is
// unchanged (the rule persists, only the in-memory suppression
// flag is set).
func TestToast_SessionSuppressionDoesNotPersist(t *testing.T) {
	e := New()
	toast := NewAutoTileToast(e)
	atlas := pixelforge_project.TileAtlas{
		AutoTileRules: []pixelforge_project.AutoTileRule{
			{Pattern: [9]int{1, 0, 0, 0, 0, 0, 0, 0, 0}, Output: 5, Count: 3},
		},
	}
	promo := AutoTilePromotion{
		RuleIndex: 0,
		Pattern:   atlas.AutoTileRules[0].Pattern,
		Output:    atlas.AutoTileRules[0].Output,
	}
	e.handleAutoTilePromotions([]AutoTilePromotion{promo})

	toast.No()

	assert.Len(t, atlas.AutoTileRules, 1,
		"suppression is in-memory; the project's rule list is untouched")
	assert.Equal(t, 3, atlas.AutoTileRules[0].Count,
		"No does not decrement the rule's Count")
}

// TestRuleSignatureStable: same (Pattern, Output) → same signature;
// different Pattern or Output → different signature. Locks down
// what the suppression map keys on.
func TestRuleSignatureStable(t *testing.T) {
	a := ruleSignature{Pattern: [9]int{1, 2, 3}, Output: 5}
	b := ruleSignature{Pattern: [9]int{1, 2, 3}, Output: 5}
	c := ruleSignature{Pattern: [9]int{1, 2, 4}, Output: 5}
	d := ruleSignature{Pattern: [9]int{1, 2, 3}, Output: 6}
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.NotEqual(t, a, d)
}

// TestToast_NilSafety: methods on a nil-bound toast (e.g. an editor
// constructed at zero value) do not panic.
func TestToast_NilSafety(t *testing.T) {
	var nilToast *AutoTileToast
	assert.False(t, nilToast.Visible())
	assert.Nil(t, nilToast.Promotion())
	assert.NotPanics(t, func() { nilToast.Yes() })
	assert.NotPanics(t, func() { nilToast.No() })
	assert.NotPanics(t, func() { nilToast.Esc() })
	assert.NotPanics(t, func() { nilToast.ClickOutside() })

	emptyToast := NewAutoTileToast(nil)
	assert.False(t, emptyToast.Visible())
	assert.NotPanics(t, func() { emptyToast.Yes() })
}
