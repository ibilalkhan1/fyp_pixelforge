package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func modalTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(128, 128)
	pixelforge.Cls()
}

func TestModal_VisibleAfterShow(t *testing.T) {
	modalTestSetup(t)
	m := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{
		BodyW: 80, BodyH: 40,
	})
	assert.False(t, m.Visible())
	m.Show()
	assert.True(t, m.Visible())
}

func TestModal_DismissFiresCallbackOnce(t *testing.T) {
	modalTestSetup(t)
	count := 0
	m := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{
		BodyW: 80, BodyH: 40,
		OnDismiss: func() { count++ },
	})
	m.Show()
	m.Dismiss()
	m.Dismiss() // already dismissed
	assert.Equal(t, 1, count)
	assert.False(t, m.Visible())
}

func TestModal_HandleEscape(t *testing.T) {
	modalTestSetup(t)
	m := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{
		BodyW: 80, BodyH: 40,
	})
	assert.False(t, m.HandleEscape(), "Esc on a hidden modal is a no-op")
	m.Show()
	assert.True(t, m.HandleEscape())
	assert.False(t, m.Visible())
}

func TestModal_BodyExposed(t *testing.T) {
	modalTestSetup(t)
	m := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{
		BodyW: 80, BodyH: 40,
	})
	assert.NotNil(t, m.Body)
	assert.Equal(t, 80, m.Body.W)
	assert.Equal(t, 40, m.Body.H)
}

func TestModal_HiddenDrawIsNoOp(t *testing.T) {
	modalTestSetup(t)
	root := pgui.New()
	m := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{
		BodyW: 60, BodyH: 30,
	})
	root.Attach(m.Element)
	assert.NotPanics(t, func() { root.Draw() })
}

func TestModalStack_EscapeRoutesToTop(t *testing.T) {
	modalTestSetup(t)
	stack := widgets.NewModalStack()
	bottom := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{BodyW: 40, BodyH: 20})
	top := widgets.NewModal(0, 0, 128, 128, widgets.ModalOptions{BodyW: 40, BodyH: 20})

	stack.Push(bottom)
	stack.Push(top)

	assert.Same(t, top, stack.Top())
	assert.True(t, stack.HandleEscape())
	assert.False(t, top.Visible())
	assert.True(t, bottom.Visible(), "lower modal must remain visible after Esc dismisses the top")
	stack.Cleanup()
	assert.Same(t, bottom, stack.Top())
}
