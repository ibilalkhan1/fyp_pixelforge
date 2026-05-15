package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// Vector2Widget edits a two-component numeric value. The schema stores
// vector2 fields as a 2-element slice/array of float64 (or int); we
// render two NumericWidget-style steppers side by side.
type Vector2Widget struct {
	F pfcomponent.FieldMetadata
}

func (w *Vector2Widget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *Vector2Widget) Draw(dst *ebiten.Image, area Rect, value any) {
	x, y := extractVector2(value)
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, area, colWidgetBorder)
	printAt(dst, fmt.Sprintf("%s: x=%.2f  y=%.2f", w.F.Name, x, y), area.X+4, area.Y+4)

	// Two pairs of -/+ buttons stacked horizontally.
	minusX := Rect{X: area.X + 4, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	plusX := Rect{X: area.X + 18, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	minusY := Rect{X: area.X + 40, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	plusY := Rect{X: area.X + 54, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	for _, r := range []Rect{minusX, plusX, minusY, plusY} {
		fillRect(dst, r, colWidgetBorder)
	}
	printAt(dst, "-x", minusX.X+1, minusX.Y)
	printAt(dst, "+x", plusX.X+1, plusX.Y)
	printAt(dst, "-y", minusY.X+1, minusY.Y)
	printAt(dst, "+y", plusY.X+1, plusY.Y)
}

func (w *Vector2Widget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	x, y := extractVector2(value)
	step := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		step = 10
	}
	minusX := Rect{X: area.X + 4, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	plusX := Rect{X: area.X + 18, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	minusY := Rect{X: area.X + 40, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	plusY := Rect{X: area.X + 54, Y: area.Y + area.H/2, W: 12, H: area.H/2 - 4}
	switch {
	case minusX.Contains(mx, my):
		x -= step
	case plusX.Contains(mx, my):
		x += step
	case minusY.Contains(mx, my):
		y -= step
	case plusY.Contains(mx, my):
		y += step
	default:
		return nil
	}
	return &EditEvent{NewValue: []float64{x, y}}
}

func extractVector2(value any) (float64, float64) {
	switch v := value.(type) {
	case []float64:
		if len(v) >= 2 {
			return v[0], v[1]
		}
	case []any:
		var x, y float64
		if len(v) > 0 {
			x, _ = asFloat(v[0])
		}
		if len(v) > 1 {
			y, _ = asFloat(v[1])
		}
		return x, y
	case [2]float64:
		return v[0], v[1]
	case [2]int:
		return float64(v[0]), float64(v[1])
	}
	return 0, 0
}
