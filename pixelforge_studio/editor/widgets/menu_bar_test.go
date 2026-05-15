package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// labelRects lay out left-to-right with each label sized by its text width.
func TestMenuBar_LabelRects(t *testing.T) {
	mb := NewMenuBar([]MenuDef{
		{Label: "File"},
		{Label: "Edit"},
	})
	area := Rect{X: 0, Y: 0, W: 200, H: MenuBarHeight}
	rects := mb.labelRects(area)
	require := assert.New(t)
	require.Len(rects, 2)
	require.Equal(8, rects[0].X)
	require.Equal(MenuBarHeight, rects[0].H)
	require.Greater(rects[1].X, rects[0].X+rects[0].W-1, "second label is to the right of the first")
}

// Reset closes any open dropdown.
func TestMenuBar_ResetClosesDropdown(t *testing.T) {
	mb := NewMenuBar([]MenuDef{{Label: "File", Items: []MenuItem{{Label: "New"}}}})
	mb.openIdx = 0
	assert.True(t, mb.IsOpen())
	mb.Reset()
	assert.False(t, mb.IsOpen())
}

// dropdownRect is anchored to the opened label and sized to the widest item.
func TestMenuBar_DropdownRectAnchoredToLabel(t *testing.T) {
	mb := NewMenuBar([]MenuDef{
		{Label: "File", Items: []MenuItem{
			{Label: "New"},
			{Label: "Save As..."},
		}},
	})
	area := Rect{X: 0, Y: 0, W: 200, H: MenuBarHeight}
	dr := mb.dropdownRect(area, 0)
	assert.Equal(t, mb.labelRects(area)[0].X, dr.X)
	assert.Equal(t, area.Y+area.H, dr.Y)
	assert.Equal(t, 2*menuItemHeight, dr.H)
}

// dropdownRect with an invalid index returns the zero rect.
func TestMenuBar_DropdownRectOutOfRange(t *testing.T) {
	mb := NewMenuBar([]MenuDef{{Label: "File"}})
	assert.Equal(t, Rect{}, mb.dropdownRect(Rect{}, -1))
	assert.Equal(t, Rect{}, mb.dropdownRect(Rect{}, 99))
}
