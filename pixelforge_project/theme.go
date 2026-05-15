package pixelforge_project

// Theme holds the palette-slot indices the editor uses for chrome
// colours plus a font reference. Slot indices are palette positions
// (0..63), not raw RGB — this matches the engine's palette-as-
// constraint discipline.
//
// The struct is additive to the schema (v1): missing or null on older
// projects falls back to the zero value, which still renders the editor
// with a reasonable dark theme.
type Theme struct {
	BackgroundSlot  int    `json:"background_slot"`
	PanelSlot       int    `json:"panel_slot"`
	PanelHeaderSlot int    `json:"panel_header_slot"`
	TextSlot        int    `json:"text_slot"`
	TextDimSlot     int    `json:"text_dim_slot"`
	AccentSlot      int    `json:"accent_slot"`
	WarningSlot     int    `json:"warning_slot"`
	FontName        string `json:"font_name"`
}

// DefaultTheme returns the theme used when a project omits the field.
// Slot indices are conservative defaults that work with the engine's
// default palette: dark bg (slot 1), readable text (slot 7), blue
// accent (slot 12).
func DefaultTheme() Theme {
	return Theme{
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

// SanitizeSlots clamps each slot index to the valid palette range so a
// hand-edited theme can't crash the renderer with an out-of-range
// index. We clamp instead of failing because the theme is meant to be
// forgiving: a typo shouldn't break the editor.
func (t *Theme) SanitizeSlots() {
	const maxSlot = 63
	for _, p := range []*int{
		&t.BackgroundSlot, &t.PanelSlot, &t.PanelHeaderSlot,
		&t.TextSlot, &t.TextDimSlot, &t.AccentSlot, &t.WarningSlot,
	} {
		if *p < 0 || *p > maxSlot {
			*p = 0
		}
	}
}
