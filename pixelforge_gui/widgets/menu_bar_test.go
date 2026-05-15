package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMenuBar_StartsClosed(t *testing.T) {
	mb := NewMenuBar(menuFixture(), MenuBarOptions{})
	assert.False(t, mb.IsOpen())
	assert.Equal(t, -1, mb.OpenIndex())
}

func TestMenuBar_OpenAndClose(t *testing.T) {
	mb := NewMenuBar(menuFixture(), MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	mb.Open(0)
	assert.True(t, mb.IsOpen())
	mb.Close()
	assert.False(t, mb.IsOpen())
}

func TestMenuBar_ClickOnLabelToggles(t *testing.T) {
	mb := NewMenuBar(menuFixture(), MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	rects := mb.LabelRects()
	consumed := mb.HandleClick(rects[0].X+5, rects[0].Y+5)
	assert.True(t, consumed)
	assert.Equal(t, 0, mb.OpenIndex())
	mb.HandleClick(rects[0].X+5, rects[0].Y+5)
	assert.Equal(t, -1, mb.OpenIndex(), "second click on the same label closes it")
}

func TestMenuBar_ClickOnDropdownItemFiresCallback(t *testing.T) {
	fired := false
	menus := []MenuDef{
		{Label: "File", Items: []MenuItem{
			{Label: "Save", OnSelect: func() { fired = true }},
		}},
	}
	mb := NewMenuBar(menus, MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	mb.Open(0)
	dr := mb.DropdownRect(0)
	mb.HandleClick(dr.X+5, dr.Y+5)
	assert.True(t, fired)
	assert.False(t, mb.IsOpen())
}

func TestMenuBar_ClickOutsideClosesDropdown(t *testing.T) {
	mb := NewMenuBar(menuFixture(), MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	mb.Open(0)
	mb.HandleClick(300, 300)
	assert.False(t, mb.IsOpen())
}

func TestMenuBar_HandleEscape(t *testing.T) {
	mb := NewMenuBar(menuFixture(), MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	assert.False(t, mb.HandleEscape())
	mb.Open(0)
	assert.True(t, mb.HandleEscape())
	assert.False(t, mb.IsOpen())
}

func TestMenuBar_ZeroItemsRendersWithoutPanic(t *testing.T) {
	mb := NewMenuBar([]MenuDef{}, MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	// Should not panic when there are no menus.
	mb.Draw()
}

func TestMenuBar_DisabledItemDoesNotFire(t *testing.T) {
	fired := false
	menus := []MenuDef{
		{Label: "File", Items: []MenuItem{
			{Label: "Save", Disabled: true, OnSelect: func() { fired = true }},
		}},
	}
	mb := NewMenuBar(menus, MenuBarOptions{})
	mb.SetBounds(0, 0, 400, 22)
	mb.Open(0)
	dr := mb.DropdownRect(0)
	mb.HandleClick(dr.X+5, dr.Y+5)
	assert.False(t, fired)
}

func menuFixture() []MenuDef {
	return []MenuDef{
		{Label: "File", Items: []MenuItem{
			{Label: "Open", Shortcut: "Ctrl+O"},
			{Separator: true},
			{Label: "Quit", Shortcut: "Ctrl+Q"},
		}},
		{Label: "Edit", Items: []MenuItem{
			{Label: "Undo", Shortcut: "Ctrl+Z"},
		}},
	}
}
