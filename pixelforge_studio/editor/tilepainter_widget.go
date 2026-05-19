// tilepainter_widget.go owns the inspector drawer for the TileAtlas
// custom widget registered in registrations.go. The drawer renders
// (top to bottom):
//
//  1. A header line showing the bound SpriteSheetRef + active-rules
//     count.
//  2. Three Brush / Bucket / Rect radio buttons that mirror the
//     TilePainter's SubMode.
//  3. A tile palette grid that picks the active tile ID.
//  4. Undo / Redo buttons backed by the editor's UndoStack.
//
// State flows: clicks here write into TilePainter via the *Editor
// accessors (SelectedTile / SetSelectedTile / PaintSubMode /
// SetPaintSubMode). The canvas's ToolPaint dispatch (U6) reads the
// same accessors when LMB events arrive, so the two surfaces always
// agree on the active tile and sub-mode.
//
// The widget is dispatched through pfcomponent.WidgetCustom. The
// drawer receives the typed *TileAtlas via DrawerContext.Owner when
// the inspector dispatches from the scene-level TileAtlas surface;
// when no typed owner is passed (e.g. the field is dispatched from
// an entity-component path that doesn't apply here), the drawer
// degrades gracefully to a "(select a TileAtlas)" hint.
package editor

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pfcomponent"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// tilepainterEditorKey is the DrawerContext.Extras key the dispatch
// path uses to inject the live *Editor pointer. Documented as a
// stable key so the editor and the drawer agree on the contract
// without forcing pfcomponent to know about the editor type.
const tilepainterEditorKey = "editor"

// tilepainterColumns is the number of tile-palette buttons rendered
// per row. Four columns matches the existing Scene-toolbar palette
// (tile_palette.go) so designers see a consistent layout no matter
// which surface they pick the active tile from.
const tilepainterColumns = 4

// tilepainterFallbackTiles is the v1 fallback tile count for the
// palette grid when no sprite-sheet decoder pipeline can slice the
// bound SpriteSheetRef. Matches tilePaletteFallbackCount.
const tilepainterFallbackTiles = tilePaletteFallbackCount

// tilepainterDraw renders the painter widget inside the inspector.
// Returns false unconditionally: the widget mutates session-only UI
// state (active tile, sub-mode) which never marks the project
// dirty per docs/solutions/dirty-state-ux.md. Undo / Redo mutate
// the project but route through the existing UndoStack which calls
// MarkDirty itself.
func tilepainterDraw(ctx pfcomponent.DrawerContext) bool {
	atlas, _ := ctx.Owner.(*pixelforge_project.TileAtlas)
	if atlas == nil {
		imgui.TextDisabled("(select a TileAtlas to paint)")
		return false
	}
	editor, _ := ctx.Extras[tilepainterEditorKey].(*Editor)

	tilepainterHeader(atlas)
	imgui.Separator()
	tilepainterSubModePicker(editor)
	imgui.Separator()
	tilepainterTilePalette(editor, atlas)
	imgui.Separator()
	tilepainterUndoRedo(editor)
	imgui.Separator()
	tilepainterActiveRulesIndicator(atlas)
	return false
}

// tilepainterHeader prints a one-line summary of which sprite sheet
// the atlas is bound to. Empty bindings produce a hint so designers
// know the cell IDs they paint are purely integers until a sheet is
// attached.
func tilepainterHeader(atlas *pixelforge_project.TileAtlas) {
	switch {
	case atlas.SpriteSheetRef == "":
		imgui.TextUnformatted("Tile Painter — no sprite sheet bound")
	default:
		imgui.TextUnformatted("Tile Painter — sheet: " + atlas.SpriteSheetRef)
	}
}

// tilepainterSubModePicker emits the three radio buttons that pick
// the active paint sub-mode. Reads / writes through *Editor so the
// toolbar palette in the Scene workspace stays in sync.
func tilepainterSubModePicker(e *Editor) {
	if e == nil {
		imgui.TextDisabled("(no editor context — sub-mode unavailable)")
		return
	}
	modes := []struct {
		label string
		mode  PaintSubMode
	}{
		{"Brush", PaintBrush},
		{"Bucket", PaintBucket},
		{"Rectangle", PaintRectangle},
	}
	current := e.PaintSubMode()
	for i, m := range modes {
		if i > 0 {
			imgui.SameLine()
		}
		if imgui.RadioButtonBool(m.label, current == m.mode) {
			e.SetPaintSubMode(m.mode)
		}
	}
}

