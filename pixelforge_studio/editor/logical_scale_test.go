package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEditor_LayoutHonorsLogicalScale(t *testing.T) {
	s := DefaultSettings()
	s.LogicalScale = 2
	e := NewWithSettings(s)
	w, h := e.Layout(1920, 1200)
	assert.Equal(t, 960, w)
	assert.Equal(t, 600, h)
}

func TestEditor_LayoutFallsBackWhenWindowTooSmall(t *testing.T) {
	s := DefaultSettings()
	s.LogicalScale = 4
	e := NewWithSettings(s)
	// 4× scale on 600×400 → 150×100 logical, below minimum.
	w, h := e.Layout(600, 400)
	assert.Equal(t, 600, w, "falls back to window dimensions")
	assert.Equal(t, 400, h)
}

func TestEditor_EffectiveLogicalScaleClampsInvalid(t *testing.T) {
	s := DefaultSettings()
	s.LogicalScale = 99
	e := NewWithSettings(s)
	assert.Equal(t, 4, e.EffectiveLogicalScale())

	s.LogicalScale = -3
	assert.Equal(t, 1, e.EffectiveLogicalScale())
}

func TestEditor_SetLogicalScaleClamps(t *testing.T) {
	s := DefaultSettings()
	e := NewWithSettings(s)
	e.SetLogicalScale(0)
	assert.Equal(t, 1, s.LogicalScale)

	e.SetLogicalScale(10)
	assert.Equal(t, 4, s.LogicalScale)
}

func TestEditor_SetLogicalScaleNoOpWhenUnchanged(t *testing.T) {
	s := DefaultSettings()
	s.LogicalScale = 2
	e := NewWithSettings(s)
	e.SetLogicalScale(2)
	assert.Equal(t, 2, s.LogicalScale)
}

func TestDefaultSettings_LogicalScaleDefault(t *testing.T) {
	s := DefaultSettings()
	assert.Equal(t, 1, s.LogicalScale)
}
