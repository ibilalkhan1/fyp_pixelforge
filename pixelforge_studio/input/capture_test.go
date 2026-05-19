package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCaptureMode_StartsInactive guards the zero-value contract: a
// freshly-constructed CaptureMode is Inactive with no in-flight
// intent.
func TestCaptureMode_StartsInactive(t *testing.T) {
	var c CaptureMode
	assert.Equal(t, CaptureInactive, c.State())
	assert.False(t, c.IsActive())
	assert.Empty(t, c.Intent())
}

// TestCaptureMode_BeginCaptureTransitionsToWaiting exercises the
// primary entry point: BeginCapture(intent) → Waiting{intent}.
func TestCaptureMode_BeginCaptureTransitionsToWaiting(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	assert.Equal(t, CaptureWaiting, c.State())
	assert.True(t, c.IsActive())
	assert.Equal(t, "input/jump", c.Intent())
}

// TestCaptureMode_OnKey_CapturesNonModifier verifies the happy path:
// in Waiting state, a non-modifier keypress captures + returns to
// Inactive with the binding value.
func TestCaptureMode_OnKey_CapturesNonModifier(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	bound, newBinding := c.OnKey(ebiten.KeySpace)
	assert.True(t, bound, "Space press should capture")
	assert.Equal(t, " ", newBinding, "Space should map to the U3 compiler value \" \"")
	assert.Equal(t, CaptureInactive, c.State(), "capture transitions to Inactive after binding")
	assert.Empty(t, c.Intent(), "in-flight intent cleared after binding")
}

// TestCaptureMode_OnKey_EscCancels covers the Esc-cancel path: the
// state machine returns to Inactive without recording a binding.
func TestCaptureMode_OnKey_EscCancels(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	bound, newBinding := c.OnKey(ebiten.KeyEscape)
	assert.False(t, bound, "Esc should not capture")
	assert.Empty(t, newBinding)
	assert.Equal(t, CaptureInactive, c.State(), "Esc transitions to Inactive")
	assert.Empty(t, c.Intent())
}

// TestCaptureMode_OnKey_ModifierIgnored is the design-lens doc-review
// finding's enforcement: pressing Shift/Ctrl/Alt alone in Waiting state
// must NOT capture — capture stays in Waiting so the designer can
// follow up with the modified key.
func TestCaptureMode_OnKey_ModifierIgnored(t *testing.T) {
	modifiers := []ebiten.Key{
		ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
	}
	for _, k := range modifiers {
		var c CaptureMode
		c.BeginCapture("input/jump")
		bound, newBinding := c.OnKey(k)
		assert.False(t, bound, "modifier %v should not capture", k)
		assert.Empty(t, newBinding, "modifier %v should not produce a binding", k)
		assert.Equal(t, CaptureWaiting, c.State(), "modifier %v should leave capture Waiting", k)
		assert.Equal(t, "input/jump", c.Intent(), "modifier %v should not clear in-flight intent", k)
	}
}

// TestCaptureMode_OnKey_InactiveIgnored documents the off-state
// behaviour: OnKey while Inactive is a no-op.
func TestCaptureMode_OnKey_InactiveIgnored(t *testing.T) {
	var c CaptureMode
	bound, newBinding := c.OnKey(ebiten.KeySpace)
	assert.False(t, bound, "Inactive OnKey should not capture")
	assert.Empty(t, newBinding)
	assert.Equal(t, CaptureInactive, c.State())
}

// TestCaptureMode_BeginCaptureSameIntent_TogglesOff covers the design-
// lens "click Capture again to cancel" finding.
func TestCaptureMode_BeginCaptureSameIntent_TogglesOff(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	require.Equal(t, CaptureWaiting, c.State())
	c.BeginCapture("input/jump")
	assert.Equal(t, CaptureInactive, c.State(), "same-intent BeginCapture toggles off")
	assert.Empty(t, c.Intent())
}

// TestCaptureMode_BeginCaptureDifferentIntent_Retargets shows that
// clicking Capture on a different row mid-capture redirects the
// state machine to the new intent without an explicit cancel.
func TestCaptureMode_BeginCaptureDifferentIntent_Retargets(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	require.Equal(t, "input/jump", c.Intent())
	c.BeginCapture("input/attack")
	assert.Equal(t, CaptureWaiting, c.State(), "different-intent BeginCapture stays Waiting")
	assert.Equal(t, "input/attack", c.Intent(), "in-flight intent updates")
}

// TestCaptureMode_Cancel_ResetsState verifies the Cancel hook used
// when the workspace loses focus / project changes.
func TestCaptureMode_Cancel_ResetsState(t *testing.T) {
	var c CaptureMode
	c.BeginCapture("input/jump")
	c.Cancel()
	assert.Equal(t, CaptureInactive, c.State())
	assert.Empty(t, c.Intent())

	// Idempotent: Cancel on Inactive is a no-op.
	c.Cancel()
	assert.Equal(t, CaptureInactive, c.State())
}

// TestCaptureMode_OnKey_LetterCapturesUpperLabel checks that letter
// keys produce the canonical uppercase pixelforge_key string.
func TestCaptureMode_OnKey_LetterCapturesUpperLabel(t *testing.T) {
	cases := []struct {
		key  ebiten.Key
		want string
	}{
		{ebiten.KeyA, "A"},
		{ebiten.KeyZ, "Z"},
		{ebiten.KeyW, "W"},
		{ebiten.KeyS, "S"},
		{ebiten.KeyDigit0, "0"},
		{ebiten.KeyDigit9, "9"},
		{ebiten.KeyArrowUp, "Up"},
		{ebiten.KeyArrowDown, "Down"},
		{ebiten.KeyArrowLeft, "Left"},
		{ebiten.KeyArrowRight, "Right"},
		{ebiten.KeyEnter, "Enter"},
		{ebiten.KeyTab, "Tab"},
		{ebiten.KeyF1, "F1"},
		{ebiten.KeyF12, "F12"},
	}
	for _, tc := range cases {
		var c CaptureMode
		c.BeginCapture("input/x")
		bound, newBinding := c.OnKey(tc.key)
		assert.True(t, bound, "%v should capture", tc.key)
		assert.Equal(t, tc.want, newBinding, "%v should bind to %q", tc.key, tc.want)
	}
}

// TestCaptureMode_NilSafe documents the nil-receiver contract — the
// workspace's CaptureMode field is a value, not a pointer, so this
// is belt-and-braces but covers any future *CaptureMode use.
func TestCaptureMode_NilSafe(t *testing.T) {
	var c *CaptureMode
	assert.Equal(t, CaptureInactive, c.State())
	assert.False(t, c.IsActive())
	assert.Empty(t, c.Intent())
	c.BeginCapture("x") // must not panic
	c.Cancel()          // must not panic
	bound, val := c.OnKey(ebiten.KeySpace)
	assert.False(t, bound)
	assert.Empty(t, val)
}
