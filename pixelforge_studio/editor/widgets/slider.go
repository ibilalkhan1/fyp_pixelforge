package widgets

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// SliderWidget renders a horizontal slider for numeric fields tagged
// pf:"slider,min..max". Dragging the track within 1% of the cursor
// position updates the field value.
type SliderWidget struct {
	F        pfcomponent.FieldMetadata
	dragging bool
}

func (s *SliderWidget) Field() pfcomponent.FieldMetadata { return s.F }

func (s *SliderWidget) Draw(dst *ebiten.Image, area Rect, value any) {
	cur, _ := asFloat(value)
	clamped := clampFloat(cur, s.F.Min, s.F.Max)

	// Background track + filled portion.
	fillRect(dst, area, colWidgetBg)

	trackY := area.Y + area.H/2 - 3
	track := Rect{X: area.X + 4, Y: trackY, W: area.W - 8, H: 6}
	fillRect(dst, track, colWidgetBorder)

	progress := 0.0
	if s.F.Max > s.F.Min {
		progress = (clamped - s.F.Min) / (s.F.Max - s.F.Min)
	}
	fill := track
	fill.W = int(float64(track.W) * progress)
	fillRect(dst, fill, colWidgetFill)

	// Knob.
	knobX := track.X + fill.W - 3
	knob := Rect{X: knobX, Y: trackY - 3, W: 6, H: 12}
	fillRect(dst, knob, colWidgetText)

	printAt(dst, fmt.Sprintf("%s: %.2f", s.F.Name, clamped), area.X+4, area.Y-12)
}

// Update returns an EditEvent when the user drags the slider. We track
// `dragging` so a click on the track starts a drag, and movement
// outside the track continues until release.
func (s *SliderWidget) Update(area Rect, value any, mx, my int, pressed bool) *EditEvent {
	if isClickJustPressed() && area.Contains(mx, my) {
		s.dragging = true
	}
	if !pressed {
		s.dragging = false
		return nil
	}
	if !s.dragging {
		return nil
	}
	track := Rect{X: area.X + 4, Y: area.Y + area.H/2 - 3, W: area.W - 8, H: 6}
	if track.W <= 0 {
		return nil
	}
	progress := float64(mx-track.X) / float64(track.W)
	progress = clampFloat(progress, 0, 1)
	newVal := s.F.Min + progress*(s.F.Max-s.F.Min)

	// Coerce to int range when the underlying type is integer.
	switch s.F.Type.Kind().String() {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return &EditEvent{NewValue: int(newVal)}
	}
	return &EditEvent{NewValue: newVal}
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
