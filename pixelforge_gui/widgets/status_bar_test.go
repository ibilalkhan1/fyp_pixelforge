package widgets

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
)

func TestStatusBar_DefaultColors(t *testing.T) {
	sb := NewStatusBar()
	assert.Equal(t, pixelforge.Color(7), sb.FgColor)
	assert.Equal(t, pixelforge.Color(5), sb.BgColor)
}

func TestStatusBar_SetBounds(t *testing.T) {
	sb := NewStatusBar()
	sb.SetBounds(10, 20, 300, 18)
	assert.Equal(t, 10, sb.X)
	assert.Equal(t, 20, sb.Y)
	assert.Equal(t, 300, sb.W)
	assert.Equal(t, 18, sb.H)
}

func TestStatusBar_DrawWithNoTextDoesNotPanic(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	sb := NewStatusBar()
	sb.SetBounds(0, 0, 320, 18)
	sb.Draw()
}

func TestStatusBar_DrawLongHintTruncates(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	sb := NewStatusBar()
	sb.SetBounds(0, 0, 80, 18)
	sb.Hint = "this hint is much longer than the bar can hold without truncation"
	sb.Draw()
}

func TestTruncateWithEllipsis(t *testing.T) {
	assert.Equal(t, "hello", truncateWithEllipsis("hello", 100))
	assert.Equal(t, "hello world.", truncateWithEllipsis("hello world is a long string", 12))
	assert.Equal(t, "", truncateWithEllipsis("anything", 0))
}
