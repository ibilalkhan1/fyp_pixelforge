package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func widgetsFilePickerOpts() widgets.FilePickerOptions {
	return widgets.FilePickerOptions{
		StartPath: ".",
	}
}

func TestChromeVisibility_DefaultsVisible(t *testing.T) {
	e := New()
	assert.False(t, e.ChromeHidden(), "chrome should start visible")
}

func TestChromeVisibility_ToggleHidesAndRestores(t *testing.T) {
	e := New()
	e.ToggleChromeVisibility()
	assert.True(t, e.ChromeHidden())
	e.ToggleChromeVisibility()
	assert.False(t, e.ChromeHidden())
}

func TestChromeVisibility_HandleEscape_TogglesWhenNoModal(t *testing.T) {
	e := New()
	e.handleEscape()
	assert.True(t, e.ChromeHidden())
	e.handleEscape()
	assert.False(t, e.ChromeHidden())
}

func TestChromeVisibility_HandleEscape_ConfirmModalSkipsToggle(t *testing.T) {
	e := New()
	e.confirmDialog.Show("Confirm?", "Are you sure?", func() {})
	e.handleEscape()
	assert.False(t, e.ChromeHidden(),
		"open confirm modal must absorb Esc precedence — chrome toggle does not fire")
}

func TestChromeVisibility_HandleEscape_FilePickerSkipsToggle(t *testing.T) {
	e := New()
	// Open the file picker via its standard Open() API; we only need
	// Visible() to report true.
	e.filePicker.Open(widgetsFilePickerOpts())
	e.handleEscape()
	assert.False(t, e.ChromeHidden())
}

func TestChromeVisibility_GameCanvasIsAllocated(t *testing.T) {
	cv := newChromeVisibility()
	cv.EnsureGameCanvas(320, 180)
	assert.Equal(t, 320, cv.GameCanvas().W())
	assert.Equal(t, 180, cv.GameCanvas().H())

	cv.EnsureGameCanvas(160, 90)
	assert.Equal(t, 160, cv.GameCanvas().W())
	assert.Equal(t, 90, cv.GameCanvas().H())
}

func TestChromeVisibility_EnsureGameCanvas_NoOpForDegenerate(t *testing.T) {
	cv := newChromeVisibility()
	assert.NotPanics(t, func() { cv.EnsureGameCanvas(0, 0) })
	assert.NotPanics(t, func() { cv.EnsureGameCanvas(-5, 100) })
}
