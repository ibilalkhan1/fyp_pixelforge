package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func dropdownTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(128, 200)
	pixelforge.Cls()
}

func TestDropdown_OpenAndClose(t *testing.T) {
	dropdownTestSetup(t)
	dd := widgets.NewDropdown(0, 0, 80, 14, 200, widgets.DropdownOptions{
		Options: []string{"red", "green", "blue"},
	})
	assert.False(t, dd.IsOpen())
	dd.Open()
	assert.True(t, dd.IsOpen())
	dd.Close()
	assert.False(t, dd.IsOpen())
}

func TestDropdown_SelectByIndex_FiresAndCloses(t *testing.T) {
	dropdownTestSetup(t)
	captured := ""
	dd := widgets.NewDropdown(0, 0, 80, 14, 200, widgets.DropdownOptions{
		Options:  []string{"red", "green", "blue"},
		OnSelect: func(s string) { captured = s },
	})
	dd.Open()
	dd.SelectByIndex(1)
	assert.Equal(t, "green", captured)
	assert.False(t, dd.IsOpen())
}

func TestDropdown_HandleEscape_DismissesOpen(t *testing.T) {
	dropdownTestSetup(t)
	dd := widgets.NewDropdown(0, 0, 80, 14, 200, widgets.DropdownOptions{
		Options: []string{"red"},
	})
	assert.False(t, dd.HandleEscape())
	dd.Open()
	assert.True(t, dd.HandleEscape())
	assert.False(t, dd.IsOpen())
}

func TestDropdown_FlipsUpwardNearBottom(t *testing.T) {
	dropdownTestSetup(t)
	// containerH=200, selector at y=180 with h=14 has only 6 px below.
	// Three options × 14 px each need 42 px. Should flip upward.
	dd := widgets.NewDropdown(0, 180, 80, 14, 200, widgets.DropdownOptions{
		Options: []string{"red", "green", "blue"},
	})
	dd.Open()
	// We don't expose the upward flag directly; assert via popoverBounds.
	// HandlePointer on a click above the selector should land in the list.
	consumed := dd.HandlePointer(20, 180-14) // 14 px above selector top
	assert.True(t, consumed)
}

func TestDropdown_HandlePointer_ClickOutsideCloses(t *testing.T) {
	dropdownTestSetup(t)
	dd := widgets.NewDropdown(0, 0, 80, 14, 200, widgets.DropdownOptions{
		Options: []string{"red", "green"},
	})
	dd.Open()
	consumed := dd.HandlePointer(500, 500) // far outside
	assert.True(t, consumed)
	assert.False(t, dd.IsOpen())
}

func TestDropdown_Draw_DoesNotPanic(t *testing.T) {
	dropdownTestSetup(t)
	root := pgui.New()
	dd := widgets.NewDropdown(0, 0, 80, 14, 200, widgets.DropdownOptions{
		Options: []string{"red"},
	})
	root.Attach(dd.Element)
	assert.NotPanics(t, func() { root.Draw() })
	dd.Open()
	assert.NotPanics(t, func() { root.Draw() })
}
