package widgets

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// CheckboxWidget for bool fields.
type CheckboxWidget struct {
	F pfcomponent.FieldMetadata
}

func (w *CheckboxWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *CheckboxWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	b, _ := asBool(value)
	box := Rect{X: area.X + 4, Y: area.Y + 4, W: area.H - 8, H: area.H - 8}
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, box, colWidgetBorder)
	if b {
		inner := Rect{X: box.X + 2, Y: box.Y + 2, W: box.W - 4, H: box.H - 4}
		fillRect(dst, inner, colWidgetFill)
	}
	printAt(dst, w.F.Name, box.X+box.W+8, area.Y+4)
}

func (w *CheckboxWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}
	if !area.Contains(mx, my) {
		return nil
	}
	b, _ := asBool(value)
	return &EditEvent{NewValue: !b}
}
