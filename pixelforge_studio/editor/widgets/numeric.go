package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// NumericWidget renders an int or float field as a compact label with
// +/- step buttons on either side. The increment defaults to 1 for
// integers and 0.1 for floats; Shift held during click → ×10 step.
//
// Used as the default for untagged numeric fields and for fields tagged
// pf:"int_field" / pf:"float_field" (the metadata's type-based
// fallbacks).
type NumericWidget struct {
	F pfcomponent.FieldMetadata
}

func (w *NumericWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *NumericWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	v, _ := asFloat(value)
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, area, colWidgetBorder)

	minusRect := Rect{X: area.X + 4, Y: area.Y + 4, W: 16, H: area.H - 8}
	plusRect := Rect{X: area.X + area.W - 20, Y: area.Y + 4, W: 16, H: area.H - 8}
	fillRect(dst, minusRect, colWidgetBorder)
	fillRect(dst, plusRect, colWidgetBorder)
	printAt(dst, "-", minusRect.X+5, minusRect.Y+1)
	printAt(dst, "+", plusRect.X+5, plusRect.Y+1)

	label := fmt.Sprintf("%s: %s", w.F.Name, formatNumeric(v, w.F.WidgetKind))
	printAt(dst, label, area.X+24, area.Y+4)
}

func (w *NumericWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	v, _ := asFloat(value)
	step := numericStep(w.F.WidgetKind)
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		step *= 10
	}
	minusRect := Rect{X: area.X + 4, Y: area.Y + 4, W: 16, H: area.H - 8}
	plusRect := Rect{X: area.X + area.W - 20, Y: area.Y + 4, W: 16, H: area.H - 8}
	switch {
	case minusRect.Contains(mx, my):
		v -= step
	case plusRect.Contains(mx, my):
		v += step
	default:
		return nil
	}
	if w.F.WidgetKind == pfcomponent.WidgetIntField {
		return &EditEvent{NewValue: int(v)}
	}
	return &EditEvent{NewValue: v}
}

func numericStep(kind pfcomponent.WidgetKind) float64 {
	if kind == pfcomponent.WidgetFloatField {
		return 0.1
	}
	return 1
}

func formatNumeric(v float64, kind pfcomponent.WidgetKind) string {
	if kind == pfcomponent.WidgetIntField {
		return fmt.Sprintf("%d", int(v))
	}
	return fmt.Sprintf("%.3g", v)
}
