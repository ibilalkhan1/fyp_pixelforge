// scene_overlay_scanline.go owns idea #3 v1 U6's 8-sprites-per-
// scanline soft-warn overlay. The overlay paints a red horizontal
// band into the scene preview ebiten.Image whenever ≥9 entities
// overlap any Y row (the NES PPU's hardware limit is 8 sprites per
// scanline; the 9th would flicker or drop).
//
// Studio-only by construction: this file lives in the editor
// package, called only from sceneGame.Draw which never reaches the
// shipped runtime. No `if release {}` flag, no conditional gate —
// the boundary is structural.
//
// Tests exercise the count + violation-range helpers; rendering
// itself is asserted via pixel-read in U9's e2e tests.
package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// ScanlineThreshold is the per-row entity count that triggers the
// warning band. The NES PPU's hardware cap is 8 sprites per
// scanline; the 9th and later flicker. Strictly-greater-than 8
// means painting starts at 9.
const ScanlineThreshold = 8

// DefaultSpriteHeight is the px height assumed for entities that
// don't carry an explicit sprite reference. Matches the NES sprite
// 8x8 default.
const DefaultSpriteHeight = 8

// ScanlineBand describes one contiguous Y range where the overlay
// should paint. The range is inclusive of YStart, exclusive of
// YEnd (matches Go's stdlib half-open convention).
type ScanlineBand struct {
	YStart int
	YEnd   int
}

// ScanlineOverlayColor is the fill color the band paints in. Red
// with ~50% alpha keeps the scene preview visible underneath while
// drawing the eye to the violating rows.
var ScanlineOverlayColor = color.RGBA{R: 0xff, G: 0x40, B: 0x40, A: 0x80}

// CountScanlineOccupancy returns a per-Y-pixel count of entities
// whose visible Y range covers that row. screenHeight bounds the
// returned slice; entities clamp into [0, screenHeight). Entities
// with negative TileY don't contribute (out-of-scene).
//
// The contract is "if counts[y] > ScanlineThreshold, the row
// violates" — callers pair this with ScanlineViolationRanges to
// turn per-row counts into paintable bands.
func CountScanlineOccupancy(entities []pixelforge_project.Entity, screenHeight, tileHeight int) []int {
	if screenHeight <= 0 {
		return nil
	}
	counts := make([]int, screenHeight)
	if tileHeight <= 0 {
		tileHeight = DefaultSpriteHeight
	}
	for _, e := range entities {
		startY := e.TileY * tileHeight
		endY := startY + DefaultSpriteHeight
		if startY < 0 {
			startY = 0
		}
		if endY > screenHeight {
			endY = screenHeight
		}
		for y := startY; y < endY; y++ {
			counts[y]++
		}
	}
	return counts
}

// ScanlineViolationRanges scans a per-Y count list and returns the
// contiguous ranges where counts[y] > threshold. Adjacent violating
// rows merge into one band so the overlay doesn't paint a striped
// pattern when many entities cluster in a horizontal block.
func ScanlineViolationRanges(counts []int, threshold int) []ScanlineBand {
	if len(counts) == 0 {
		return nil
	}
	var bands []ScanlineBand
	inBand := false
	start := 0
	for y, c := range counts {
		violates := c > threshold
		switch {
		case violates && !inBand:
			start = y
			inBand = true
		case !violates && inBand:
			bands = append(bands, ScanlineBand{YStart: start, YEnd: y})
			inBand = false
		}
	}
	if inBand {
		bands = append(bands, ScanlineBand{YStart: start, YEnd: len(counts)})
	}
	return bands
}

// PaintScanlineOverlay renders the violation bands into dst. Skipped
// when overlays.ScanlineEnabled is false or the scene is empty.
// Called from sceneGame.Draw after the entity render so the bands
// composite on top.
func PaintScanlineOverlay(dst *ebiten.Image, scene *pixelforge_project.Scene, overlays pixelforge_project.EditorOverlays) {
	if dst == nil || scene == nil {
		return
	}
	if !overlays.ScanlineEnabled {
		return
	}
	if len(scene.Entities) <= ScanlineThreshold {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	tileHeight := DefaultSpriteHeight
	counts := CountScanlineOccupancy(scene.Entities, h, tileHeight)
	bands := ScanlineViolationRanges(counts, ScanlineThreshold)
	for _, b := range bands {
		paintBand(dst, w, b.YStart, b.YEnd-b.YStart)
	}
	_ = w
}

func paintBand(dst *ebiten.Image, width, y, height int) {
	vector.DrawFilledRect(
		dst,
		0, float32(y),
		float32(width), float32(height),
		ScanlineOverlayColor,
		false,
	)
}

// ScanlineOverlayTooltip is the explanatory text the View menu's
// hover hint and the band's mouse-hover tooltip render. Designers
// don't have to read NESdev forums to understand why the band is
// there — the editor explains it inline.
const ScanlineOverlayTooltip = "NES would flicker — more than 8 sprites on this scanline."
