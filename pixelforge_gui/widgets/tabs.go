package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// TabsOptions configures a Tabs widget.
type TabsOptions struct {
	Labels       []string
	Selected     int
	BgColor      pixelforge.Color
	ActiveBg     pixelforge.Color
	BorderColor  pixelforge.Color
	AccentColor  pixelforge.Color
	TextColor    pixelforge.Color
	TabWidth     int
	OnSelect     func(idx int)
}

// Tabs is a horizontal tab strip. Each tab is a fixed-width clickable
// region; the active tab gets a 2-pixel accent stripe on its bottom edge.
type Tabs struct {
	*pgui.Element

	Labels      []string
	Selected    int
	BgColor     pixelforge.Color
	ActiveBg    pixelforge.Color
	BorderColor pixelforge.Color
	AccentColor pixelforge.Color
	TextColor   pixelforge.Color
	TabWidth    int

	OnSelect func(idx int)
}

// NewTabs constructs a Tabs widget rooted at (x, y, w, h).
func NewTabs(x, y, w, h int, opts TabsOptions) *Tabs {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.ActiveBg == 0 {
		opts.ActiveBg = 5
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.AccentColor == 0 {
		opts.AccentColor = 12
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	if opts.TabWidth == 0 {
		opts.TabWidth = 80
	}
	t := &Tabs{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Labels:      append([]string(nil), opts.Labels...),
		Selected:    opts.Selected,
		BgColor:     opts.BgColor,
		ActiveBg:    opts.ActiveBg,
		BorderColor: opts.BorderColor,
		AccentColor: opts.AccentColor,
		TextColor:   opts.TextColor,
		TabWidth:    opts.TabWidth,
		OnSelect:    opts.OnSelect,
	}
	t.Element.OnDraw = func(ev pgui.DrawEvent) {
		t.draw()
	}
	t.Element.OnTap = func(ev pgui.Event) {
		t.handleTap()
	}
	return t
}

// Select moves the active tab to idx and fires OnSelect.
func (t *Tabs) Select(idx int) {
	if idx < 0 || idx >= len(t.Labels) {
		return
	}
	if t.Selected == idx {
		return
	}
	t.Selected = idx
	if t.OnSelect != nil {
		t.OnSelect(idx)
	}
}

// SelectNext moves selection forward by one (wraps).
func (t *Tabs) SelectNext() {
	if len(t.Labels) == 0 {
		return
	}
	t.Select((t.Selected + 1) % len(t.Labels))
}

// SelectPrev moves selection backward by one (wraps).
func (t *Tabs) SelectPrev() {
	if len(t.Labels) == 0 {
		return
	}
	t.Select((t.Selected - 1 + len(t.Labels)) % len(t.Labels))
}

func (t *Tabs) handleTap() {
	// Pointer is in element-local coordinates already. Compute the
	// hovered tab index from the mouse position; we conservatively use
	// the pixelforge_gui internal accounting by checking the pointer
	// against tab rects below.
	//
	// We mirror pigui.Element.Update's mousePosition math.
	// The pgui Event doesn't expose pointer coordinates directly, so we
	// query via the pimouse package the same way Element does.
	t.dispatchClick()
}

func (t *Tabs) dispatchClick() {
	// Compute the local mouse position. pgui.Element shifts the camera
	// before invoking the callback; we reconstruct the local x by
	// subtracting the element's screen-space origin.
	mx, _ := pguiPointerLocal(t.Element)
	if mx < 0 {
		return
	}
	idx := mx / t.TabWidth
	if idx >= 0 && idx < len(t.Labels) {
		t.Select(idx)
	}
}

func (t *Tabs) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(t.BgColor)
	pixelforge.RectFill(0, 0, t.W-1, t.H-1)

	font := pgui.DefaultFont()
	for i, label := range t.Labels {
		tabX := i * t.TabWidth
		if tabX >= t.W {
			break
		}
		tabW := t.TabWidth
		if tabX+tabW > t.W {
			tabW = t.W - tabX
		}

		if i == t.Selected {
			pixelforge.SetColor(t.ActiveBg)
			pixelforge.RectFill(tabX, 0, tabX+tabW-1, t.H-1)
			// Accent stripe at the bottom.
			pixelforge.SetColor(t.AccentColor)
			pixelforge.RectFill(tabX, t.H-2, tabX+tabW-1, t.H-1)
		}

		pixelforge.SetColor(t.TextColor)
		textW, textH := font.Measure(label)
		_, _ = font.Print(label,
			tabX+(tabW-textW)/2,
			(t.H-textH)/2)
	}
}
