package widgets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
)

func buttonTestSetup(t *testing.T) {
	t.Helper()
	pixelforge.SetScreenSize(64, 32)
	pixelforge.Cls()
}

func TestNewButton_Defaults(t *testing.T) {
	buttonTestSetup(t)
	b := widgets.NewButton(0, 0, 40, 16, widgets.ButtonOptions{Label: "OK"})
	assert.Equal(t, "OK", b.Label)
	assert.False(t, b.Disabled)
}

func TestButton_OnTapFires_OnElementTap(t *testing.T) {
	buttonTestSetup(t)
	fired := 0
	b := widgets.NewButton(0, 0, 40, 16, widgets.ButtonOptions{
		Label: "OK",
		OnTap: func() { fired++ },
	})
	// Directly drive the element's OnTap callback as the GUI loop would.
	b.Element.OnTap(pgui.Event{Element: b.Element, HasPointer: true})
	assert.Equal(t, 1, fired)
}

func TestButton_DisabledSuppressesOnTap(t *testing.T) {
	buttonTestSetup(t)
	fired := 0
	b := widgets.NewButton(0, 0, 40, 16, widgets.ButtonOptions{
		Label:    "OK",
		Disabled: true,
		OnTap:    func() { fired++ },
	})
	b.Element.OnTap(pgui.Event{Element: b.Element, HasPointer: true})
	assert.Equal(t, 0, fired)
}

func TestButton_Draw_DoesNotPanic(t *testing.T) {
	buttonTestSetup(t)
	root := pgui.New()
	b := widgets.NewButton(2, 2, 40, 16, widgets.ButtonOptions{Label: "OK"})
	root.Attach(b.Element)
	assert.NotPanics(t, func() { root.Draw() })
}
