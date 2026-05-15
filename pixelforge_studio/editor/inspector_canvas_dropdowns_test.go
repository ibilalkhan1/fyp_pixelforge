package editor

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanvasDropdownFor_MapsKnownWidgets(t *testing.T) {
	cases := map[pfcomponent.WidgetKind]canvasDropdownKind{
		pfcomponent.WidgetPaletteColor: canvasDropdownColorPicker,
		pfcomponent.WidgetSpriteRef:    canvasDropdownSpriteRef,
		pfcomponent.WidgetAudioRef:     canvasDropdownAudioRef,
		pfcomponent.WidgetEventTopic:   canvasDropdownEventTopic,
		pfcomponent.WidgetEnum:         canvasDropdownEnum,
	}
	for kind, want := range cases {
		got, ok := canvasDropdownFor(pfcomponent.FieldMetadata{WidgetKind: kind})
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
}

func TestCanvasDropdownFor_RejectsUnknownWidgets(t *testing.T) {
	_, ok := canvasDropdownFor(pfcomponent.FieldMetadata{WidgetKind: pfcomponent.WidgetText})
	assert.False(t, ok)
}

func TestCanvasDropdownWidget_SetOptionsIsIdempotent(t *testing.T) {
	w := newCanvasDropdownWidget(canvasDropdownSpriteRef, pfcomponent.FieldMetadata{Name: "Sprite"})
	w.SetOptions([]string{"a", "b"})
	first := w.Dropdown.Options
	w.SetOptions([]string{"a", "b"})
	assert.Equal(t, first, w.Dropdown.Options)
}

func TestCanvasDropdownWidget_PendingFromCallback(t *testing.T) {
	w := newCanvasDropdownWidget(canvasDropdownEnum, pfcomponent.FieldMetadata{Name: "Tool"})
	w.SetOptions([]string{"select", "place", "delete", "paint"})
	w.Dropdown.SelectByIndex(2)
	val, ok := w.ConsumePending()
	require.True(t, ok)
	assert.Equal(t, "delete", val)

	// Pending is one-shot.
	_, ok = w.ConsumePending()
	assert.False(t, ok)
}

func TestCanvasDropdownWidget_IsEmpty(t *testing.T) {
	w := newCanvasDropdownWidget(canvasDropdownAudioRef, pfcomponent.FieldMetadata{Name: "Audio"})
	assert.True(t, w.IsEmpty())
	w.SetOptions([]string{"a"})
	assert.False(t, w.IsEmpty())
}
