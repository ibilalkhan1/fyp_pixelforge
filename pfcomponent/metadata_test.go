package pfcomponent

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// parseRange handles whitespace tolerance and float forms.
func TestParseRange(t *testing.T) {
	cases := []struct {
		in      string
		lo, hi  float64
		wantErr bool
	}{
		{"0..10", 0, 10, false},
		{"0.5..1.5", 0.5, 1.5, false},
		{" 0 .. 100 ", 0, 100, false},
		{"abc..1", 0, 0, true},
		{"0..xyz", 0, 0, true},
		{"no-range", 0, 0, true},
	}
	for _, c := range cases {
		lo, hi, err := parseRange(c.in)
		if c.wantErr {
			assert.Error(t, err, "expected error for %q", c.in)
			continue
		}
		assert.NoError(t, err, "input %q", c.in)
		assert.Equal(t, c.lo, lo, "lo for %q", c.in)
		assert.Equal(t, c.hi, hi, "hi for %q", c.in)
	}
}

// splitEnumOptions trims and drops empties.
func TestSplitEnumOptions(t *testing.T) {
	assert.Equal(t, []string{"A", "B", "C"}, splitEnumOptions("A|B|C"))
	assert.Equal(t, []string{"A", "B"}, splitEnumOptions(" A | B | "))
	assert.Empty(t, splitEnumOptions(""))
}

// parseMaxLen extracts maxlen=N or returns 0.
func TestParseMaxLen(t *testing.T) {
	assert.Equal(t, 64.0, parseMaxLen("maxlen=64"))
	assert.Equal(t, 16.0, parseMaxLen("other,maxlen=16"))
	assert.Equal(t, 0.0, parseMaxLen("nope"))
	assert.Equal(t, 0.0, parseMaxLen("maxlen=abc"))
	assert.Equal(t, 0.0, parseMaxLen(""))
}

// widgetFromType picks sensible defaults for common Go kinds.
func TestWidgetFromType(t *testing.T) {
	type S struct {
		B   bool
		F32 float32
		F64 float64
		I   int
		U8  uint8
		Str string
		Sl  []int
	}
	rt := reflect.TypeOf(S{})
	assert.Equal(t, WidgetCheckbox, widgetFromType(rt.Field(0).Type))
	assert.Equal(t, WidgetFloatField, widgetFromType(rt.Field(1).Type))
	assert.Equal(t, WidgetFloatField, widgetFromType(rt.Field(2).Type))
	assert.Equal(t, WidgetIntField, widgetFromType(rt.Field(3).Type))
	assert.Equal(t, WidgetIntField, widgetFromType(rt.Field(4).Type))
	assert.Equal(t, WidgetText, widgetFromType(rt.Field(5).Type))
	assert.Equal(t, WidgetDefault, widgetFromType(rt.Field(6).Type))
}
