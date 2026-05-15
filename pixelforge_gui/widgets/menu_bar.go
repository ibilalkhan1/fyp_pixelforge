package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// MenuItem is a single dropdown entry.
type MenuItem struct {
	Label     string
	Shortcut  string
	OnSelect  func()
	Separator bool
	Disabled  bool
}

// MenuDef is one top-level menu.
type MenuDef struct {
	Label string
	Items []MenuItem
}

// MenuBarOptions configures a MenuBar's colors. Zero values fall back
// to sensible defaults.
type MenuBarOptions struct {
	BgColor       pixelforge.Color
	OpenColor     pixelforge.Color
	DropdownColor pixelforge.Color
	BorderColor   pixelforge.Color
	TextColor     pixelforge.Color
}

// MenuBar is the canvas-resident menu strip. It mirrors the native
// (overlay) MenuBar's API + behaviour 1:1 so editor code can swap
// between them.
type MenuBar struct {
	Menus []MenuDef

	X, Y, W, H int

	openIdx int

	bgColor       pixelforge.Color
	openColor     pixelforge.Color
	dropdownColor pixelforge.Color
	borderColor   pixelforge.Color
	textColor     pixelforge.Color
}

// MenuBarHeight matches the native bar's height so existing chrome
// layouts don't shift when the canvas bar replaces it.
const MenuBarHeight = 22

// MenuItemHeight is the per-item dropdown row height.
const MenuItemHeight = 18

const (
	menuLabelPadding = 12
	menuGlyphW       = 4 // cofont 4-px-wide glyphs
)

