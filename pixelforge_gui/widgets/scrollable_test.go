package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func scrollableTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(64, 32)
	pixelforge.Cls()
}

func TestScrollable_ScrollClampsAtZero(t *testing.T) {
	scrollableTestSetup(t)
	s := widgets.NewScrollable(0, 0, 40, 60, widgets.ScrollableOptions{ContentH: 200})
	s.Scroll(-30)
	assert.Equal(t, 0, s.Offset())
}

func TestScrollable_ScrollClampsAtMax(t *testing.T) {
	scrollableTestSetup(t)
	s := widgets.NewScrollable(0, 0, 40, 60, widgets.ScrollableOptions{ContentH: 200})
	s.Scroll(9999)
	assert.Equal(t, s.MaxOffset(), s.Offset())
}

func TestScrollable_ScrollByMovesByStep(t *testing.T) {
	scrollableTestSetup(t)
	s := widgets.NewScrollable(0, 0, 40, 60, widgets.ScrollableOptions{ContentH: 500})
	s.ScrollBy(5)
	assert.Equal(t, 5*widgets.ScrollStep, s.Offset())
	// Content child should be repositioned to reflect the offset.
	assert.Equal(t, -5*widgets.ScrollStep, s.Content.Y)
}

func TestScrollable_ShortContentHasNoMaxOffset(t *testing.T) {
	scrollableTestSetup(t)
	s := widgets.NewScrollable(0, 0, 40, 60, widgets.ScrollableOptions{ContentH: 30})
	assert.Equal(t, 0, s.MaxOffset())
	s.ScrollBy(3)
	assert.Equal(t, 0, s.Offset(), "short content should never scroll")
}

func TestScrollable_SetContentH_ResizesContentAndClamps(t *testing.T) {
	scrollableTestSetup(t)
	s := widgets.NewScrollable(0, 0, 40, 60, widgets.ScrollableOptions{ContentH: 500})
	s.Scroll(400)
	s.SetContentH(80) // new max offset = 20
	assert.Equal(t, 20, s.Offset())
}

func TestScrollable_Draw_DoesNotPanic(t *testing.T) {
	scrollableTestSetup(t)
	root := pgui.New()
	s := widgets.NewScrollable(0, 0, 40, 30, widgets.ScrollableOptions{ContentH: 200})
	root.Attach(s.Element)
	assert.NotPanics(t, func() { root.Draw() })
}
