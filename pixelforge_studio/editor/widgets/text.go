package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// TextWidget is a simple inline text editor. M0-M2 keeps text input
// minimal: clicking the widget gives it focus; subsequent typed
// characters append, Backspace removes one, Enter commits. Cursor
// positioning, selection, and IME are deferred to M3.
type TextWidget struct {
	F       pfcomponent.FieldMetadata
	focused bool
	buf     []rune
}

func (w *TextWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *TextWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	fillRect(dst, area, colWidgetBg)
	c := colWidgetBorder
	if w.focused {
		c = colWidgetFill
	}
	strokeRect(dst, area, c)

	display, _ := asString(value)
	if w.focused {
		display = string(w.buf)
	}
	printAt(dst, fmt.Sprintf("%s:", w.F.Name), area.X+4, area.Y+4)
	printAt(dst, display, area.X+4, area.Y+area.H/2)
}

func (w *TextWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	if isClickJustPressed() {
		if area.Contains(mx, my) {
			if !w.focused {
				current, _ := asString(value)
				w.buf = []rune(current)
			}
			w.focused = true
		} else {
			if w.focused {
				w.focused = false
				return &EditEvent{NewValue: string(w.buf)}
			}
		}
	}
	if !w.focused {
		return nil
	}
	// Read typed characters.
	w.buf = append(w.buf, ebiten.AppendInputChars(nil)...)
	if w.F.Max > 0 && float64(len(w.buf)) > w.F.Max {
		w.buf = w.buf[:int(w.F.Max)]
	}
	if ebiten.IsKeyPressed(ebiten.KeyBackspace) && len(w.buf) > 0 {
		w.buf = w.buf[:len(w.buf)-1]
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		w.focused = false
		return &EditEvent{NewValue: string(w.buf)}
	}
	return nil
}
