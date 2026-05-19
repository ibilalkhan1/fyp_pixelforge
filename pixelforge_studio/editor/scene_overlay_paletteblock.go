// scene_overlay_paletteblock.go owns idea #3 v1 U7's 2x2 BG
// palette-block consistency overlay. The overlay outlines tile
// blocks where painted cells exist but no explicit
// NESPaletteBlock assignment has been made — those blocks would
// render with bg_0 by default on the real NES, which is rarely
// the designer's intent.
//
// v1 flags "painted but unassigned" blocks. When v2 adds per-tile
// palette overrides, the same overlay extends to flag genuine
// intra-block conflicts (e.g. tile A bound to bg_1, tile B bound
// to bg_2 in the same 2x2 block).
//
// Studio-only by construction: called from sceneGame.Draw which
// never reaches the shipped runtime.
package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// PaletteBlockOverlayColor is the outline color for unassigned
// palette blocks. Yellow with full alpha — bright enough to spot
// without obscuring the underlying tile content.
var PaletteBlockOverlayColor = color.RGBA{R: 0xff, G: 0xd0, B: 0x40, A: 0xff}

// PaletteBlockViolation describes one 2x2 block that surfaces a
// soft-warn outline. BlockCol / BlockRow are coords in the 2x2-
// block grid (tileCol/2, tileRow/2); PixelX / PixelY name the
// top-left scene-space pixel coordinate the outline rectangle
// starts at; PixelW / PixelH name the rectangle's size (always
// 2 * tile dimensions for the 2x2 block).
type PaletteBlockViolation struct {
	BlockCol int
	BlockRow int
	PixelX   int
	PixelY   int
	PixelW   int
	PixelH   int
}

// FindPaletteBlockViolations walks the TileAtlas in 2x2 windows
// and returns every block where at least one tile is painted
// (Grid[r][c] != 0) AND the corresponding NESPaletteBlock entry is
// the unassigned sentinel (-1). Empty blocks (no painted tiles)
// don't violate — empty regions naturally render as the default
// background.
func FindPaletteBlockViolations(atlas *pixelforge_project.TileAtlas) []PaletteBlockViolation {
	if atlas == nil || len(atlas.Grid) == 0 {
		return nil
	}
	tw := atlas.TileW
	if tw <= 0 {
		tw = DefaultSpriteHeight
	}
	th := atlas.TileH
	if th <= 0 {
		th = DefaultSpriteHeight
	}
	var out []PaletteBlockViolation
	for blockRow := 0; blockRow*2 < len(atlas.Grid); blockRow++ {
		for blockCol := 0; blockCol*2 < maxRowLen(atlas.Grid); blockCol++ {
			if !blockHasPaintedCell(atlas.Grid, blockCol, blockRow) {
				continue
			}
			if blockExplicitlyAssigned(atlas, blockRow, blockCol) {
				continue
			}
			out = append(out, PaletteBlockViolation{
				BlockCol: blockCol,
				BlockRow: blockRow,
				PixelX:   blockCol * 2 * tw,
				PixelY:   blockRow * 2 * th,
				PixelW:   2 * tw,
				PixelH:   2 * th,
			})
		}
	}
	return out
}

// blockHasPaintedCell reports whether any of the 2x2 tile cells at
// block (blockCol, blockRow) carries a non-zero tile ID.
func blockHasPaintedCell(grid [][]int, blockCol, blockRow int) bool {
	for dr := 0; dr < 2; dr++ {
		row := blockRow*2 + dr
		if row >= len(grid) {
			break
		}
		for dc := 0; dc < 2; dc++ {
			col := blockCol*2 + dc
			if col >= len(grid[row]) {
				break
			}
			if grid[row][col] != 0 {
				return true
			}
		}
	}
	return false
}

// blockExplicitlyAssigned reports whether NESPaletteBlock has a
// non-sentinel value at the supplied block. The sentinel (-1) means
// "unassigned"; 0..3 mean "explicitly chosen sub-palette." Default-
// zero in a never-populated matrix slot is also the unassigned case
// when the slot is out of range.
func blockExplicitlyAssigned(atlas *pixelforge_project.TileAtlas, blockRow, blockCol int) bool {
	if blockRow >= len(atlas.NESPaletteBlock) {
		return false
	}
	row := atlas.NESPaletteBlock[blockRow]
	if blockCol >= len(row) {
		return false
	}
	v := row[blockCol]
	return v >= 0 && v < NESPaletteBlockMaxIndex
}

func maxRowLen(grid [][]int) int {
	max := 0
	for _, row := range grid {
		if len(row) > max {
			max = len(row)
		}
	}
	return max
}

// PaintPaletteBlockOverlay outlines every violation in dst. Skipped
// when overlays.PaletteBlockEnabled is false or no atlas exists.
func PaintPaletteBlockOverlay(dst *ebiten.Image, scene *pixelforge_project.Scene, overlays pixelforge_project.EditorOverlays) {
	if dst == nil || scene == nil {
		return
	}
	if !overlays.PaletteBlockEnabled {
		return
	}
	if len(scene.TileAtlases) == 0 {
		return
	}
	atlas := &scene.TileAtlases[0]
	violations := FindPaletteBlockViolations(atlas)
	for _, v := range violations {
		paintOutline(dst, v.PixelX, v.PixelY, v.PixelW, v.PixelH)
	}
}

func paintOutline(dst *ebiten.Image, x, y, w, h int) {
	vector.StrokeRect(
		dst,
		float32(x), float32(y),
		float32(w), float32(h),
		1.5,
		PaletteBlockOverlayColor,
		false,
	)
}

// PaletteBlockOverlayTooltip is the explanatory copy designers see
// when hovering an outlined block. Names the constraint in
// designer-readable language.
const PaletteBlockOverlayTooltip = "Real NES requires every 2x2 background block share one palette quadrant."
