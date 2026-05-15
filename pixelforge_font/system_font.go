package pixelforge_font

import (
	"image"
	"image/color"

	"golang.org/x/image/font/basicfont"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

// NewSystemSheet builds a pifont.Sheet backed by
// golang.org/x/image/font/basicfont's Face7x13. The result is a 7×13
// per-glyph sheet — bigger than cofont's 4×8 so the editor reads more
// comfortably even at scale 1×.
//
// Each ASCII printable (and U+FFFD as a tofu fallback) is rasterised
// into a pixelforge.Canvas; the canvases share the sheet's color
// table via FgColor / BgColor.
func NewSystemSheet() Sheet {
	face := basicfont.Face7x13
	sheet := Sheet{
		Height:  face.Height,
		FgColor: 1,
		BgColor: 0,
		Chars:   map[rune]pixelforge.Sprite{},
	}
	for _, r := range face.Ranges {
		for ch := r.Low; ch < r.High; ch++ {
			glyph := rasteriseGlyph(face, ch)
			sheet.Chars[ch] = glyph
		}
	}
	return sheet
}

// rasteriseGlyph extracts the glyph for rune ch out of face's Mask
// into a fresh pixelforge.Sprite. Pixels darker than 128 in the mask
// alpha channel become FgColor; the rest stay BgColor.
func rasteriseGlyph(face *basicfont.Face, ch rune) pixelforge.Sprite {
	w := face.Width
	h := face.Height
	canvas := pixelforge.NewCanvas(w, h)
	// Find ch in Ranges.
	offset := -1
	for _, r := range face.Ranges {
		if ch >= r.Low && ch < r.High {
			offset = r.Offset + int(ch-r.Low)
			break
		}
	}
	if offset < 0 {
		return pixelforge.Sprite{Area: pixelforge.IntArea{W: w, H: h}, Source: canvas}
	}
	rect := image.Rect(0, offset*h, face.Width, (offset+1)*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mx := rect.Min.X + x
			my := rect.Min.Y + y
			c := face.Mask.At(mx, my)
			_, _, _, a := c.RGBA()
			if a >= 0x8000 {
				canvas.Set(x, y, 1)
			}
		}
	}
	_ = color.Black
	return pixelforge.Sprite{Area: canvas.EntireArea(), Source: canvas}
}
