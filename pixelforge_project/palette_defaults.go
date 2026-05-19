package pixelforge_project

// palette_defaults.go owns the load-time backfill discipline idea #3
// v1 introduces for PaletteData. Pre-v1 .pforge files (and any
// project that omits the new sub-palette overlays) get the canonical
// bg_0..bg_3 + sprite_0..sprite_3 assignments applied at load so the
// inspector and the new palette workspace see populated overlays
// without manual editing.
//
// applyDefaults follows the additive-omitempty + sanitize discipline
// in docs/solutions/editor-pforge-schema-shape.md: zero-valued
// overlays receive defaults, out-of-range slot indices clamp to
// 0..MaxColors-1, no input is ever rejected.

// ApplyDefaults backfills the idea-#3 sub-palette overlays when they
// are zero-valued, then clamps every slot index into the legal range
// [0, MaxColors). Idempotent: re-running on a populated palette
// preserves designer overrides because the populated-check is a
// per-name string compare, not a per-slot fight against existing
// data.
//
// Exported so packages outside pixelforge_project (e.g.
// pixelforge_studio/palette's auto-pick tests) can apply the
// canonical defaults to a freshly-constructed project without
// going through the full Load() pipeline.
//
// Called from Project.applyDefaults (the existing load-time hook in
// project.go) so legacy projects pick up the overlays automatically.
func (p *PaletteData) ApplyDefaults() {
	if p == nil {
		return
	}
	if isZeroSubPaletteArray(p.BGSubPalettes) {
		p.BGSubPalettes = DefaultBGSubPalettes()
	}
	if isZeroSubPaletteArray(p.SpriteSubPalettes) {
		p.SpriteSubPalettes = DefaultSpriteSubPalettes()
	}
	clampSubPaletteSlots(p.BGSubPalettes[:])
	clampSubPaletteSlots(p.SpriteSubPalettes[:])
}

// DefaultBGSubPalettes returns the canonical background-sub-palette
// assignment idea #3 v1 ships with. Slot 0 stays reserved
// transparent (per palette.go conventions); slots 1..16 distribute
// 4 per sub-palette in linear order so the defaults are predictable
// and easy to spot in the Palette workspace.
func DefaultBGSubPalettes() [4]SubPalette {
	return [4]SubPalette{
		{Name: "bg_0", Slots: [4]int{1, 2, 3, 4}},
		{Name: "bg_1", Slots: [4]int{5, 6, 7, 8}},
		{Name: "bg_2", Slots: [4]int{9, 10, 11, 12}},
		{Name: "bg_3", Slots: [4]int{13, 14, 15, 16}},
	}
}

// DefaultSpriteSubPalettes returns the canonical sprite-sub-palette
// assignment. Slots 17..32 are reserved so the sprite overlays don't
// fight the background overlays for the same base colors on a fresh
// project.
func DefaultSpriteSubPalettes() [4]SubPalette {
	return [4]SubPalette{
		{Name: "sprite_0", Slots: [4]int{17, 18, 19, 20}},
		{Name: "sprite_1", Slots: [4]int{21, 22, 23, 24}},
		{Name: "sprite_2", Slots: [4]int{25, 26, 27, 28}},
		{Name: "sprite_3", Slots: [4]int{29, 30, 31, 32}},
	}
}

// DefaultSubPaletteName is the sprite_0 fallback the SpriteAsset
// loader uses when an asset has no explicit SubPalette set. Exposed
// so tests + the inspector dropdown can share one source of truth.
const DefaultSubPaletteName = "sprite_0"

// isZeroSubPaletteArray reports whether the supplied 4-slot array
// has never been populated. A populated array always has at least
// one Name set (defaults always carry names); the zero-array check
// is therefore a Name == "" check on every entry. This is more
// robust than a full-struct comparison because designer-cleared
// arrays (all Names blanked) round-trip the same way as never-set
// arrays.
func isZeroSubPaletteArray(arr [4]SubPalette) bool {
	for _, sp := range arr {
		if sp.Name != "" {
			return false
		}
	}
	return true
}

// clampSubPaletteSlots repairs out-of-range slot indices in every
// sub-palette of the supplied slice. Values outside [0, MaxColors)
// clamp to the nearer end so the inspector and the renderer never
// see a panic-inducing index. Per the sanitize discipline: repair,
// never reject.
func clampSubPaletteSlots(palettes []SubPalette) {
	for i := range palettes {
		for j := range palettes[i].Slots {
			s := palettes[i].Slots[j]
			if s < 0 {
				s = 0
			}
			if s >= MaxColors {
				s = MaxColors - 1
			}
			palettes[i].Slots[j] = s
		}
	}
}
