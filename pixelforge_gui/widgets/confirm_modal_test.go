package widgets

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
)

func TestConfirmModal_StartsHidden(t *testing.T) {
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	assert.False(t, c.Visible())
}

func TestConfirmModal_ShowAndOK(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	fired := false
	c.Show("title", "message", func() { fired = true }, nil)
	c.Draw() // populate okRect / cancelRect
	assert.True(t, c.HandleClick(c.okRect.X+5, c.okRect.Y+5))
	assert.True(t, fired)
	assert.False(t, c.Visible())
}

func TestConfirmModal_ShowAndCancel(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	cancelled := false
	c.Show("title", "msg", nil, func() { cancelled = true })
	c.Draw()
	assert.True(t, c.HandleClick(c.cancelRect.X+5, c.cancelRect.Y+5))
	assert.True(t, cancelled)
	assert.False(t, c.Visible())
}

func TestConfirmModal_EscapeFiresCancel(t *testing.T) {
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	cancelled := false
	c.Show("t", "m", nil, func() { cancelled = true })
	assert.True(t, c.HandleEscape())
	assert.True(t, cancelled)
	assert.False(t, c.Visible())
}

func TestConfirmModal_ClickWhileHiddenIgnored(t *testing.T) {
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	assert.False(t, c.HandleClick(10, 10))
}

func TestConfirmModal_HideDoesNotFireCallbacks(t *testing.T) {
	c := NewConfirmModal(0, 0, 320, 180, ConfirmModalOptions{})
	fired, cancelled := false, false
	c.Show("t", "m", func() { fired = true }, func() { cancelled = true })
	c.Hide()
	assert.False(t, fired)
	assert.False(t, cancelled)
}
