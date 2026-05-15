package palette

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// parseHexColor accepts #RRGGBB and rejects malformed forms.
func TestParseHexColor(t *testing.T) {
	c, ok := parseHexColor("#ff8800")
	assert.True(t, ok)
	assert.Equal(t, uint8(0xff), c.R)
	assert.Equal(t, uint8(0x88), c.G)
	assert.Equal(t, uint8(0x00), c.B)

	_, ok = parseHexColor("garbage")
	assert.False(t, ok)
	_, ok = parseHexColor("#zzzzzz")
	assert.False(t, ok)
	// v1 rejects 3-digit shorthand.
	_, ok = parseHexColor("#fff")
	assert.False(t, ok)
}

// formatHexColor matches the project's storage format and clamps.
func TestFormatHexColor(t *testing.T) {
	assert.Equal(t, "#ff8800", formatHexColor(255, 136, 0))
	assert.Equal(t, "#ff8800", formatHexColor(999, 136, -5)) // clamped
	assert.Equal(t, "#000000", formatHexColor(0, 0, 0))
}

// hitTestSwatch resolves window-space coords to a palette slot.
func TestGrid_HitTestSwatch(t *testing.T) {
	g := NewGrid()
	area := widgets.Rect{X: 0, Y: 0, W: 400, H: 400}
	// Slot 0 — top-left swatch.
	r0 := g.swatchRect(area, 0)
	mid := func(r widgets.Rect) (int, int) { return r.X + r.W/2, r.Y + r.H/2 }
	mx, my := mid(r0)
	assert.Equal(t, 0, g.HitTestSwatch(area, mx, my))

	// Slot 9 — second row, second column.
	r9 := g.swatchRect(area, 9)
	mx, my = mid(r9)
	assert.Equal(t, 9, g.HitTestSwatch(area, mx, my))

	// Miss between swatches.
	assert.Equal(t, -1, g.HitTestSwatch(area, area.X+area.W-1, area.Y))
}

// SetSelectedSlot clamps and round-trips.
func TestGrid_SelectedSlotRoundTrip(t *testing.T) {
	g := NewGrid()
	g.SetSelectedSlot(8)
	assert.Equal(t, 8, g.SelectedSlot())
	g.SetSelectedSlot(-1) // out of range → no-op
	assert.Equal(t, 8, g.SelectedSlot())
	g.SetSelectedSlot(pixelforge_project.MaxColors) // out of range → no-op
	assert.Equal(t, 8, g.SelectedSlot())
}

// rgbPicker.Confirm via the hex input writes to the project slot.
func TestRGBPicker_ConfirmHex(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	picker := &rgbPicker{}
	picker.Reset(p.Palette.Base[8])
	picker.SetHex("#ff0000")

	ok := picker.Confirm(p, 8)
	assert.True(t, ok)
	assert.Equal(t, "#ff0000", p.Palette.Base[8])
}

// rgbPicker.Confirm via channel sliders writes the same color as the
// hex equivalent.
func TestRGBPicker_ConfirmRGBChannels(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	picker := &rgbPicker{}
	picker.Reset(p.Palette.Base[8])
	picker.SetRGB(255, 0, 0)
	picker.SetHex("") // ensure hex input doesn't override channels
	ok := picker.Confirm(p, 8)
	assert.True(t, ok)
	assert.Equal(t, "#ff0000", p.Palette.Base[8])
}

// Confirm with a malformed hex (looks like one) rejects.
func TestRGBPicker_ConfirmRejectsMalformedHex(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	picker := &rgbPicker{}
	picker.Reset(p.Palette.Base[8])
	picker.SetHex("#garbage")
	ok := picker.Confirm(p, 8)
	assert.False(t, ok)
	assert.True(t, picker.hexErr)
	// Project unchanged.
	assert.Equal(t, pixelforge_project.NewProject("t").Palette.Base[8], p.Palette.Base[8])
}

// Editing slot 0 saves the RGB but does not flip transparency.
func TestRGBPicker_Slot0AllowedToEdit(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	picker := &rgbPicker{}
	picker.Reset(p.Palette.Base[0])
	picker.SetHex("#abcdef")
	ok := picker.Confirm(p, 0)
	assert.True(t, ok)
	assert.Equal(t, "#abcdef", p.Palette.Base[0])
}

// Reset populates RGB from a hex string.
func TestRGBPicker_ResetParsesHex(t *testing.T) {
	picker := &rgbPicker{}
	picker.Reset("#ff8800")
	assert.Equal(t, 255, picker.r)
	assert.Equal(t, 136, picker.g)
	assert.Equal(t, 0, picker.b)
}

// CurrentHex round-trips through formatHexColor.
func TestRGBPicker_CurrentHexRoundTrip(t *testing.T) {
	picker := &rgbPicker{}
	picker.SetRGB(255, 0, 128)
	assert.Equal(t, "#ff0080", picker.CurrentHex())
}

// Ensure unused color import (debug helper) doesn't trip the compiler.
var _ = color.RGBA{}
