package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEditor_ResizeLeftPanel_AppliesDelta(t *testing.T) {
	s := DefaultSettings()
	e := NewWithSettings(s)
	// Drive a layout so chrome.WindowW is populated.
	e.chrome.recompute(1280, 800)
	before := e.chrome.LeftPanelW
	e.ResizeLeftPanel(30)
	assert.Equal(t, before+30, e.chrome.LeftPanelW)
	assert.Equal(t, before+30, s.LeftPanelW)
}

func TestEditor_ResizeLeftPanel_ClampsMin(t *testing.T) {
	s := DefaultSettings()
	e := NewWithSettings(s)
	e.chrome.recompute(1280, 800)
	e.ResizeLeftPanel(-9999)
	assert.GreaterOrEqual(t, e.chrome.LeftPanelW, minLeftPanelW)
}

func TestEditor_ResizeRightPanel_AppliesDelta(t *testing.T) {
	s := DefaultSettings()
	e := NewWithSettings(s)
	e.chrome.recompute(1280, 800)
	before := e.chrome.RightPanelW
	e.ResizeRightPanel(20)
	assert.Equal(t, before+20, e.chrome.RightPanelW)
	assert.Equal(t, before+20, s.RightPanelW)
}

func TestEditor_PersistedPanelWidthsApplyOnStartup(t *testing.T) {
	s := DefaultSettings()
	s.LeftPanelW = 320
	s.RightPanelW = 340
	e := NewWithSettings(s)
	assert.Equal(t, 320, e.chrome.LeftPanelW)
	assert.Equal(t, 340, e.chrome.RightPanelW)
}

func TestChromeLayout_LeftGutterRect(t *testing.T) {
	l := defaultChromeLayout()
	l.recompute(1280, 800)
	g := l.LeftGutterRect()
	// 4px wide; sits at the canvas's left edge minus 2.
	assert.Equal(t, 4, g.W)
	assert.Equal(t, l.Canvas.X-2, g.X)
}

func TestChromeLayout_RightGutterRect(t *testing.T) {
	l := defaultChromeLayout()
	l.recompute(1280, 800)
	g := l.RightGutterRect()
	assert.Equal(t, 4, g.W)
	assert.Equal(t, l.Canvas.X+l.Canvas.W-2, g.X)
}
