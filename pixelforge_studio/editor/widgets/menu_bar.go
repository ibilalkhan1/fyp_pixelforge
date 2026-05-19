package widgets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// MenuItem is a single dropdown entry under a top-level menu.
type MenuItem struct {
	Label    string
	Shortcut string // optional "Ctrl+S" text; rendered right-aligned
	OnSelect func()
	// Separator, when true, renders a thin divider in place of an item.
	Separator bool
	// Disabled items render dim and ignore clicks.
	Disabled bool
	// Checked, when true, renders a checkmark glyph before Label.
	// Carries no behaviour beyond the visual cue; OnSelect is still
	// the click target. Used by toggle-style menu entries (idea #3
	// v1 U8 introduces the first ones — the overlay toggles).
	Checked bool
}

// MenuDef is one top-level menu (File / Edit / View / Help).
type MenuDef struct {
	Label string
	Items []MenuItem
}

// MenuBar is the top-of-window menu strip. Click a top-level label to
// open its dropdown; click an item to fire its callback.
type MenuBar struct {
	Menus []MenuDef

	openIdx int // -1 when closed
}

// NewMenuBar returns a closed menu bar with the supplied definitions.
func NewMenuBar(menus []MenuDef) *MenuBar {
	return &MenuBar{Menus: menus, openIdx: -1}
}

// Reset closes any open dropdown.
func (m *MenuBar) Reset() { m.openIdx = -1 }

// IsOpen reports whether any dropdown is currently open.
func (m *MenuBar) IsOpen() bool { return m.openIdx >= 0 }

// MenuBarHeight is the menu strip's vertical pixel size.
const MenuBarHeight = 22

// labelWidth is the per-character pixel width used to lay out menu
// titles. The debug font is roughly 6px/char; we pad each side by 12.
const labelPadding = 12
const labelGlyphW = 6

// labelRects returns one Rect per top-level menu in window-pixel space.
func (m *MenuBar) labelRects(area Rect) []Rect {
	rects := make([]Rect, len(m.Menus))
	x := area.X + 8
	for i, md := range m.Menus {
		w := len(md.Label)*labelGlyphW + labelPadding
		rects[i] = Rect{X: x, Y: area.Y, W: w, H: area.H}
		x += w
	}
	return rects
}

// Update routes input to the menu bar. area is the menu strip's rect.
// Returns true when input was consumed (so the chrome below should
// skip handling it this frame).
func (m *MenuBar) Update(area Rect) bool {
	mx, my := ebiten.CursorPosition()
	consumed := false

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && m.IsOpen() {
		m.openIdx = -1
		return true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Click on a top-level label toggles its dropdown.
		for i, lr := range m.labelRects(area) {
			if lr.Contains(mx, my) {
				if m.openIdx == i {
					m.openIdx = -1
				} else {
					m.openIdx = i
				}
				return true
			}
		}
		if m.IsOpen() {
			dr := m.dropdownRect(area, m.openIdx)
			if dr.Contains(mx, my) {
				idx := (my - dr.Y) / menuItemHeight
				if idx >= 0 && idx < len(m.Menus[m.openIdx].Items) {
					item := m.Menus[m.openIdx].Items[idx]
					m.openIdx = -1
					if !item.Separator && !item.Disabled && item.OnSelect != nil {
						item.OnSelect()
					}
				}
				return true
			}
			// Click outside dropdown closes it.
			m.openIdx = -1
			consumed = true
		}
	}
	return consumed
}

// dropdownRect computes the rect of the dropdown panel for menu i.
func (m *MenuBar) dropdownRect(area Rect, i int) Rect {
	rects := m.labelRects(area)
	if i < 0 || i >= len(rects) {
		return Rect{}
	}
	anchor := rects[i]
	maxLabel := 0
	for _, it := range m.Menus[i].Items {
		w := len(it.Label) * labelGlyphW
		if it.Shortcut != "" {
			w += len(it.Shortcut)*labelGlyphW + 24
		}
		if w > maxLabel {
			maxLabel = w
		}
	}
	bodyW := maxLabel + 24
	if bodyW < anchor.W {
		bodyW = anchor.W
	}
	bodyH := len(m.Menus[i].Items) * menuItemHeight
	return Rect{X: anchor.X, Y: anchor.Y + anchor.H, W: bodyW, H: bodyH}
}

// Draw paints the menu bar and any open dropdown into the destination.
func (m *MenuBar) Draw(dst *ebiten.Image, area Rect) {
	fillRect(dst, area, colMenuBarBg)
	rects := m.labelRects(area)
	for i, md := range m.Menus {
		lr := rects[i]
		if i == m.openIdx {
			fillRect(dst, lr, colMenuBarOpen)
		}
		printAt(dst, md.Label, lr.X+labelPadding/2, lr.Y+3)
	}
	if !m.IsOpen() {
		return
	}
	dr := m.dropdownRect(area, m.openIdx)
	fillRect(dst, dr, colMenuDropdown)
	strokeRect(dst, dr, colWidgetBorder)
	for i, it := range m.Menus[m.openIdx].Items {
		row := Rect{X: dr.X, Y: dr.Y + i*menuItemHeight, W: dr.W, H: menuItemHeight}
		if it.Separator {
			midY := row.Y + row.H/2
			fillRect(dst, Rect{X: row.X + 6, Y: midY, W: row.W - 12, H: 1}, colWidgetBorder)
			continue
		}
		printAt(dst, it.Label, row.X+8, row.Y+3)
		if it.Shortcut != "" {
			x := row.X + row.W - len(it.Shortcut)*labelGlyphW - 8
			printAt(dst, it.Shortcut, x, row.Y+3)
		}
	}
}

const menuItemHeight = 18

var (
	colMenuBarBg    = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xff}
	colMenuBarOpen  = color.RGBA{R: 0x32, G: 0x32, B: 0x40, A: 0xff}
	colMenuDropdown = color.RGBA{R: 0x22, G: 0x22, B: 0x2c, A: 0xff}
)
