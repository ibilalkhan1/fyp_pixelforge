package pixelforge_project

// MaxColors mirrors the engine constant. Duplicated here to keep
// pixelforge_project free of import cycles with pixelforge itself.
const MaxColors = 64

// PaletteData carries the 64-color palette, the 4 ColorTables, named
// non-destructive presets, and animation curves over individual slots.
type PaletteData struct {
	// Base is the 64 RGB colors. Each entry is a "#RRGGBB" hex
	// string for human-readable JSON; round-trips losslessly.
	Base [MaxColors]string `json:"base"`

	// ColorTables is the 4-table mapping grid (table × source × target
	// → palette index). The engine selects a table via (source|target)
	// >> 6 at draw time.
	ColorTables [4][MaxColors][MaxColors]uint8 `json:"color_tables"`

	// Presets are Lightroom-style non-destructive overlays. Each
	// preset overrides a subset of palette slots or color-table cells;
	// the editor composes the active preset stack over the base.
	Presets []ColorTablePreset `json:"presets"`

	// Animations drive palette slots over time, fired by events or by
	// the runtime clock. Populated by the M2 palette animator.
	Animations []PaletteAnimation `json:"animations"`
}

// ColorTablePreset is a named override applied on top of the base
// palette. Either a palette-slot override, a color-table cell override,
// or both. Toggleable from the M2 preset stack panel.
type ColorTablePreset struct {
	Name string `json:"name"`

	// PaletteOverrides[slot] = "#RRGGBB" replaces a base palette color
	// while the preset is active. Missing slots fall through to base.
	PaletteOverrides map[int]string `json:"palette_overrides,omitempty"`

	// ColorTableOverrides[table][source][target] = new palette index.
	// Empty maps round-trip as "{}" not "null" via the saver.
	ColorTableOverrides []ColorTableCellOverride `json:"color_table_overrides,omitempty"`
}

// ColorTableCellOverride targets one cell in one of the four tables.
// Stored as a flat list (not a sparse 3-D map) so JSON diffs stay small
// even when only a handful of cells change.
type ColorTableCellOverride struct {
	Table  int   `json:"table"`
	Source int   `json:"source"`
	Target int   `json:"target"`
	Value  uint8 `json:"value"`
}

// PaletteAnimation drives one palette slot over time. The editor
// surfaces this as a keyframe timeline in M2; the runtime samples the
// curve each frame to push fresh colors into pixelforge.Palette.
type PaletteAnimation struct {
	// Slot is the palette index this animation targets.
	Slot int `json:"slot"`

	// Keyframes is an ordered list of (time, color) tuples. Time is
	// in seconds since trigger; the runtime interpolates with the
	// chosen Easing.
	Keyframes []PaletteKeyframe `json:"keyframes"`

	// Easing names the interpolation curve: "linear", "ease_in",
	// "ease_out", "ease_in_out", "step".
	Easing string `json:"easing"`

	// TriggerEvent fires the animation. Empty means "run on scene
	// start". M5 visual scripting can publish events to drive this.
	TriggerEvent string `json:"trigger_event"`

	// Loop replays the animation after the last keyframe. False holds
	// the final color.
	Loop bool `json:"loop"`
}

// PaletteKeyframe is one (time, color) pair on a PaletteAnimation.
type PaletteKeyframe struct {
	Time  float64 `json:"time"`
	Color string  `json:"color"` // "#RRGGBB"
}

// DefaultPalette returns a fresh palette initialised to a neutral grey
// ramp. The base palette is rebuilt by the M2 importer when the user
// drops their first sprite; until then this gives the runtime
// something safe to render.
func DefaultPalette() PaletteData {
	pd := PaletteData{
		Presets:    []ColorTablePreset{},
		Animations: []PaletteAnimation{},
	}
	// 64 entries of an evenly-spaced grey ramp so the canvas isn't
	// pitch black on a fresh project. We'll keep slot 0 as transparent
	// (black) to match the engine's existing transparency rules.
	for i := 0; i < MaxColors; i++ {
		v := byte((i * 255) / (MaxColors - 1))
		pd.Base[i] = formatRGB(v, v, v)
	}
	// Identity ColorTables: each cell maps source→target as source.
	for t := 0; t < 4; t++ {
		for s := 0; s < MaxColors; s++ {
			for d := 0; d < MaxColors; d++ {
				pd.ColorTables[t][s][d] = uint8(s)
			}
		}
	}
	return pd
}

// formatRGB renders one color as "#RRGGBB". Kept inline to avoid pulling
// the fmt package into the hot serialization path. (Save/Load already
// pays a bigger fixed cost via encoding/json.)
func formatRGB(r, g, b byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 7)
	out[0] = '#'
	out[1] = hex[r>>4]
	out[2] = hex[r&0xf]
	out[3] = hex[g>>4]
	out[4] = hex[g&0xf]
	out[5] = hex[b>>4]
	out[6] = hex[b&0xf]
	return string(out)
}
