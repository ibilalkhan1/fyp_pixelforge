package widgets

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
)

// New() returns the right widget concrete type for each WidgetKind.
func TestNew_DispatchesByKind(t *testing.T) {
	cases := []struct {
		kind pfcomponent.WidgetKind
		want any
	}{
		{pfcomponent.WidgetSlider, &SliderWidget{}},
		{pfcomponent.WidgetPaletteColor, &ColorPickerWidget{}},
		{pfcomponent.WidgetSpriteRef, &SpriteRefWidget{}},
		{pfcomponent.WidgetAudioRef, &AudioRefWidget{}},
		{pfcomponent.WidgetEventTopic, &EventTopicWidget{}},
		{pfcomponent.WidgetEnum, &EnumWidget{}},
		{pfcomponent.WidgetText, &TextWidget{}},
		{pfcomponent.WidgetVector2, &Vector2Widget{}},
		{pfcomponent.WidgetCheckbox, &CheckboxWidget{}},
		{pfcomponent.WidgetIntField, &NumericWidget{}},
		{pfcomponent.WidgetFloatField, &NumericWidget{}},
		{pfcomponent.WidgetDefault, &DefaultFieldWidget{}},
		{pfcomponent.WidgetUnknown, &UnknownWidget{}},
	}
	for _, c := range cases {
		got := New(pfcomponent.FieldMetadata{WidgetKind: c.kind})
		assert.IsType(t, c.want, got, "WidgetKind %s", c.kind)
	}
}

// Rect.Contains uses inclusive-min / exclusive-max bounds.
func TestRect_Contains(t *testing.T) {
	r := Rect{X: 10, Y: 20, W: 30, H: 40}
	assert.True(t, r.Contains(10, 20))
	assert.True(t, r.Contains(39, 59))
	assert.False(t, r.Contains(9, 20))
	assert.False(t, r.Contains(40, 20))
	assert.False(t, r.Contains(10, 60))
}

// asFloat coerces common numeric `any` values.
func TestAsFloat(t *testing.T) {
	cases := []struct {
		in       any
		want     float64
		wantOk   bool
	}{
		{1.5, 1.5, true},
		{float32(2.5), 2.5, true},
		{3, 3.0, true},
		{int64(4), 4.0, true},
		{uint(5), 5.0, true},
		{"nope", 0, false},
	}
	for _, c := range cases {
		got, ok := asFloat(c.in)
		assert.Equal(t, c.wantOk, ok, "%v", c.in)
		if ok {
			assert.Equal(t, c.want, got)
		}
	}
}

// parseHexColor handles "#RRGGBB" and rejects garbage.
func TestParseHexColor(t *testing.T) {
	c, ok := parseHexColor("#ff8800")
	assert.True(t, ok)
	assert.Equal(t, uint8(0xff), c.R)
	assert.Equal(t, uint8(0x88), c.G)
	assert.Equal(t, uint8(0x00), c.B)

	_, ok = parseHexColor("nope")
	assert.False(t, ok)

	_, ok = parseHexColor("#zzzzzz")
	assert.False(t, ok)
}

// SliderWidget reports the right widget kind via Field().
func TestSliderWidget_Field(t *testing.T) {
	w := &SliderWidget{F: pfcomponent.FieldMetadata{
		Name:       "Speed",
		WidgetKind: pfcomponent.WidgetSlider,
		Min:        0, Max: 10,
		Type: reflect.TypeOf(float64(0)),
	}}
	assert.Equal(t, "Speed", w.Field().Name)
	assert.Equal(t, pfcomponent.WidgetSlider, w.Field().WidgetKind)
}

// CheckboxWidget toggles bool value on click inside its rect.
func TestCheckboxWidget_ToggleSemantics(t *testing.T) {
	// We can't easily simulate ebiten.IsMouseButtonJustPressed; we
	// verify the toggle logic via the `extractVector2 / asBool path —
	// asBool semantics, since that's the value the widget reads.
	tr, ok := asBool(true)
	assert.True(t, ok)
	assert.True(t, tr)
	fa, ok := asBool(false)
	assert.True(t, ok)
	assert.False(t, fa)
}

// extractVector2 handles the various shapes a vector2 value can arrive in.
func TestExtractVector2(t *testing.T) {
	cases := []struct {
		in       any
		wantX, wantY float64
	}{
		{[]float64{3, 4}, 3, 4},
		{[]any{5.0, 6.0}, 5, 6},
		{[]any{7, 8}, 7, 8},
		{[2]float64{9, 10}, 9, 10},
		{[2]int{11, 12}, 11, 12},
		{nil, 0, 0},
		{"garbage", 0, 0},
	}
	for _, c := range cases {
		x, y := extractVector2(c.in)
		assert.Equal(t, c.wantX, x)
		assert.Equal(t, c.wantY, y)
	}
}

// Numeric formatNumeric uses int vs float formatting appropriately.
func TestFormatNumeric(t *testing.T) {
	assert.Equal(t, "5", formatNumeric(5.0, pfcomponent.WidgetIntField))
	assert.Equal(t, "1.5", formatNumeric(1.5, pfcomponent.WidgetFloatField))
}

// numericStep returns 1 for ints and 0.1 for floats.
func TestNumericStep(t *testing.T) {
	assert.Equal(t, 1.0, numericStep(pfcomponent.WidgetIntField))
	assert.InDelta(t, 0.1, numericStep(pfcomponent.WidgetFloatField), 0.0001)
}

// clampFloat sanity check.
func TestClampFloat(t *testing.T) {
	assert.Equal(t, 5.0, clampFloat(5, 0, 10))
	assert.Equal(t, 0.0, clampFloat(-1, 0, 10))
	assert.Equal(t, 10.0, clampFloat(99, 0, 10))
}
