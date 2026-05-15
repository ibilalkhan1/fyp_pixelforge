package palette

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

const (
	matrixCellSize = 4
	matrixTables   = 4
)

// Matrix renders the 4 ColorTable matrices (4 × 64 × 64 cells) as a
// vertical stack. Clicking a cell pops a small slot picker; the chosen
// slot index is written into project.Palette.ColorTables[T][src][dst].
type Matrix struct {
	scrollOffset int

	// pickerOpen indicates the slot-picker popover for cell (table,
	// src, dst) is visible. Implementation is intentionally tiny — a
	// 64-swatch grid that writes to the cell on click.
	pickerOpen   bool
	pickerTable  int
	pickerSrc    int
	pickerDst    int
}

// NewMatrix returns a fresh matrix view.
func NewMatrix() *Matrix { return &Matrix{} }

// Update routes input. Returns true while owning input.
func (m *Matrix) Update(area widgets.Rect, p *pixelforge_project.Project, e *editor.Editor) bool {
	_, wy := ebiten.Wheel()
	if wy != 0 {
		m.scrollOffset -= int(wy * matrixCellSize * 4)
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		maxScroll := matrixTables*matrixTableHeight() - area.H
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}
	}

	if m.pickerOpen {
		return m.updatePicker(area, p, e)
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	mx, my := ebiten.CursorPosition()
	table, src, dst := m.HitTest(area, mx, my)
	if table < 0 {
		return false
	}
	m.pickerOpen = true
	m.pickerTable = table
	m.pickerSrc = src
	m.pickerDst = dst
	return true
}

// updatePicker handles the slot-picker popover. On confirm the chosen
// slot value is written to the cell and the picker dismisses.
func (m *Matrix) updatePicker(area widgets.Rect, p *pixelforge_project.Project, e *editor.Editor) bool {
	mx, my := ebiten.CursorPosition()
	body := m.pickerBody(area)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.pickerOpen = false
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if !body.Contains(mx, my) {
		m.pickerOpen = false
		return true
	}
	for slot := 0; slot < pixelforge_project.MaxColors; slot++ {
		sr := m.pickerSlotRect(body, slot)
		if sr.Contains(mx, my) {
			p.Palette.ColorTables[m.pickerTable][m.pickerSrc][m.pickerDst] = uint8(slot)
			m.pickerOpen = false
			if e != nil {
				e.MarkDirty()
				e.SetStatusMessage(fmt.Sprintf("table=%d (%d→%d) = %d", m.pickerTable, m.pickerSrc, m.pickerDst, slot))
			}
			return true
		}
	}
	return true
}

// HitTest returns (table, src, dst) at the window-space (mx, my), or
// (-1,-1,-1) when no cell is under the cursor.
func (m *Matrix) HitTest(area widgets.Rect, mx, my int) (int, int, int) {
	relY := my - area.Y + m.scrollOffset
	tableHeight := matrixTableHeight()
	table := relY / tableHeight
	if table < 0 || table >= matrixTables {
		return -1, -1, -1
	}
	innerY := relY - table*tableHeight - matrixHeaderH
	if innerY < 0 || innerY >= pixelforge_project.MaxColors*matrixCellSize {
		return -1, -1, -1
	}
	relX := mx - area.X - matrixGutter
	if relX < 0 || relX >= pixelforge_project.MaxColors*matrixCellSize {
		return -1, -1, -1
	}
	src := innerY / matrixCellSize
	dst := relX / matrixCellSize
	return table, src, dst
}

