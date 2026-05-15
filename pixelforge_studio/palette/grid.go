// Package palette implements the Pixelforge Studio palette workspace —
// the 64-swatch grid, the four ColorTable matrices, Lightroom-style
// preset stacks, animation timelines, paint tools, and the PNG drop-
// import pipeline.
//
// The package is self-contained: only the editor's main.go and the
// pixelforge_studio binary import it. It depends on
// pixelforge_studio/editor (Workspace interface, Editor type) but never
// the other way round.
package palette

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

const (
	// gridColumns × gridRows = MaxColors (64).
	gridColumns = 8
	gridRows    = 8

	// swatchSize / swatchGap match the inspector's color picker but a
	// bit larger because the workspace has more room.
	gridSwatchSize = 28
	gridSwatchGap  = 2
)

// Grid is the 64-swatch palette grid. Clicking a swatch opens a small
// RGB picker popover anchored below the swatch.
type Grid struct {
	selectedSlot int
	picker       rgbPicker
	pickerOpen   bool

	// onAnimate is invoked when the user right-clicks a swatch. The
	// workspace wires this to its animation timeline popover (U19).
	onAnimate func(slot int)
}

// NewGrid returns a fresh grid with slot 0 selected.
func NewGrid() *Grid { return &Grid{} }

// SelectedSlot returns the currently-highlighted palette slot.
func (g *Grid) SelectedSlot() int { return g.selectedSlot }

// SetSelectedSlot selects a slot programmatically.
func (g *Grid) SetSelectedSlot(slot int) {
	if slot < 0 || slot >= pixelforge_project.MaxColors {
		return
	}
	g.selectedSlot = slot
}

// OnAnimate registers the right-click → animate callback.
func (g *Grid) OnAnimate(cb func(slot int)) { g.onAnimate = cb }

// PickerOpen reports whether the inline picker is currently visible.
func (g *Grid) PickerOpen() bool { return g.pickerOpen }

// Update handles input within area. Returns true if input was consumed.
func (g *Grid) Update(area widgets.Rect, p *pixelforge_project.Project, e *editor.Editor) bool {
	mx, my := ebiten.CursorPosition()
	if g.pickerOpen {
		if g.picker.Update(area, p, e, g) {
			return true
		}
		// Click outside the picker dismisses it.
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.picker.bodyRect(area, g.selectedSlot).Contains(mx, my) {
			g.pickerOpen = false
			return true
		}
		return true
	}
	// Left-click swatch → select + open picker.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if slot := g.hitTestSwatch(area, mx, my); slot >= 0 {
			g.selectedSlot = slot
			g.picker.Reset(p.Palette.Base[slot])
			g.pickerOpen = true
			return true
		}
	}
	// Right-click swatch → fire animate callback.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if slot := g.hitTestSwatch(area, mx, my); slot >= 0 {
			g.selectedSlot = slot
			if g.onAnimate != nil {
				g.onAnimate(slot)
			}
			return true
		}
	}
	return false
}

// Draw paints the grid (and any open picker) into area.
func (g *Grid) Draw(dst *ebiten.Image, area widgets.Rect, p *pixelforge_project.Project) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	for slot := 0; slot < pixelforge_project.MaxColors; slot++ {
		r := g.swatchRect(area, slot)
		if slot == 0 {
			drawCheckerboard(dst, r)
		} else {
			c, _ := parseHexColor(p.Palette.Base[slot])
			vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
		}
		if slot == g.selectedSlot {
			vector.StrokeRect(dst, float32(r.X-1), float32(r.Y-1), float32(r.W+2), float32(r.H+2), 1, colGridSelected, false)
		}
	}
	if g.pickerOpen {
		g.picker.Draw(dst, area, g.selectedSlot, p)
	}
}

// HitTestSwatch exposes the swatch under (mx, my); returns -1 on miss.
func (g *Grid) HitTestSwatch(area widgets.Rect, mx, my int) int {
	return g.hitTestSwatch(area, mx, my)
}

func (g *Grid) hitTestSwatch(area widgets.Rect, mx, my int) int {
	for slot := 0; slot < pixelforge_project.MaxColors; slot++ {
		r := g.swatchRect(area, slot)
		if r.Contains(mx, my) {
			return slot
		}
	}
	return -1
}

// swatchRect returns the rect for slot inside the workspace area.
func (g *Grid) swatchRect(area widgets.Rect, slot int) widgets.Rect {
	totalW := gridColumns*gridSwatchSize + (gridColumns-1)*gridSwatchGap
	totalH := gridRows*gridSwatchSize + (gridRows-1)*gridSwatchGap
	originX := area.X + (area.W-totalW)/2
	originY := area.Y + (area.H-totalH)/2
	col := slot % gridColumns
	row := slot / gridColumns
	return widgets.Rect{
		X: originX + col*(gridSwatchSize+gridSwatchGap),
		Y: originY + row*(gridSwatchSize+gridSwatchGap),
		W: gridSwatchSize,
		H: gridSwatchSize,
	}
}

