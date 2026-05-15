package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Confirm fires the handler and hides the dialog.
func TestConfirmDialog_ConfirmFiresHandler(t *testing.T) {
	calls := 0
	c := NewConfirmDialog()
	c.Show("title", "msg", func() { calls++ })
	assert.True(t, c.Visible())
	c.Confirm()
	assert.False(t, c.Visible())
	assert.Equal(t, 1, calls)
}

// Cancel hides without running the handler.
func TestConfirmDialog_CancelSkipsHandler(t *testing.T) {
	calls := 0
	c := NewConfirmDialog()
	c.Show("title", "msg", func() { calls++ })
	c.Cancel()
	assert.False(t, c.Visible())
	assert.Equal(t, 0, calls)
}

// Confirm with nil handler does not panic.
func TestConfirmDialog_ConfirmNilHandler(t *testing.T) {
	c := NewConfirmDialog()
	c.Show("title", "msg", nil)
	assert.NotPanics(t, c.Confirm)
	assert.False(t, c.Visible())
}

// PromptIfDirty: clean project → action runs immediately, no modal.
func TestEditor_PromptIfDirtyCleanRunsImmediately(t *testing.T) {
	e := New()
	calls := 0
	e.PromptIfDirty("t", "m", func() { calls++ })
	assert.False(t, e.confirmDialog.Visible())
	assert.Equal(t, 1, calls)
}

// PromptIfDirty: dirty project → action runs only on confirm.
func TestEditor_PromptIfDirtyDirtyOpensModal(t *testing.T) {
	e := New()
	e.MarkDirty()
	calls := 0
	e.PromptIfDirty("t", "m", func() { calls++ })
	assert.True(t, e.confirmDialog.Visible())
	assert.Equal(t, 0, calls)

	e.confirmDialog.Confirm()
	assert.Equal(t, 1, calls)
}

// PromptIfDirty: Cancel leaves the project untouched.
func TestEditor_PromptIfDirtyDirtyCancel(t *testing.T) {
	e := New()
	e.MarkDirty()
	calls := 0
	e.PromptIfDirty("t", "m", func() { calls++ })
	e.confirmDialog.Cancel()
	assert.Equal(t, 0, calls)
}
