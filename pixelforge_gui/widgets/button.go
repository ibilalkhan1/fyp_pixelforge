package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// ButtonOptions configures a Button. Zero-valued colours fall back to
// the M0-M2 chrome palette defaults.
type ButtonOptions struct {
	Label        string
	FgColor      pixelforge.Color
	BgColor      pixelforge.Color
	HoverBgColor pixelforge.Color
	BorderColor  pixelforge.Color
	DimFgColor   pixelforge.Color
	Disabled     bool
	OnTap        func()
}

// Button is a clickable rectangle that fires OnTap on release while
// the pointer is still inside. Pressed buttons offset their label by
// 1px down/right for classic press feedback.
type Button struct {
	*pgui.Element

	Label        string
	FgColor      pixelforge.Color
	BgColor      pixelforge.Color
	HoverBgColor pixelforge.Color
	BorderColor  pixelforge.Color
	DimFgColor   pixelforge.Color
	Disabled     bool

	OnTap func()
}

// NewButton constructs a Button rooted at (x, y, w, h).
func NewButton(x, y, w, h int, opts ButtonOptions) *Button {
	if opts.FgColor == 0 {
		opts.FgColor = 7
	}
	if opts.BgColor == 0 {
		opts.BgColor = 12
	}
	if opts.HoverBgColor == 0 {
		opts.HoverBgColor = 28
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.DimFgColor == 0 {
		opts.DimFgColor = 5
	}
	b := &Button{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Label:        opts.Label,
		FgColor:      opts.FgColor,
		BgColor:      opts.BgColor,
		HoverBgColor: opts.HoverBgColor,
		BorderColor:  opts.BorderColor,
		DimFgColor:   opts.DimFgColor,
		Disabled:     opts.Disabled,
		OnTap:        opts.OnTap,
	}
	b.Element.OnDraw = func(ev pgui.DrawEvent) {
		b.draw(ev)
	}
	b.Element.OnTap = func(ev pgui.Event) {
		if b.Disabled {
			return
		}
		if b.OnTap != nil {
			b.OnTap()
		}
	}
	return b
}

// SetLabel updates the button label.
func (b *Button) SetLabel(s string) { b.Label = s }

// SetDisabled toggles the disabled state.
func (b *Button) SetDisabled(v bool) { b.Disabled = v }

func (b *Button) draw(ev pgui.DrawEvent) {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	bg := b.BgColor
	if ev.HasPointer && !b.Disabled {
		bg = b.HoverBgColor
	}
	if b.Disabled {
		// Halve apparent contrast: use a darker bg via slot 0 when
		// callers haven't supplied a dedicated disabled colour.
		bg = b.BgColor
	}

	pixelforge.SetColor(bg)
	pixelforge.RectFill(0, 0, b.W-1, b.H-1)

	pixelforge.SetColor(b.BorderColor)
	pixelforge.Rect(0, 0, b.W-1, b.H-1)

	if b.Label == "" {
		return
	}
	fg := b.FgColor
	if b.Disabled {
		fg = b.DimFgColor
	}
	pixelforge.SetColor(fg)

	font := pgui.DefaultFont()
	textW, textH := font.Measure(b.Label)
	x := (b.W - textW) / 2
	y := (b.H - textH) / 2
	if ev.Pressed && !b.Disabled {
		// Classic press feedback: nudge label 1px down/right.
		x++
		y++
	}
	_, _ = font.Print(b.Label, x, y)
}
