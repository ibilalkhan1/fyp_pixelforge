// Package widgets contains higher-level GUI widgets composed from
// pixelforge_gui.Element. These widgets are usable by the editor and by
// user games: every widget draws via engine primitives only, never
// reaching out to ebitenutil or native Ebitengine drawing.
package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// PanelOptions configures a Panel. Zero values mean: no title, slot 1
// background, slot 6 border, no padding.
type PanelOptions struct {
	Title       string
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	TitleColor  pixelforge.Color
	TitleHeight int // pixels; defaults to 0 (no title bar) when Title is "".
}

// Panel is a background + optional border + optional title strip. It is
// the canvas-resident equivalent of the M0-M2 native panel chrome.
type Panel struct {
	*pgui.Element

	Title       string
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	TitleColor  pixelforge.Color
	TitleHeight int

	// Body is the inner element children should be attached to so that
	// the title strip does not overlap them. When the panel has no
	// title, Body == Element.
	Body *pgui.Element
}

// NewPanel constructs a Panel rooted at (x, y, w, h).
//
// When opts.Title is non-empty, a title strip is drawn at the top with
// TitleHeight pixels reserved. Children should be attached to Panel.Body
// so they appear below the title strip.
func NewPanel(x, y, w, h int, opts PanelOptions) *Panel {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.TitleColor == 0 {
		opts.TitleColor = 7
	}
	titleH := opts.TitleHeight
	if opts.Title != "" && titleH == 0 {
		// Default title strip is just tall enough for one line of the
		// 4x8 cofont plus 4px of vertical padding (2 top, 2 bottom).
		titleH = 12
	}
	p := &Panel{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Title:       opts.Title,
		BgColor:     opts.BgColor,
		BorderColor: opts.BorderColor,
		TitleColor:  opts.TitleColor,
		TitleHeight: titleH,
	}
	p.Element.OnDraw = func(ev pgui.DrawEvent) {
		p.draw()
	}
	if titleH > 0 {
		// Body is a child element positioned just below the title strip.
		body := pgui.Attach(p.Element, 0, titleH, w, h-titleH)
		p.Body = body
	} else {
		p.Body = p.Element
	}
	return p
}

func (p *Panel) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Background fill across the full panel area, then border outline.
	pixelforge.SetColor(p.BgColor)
	pixelforge.RectFill(0, 0, p.W-1, p.H-1)

	if p.TitleHeight > 0 {
		// Title strip background using a slightly different colour
		// derived from the border colour (palette slot 0 by convention).
		pixelforge.SetColor(p.BorderColor)
		pixelforge.RectFill(0, 0, p.W-1, p.TitleHeight-1)
		// Title text. The Print call honours the current draw colour
		// via the colortable-aware path in pifont.Sheet.Print.
		pixelforge.SetColor(p.TitleColor)
		font := pgui.DefaultFont()
		_, _ = font.Print(p.Title, 4, 2)
	}

	pixelforge.SetColor(p.BorderColor)
	pixelforge.Rect(0, 0, p.W-1, p.H-1)
}