// Draw paints the matrix view.
func (m *Matrix) Draw(dst *ebiten.Image, area widgets.Rect, p *pixelforge_project.Project) {
	vector.DrawFilledRect(dst, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), colMatrixBg, false)

	rowH := matrixTableHeight()
	for t := 0; t < matrixTables; t++ {
		tableTop := area.Y + t*rowH - m.scrollOffset
		if tableTop+rowH < area.Y || tableTop >= area.Y+area.H {
			continue
		}
		headerRect := widgets.Rect{X: area.X, Y: tableTop, W: area.W, H: matrixHeaderH}
		vector.DrawFilledRect(dst, float32(headerRect.X), float32(headerRect.Y), float32(headerRect.W), float32(headerRect.H), colMatrixHeader, false)
		ebitenutilPrint(dst, fmt.Sprintf("ColorTable %d", t), headerRect.X+6, headerRect.Y+2)

		// Cells.
		baseX := area.X + matrixGutter
		baseY := tableTop + matrixHeaderH
		for src := 0; src < pixelforge_project.MaxColors; src++ {
			for dt := 0; dt < pixelforge_project.MaxColors; dt++ {
				cell := widgets.Rect{
					X: baseX + dt*matrixCellSize,
					Y: baseY + src*matrixCellSize,
					W: matrixCellSize, H: matrixCellSize,
				}
				val := p.Palette.ColorTables[t][src][dt]
				c, _ := parseHexColor(p.Palette.Base[val])
				vector.DrawFilledRect(dst, float32(cell.X), float32(cell.Y), float32(cell.W), float32(cell.H), c, false)
				if access := pixelforge.ColorTableAccesses[t][src][dt]; access > 0 {
					tint := heatTint(access)
					vector.DrawFilledRect(dst, float32(cell.X), float32(cell.Y), float32(cell.W), float32(cell.H), tint, false)
				}
			}
		}
	}

	if m.pickerOpen {
		m.drawPicker(dst, area, p)
	}
}

// drawPicker renders the slot picker overlay.
func (m *Matrix) drawPicker(dst *ebiten.Image, area widgets.Rect, p *pixelforge_project.Project) {
	body := m.pickerBody(area)
	vector.DrawFilledRect(dst, float32(body.X), float32(body.Y), float32(body.W), float32(body.H), colMatrixPickerBg, false)
	vector.StrokeRect(dst, float32(body.X), float32(body.Y), float32(body.W), float32(body.H), 1, colMatrixHeader, false)
	ebitenutilPrint(dst, fmt.Sprintf("Pick value: table=%d src=%d dst=%d", m.pickerTable, m.pickerSrc, m.pickerDst), body.X+6, body.Y+2)
	for slot := 0; slot < pixelforge_project.MaxColors; slot++ {
		r := m.pickerSlotRect(body, slot)
		if slot == 0 {
			drawCheckerboard(dst, r)
			continue
		}
		c, _ := parseHexColor(p.Palette.Base[slot])
		vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
	}
}

func (m *Matrix) pickerBody(area widgets.Rect) widgets.Rect {
	const w, h = 220, 144
	x := area.X + (area.W-w)/2
	y := area.Y + (area.H-h)/2
	return widgets.Rect{X: x, Y: y, W: w, H: h}
}

func (m *Matrix) pickerSlotRect(body widgets.Rect, slot int) widgets.Rect {
	const cell = 20
	const cols = 8
	col := slot % cols
	row := slot / cols
	return widgets.Rect{X: body.X + 10 + col*(cell+2), Y: body.Y + 18 + row*(cell+2), W: cell, H: cell}
}

// heatTint maps an access count to a transparent overlay color. The
// scale is logarithmic-ish: nonzero counts surface readably without
// drowning out the underlying palette swatch.
func heatTint(access uint64) color.RGBA {
	if access == 0 {
		return color.RGBA{}
	}
	// Cap at a reasonable visual saturation.
	alpha := uint8(40)
	if access > 100 {
		alpha = 90
	}
	if access > 10000 {
		alpha = 140
	}
	return color.RGBA{R: 0xff, G: 0x9d, B: 0x40, A: alpha}
}

const (
	matrixHeaderH = 14
	matrixGutter  = 4
)

func matrixTableHeight() int {
	return matrixHeaderH + pixelforge_project.MaxColors*matrixCellSize + 8
}

var (
	colMatrixBg       = color.RGBA{R: 0x12, G: 0x12, B: 0x18, A: 0xff}
	colMatrixHeader   = color.RGBA{R: 0x2a, G: 0x2a, B: 0x36, A: 0xff}
	colMatrixPickerBg = color.RGBA{R: 0x22, G: 0x22, B: 0x2c, A: 0xff}
)