// drawCheckerboard renders the slot-0 transparency marker.
func drawCheckerboard(dst *ebiten.Image, r widgets.Rect) {
	const cell = 4
	for y := 0; y < r.H; y += cell {
		for x := 0; x < r.W; x += cell {
			c := color.RGBA{R: 0x10, G: 0x10, B: 0x16, A: 0xff}
			if ((x/cell)+(y/cell))%2 == 0 {
				c = color.RGBA{R: 0x30, G: 0x30, B: 0x38, A: 0xff}
			}
			vector.DrawFilledRect(dst, float32(r.X+x), float32(r.Y+y), cell, cell, c, false)
		}
	}
}

// rgbPicker is the inline RGB editor that pops below a swatch.
type rgbPicker struct {
	r, g, b   int
	hexBuf    []rune
	hexFocus  bool
	hexErr    bool
}

// Reset seeds the picker with the hex color currently in the slot.
func (p *rgbPicker) Reset(hex string) {
	c, ok := parseHexColor(hex)
	if !ok {
		c = color.RGBA{}
	}
	p.r = int(c.R)
	p.g = int(c.G)
	p.b = int(c.B)
	p.hexBuf = []rune(hex)
	p.hexFocus = false
	p.hexErr = false
}

// SetRGB seeds the picker with explicit channel values (test helper).
func (p *rgbPicker) SetRGB(r, g, b int) {
	p.r, p.g, p.b = r, g, b
	p.hexBuf = []rune(formatHexColor(r, g, b))
	p.hexErr = false
}

// SetHex seeds the picker's hex input.
func (p *rgbPicker) SetHex(hex string) {
	p.hexBuf = []rune(hex)
}

// CurrentHex returns the committed color as "#RRGGBB".
func (p *rgbPicker) CurrentHex() string {
	return formatHexColor(p.r, p.g, p.b)
}

// Confirm commits the current channel values to slot. Returns false
// (rejected) when the hex input is set but malformed.
func (p *rgbPicker) Confirm(project *pixelforge_project.Project, slot int) bool {
	// If the hex input is the freshest edit, parse it; otherwise
	// commit the channel sliders.
	hex := string(p.hexBuf)
	if hex != "" && looksLikeHexColor(hex) {
		c, ok := parseHexColor(hex)
		if !ok {
			p.hexErr = true
			return false
		}
		p.r, p.g, p.b = int(c.R), int(c.G), int(c.B)
	}
	if project != nil {
		project.Palette.Base[slot] = formatHexColor(p.r, p.g, p.b)
	}
	return true
}

// Update routes input for the picker. Returns true while the picker
// owns input.
func (p *rgbPicker) Update(area widgets.Rect, project *pixelforge_project.Project, e *editor.Editor, g *Grid) bool {
	body := p.bodyRect(area, g.selectedSlot)
	mx, my := ebiten.CursorPosition()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.pickerOpen = false
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if p.Confirm(project, g.selectedSlot) {
			g.pickerOpen = false
			if e != nil {
				e.MarkDirty()
				e.SetStatusMessage(fmt.Sprintf("slot %d = %s", g.selectedSlot, p.CurrentHex()))
			}
		}
		return true
	}

	if !body.Contains(mx, my) {
		return false
	}
	// Hex input gets focus when clicked.
	hexRect := widgets.Rect{X: body.X + 8, Y: body.Y + 64, W: body.W - 24, H: 18}
	confirmRect := widgets.Rect{X: body.X + body.W - 70, Y: body.Y + body.H - 26, W: 60, H: 20}
	cancelRect := widgets.Rect{X: body.X + 8, Y: body.Y + body.H - 26, W: 60, H: 20}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case hexRect.Contains(mx, my):
			p.hexFocus = true
		case confirmRect.Contains(mx, my):
			if p.Confirm(project, g.selectedSlot) {
				g.pickerOpen = false
				if e != nil {
					e.MarkDirty()
					e.SetStatusMessage(fmt.Sprintf("slot %d = %s", g.selectedSlot, p.CurrentHex()))
				}
			}
			return true
		case cancelRect.Contains(mx, my):
			g.pickerOpen = false
			return true
		default:
			p.hexFocus = false
		}
	}

	if p.hexFocus {
		p.hexBuf = append(p.hexBuf, ebiten.AppendInputChars(nil)...)
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(p.hexBuf) > 0 {
			p.hexBuf = p.hexBuf[:len(p.hexBuf)-1]
		}
		if len(p.hexBuf) > 7 {
			p.hexBuf = p.hexBuf[:7]
		}
	}
	return true
}

