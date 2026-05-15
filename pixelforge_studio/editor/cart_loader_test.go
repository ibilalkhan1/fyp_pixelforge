package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmbeddedEditorProject(t *testing.T) {
	p := LoadEmbeddedEditorProject()
	require.NotNil(t, p, "embedded editor.pforge must parse")
	assert.Equal(t, "editor", p.Name)
	assert.Equal(t, EditorCanvasW, p.ScreenWidth)
	assert.Equal(t, EditorCanvasH, p.ScreenHeight)
}

func TestLoadEditorTheme_PopulatesNonZeroSlots(t *testing.T) {
	th := loadEditorTheme()
	require.NotNil(t, th)
	// Theme should come from the embedded fixture; cofont font name proves R2.
	assert.Equal(t, "cofont", th.FontName)
	assert.NotEqual(t, uint8(0), th.TextSlot)
}

func TestCartUsesLoadedTheme(t *testing.T) {
	e := New()
	require.NotNil(t, e.Cart())
	require.NotNil(t, e.Cart().Theme())
	assert.Equal(t, "cofont", e.Cart().Theme().FontName,
		"cart's theme should be sourced from the embedded editor.pforge")
}
