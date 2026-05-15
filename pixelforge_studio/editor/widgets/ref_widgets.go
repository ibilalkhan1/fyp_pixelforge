package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// dropdown is the shared dropdown implementation used by sprite,
// audio, event, and enum widgets. Each widget instantiates its own
// dropdown with the right option list.
type dropdown struct {
	field   pfcomponent.FieldMetadata
	options []string
	open    bool
}

func (d *dropdown) draw(dst *ebiten.Image, area Rect, current string, name string) {
	fillRect(dst, area, colWidgetBg)
	strokeRect(dst, area, colWidgetBorder)
	printAt(dst, fmt.Sprintf("%s: %s", name, displayOrPlaceholder(current)), area.X+4, area.Y+4)
	caret := Rect{X: area.X + area.W - 12, Y: area.Y + 4, W: 8, H: 8}
	fillRect(dst, caret, colWidgetBorder)

	if !d.open {
		return
	}
	listRect := Rect{X: area.X, Y: area.Y + area.H, W: area.W, H: len(d.options) * 14}
	fillRect(dst, listRect, colWidgetBg)
	strokeRect(dst, listRect, colWidgetBorder)
	for i, opt := range d.options {
		y := listRect.Y + i*14
		printAt(dst, opt, listRect.X+6, y+1)
	}
}

func (d *dropdown) update(area Rect, current string, mx, my int) *EditEvent {
	if !isClickJustPressed() {
		return nil
	}
	if area.Contains(mx, my) {
		d.open = !d.open
		return nil
	}
	if !d.open {
		return nil
	}
	listRect := Rect{X: area.X, Y: area.Y + area.H, W: area.W, H: len(d.options) * 14}
	if !listRect.Contains(mx, my) {
		d.open = false
		return nil
	}
	idx := (my - listRect.Y) / 14
	if idx < 0 || idx >= len(d.options) {
		d.open = false
		return nil
	}
	d.open = false
	return &EditEvent{NewValue: d.options[idx]}
}

func displayOrPlaceholder(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// SpriteRefWidget: dropdown over Context.SpriteNames.
type SpriteRefWidget struct {
	F   pfcomponent.FieldMetadata
	Ctx *Context
	dd  dropdown
}

func (w *SpriteRefWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *SpriteRefWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	w.refreshOptions()
	cur, _ := asString(value)
	w.dd.draw(dst, area, cur, w.F.Name)
}

func (w *SpriteRefWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	w.refreshOptions()
	cur, _ := asString(value)
	return w.dd.update(area, cur, mx, my)
}

func (w *SpriteRefWidget) refreshOptions() {
	if w.Ctx != nil {
		w.dd.options = w.Ctx.SpriteNames
	} else {
		w.dd.options = nil
	}
}

// AudioRefWidget.
type AudioRefWidget struct {
	F   pfcomponent.FieldMetadata
	Ctx *Context
	dd  dropdown
}

func (w *AudioRefWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *AudioRefWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	w.refreshOptions()
	cur, _ := asString(value)
	w.dd.draw(dst, area, cur, w.F.Name)
}

func (w *AudioRefWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	w.refreshOptions()
	cur, _ := asString(value)
	return w.dd.update(area, cur, mx, my)
}

func (w *AudioRefWidget) refreshOptions() {
	if w.Ctx != nil {
		w.dd.options = w.Ctx.AudioNames
	} else {
		w.dd.options = nil
	}
}

// EventTopicWidget — picks from Context.EventTopics.
type EventTopicWidget struct {
	F   pfcomponent.FieldMetadata
	Ctx *Context
	dd  dropdown
}

func (w *EventTopicWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *EventTopicWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	w.refreshOptions()
	cur, _ := asString(value)
	w.dd.draw(dst, area, cur, w.F.Name)
}

func (w *EventTopicWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	w.refreshOptions()
	cur, _ := asString(value)
	return w.dd.update(area, cur, mx, my)
}

func (w *EventTopicWidget) refreshOptions() {
	if w.Ctx != nil {
		w.dd.options = w.Ctx.EventTopics
	} else {
		w.dd.options = nil
	}
}

// EnumWidget — picks from the metadata's static Options.
type EnumWidget struct {
	F  pfcomponent.FieldMetadata
	dd dropdown
}

func (w *EnumWidget) Field() pfcomponent.FieldMetadata { return w.F }

func (w *EnumWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	w.dd.options = w.F.Options
	cur, _ := asString(value)
	w.dd.draw(dst, area, cur, w.F.Name)
}

func (w *EnumWidget) Update(area Rect, value any, mx, my int, _ bool) *EditEvent {
	w.dd.options = w.F.Options
	cur, _ := asString(value)
	return w.dd.update(area, cur, mx, my)
}