// Draw paints the picker popover anchored to the currently-selected swatch.
func (p *rgbPicker) Draw(dst *ebiten.Image, area widgets.Rect, slot int, project *pixelforge_project.Project) {
	body := p.bodyRect(area, slot)
	vector.DrawFilledRect(dst, float32(body.X), float32(body.Y), float32(body.W), float32(body.H), colPickerBg, false)
	vector.StrokeRect(dst, float32(body.X), float32(body.Y), float32(body.W), float32(body.H), 1, colPickerBorder, false)

	ebitenutilPrint(dst, fmt.Sprintf("Slot %d", slot), body.X+8, body.Y+4)
	ebitenutilPrint(dst, fmt.Sprintf("R %d  G %d  B %d", p.r, p.g, p.b), body.X+8, body.Y+22)
	hexRect := widgets.Rect{X: body.X + 8, Y: body.Y + 64, W: body.W - 24, H: 18}
	c := colPickerInputBg
	if p.hexErr {
		c = color.RGBA{R: 0x80, G: 0x20, B: 0x20, A: 0xff}
	}
	vector.DrawFilledRect(dst, float32(hexRect.X), float32(hexRect.Y), float32(hexRect.W), float32(hexRect.H), c, false)
	ebitenutilPrint(dst, string(p.hexBuf), hexRect.X+4, hexRect.Y+1)

	if slot == 0 {
		ebitenutilPrint(dst, "(transparent)", body.X+8, body.Y+44)
	}

	// Color chip preview.
	chip := widgets.Rect{X: body.X + body.W - 28, Y: body.Y + 8, W: 18, H: 18}
	vector.DrawFilledRect(dst, float32(chip.X), float32(chip.Y), float32(chip.W), float32(chip.H),
		color.RGBA{R: uint8(p.r), G: uint8(p.g), B: uint8(p.b), A: 0xff}, false)

	confirm := widgets.Rect{X: body.X + body.W - 70, Y: body.Y + body.H - 26, W: 60, H: 20}
	cancel := widgets.Rect{X: body.X + 8, Y: body.Y + body.H - 26, W: 60, H: 20}
	vector.DrawFilledRect(dst, float32(confirm.X), float32(confirm.Y), float32(confirm.W), float32(confirm.H), colPickerConfirm, false)
	vector.DrawFilledRect(dst, float32(cancel.X), float32(cancel.Y), float32(cancel.W), float32(cancel.H), colPickerCancel, false)
	ebitenutilPrint(dst, "OK", confirm.X+18, confirm.Y+3)
	ebitenutilPrint(dst, "Cancel", cancel.X+10, cancel.Y+3)
	_ = project
}

func (p *rgbPicker) bodyRect(area widgets.Rect, slot int) widgets.Rect {
	const w, h = 200, 120
	col := slot % gridColumns
	row := slot / gridColumns
	totalW := gridColumns*gridSwatchSize + (gridColumns-1)*gridSwatchGap
	totalH := gridRows*gridSwatchSize + (gridRows-1)*gridSwatchGap
	originX := area.X + (area.W-totalW)/2
	originY := area.Y + (area.H-totalH)/2
	x := originX + col*(gridSwatchSize+gridSwatchGap)
	y := originY + (row+1)*(gridSwatchSize+gridSwatchGap)
	// Clamp to area so the popover never paints outside the workspace.
	if x+w > area.X+area.W {
		x = area.X + area.W - w
	}
	if y+h > area.Y+area.H {
		y = originY + row*(gridSwatchSize+gridSwatchGap) - h - 2
	}
	_ = totalH
	return widgets.Rect{X: x, Y: y, W: w, H: h}
}

// parseHexColor accepts "#RRGGBB" (case-insensitive). Returns ok=false
// for any other shape — 3-digit shorthand is rejected for v1
// simplicity, per the plan's open decision.
func parseHexColor(s string) (color.RGBA, bool) {
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{}, false
	}
	r, ok1 := parseHexByte(s[1:3])
	g, ok2 := parseHexByte(s[3:5])
	b, ok3 := parseHexByte(s[5:7])
	if !ok1 || !ok2 || !ok3 {
		return color.RGBA{}, false
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, true
}

// looksLikeHexColor recognises "#"-prefixed strings even when malformed
// so the picker can distinguish empty-input from typo-input.
func looksLikeHexColor(s string) bool {
	return strings.HasPrefix(s, "#")
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
	}
	return 0, false
}

// formatHexColor renders r/g/b as "#RRGGBB".
func formatHexColor(r, g, b int) string {
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 7)
	out[0] = '#'
	out[1] = hex[r>>4]
	out[2] = hex[r&0xf]
	out[3] = hex[g>>4]
	out[4] = hex[g&0xf]
	out[5] = hex[b>>4]
	out[6] = hex[b&0xf]
	return string(out)
}

var (
	colGridSelected  = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	colPickerBg      = color.RGBA{R: 0x22, G: 0x22, B: 0x2c, A: 0xff}
	colPickerBorder  = color.RGBA{R: 0x44, G: 0x44, B: 0x50, A: 0xff}
	colPickerInputBg = color.RGBA{R: 0x18, G: 0x18, B: 0x22, A: 0xff}
	colPickerConfirm = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	colPickerCancel  = color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff}
)
