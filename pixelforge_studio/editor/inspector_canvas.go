package editor

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// DrawCanvas renders the inspector chrome (panel, component headers,
// per-field labels + value previews) into the editor cart's canvas. It
// is the canvas-resident half of the M3 hybrid inspector — actual
// editing still happens through the native overlay path layered on top.
//
// area is the inspector rect in canvas-local coordinates.
func (i *Inspector) DrawCanvas(
	area widgets.Rect,
	project *pixelforge_project.Project,
	entity *pixelforge_project.Entity,
	theme *EditorTheme,
) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	if theme == nil {
		theme = DefaultEditorTheme()
	}

	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Inspector panel chrome.
	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(area.X, area.Y, area.X+area.W-1, area.Y+area.H-1)

	pixelforge.SetColor(theme.PanelHeaderSlot)
	pixelforge.RectFill(area.X, area.Y, area.X+area.W-1, area.Y+15)
	pixelforge.SetColor(theme.TextSlot)
	pcofont("INSPECTOR", area.X+8, area.Y+4)

	if entity == nil {
		pixelforge.SetColor(theme.TextDimSlot)
		pcofont("(no selection)", area.X+8, area.Y+24)
		return
	}

	y := area.Y + 22
	pixelforge.SetColor(theme.TextSlot)
	pcofont(entity.Name, area.X+8, y)
	y += 12

	for _, comp := range entity.Components {
		md, ok := pfcomponent.Get(comp.Type)
		if !ok {
			pixelforge.SetColor(theme.WarningSlot)
			pixelforge.RectFill(area.X+4, y, area.X+area.W-5, y+12)
			pixelforge.SetColor(theme.TextSlot)
			pcofont("(unknown) "+comp.Type, area.X+8, y+2)
			y += 16
			continue
		}
		pixelforge.SetColor(theme.PanelHeaderSlot)
		pixelforge.RectFill(area.X+4, y, area.X+area.W-5, y+14)
		pixelforge.SetColor(theme.TextSlot)
		pcofont(md.Name, area.X+8, y+3)
		y += 16

		for _, f := range md.Fields {
			drawInspectorFieldCanvas(area, y, f, comp.Values[f.JSONKey], theme)
			y += 18
		}
		y += 4
	}
	_ = project
}

func drawInspectorFieldCanvas(area widgets.Rect, y int, f pfcomponent.FieldMetadata, val any, theme *EditorTheme) {
	pixelforge.SetColor(theme.TextSlot)
	pcofont(f.Name, area.X+8, y)
	// Value preview.
	pixelforge.SetColor(theme.TextDimSlot)
	pcofont(formatFieldValue(val), area.X+area.W/2, y)
}

func formatFieldValue(v any) string {
	if v == nil {
		return "-"
	}
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%.2f", x)
	case int:
		return fmt.Sprintf("%d", x)
	case map[string]any:
		// Vector2: {x, y}
		if xv, ok := x["x"]; ok {
			if yv, ok2 := x["y"]; ok2 {
				return fmt.Sprintf("%v,%v", xv, yv)
			}
		}
		return fmt.Sprintf("%v", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