// tilepainterTilePalette renders the active tile picker grid. Click
// selects the tile ID; the canvas dispatch reads it via
// e.SelectedTile() when painting.
func tilepainterTilePalette(e *Editor, atlas *pixelforge_project.TileAtlas) {
	imgui.TextUnformatted(tilepainterPaletteHeader(atlas))
	count := tilepainterTileCount(atlas)
	selected := -1
	if e != nil {
		selected = e.SelectedTile()
	}
	for i := 0; i < count; i++ {
		if i%tilepainterColumns != 0 {
			imgui.SameLine()
		}
		label := fmt.Sprintf("T%d", i)
		isSelected := i == selected
		if isSelected {
			// Pressed-state radio button to highlight the active tile;
			// matches the Scene-toolbar palette convention.
			imgui.RadioButtonBool(label, true)
			if e != nil && imgui.IsItemClicked() {
				e.SetSelectedTile(i)
			}
			continue
		}
		if imgui.Button(label) && e != nil {
			e.SetSelectedTile(i)
		}
	}
}

// tilepainterPaletteHeader composes the palette's header line so a
// designer who hasn't bound a sheet still understands "paint = write
// integer IDs into the grid; rendering needs a sheet to look right."
func tilepainterPaletteHeader(atlas *pixelforge_project.TileAtlas) string {
	if atlas.SpriteSheetRef == "" {
		return "Palette (no sheet — IDs only)"
	}
	return "Palette"
}

// tilepainterTileCount picks how many tile buttons to render. v1
// hard-codes the fallback count because the sheet-decoder pipeline
// that would yield a sheet-derived count doesn't exist yet (same
// reasoning as tile_palette.go's tilePaletteCount).
func tilepainterTileCount(_ *pixelforge_project.TileAtlas) int {
	return tilepainterFallbackTiles
}

// tilepainterUndoRedo emits the two undo / redo buttons. Both wire
// into the editor's UndoStack, which performs the project mutation
// and fires MarkDirty itself — the drawer doesn't double-mark.
func tilepainterUndoRedo(e *Editor) {
	if e == nil || e.UndoStack() == nil {
		imgui.TextDisabled("(undo unavailable)")
		return
	}
	stack := e.UndoStack()
	if imgui.Button("Undo") {
		stack.Undo()
	}
	imgui.SameLine()
	if imgui.Button("Redo") {
		stack.Redo()
	}
}

// tilepainterActiveRulesIndicator reports how many of the atlas's
// AutoTileRules are currently active (Count >= threshold). The
// indicator is read-only in v1 — rule management lands in v2 per
// the scope boundaries. Designers can hover the line to see a
// short summary of the active rules' Output values, which helps
// them recognise what the synth has been learning while they paint.
func tilepainterActiveRulesIndicator(atlas *pixelforge_project.TileAtlas) {
	active := tilepainterCountActiveRules(atlas)
	switch active {
	case 0:
		imgui.TextDisabled("Auto-rules: none active yet")
	case 1:
		imgui.TextUnformatted("Auto-rules: 1 active")
	default:
		imgui.TextUnformatted(fmt.Sprintf("Auto-rules: %d active", active))
	}
	if imgui.IsItemHovered() && active > 0 {
		imgui.SetTooltip(tilepainterActiveRulesTooltip(atlas))
	}
}

// tilepainterCountActiveRules returns the number of auto-rules that
// have crossed the activation threshold. Pulled out as a helper so
// tests can assert on the count without driving imgui.
func tilepainterCountActiveRules(atlas *pixelforge_project.TileAtlas) int {
	n := 0
	for _, r := range atlas.AutoTileRules {
		if r.Count >= activeRuleThreshold() {
			n++
		}
	}
	return n
}

