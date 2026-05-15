package widgets

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// ColorPickerWidget renders a 64-swatch palette grid; clicking a
// swatch sets the field to the swatch's palette index.
//
// Context.PaletteColors must hold 64 "#RRGGBB" strings for the preview
// to match the live palette. With an empty context, swatches render in
// a neutral grey.
type ColorPickerWidget struct {
	F   pfcomponent.FieldMetadata
	Ctx *Context
}

func (w *ColorPickerWidget) Field() pfcomponent.FieldMetadata { return w.F }

const (
	paletteColumns = 16
	paletteRows    = 4
	swatchSize     = 12
	swatchGap      = 1
)

func (w *ColorPickerWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	idx, _ := asFloat(value)

	fillRect(dst, area, colWidgetBg)

	for i := 0; i < paletteColumns*paletteRows; i++ {
		col := i % paletteColumns
		row := i / paletteColumns
		x := area.X + 4 + col*(swatchSize+swatchGap)
		y := area.Y + 12 + row*(swatchSize+swatchGap)
		c := paletteSwatchColor(w.Ctx, i)
		fillRect(dst, Rect{X: x, Y: y, W: swatchSize, H: swatchSize}, c)
		if int(idx) == i {
			strokeRect(dst, Rect{X: x - 1, Y: y - 1, W: swatchSize + 2, H: swatchSize + 2}, colWidgetText)
		}
	}
	printAt(dst, fmt.Sprintf("%s: %d", w.F.Name, int(idx)), area.X+4, area.Y)
}

func (w *ColorPickerWidget) Update(area Rect, value any, mx, my int, pressed bool) *EditEvent {
	if !isClickJustPressed() {
		return nil
	}
	if !area.Contains(mx, my) {
		return nil
	}
	for i := 0; i < paletteColumns*paletteRows; i++ {
		col := i % paletteColumns
		row := i / paletteColumns
		x := area.X + 4 + col*(swatchSize+swatchGap)
		y := area.Y + 12 + row*(swatchSize+swatchGap)
		sr := Rect{X: x, Y: y, W: swatchSize, H: swatchSize}
		if sr.Contains(mx, my) {
			return &EditEvent{NewValue: i}
		}
	}
	return nil
}

// paletteSwatchColor returns the RGB to render swatch i with. Falls
// back to a deterministic grey ramp when the palette is unknown.
func paletteSwatchColor(ctx *Context, i int) color.RGBA {
	if ctx != nil && i < len(ctx.PaletteColors) {
		if c, ok := parseHexColor(ctx.PaletteColors[i]); ok {
			return c
		}
	}
	v := uint8((i * 255) / 63)
	return color.RGBA{R: v, G: v, B: v, A: 0xff}
}

// parseHexColor accepts "#RRGGBB" or "#RGB" forms.
func parseHexColor(s string) (color.RGBA, bool) {
	if len(s) == 7 && s[0] == '#' {
		r, ok1 := parseHexByte(s[1:3])
		g, ok2 := parseHexByte(s[3:5])
		b, ok3 := parseHexByte(s[5:7])
		if ok1 && ok2 && ok3 {
			return color.RGBA{R: r, G: g, B: b, A: 0xff}, true
		}
	}
	return color.RGBA{}, false
}

func parseHexByte(s string) (uint8, bool) {
	if len(s) != 2 {
		return 0, false
	}
	hi, ok := hexNibble(s[0])
	if !ok {
		return 0, false
	}
	lo, ok := hexNibble(s[1])
	if !ok {
		return 0, false
	}
	return hi<<4 | lo, true
}

func hexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}
