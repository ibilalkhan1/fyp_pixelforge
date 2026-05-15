package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// dismiss() flips Visible and fires OnDismiss exactly once.
func TestModal_DismissCallbackFires(t *testing.T) {
	calls := 0
	m := &Modal{
		Visible:   true,
		Body:      Rect{X: 10, Y: 10, W: 100, H: 100},
		OnDismiss: func() { calls++ },
	}
	m.dismiss()
	assert.False(t, m.Visible)
	assert.Equal(t, 1, calls)
}

// dismiss() on a Modal without an OnDismiss callback does not panic.
func TestModal_DismissNilCallback(t *testing.T) {
	m := &Modal{Visible: true}
	assert.NotPanics(t, func() { m.dismiss() })
	assert.False(t, m.Visible)
}

// HandleDismiss returns false when the modal is hidden.
func TestModal_HandleDismissReturnsFalseWhenHidden(t *testing.T) {
	m := &Modal{Visible: false}
	assert.False(t, m.HandleDismiss(0, 0))
}
