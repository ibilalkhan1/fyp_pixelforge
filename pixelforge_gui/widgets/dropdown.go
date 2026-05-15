package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// DropdownOptions configures a Dropdown widget.
type DropdownOptions struct {
	Options     []string
	Selected    string
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	FgColor     pixelforge.Color
	HoverColor  pixelforge.Color
	OnSelect    func(value string)
}

// Dropdown is a single-select combo: a selector button that, when
// clicked, shows a popover listing the options. Click-outside or Esc
// closes without selecting; clicking an option fires OnSelect and
// closes.
type Dropdown struct {
	*pgui.Element

	Options     []string
	Selected    string
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	FgColor     pixelforge.Color
	HoverColor  pixelforge.Color
	OnSelect    func(value string)

	open       bool
	popoverH   int // height per option row
	upward     bool // anchor list upward when near canvas bottom
	containerH int  // for upward-flip calculations
}

// NewDropdown constructs a Dropdown rooted at (x, y, w, h). containerH
// is the maximum Y inside the parent canvas at which the popover may
// extend; pass 0 to disable upward flipping.
func NewDropdown(x, y, w, h, containerH int, opts DropdownOptions) *Dropdown {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.FgColor == 0 {
		opts.FgColor = 7
	}
	if opts.HoverColor == 0 {
		opts.HoverColor = 12
	}
	d := &Dropdown{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Options:     append([]string(nil), opts.Options...),
		Selected:    opts.Selected,
		BgColor:     opts.BgColor,
		BorderColor: opts.BorderColor,
		FgColor:     opts.FgColor,
		HoverColor:  opts.HoverColor,
		OnSelect:    opts.OnSelect,
		popoverH:    h,
		containerH:  containerH,
	}
	d.Element.OnDraw = func(ev pgui.DrawEvent) {
		d.draw()
	}
	d.Element.OnTap = func(ev pgui.Event) {
		d.toggle()
	}
	return d
}

// Open opens the popover.
func (d *Dropdown) Open() { d.open = true; d.recomputeFlip() }

// Close closes the popover without selecting.
func (d *Dropdown) Close() { d.open = false }

// IsOpen reports whether the popover is currently visible.
func (d *Dropdown) IsOpen() bool { return d.open }

// SetOptions replaces the dropdown's option list. The selected value
// is retained when present in the new options, otherwise cleared.
func (d *Dropdown) SetOptions(opts []string) {
	d.Options = append([]string(nil), opts...)
	if d.Selected != "" {
		found := false
		for _, o := range d.Options {
			if o == d.Selected {
				found = true
				break
			}
		}
		if !found {
			d.Selected = ""
		}
	}
	if d.open {
		d.recomputeFlip()
	}
}

// SelectByIndex picks the option at idx, fires OnSelect, and closes.
func (d *Dropdown) SelectByIndex(idx int) {
	if idx < 0 || idx >= len(d.Options) {
		return
	}
	d.Selected = d.Options[idx]
	if d.OnSelect != nil {
		d.OnSelect(d.Selected)
	}
	d.open = false
}

// HandlePointer routes a click at global (mx, my) — if inside an option
// row, that option is selected. Returns true when the click was
// consumed (either selected or click-outside-close).
func (d *Dropdown) HandlePointer(mx, my int) bool {
	if !d.open {
		return false
	}
	listY, listH := d.popoverBounds()
	listX0 := d.X
	listX1 := d.X + d.W
	if mx >= listX0 && mx < listX1 && my >= listY && my < listY+listH {
		idx := (my - listY) / d.popoverH
		d.SelectByIndex(idx)
		return true
	}
	// Click outside closes the dropdown.
	d.open = false
	return true
}

// HandleEscape dismisses the popover. Returns true if it was open.
func (d *Dropdown) HandleEscape() bool {
	if !d.open {
		return false
	}
	d.open = false
	return true
}

func (d *Dropdown) toggle() {
	if d.open {
		d.open = false
		return
	}
	d.open = true
	d.recomputeFlip()
}

// recomputeFlip decides whether to anchor the option list above or
// below the selector based on remaining vertical space.
func (d *Dropdown) recomputeFlip() {
	if d.containerH <= 0 {
		d.upward = false
		return
	}
	listH := len(d.Options) * d.popoverH
	bottomSpace := d.containerH - (d.Y + d.H)
	if listH > bottomSpace && d.Y >= listH {
		d.upward = true
		return
	}
	d.upward = false
}

// popoverBounds returns the global Y origin and height of the option
// list when open.
func (d *Dropdown) popoverBounds() (y, h int) {
	h = len(d.Options) * d.popoverH
	if d.upward {
		y = d.Y - h
	} else {
		y = d.Y + d.H
	}
	return y, h
}

func (d *Dropdown) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Selector chrome.
	pixelforge.SetColor(d.BgColor)
	pixelforge.RectFill(0, 0, d.W-1, d.H-1)
	pixelforge.SetColor(d.BorderColor)
	pixelforge.Rect(0, 0, d.W-1, d.H-1)
	pixelforge.SetColor(d.FgColor)
	font := pgui.DefaultFont()
	textY := (d.H - font.LineHeight()) / 2
	_, _ = font.Print(d.Selected, 4, textY)

	if !d.open {
		return
	}
	// Popover. Coordinates are element-local: offset rows by
	// popover-relative-to-selector.
	_, listH := d.popoverBounds()
	var startY int
	if d.upward {
		startY = -listH
	} else {
		startY = d.H
	}
	pixelforge.SetColor(d.BgColor)
	pixelforge.RectFill(0, startY, d.W-1, startY+listH-1)
	pixelforge.SetColor(d.BorderColor)
	pixelforge.Rect(0, startY, d.W-1, startY+listH-1)

	for i, opt := range d.Options {
		rowY := startY + i*d.popoverH
		pixelforge.SetColor(d.FgColor)
		_, _ = font.Print(opt, 4, rowY+(d.popoverH-font.LineHeight())/2)
	}
}
