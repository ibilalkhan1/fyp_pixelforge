package editor

import "github.com/ibilalkhan1/fyp_pixelforge"

// EditorTheme holds the editor's chrome palette + font reference. It is
// populated by loading editor.pforge at startup (U27). Zero values
// fall back to the M0-M2 dark theme palette so the cart works even
// without a loaded fixture.
type EditorTheme struct {
	BackgroundSlot  pixelforge.Color
	PanelSlot       pixelforge.Color
	PanelHeaderSlot pixelforge.Color
	TextSlot        pixelforge.Color
	TextDimSlot     pixelforge.Color
	AccentSlot      pixelforge.Color
	WarningSlot     pixelforge.Color
	FontName        string
}

// DefaultEditorTheme returns the theme the cart uses before
// editor.pforge has been loaded. Slot indices match a sensible default
// pixelforge palette mapping (dark bg, light text, blue accent).
func DefaultEditorTheme() *EditorTheme {
	return &EditorTheme{
		BackgroundSlot:  1,
		PanelSlot:       2,
		PanelHeaderSlot: 5,
		TextSlot:        7,
		TextDimSlot:     6,
		AccentSlot:      12,
		WarningSlot:     9,
		FontName:        "cofont",
	}
}