// NewMenuBar returns a closed menu bar with the supplied definitions.
func NewMenuBar(menus []MenuDef, opts MenuBarOptions) *MenuBar {
	if opts.BgColor == 0 {
		opts.BgColor = 5
	}
	if opts.OpenColor == 0 {
		opts.OpenColor = 12
	}
	if opts.DropdownColor == 0 {
		opts.DropdownColor = 2
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	return &MenuBar{
		Menus:         menus,
		openIdx:       -1,
		bgColor:       opts.BgColor,
		openColor:     opts.OpenColor,
		dropdownColor: opts.DropdownColor,
		borderColor:   opts.BorderColor,
		textColor:     opts.TextColor,
	}
}

// IsOpen reports whether any dropdown is currently open.
func (m *MenuBar) IsOpen() bool { return m.openIdx >= 0 }

// OpenIndex returns the index of the open menu, or -1 when closed.
func (m *MenuBar) OpenIndex() int { return m.openIdx }

// Close dismisses any open dropdown.
func (m *MenuBar) Close() { m.openIdx = -1 }

// Open opens the dropdown for menu i.
func (m *MenuBar) Open(i int) {
	if i < 0 || i >= len(m.Menus) {
		return
	}
	m.openIdx = i
}

// SetBounds positions the menu bar.
func (m *MenuBar) SetBounds(x, y, w, h int) {
	m.X, m.Y, m.W, m.H = x, y, w, h
}

// LabelRects returns one (x, w) pair per top-level menu in workspace-
// local coords. The y / h are the bar's bounds.
func (m *MenuBar) LabelRects() []IntRect {
	rects := make([]IntRect, len(m.Menus))
	x := m.X + 8
	for i, md := range m.Menus {
		w := len(md.Label)*menuGlyphW + menuLabelPadding
		rects[i] = IntRect{X: x, Y: m.Y, W: w, H: m.H}
		x += w
	}
	return rects
}

// IntRect is a plain int-coords rectangle. Mirrors widgets.Rect so the
// canvas widget bank does not depend on the studio's widgets package.
type IntRect struct {
	X, Y, W, H int
}

// Contains reports whether (px, py) lies inside r.
func (r IntRect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.W && py >= r.Y && py < r.Y+r.H
}

// DropdownRect computes the dropdown panel rect for menu i.
func (m *MenuBar) DropdownRect(i int) IntRect {
	rects := m.LabelRects()
	if i < 0 || i >= len(rects) {
		return IntRect{}
	}
	anchor := rects[i]
	maxLabel := 0
	for _, it := range m.Menus[i].Items {
		w := len(it.Label) * menuGlyphW
		if it.Shortcut != "" {
			w += len(it.Shortcut)*menuGlyphW + 24
		}
		if w > maxLabel {
			maxLabel = w
		}
	}
	bodyW := maxLabel + 24
	if bodyW < anchor.W {
		bodyW = anchor.W
	}
	bodyH := len(m.Menus[i].Items) * MenuItemHeight
	return IntRect{X: anchor.X, Y: anchor.Y + anchor.H, W: bodyW, H: bodyH}
}

// HandleClick processes a left-click at (px, py). Returns true when
// the click was consumed.
func (m *MenuBar) HandleClick(px, py int) bool {
	for i, lr := range m.LabelRects() {
		if lr.Contains(px, py) {
			if m.openIdx == i {
				m.openIdx = -1
			} else {
				m.openIdx = i
			}
			return true
		}
	}
	if m.IsOpen() {
		dr := m.DropdownRect(m.openIdx)
		if dr.Contains(px, py) {
			idx := (py - dr.Y) / MenuItemHeight
			if idx >= 0 && idx < len(m.Menus[m.openIdx].Items) {
				item := m.Menus[m.openIdx].Items[idx]
				m.openIdx = -1
				if !item.Separator && !item.Disabled && item.OnSelect != nil {
					item.OnSelect()
				}
			}
			return true
		}
		m.openIdx = -1
		return true
	}
	return false
}

// HandleEscape dismisses any open dropdown. Returns true when consumed.
func (m *MenuBar) HandleEscape() bool {
	if !m.IsOpen() {
		return false
	}
	m.openIdx = -1
	return true
}

// Draw paints the menu bar + open dropdown using engine primitives.
// Caller is responsible for setting the camera if the bar is rendered
// inside a parent canvas.
func (m *MenuBar) Draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(m.bgColor)
	pixelforge.RectFill(m.X, m.Y, m.X+m.W-1, m.Y+m.H-1)

	rects := m.LabelRects()
	for i, md := range m.Menus {
		lr := rects[i]
		if i == m.openIdx {
			pixelforge.SetColor(m.openColor)
			pixelforge.RectFill(lr.X, lr.Y, lr.X+lr.W-1, lr.Y+lr.H-1)
		}
		pixelforge.SetColor(m.textColor)
		pixelforge_cofont.Print(md.Label, lr.X+menuLabelPadding/2, lr.Y+(lr.H-pixelforge_cofont.Sheet.Height)/2)
	}
	if !m.IsOpen() {
		return
	}
	dr := m.DropdownRect(m.openIdx)
	pixelforge.SetColor(m.dropdownColor)
	pixelforge.RectFill(dr.X, dr.Y, dr.X+dr.W-1, dr.Y+dr.H-1)
	pixelforge.SetColor(m.borderColor)
	pixelforge.Rect(dr.X, dr.Y, dr.X+dr.W-1, dr.Y+dr.H-1)
	for i, it := range m.Menus[m.openIdx].Items {
		rowY := dr.Y + i*MenuItemHeight
		if it.Separator {
			pixelforge.SetColor(m.borderColor)
			pixelforge.Line(dr.X+6, rowY+MenuItemHeight/2, dr.X+dr.W-6, rowY+MenuItemHeight/2)
			continue
		}
		c := m.textColor
		if it.Disabled {
			c = m.borderColor
		}
		pixelforge.SetColor(c)
		pixelforge_cofont.Print(it.Label, dr.X+8, rowY+3)
		if it.Shortcut != "" {
			x := dr.X + dr.W - len(it.Shortcut)*menuGlyphW - 8
			pixelforge_cofont.Print(it.Shortcut, x, rowY+3)
		}
	}
}
