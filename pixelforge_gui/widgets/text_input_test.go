package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func textInputTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(128, 32)
	pixelforge.Cls()
}

func TestTextInput_AppendRune(t *testing.T) {
	textInputTestSetup(t)
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{})
	ti.AppendRune('A')
	ti.AppendRune('B')
	assert.Equal(t, "AB", ti.Value())
	assert.Equal(t, 2, ti.Cursor())
}

func TestTextInput_BackspaceRemovesLastRune(t *testing.T) {
	textInputTestSetup(t)
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{Initial: "ABC"})
	ti.CursorEnd()
	ti.Backspace()
	assert.Equal(t, "AB", ti.Value())
	assert.Equal(t, 2, ti.Cursor())
}

func TestTextInput_BackspaceAtZeroIsNoOp(t *testing.T) {
	textInputTestSetup(t)
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{Initial: "AB"})
	ti.CursorHome()
	ti.Backspace()
	assert.Equal(t, "AB", ti.Value())
	assert.Equal(t, 0, ti.Cursor())
}

func TestTextInput_MoveCursor(t *testing.T) {
	textInputTestSetup(t)
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{Initial: "ABC"})
	ti.CursorEnd()
	ti.MoveCursor(-1)
	assert.Equal(t, 2, ti.Cursor())
	ti.MoveCursor(99)
	assert.Equal(t, 3, ti.Cursor(), "cursor clamps to end")
	ti.MoveCursor(-99)
	assert.Equal(t, 0, ti.Cursor(), "cursor clamps to zero")
}

func TestTextInput_MaxRunesCap(t *testing.T) {
	textInputTestSetup(t)
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{MaxRunes: 3})
	for _, r := range "ABCDE" {
		ti.AppendRune(r)
	}
	assert.Equal(t, "ABC", ti.Value())
}

func TestTextInput_OnSubmitFires(t *testing.T) {
	textInputTestSetup(t)
	captured := ""
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{
		Initial:  "READY",
		OnSubmit: func(s string) { captured = s },
	})
	ti.Submit()
	assert.Equal(t, "READY", captured)
}

func TestTextInput_OnChangeFires(t *testing.T) {
	textInputTestSetup(t)
	changes := 0
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{
		OnChange: func(s string) { changes++ },
	})
	ti.AppendRune('X')
	assert.Equal(t, 1, changes)
}

func TestTextInput_FocusRegistration(t *testing.T) {
	textInputTestSetup(t)
	fm := pgui.NewFocusManager()
	ti1 := widgets.NewTextInput(0, 0, 60, 12, fm, widgets.TextInputOptions{})
	ti2 := widgets.NewTextInput(0, 20, 60, 12, fm, widgets.TextInputOptions{})

	assert.Len(t, fm.Order(), 2)
	fm.Focus(ti1.Element)
	assert.True(t, ti1.Focused())
	assert.False(t, ti2.Focused())
	fm.Tab(true)
	assert.True(t, ti2.Focused())
	assert.False(t, ti1.Focused())
}

func TestTextInput_Draw_DoesNotPanic(t *testing.T) {
	textInputTestSetup(t)
	root := pgui.New()
	ti := widgets.NewTextInput(0, 0, 60, 12, nil, widgets.TextInputOptions{Initial: "ABC"})
	root.Attach(ti.Element)
	assert.NotPanics(t, func() { root.Draw() })
}
