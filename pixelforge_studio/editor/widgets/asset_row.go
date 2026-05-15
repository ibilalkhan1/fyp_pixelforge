package widgets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// AssetRow renders one row of the editor's asset browser. The editor's
// AssetBrowser calls into this from its Draw loop; the renderer is
// owned by the widgets package so future themes can swap it.
type AssetRow struct {
	Rect      Rect
	Title     string
	Detail    string
	Selected  bool
	Iconish   bool // true → render a small accent square in place of a thumbnail
	IconColor color.RGBA
}

// Draw paints the row.
func (a AssetRow) Draw(dst *ebiten.Image) {
	bg := colAssetRowBg
	if a.Selected {
		bg = colAssetRowSelectedBg
	}
	fillRect(dst, a.Rect, bg)
	icon := Rect{X: a.Rect.X + 4, Y: a.Rect.Y + 4, W: a.Rect.H - 8, H: a.Rect.H - 8}
	c := a.IconColor
	if c == (color.RGBA{}) {
		if a.Iconish {
			c = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
		} else {
			c = color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff}
		}
	}
	fillRect(dst, icon, c)
	printAt(dst, a.Title, a.Rect.X+icon.W+12, a.Rect.Y+4)
	if a.Detail != "" {
		printAt(dst, a.Detail, a.Rect.X+icon.W+12, a.Rect.Y+18)
	}
}

var (
	colAssetRowBg         = color.RGBA{R: 0x1a, G: 0x1a, B: 0x22, A: 0xff}
	colAssetRowSelectedBg = color.RGBA{R: 0x2a, G: 0x2a, B: 0x40, A: 0xff}
)
