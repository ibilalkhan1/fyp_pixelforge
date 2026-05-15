package palette

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// DrawCanvas renders the palette workspace into the editor cart's
// canvas using engine primitives. The hybrid M3 scope: the workspace
// frame, section headers, and the 64-swatch grid render via
// pixelforge.RectFill / Rect + pixelforge_cofont.Print. The matrix,
// presets, and animator views fall back to the native overlay path
// during M3 — the canvas-resident render delivers R1 partial dogfooding
// for the highest-visibility surface (the swatch grid).
func (w *Workspace) DrawCanvas(rel widgets.Rect, e *editor.Editor) {
	if rel.W <= 0 || rel.H <= 0 || e == nil || e.Project() == nil {
		return
	}
	theme := editor.DefaultEditorTheme()
	if c := e.Cart(); c != nil && c.Theme() != nil {
		theme = c.Theme()
	}

	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Workspace background.
	pixelforge.SetColor(theme.BackgroundSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	// Section header strip.
	pixelforge.SetColor(theme.PanelHeaderSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+15)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("PALETTE", rel.X+8, rel.Y+4)

	p := e.Project()
	if p == nil {
		return
	}

	// 64-swatch grid: 8x8.
	const cols, rows = 8, 8
	const swSize = 24
	const swGap = 2
	gridX := rel.X + 16
	gridY := rel.Y + 28
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			slot := r*cols + c
			x := gridX + c*(swSize+swGap)
			y := gridY + r*(swSize+swGap)
			// Slot 0 is the canonical "transparent" — render a small
			// checkerboard so the user can tell it apart from the bg.
			if slot == 0 {
				drawCheckerboardCanvas(x, y, swSize, theme)
			} else {
				pixelforge.SetColor(pixelforge.Color(slot))
				pixelforge.RectFill(x, y, x+swSize-1, y+swSize-1)
			}
			if slot == w.grid.SelectedSlot() {
				pixelforge.SetColor(theme.TextSlot)
				pixelforge.Rect(x-1, y-1, x+swSize, y+swSize)
			}
		}
	}

	// Presets section header on the right.
	presetX := gridX + cols*(swSize+swGap) + 24
	pixelforge.SetColor(theme.PanelHeaderSlot)
	pixelforge.RectFill(presetX, rel.Y+16, rel.X+rel.W-16, rel.Y+30)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("PRESETS", presetX+4, rel.Y+20)
	// Preset rows list (visual only in M3 canvas path).
	presetY := rel.Y + 36
	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(presetX, presetY, rel.X+rel.W-16, presetY+200)
	pixelforge.SetColor(theme.TextDimSlot)
	pixelforge_cofont.Print("(see native overlay for editing)",
		presetX+8, presetY+8)
}

func drawCheckerboardCanvas(x, y, size int, theme *editor.EditorTheme) {
	const cell = 4
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)
	for j := 0; j < size; j += cell {
		for i := 0; i < size; i += cell {
			if ((i/cell)+(j/cell))%2 == 0 {
				pixelforge.SetColor(theme.PanelSlot)
			} else {
				pixelforge.SetColor(theme.PanelHeaderSlot)
			}
			x1 := x + i
			y1 := y + j
			x2 := x1 + cell - 1
			y2 := y1 + cell - 1
			if x2 > x+size-1 {
				x2 = x + size - 1
			}
			if y2 > y+size-1 {
				y2 = y + size - 1
			}
			pixelforge.RectFill(x1, y1, x2, y2)
		}
	}
}
