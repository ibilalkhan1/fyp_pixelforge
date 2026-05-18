package editor

// tile_palette.go renders idea #1 v1's tile palette panel inside the
// Scene workspace whenever the Paint tool is active. The panel lists
// the tiles available on the active TilemapLayer's bound
// SpriteSheetRef as clickable buttons; clicking one sets the painter's
// ActiveTileID.
//
// Fallback rendering: the engine-side sheet-decoder pipeline that
// would slice a SpriteSheetRef into individual tile thumbnails does
// not exist yet (U5 of this plan documented the SheetResolver gap).
// For v1 the palette renders a numbered text-button grid ("Tile 0",
// "Tile 1", ...) up to a small default count. When the decoder
// pipeline lands, the palette swaps the text buttons for ImageButton
// thumbnails sourced from the same SheetResolver the canvas already
// owns — the palette's selected-tile state contract stays unchanged,
// which is what the tests pin.

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// tilePaletteFallbackCount is the number of numbered text buttons the
// fallback palette renders when no sheet is bound (or no decoder
// pipeline can slice the bound sheet). 16 tiles is enough to cover
// the typical Mario-class strip's terrain + decoration set without
// flooding the panel. Exposed as a constant so tests reference the
// same source of truth as the renderer.
const tilePaletteFallbackCount = 16

// TilePalette is the editor-side state for the palette panel. v1
// holds only what the renderer reads back each frame: a reference to
// the painter (so clicks can mutate ActiveTileID directly) and a
// nullable layer pointer (so the palette can show "no layer bound"
// when the scene's Tilemaps is empty).
//
// The palette doesn't own the active tile ID — that's the painter's
// concern, and the painter is what the canvas dispatch reads at
// paint time. This keeps the single-source-of-truth in one place
// (TilePainter.ActiveTileID) rather than scattering tile-selection
// state across the editor.
type TilePalette struct {
	Painter *TilePainter
}

// NewTilePalette returns a palette bound to painter. Nil painter is
// allowed (the Render path no-ops); tests that don't care about
// click dispatch can pass nil.
func NewTilePalette(painter *TilePainter) *TilePalette {
	return &TilePalette{Painter: painter}
}

// Render emits the palette's ImGui widgets into the current window.
// The caller (SceneWorkspace.renderToolbar) is responsible for the
// window-begin/end framing; this function just lays out tile
// buttons. layer may be nil — the renderer falls back to a "no layer"
// hint without crashing.
//
// Tile count strategy:
//
//   - When layer.SpriteSheetRef is set AND a sheet resolution
//     pipeline lands in a future unit, the count comes from the
//     decoded sheet (one button per frame). For v1 the resolver
//     pipeline doesn't exist, so we always fall through to the
//     fallback count regardless of binding state.
//   - Fallback: tilePaletteFallbackCount numbered text buttons. The
//     designer can still author meaningful levels — the cell values
//     they paint are integers; the sheet binding only matters at
//     render time (which is also gated behind the same missing
//     decoder).
//
// Clicking a button writes ActiveTileID directly on the painter; no
// MarkDirty fires because selecting a tile isn't a project mutation
// (the project state changes only when the painter actually paints).
func (p *TilePalette) Render(layer *pixelforge_project.TilemapLayer) {
	if p == nil || p.Painter == nil {
		return
	}
	count := tilePaletteCount(layer)

	// Layer hint at the top of the panel so designers know which
	// SpriteSheetRef the cell IDs are addressing (or that none is
	// bound). "no layer" is the worst-case fallback when the scene
	// has no Tilemaps yet — paint-on-nothing is the no-op contract.
	imgui.TextUnformatted(paletteHeader(layer))
	imgui.Separator()

	// Layout: 4 buttons per row gives a 4×4 grid for the default
	// 16-tile fallback, which fits comfortably in a typical
	// inspector-width panel without horizontal scrolling.
	const cols = 4
	for i := 0; i < count; i++ {
		if i%cols != 0 {
			imgui.SameLine()
		}
		label := fmt.Sprintf("Tile %d", i)
		active := p.Painter.ActiveTileID == i
		if active {
			// Visually distinguish the selected tile with a
			// pressed-state radio button rather than a separate
			// pushed-color treatment; keeps the palette readable
			// without theming work.
			imgui.RadioButtonBool(label, true)
			if imgui.IsItemClicked() {
				p.Painter.SetActiveTile(i)
			}
		} else if imgui.Button(label) {
			p.Painter.SetActiveTile(i)
		}
	}
}

// tilePaletteCount picks how many tile buttons to render for layer.
// For v1 the answer is always tilePaletteFallbackCount because the
// sheet decoder doesn't exist yet; the signature accepts layer so a
// future unit can swap in a sheet-derived count without changing
// callers.
func tilePaletteCount(_ *pixelforge_project.TilemapLayer) int {
	return tilePaletteFallbackCount
}

// paletteHeader composes the header line above the tile buttons.
// Designers need to see whether a sheet is bound, even if v1's
// renderer can't actually display it yet — knowing "no sheet, paint
// is just integer IDs" is itself useful context.
func paletteHeader(layer *pixelforge_project.TilemapLayer) string {
	if layer == nil {
		return "Tile Palette (no layer)"
	}
	if layer.SpriteSheetRef == "" {
		return "Tile Palette (no sheet bound)"
	}
	return fmt.Sprintf("Tile Palette: %s", layer.SpriteSheetRef)
}
