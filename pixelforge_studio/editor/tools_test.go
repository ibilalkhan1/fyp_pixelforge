package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// SetTool / Tool round-trip cleanly through the editor.
func TestEditor_ToolGetterSetter(t *testing.T) {
	e := New()
	assert.Equal(t, ToolSelect, e.Tool())
	e.SetTool(ToolPlace)
	assert.Equal(t, ToolPlace, e.Tool())
	e.SetTool(ToolDelete)
	assert.Equal(t, ToolDelete, e.Tool())
	e.SetTool(ToolPaint)
	assert.Equal(t, ToolPaint, e.Tool())
}

// Tool.String produces the labels the status bar prints.
func TestTool_StringLabels(t *testing.T) {
	cases := map[Tool]string{
		ToolSelect: "select",
		ToolPlace:  "place",
		ToolDelete: "delete",
		ToolPaint:  "paint",
	}
	for tool, want := range cases {
		assert.Equal(t, want, tool.String())
	}
	assert.Equal(t, "?", Tool(99).String())
}