// SetTileAtlasBlockPalette writes the supplied sub-palette index
// into atlas.NESPaletteBlock at the block coordinates derived from
// the supplied tile cell (col, row). Auto-grows the
// NESPaletteBlock matrix when the destination block falls outside
// the existing bounds, padding new entries with the unassigned
// sentinel (-1).
//
// Returns true when the matrix actually changed so callers can
// decide whether to MarkDirty. Designer interactions go through
// this helper (instead of mutating NESPaletteBlock directly) so the
// growth + sentinel discipline lives in one place.
//
// Block coordinates: NES authoring works in 2x2 tile blocks, so
// blockRow = tileRow/2, blockCol = tileCol/2. A tile click at
// (5, 7) writes blockRow=3, blockCol=2.
func SetTileAtlasBlockPalette(atlas *pixelforge_project.TileAtlas, tileCol, tileRow, subPaletteIndex int) bool {
	if atlas == nil {
		return false
	}
	if tileCol < 0 || tileRow < 0 {
		return false
	}
	if subPaletteIndex < UnassignedNESPaletteBlock || subPaletteIndex >= NESPaletteBlockMaxIndex {
		return false
	}
	blockRow := tileRow / 2
	blockCol := tileCol / 2

	for len(atlas.NESPaletteBlock) <= blockRow {
		atlas.NESPaletteBlock = append(atlas.NESPaletteBlock, []int{})
	}
	for len(atlas.NESPaletteBlock[blockRow]) <= blockCol {
		atlas.NESPaletteBlock[blockRow] = append(atlas.NESPaletteBlock[blockRow], UnassignedNESPaletteBlock)
	}

	old := atlas.NESPaletteBlock[blockRow][blockCol]
	if old == subPaletteIndex {
		return false
	}
	atlas.NESPaletteBlock[blockRow][blockCol] = subPaletteIndex
	return true
}

// LookupTileAtlasBlockPalette returns the sub-palette index assigned
// to the 2x2 block containing tile (col, row). Returns
// UnassignedNESPaletteBlock (-1) when the block has not been
// explicitly assigned. Out-of-range coordinates also return the
// unassigned sentinel so callers don't have to bound-check.
func LookupTileAtlasBlockPalette(atlas *pixelforge_project.TileAtlas, tileCol, tileRow int) int {
	if atlas == nil || tileCol < 0 || tileRow < 0 {
		return UnassignedNESPaletteBlock
	}
	blockRow := tileRow / 2
	blockCol := tileCol / 2
	if blockRow >= len(atlas.NESPaletteBlock) {
		return UnassignedNESPaletteBlock
	}
	row := atlas.NESPaletteBlock[blockRow]
	if blockCol >= len(row) {
		return UnassignedNESPaletteBlock
	}
	return row[blockCol]
}

// UnassignedNESPaletteBlock is the sentinel value the per-block
// palette picker (and the U7 overlay) treat as "this block has no
// explicit assignment, render with bg_0 by default." Distinguished
// from 0 (explicit bg_0 choice) so the overlay can flag genuine
// gaps without false-positive on intentional bg_0 picks.
const UnassignedNESPaletteBlock = -1

// NESPaletteBlockMaxIndex is the exclusive upper bound for valid
// sub-palette indices. Project carries 4 BG sub-palettes so legal
// values are 0..3; the picker rejects anything outside that range.
const NESPaletteBlockMaxIndex = 4

// tilepainterActiveRulesTooltip composes a short summary of each
// active rule's Output value. Designers don't see the patterns
// themselves (those are 3x3 int arrays — not designer-readable);
// outputs are the actionable bit.
func tilepainterActiveRulesTooltip(atlas *pixelforge_project.TileAtlas) string {
	out := ""
	for _, r := range atlas.AutoTileRules {
		if r.Count < activeRuleThreshold() {
			continue
		}
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("→tile %d", r.Output)
	}
	return "Active outputs: " + out
}

// activeRuleThreshold reads the palette package's exported
// AutoTileActivationThreshold via the package import in U6 of plan-
// 003; here we hard-code it to 3 to avoid a dependency on the
// palette package from this file. Kept in lockstep with the constant
// via tilepainterActiveThresholdAlignment in tilepainter_widget_test.go.
func activeRuleThreshold() int {
	return 3
}
