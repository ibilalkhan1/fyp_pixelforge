package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// DefaultFieldWidget renders a read-only label for a field whose
// metadata gave us no recognisable widget. The user sees the field
// name and current value; edits require a registered widget kind.
type DefaultFieldWidget struct {
	F pfcomponent.FieldMetadata
}

func (w *DefaultFieldWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *DefaultFieldWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, area, colWidgetBorder)
	printAt(dst, fmt.Sprintf("%s: %v", w.F.Name, value), area.X+4, area.Y+4)
}

func (w *DefaultFieldWidget) Update(_ Rect, _ any, _, _ int, _ bool) *EditEvent {
	return nil
}

// UnknownWidget is rendered for fields whose pf:"..." tag named a kind
// this editor doesn't recognise. Shows the value with a warning glyph
// so the user knows the data is preserved but uneditable. Forward-
// compat for newer schemas opened in older editors.
type UnknownWidget struct {
	F pfcomponent.FieldMetadata
}

func (w *UnknownWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *UnknownWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, area, colWarning)
	printAt(dst, fmt.Sprintf("(unknown widget) %s: %v", w.F.Name, value), area.X+4, area.Y+4)
}

func (w *UnknownWidget) Update(_ Rect, _ any, _, _ int, _ bool) *EditEvent {
	return nil
}
