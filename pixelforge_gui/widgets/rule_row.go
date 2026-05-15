package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// RuleRowOptions configures a RuleRow widget.
type RuleRowOptions struct {
	Indent      int
	Conditions  []string
	Actions     []string
	BgColor     pixelforge.Color
	HoverColor  pixelforge.Color
	TextColor   pixelforge.Color
	BorderColor pixelforge.Color
	OnSelect    func(col, idx int)
	OnAdd       func(col int)
}

// RuleRow is the visual representation of one EventSheetRule: two
// columns (conditions on the left, actions on the right) separated
// by a divider, with a "+" affordance at the end of each column.
// Indent shifts the row right by `Indent * 16` px to render nested
// rules.
//
// Conditions and Actions are passed as pre-rendered strings (the
// host formats them for the rule's catalog Kinds). OnSelect fires
// with (col, idx) when the user clicks a condition or action;
// OnAdd(col) fires when the user clicks the "+" at the end of a column.
type RuleRow struct {
	*pgui.Element

	Indent      int
	Conditions  []string
	Actions     []string
	BgColor     pixelforge.Color
	HoverColor  pixelforge.Color
	TextColor   pixelforge.Color
	BorderColor pixelforge.Color

	OnSelect func(col, idx int)
	OnAdd    func(col int)
}

// NewRuleRow constructs a RuleRow rooted at (x, y, w, h).
func NewRuleRow(x, y, w, h int, opts RuleRowOptions) *RuleRow {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.HoverColor == 0 {
		opts.HoverColor = 12
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	r := &RuleRow{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Indent:      opts.Indent,
		Conditions:  append([]string(nil), opts.Conditions...),
		Actions:     append([]string(nil), opts.Actions...),
		BgColor:     opts.BgColor,
		HoverColor:  opts.HoverColor,
		TextColor:   opts.TextColor,
		BorderColor: opts.BorderColor,
		OnSelect:    opts.OnSelect,
		OnAdd:       opts.OnAdd,
	}
	r.Element.OnDraw = func(_ pgui.DrawEvent) { r.draw() }
	r.Element.OnTap = func(_ pgui.Event) { r.dispatchTap() }
	return r
}

// HitTest returns the (col, idx) clicked at element-local (mx, my),
// or (-1, -1) if no condition/action hit. col=0 conditions, col=1
// actions. Pass idx == len(items) when the click landed on the "+"
// affordance of that column.
func (r *RuleRow) HitTest(mx, my int) (col, idx int) {
	indentPx := r.Indent * 16
	contentX := indentPx
	if mx < contentX || my < 0 || my >= r.H {
		return -1, -1
	}
	colWidth := (r.W - indentPx) / 2
	rightX := contentX + colWidth

	lineH := 9
	if mx < rightX {
		// Left column (conditions).
		row := my / lineH
		if row < len(r.Conditions) {
			return 0, row
		}
		if row == len(r.Conditions) {
			return 0, len(r.Conditions) // "+"
		}
		return -1, -1
	}
	// Right column (actions).
	row := my / lineH
	if row < len(r.Actions) {
		return 1, row
	}
	if row == len(r.Actions) {
		return 1, len(r.Actions) // "+"
	}
	return -1, -1
}

func (r *RuleRow) dispatchTap() {
	mx, my := pguiPointerLocal(r.Element)
	col, idx := r.HitTest(mx, my)
	if col < 0 {
		return
	}
	// "+" sentinel.
	var nItems int
	if col == 0 {
		nItems = len(r.Conditions)
	} else {
		nItems = len(r.Actions)
	}
	if idx == nItems {
		if r.OnAdd != nil {
			r.OnAdd(col)
		}
		return
	}
	if r.OnSelect != nil {
		r.OnSelect(col, idx)
	}
}

func (r *RuleRow) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(r.BgColor)
	pixelforge.RectFill(0, 0, r.W-1, r.H-1)

	indentPx := r.Indent * 16
	contentX := indentPx
	colWidth := (r.W - indentPx) / 2

	font := pgui.DefaultFont()
	pixelforge.SetColor(r.TextColor)

	// Left column: conditions stacked.
	for i, c := range r.Conditions {
		_, _ = font.Print(truncate(c, colWidth-4, font), contentX+2, i*9)
	}
	// "+" affordance.
	pixelforge.SetColor(r.BorderColor)
	_, _ = font.Print("+", contentX+2, len(r.Conditions)*9)

	// Divider.
	pixelforge.SetColor(r.BorderColor)
	pixelforge.RectFill(contentX+colWidth-1, 0, contentX+colWidth, r.H-1)

	// Right column: actions stacked.
	rightX := contentX + colWidth + 2
	pixelforge.SetColor(r.TextColor)
	for i, a := range r.Actions {
		_, _ = font.Print(truncate(a, colWidth-4, font), rightX, i*9)
	}
	pixelforge.SetColor(r.BorderColor)
	_, _ = font.Print("+", rightX, len(r.Actions)*9)
}

// truncate clips s so its rendered width fits within maxPx, appending
// an ellipsis if it overflows. v1 estimates by glyph count via the
// font's average advance.
func truncate(s string, maxPx int, font pgui.Font) string {
	if s == "" {
		return s
	}
	w, _ := font.Measure(s)
	if w <= maxPx {
		return s
	}
	// Binary-search-style shorten until it fits.
	out := s
	for len(out) > 0 {
		out = out[:len(out)-1]
		ww, _ := font.Measure(out + "…")
		if ww <= maxPx {
			return out + "…"
		}
	}
	return "…"
}
