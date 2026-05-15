package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func tabsTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(320, 32)
	pixelforge.Cls()
}

func TestTabs_SelectFiresCallback(t *testing.T) {
	tabsTestSetup(t)
	fired := -1
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{
		Labels:   []string{"Scene", "Palette", "Audio"},
		Selected: 0,
		OnSelect: func(idx int) { fired = idx },
	})
	tabs.Select(2)
	assert.Equal(t, 2, fired)
	assert.Equal(t, 2, tabs.Selected)
}

func TestTabs_SelectOutOfRange_NoOp(t *testing.T) {
	tabsTestSetup(t)
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{
		Labels: []string{"A", "B"},
	})
	tabs.Select(99)
	assert.Equal(t, 0, tabs.Selected)
}

func TestTabs_SelectSame_DoesNotRefire(t *testing.T) {
	tabsTestSetup(t)
	count := 0
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{
		Labels:   []string{"A", "B"},
		Selected: 0,
		OnSelect: func(idx int) { count++ },
	})
	tabs.Select(0)
	assert.Equal(t, 0, count, "selecting the already-active tab must not fire")
}

func TestTabs_SelectNextWraps(t *testing.T) {
	tabsTestSetup(t)
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{
		Labels:   []string{"A", "B", "C"},
		Selected: 2,
	})
	tabs.SelectNext()
	assert.Equal(t, 0, tabs.Selected)
}

func TestTabs_SelectPrevWraps(t *testing.T) {
	tabsTestSetup(t)
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{
		Labels:   []string{"A", "B", "C"},
		Selected: 0,
	})
	tabs.SelectPrev()
	assert.Equal(t, 2, tabs.Selected)
}

func TestTabs_EmptyLabels_DrawNoPanic(t *testing.T) {
	tabsTestSetup(t)
	tabs := widgets.NewTabs(0, 0, 240, 22, widgets.TabsOptions{Labels: nil})
	root := pgui.New()
	root.Attach(tabs.Element)
	assert.NotPanics(t, func() { root.Draw() })
}
