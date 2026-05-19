package pixelforge_project

// editor_overlays.go owns the per-project soft-warn overlay flags
// idea #3 v1 introduces. Designers can toggle each overlay from the
// View menu; the toggle state persists with the .pforge file so a
// project opened on another machine starts with the same set of
// authoring-time hints enabled.
//
// Persistence detail: omitempty would discard the struct when both
// flags are false (a designer who turned both off would see them
// flip back on after reload). To keep the round-trip honest, the
// custom MarshalJSON below emits the struct whenever the project
// has *ever* touched the overlays — concretely, whenever
// EditorOverlaysSet is true. applyDefaults flips the flags on for
// new projects and sets the Set bit so subsequent saves emit a
// stable shape.

// EditorOverlays carries the per-project soft-warn overlay toggle
// state. Two flags in v1: the 8-sprites-per-scanline warning (red
// horizontal band) and the 2x2 BG palette-block consistency check
// (yellow outline). Both default to true via applyDefaults.
type EditorOverlays struct {
	// ScanlineEnabled gates the 8-sprite-per-scanline overlay (U6).
	ScanlineEnabled bool `json:"scanline_enabled,omitempty"`

	// PaletteBlockEnabled gates the 2x2 BG palette-block consistency
	// overlay (U7).
	PaletteBlockEnabled bool `json:"palette_block_enabled,omitempty"`

	// Set is a private-by-naming-convention sentinel that records
	// "this project has explicitly authored its overlay state."
	// Without it the omitempty discipline would drop a
	// {false,false} struct and the next load would default both
	// flags back on, losing the designer's intent. applyDefaults
	// flips Set true on first load so a freshly-defaulted project
	// also round-trips faithfully.
	Set bool `json:"set,omitempty"`
}

// DefaultEditorOverlays returns the canonical state new projects
// start in: both overlays enabled, Set true so the struct survives
// round-trip after a designer toggles either flag off.
func DefaultEditorOverlays() EditorOverlays {
	return EditorOverlays{
		ScanlineEnabled:     true,
		PaletteBlockEnabled: true,
		Set:                 true,
	}
}

// applyDefaults ensures every loaded project carries explicit
// overlay state. Pre-v1 files (Set == false) receive the default
// "both enabled" assignment; projects that already wrote their
// state (Set == true) preserve their flags exactly.
func (o *EditorOverlays) applyDefaults() {
	if o == nil {
		return
	}
	if !o.Set {
		*o = DefaultEditorOverlays()
	}
}
