package widgets_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/stretchr/testify/assert"
)

func TestCodeBlock_LineCount(t *testing.T) {
	cb := widgets.NewCodeBlock(0, 0, 200, 100, widgets.CodeBlockOptions{
		Text: "line 1\nline 2\nline 3",
	})
	assert.Equal(t, 3, cb.LineCount())
}

func TestCodeBlock_EmptyText(t *testing.T) {
	cb := widgets.NewCodeBlock(0, 0, 200, 100, widgets.CodeBlockOptions{})
	assert.Equal(t, 0, cb.LineCount())
}

func TestCodeBlock_SetText(t *testing.T) {
	cb := widgets.NewCodeBlock(0, 0, 200, 100, widgets.CodeBlockOptions{Text: "a"})
	cb.SetText("b\nc")
	assert.Equal(t, 2, cb.LineCount())
}
