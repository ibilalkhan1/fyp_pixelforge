package editor

import (
	"bytes"
	"embed"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

//go:embed cart_assets/editor.pforge
var editorCartFS embed.FS

// LoadEmbeddedEditorProject parses the embedded editor.pforge fixture
// and returns the in-memory project. The project's Theme drives chrome
// colours; its asset list (currently empty) seeds the cart's sprite
// references.
//
// Returns nil on parse failure so the cart falls back to its default
// theme; the failure is logged but never fatal — a broken fixture must
// not crash the studio.
func LoadEmbeddedEditorProject() *pixelforge_project.Project {
	data, err := editorCartFS.ReadFile("cart_assets/editor.pforge")
	if err != nil {
		log.Printf("pixelforge_studio: failed to read embedded editor.pforge: %v", err)
		return nil
	}
	p, err := pixelforge_project.LoadReader(bytes.NewReader(data))
	if err != nil {
		log.Printf("pixelforge_studio: failed to parse embedded editor.pforge: %v", err)
		return nil
	}
	return p
}

// loadEditorTheme returns the theme the cart uses for its chrome.
// Sourced from the embedded editor.pforge fixture (R2 dogfooding);
// falls back to DefaultEditorTheme if the fixture fails to load.
func loadEditorTheme() *EditorTheme {
	p := LoadEmbeddedEditorProject()
	if p == nil {
		return DefaultEditorTheme()
	}
	return themeFromProject(p.Theme)
}

func themeFromProject(t pixelforge_project.Theme) *EditorTheme {
	return &EditorTheme{
		BackgroundSlot:  uint8(t.BackgroundSlot),
		PanelSlot:       uint8(t.PanelSlot),
		PanelHeaderSlot: uint8(t.PanelHeaderSlot),
		TextSlot:        uint8(t.TextSlot),
		TextDimSlot:     uint8(t.TextDimSlot),
		AccentSlot:      uint8(t.AccentSlot),
		WarningSlot:     uint8(t.WarningSlot),
		FontName:        t.FontName,
	}
}
