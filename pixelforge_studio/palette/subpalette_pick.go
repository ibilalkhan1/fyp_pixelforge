// subpalette_pick.go owns idea #3 v1 U2's auto-pick algorithm:
// given an image and a project, return the sub-palette (BG or
// Sprite) whose 4 slots best fit the image's color content. Used
// by the import pipeline when the designer doesn't pre-select a
// target sub-palette.
//
// Algorithm: for each candidate sub-palette in the requested family
// (4 candidates), compute the mean per-pixel RGB Euclidean distance
// from each sampled pixel to the nearest of that sub-palette's 4
// colors. Return the sub-palette with the lowest total distance.
//
// Subsampling: images larger than 64x64 are sampled on a 16x16
// regular grid (~256 samples) to bound runtime — full-image
// distance computation on a 1024x1024 source is 1M comparisons per
// candidate * 4 candidates = 4M comparisons, which would push the
// modal beyond the 60fps frame budget. The subsample keeps the
// auto-pick well under 50ms on typical hardware while preserving
// the dominant-color signal.
package palette

import (
	"image"
	"image/color"
	"math"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// SubPaletteFamily picks which of the project's two sub-palette
// arrays the auto-pick searches across. Imports default to
// FamilySprite (most imports are sprites); the per-tile palette
// picker uses FamilyBG.
type SubPaletteFamily int

const (
	FamilySprite SubPaletteFamily = iota
	FamilyBG
)

// String returns the lowercase identifier for the family — matches
// the convention used by the pf-tag widget=subpalette dispatch.
func (f SubPaletteFamily) String() string {
	switch f {
	case FamilyBG:
		return "bg"
	case FamilySprite:
		fallthrough
	default:
		return "sprite"
	}
}

// PickBestSubPalette returns the name of the sub-palette in the
// requested family whose 4 slot colors most closely fit the input
// image, plus the mean per-pixel RGB distance to the chosen palette
// (the score used by U4's warning banner threshold).
//
// Ties resolve to the lowest-indexed sub-palette so two identical
// runs against the same image return the same answer.
//
// Failure modes:
//   - nil project, nil image, or zero-size image → returns the
//     family's first sub-palette name with score 0.
//   - project with all-zero sub-palette overlays → callers should
//     run PaletteData.applyDefaults first; this function does not
//     mutate the project.
func PickBestSubPalette(
	img image.Image,
	p *pixelforge_project.Project,
	family SubPaletteFamily,
) (subPaletteName string, score float64) {
	if p == nil {
		return defaultNameFor(family), 0
	}
	palettes := p.Palette.SpriteSubPalettes
	if family == FamilyBG {
		palettes = p.Palette.BGSubPalettes
	}
	if img == nil {
		return palettes[0].Name, 0
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return palettes[0].Name, 0
	}

	// Pre-resolve each sub-palette's 4 slot colors to RGB tuples so
	// the inner per-pixel loop avoids hex-parse work per comparison.
	subColors := make([][4]color.RGBA, len(palettes))
	for i, sp := range palettes {
		for j, slot := range sp.Slots {
			if slot < 0 || slot >= pixelforge_project.MaxColors {
				continue
			}
			c, ok := parseHexColor(p.Palette.Base[slot])
			if !ok {
				continue
			}
			subColors[i][j] = c
		}
	}

	samples := collectSamplePixels(img, w, h)
	if len(samples) == 0 {
		return palettes[0].Name, 0
	}

	bestIdx := 0
	bestScore := math.MaxFloat64
	for i := range palettes {
		total := 0.0
		for _, px := range samples {
			total += nearestDistance(px, subColors[i])
		}
		mean := total / float64(len(samples))
		if mean < bestScore {
			bestScore = mean
			bestIdx = i
		}
	}
	return palettes[bestIdx].Name, bestScore
}

// MeanDistanceToSubPalette computes the mean per-pixel RGB
// Euclidean distance from the image's sampled pixels to the nearest
// color in the named sub-palette. U4's diff modal calls this with
// the chosen sub-palette to drive the warning-banner threshold.
//
// Returns 0 when the sub-palette name is unknown or the image is
// empty — callers gate the warning on threshold comparison, so the
// safe-fallback "no warning" is the right behaviour for absent
// inputs.
func MeanDistanceToSubPalette(
	img image.Image,
	p *pixelforge_project.Project,
	subPaletteName string,
) float64 {
	if img == nil || p == nil {
		return 0
	}
	sp, found := findSubPaletteByName(p, subPaletteName)
	if !found {
		return 0
	}
	var slotColors [4]color.RGBA
	for j, slot := range sp.Slots {
		if slot < 0 || slot >= pixelforge_project.MaxColors {
			continue
		}
		c, ok := parseHexColor(p.Palette.Base[slot])
		if !ok {
			continue
		}
		slotColors[j] = c
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return 0
	}
	samples := collectSamplePixels(img, w, h)
	if len(samples) == 0 {
		return 0
	}
	total := 0.0
	for _, px := range samples {
		total += nearestDistance(px, slotColors)
	}
	return total / float64(len(samples))
}

// findSubPaletteByName looks up a sub-palette by name across both
// families. Returns the entry + true on hit.
func findSubPaletteByName(p *pixelforge_project.Project, name string) (pixelforge_project.SubPalette, bool) {
	for _, sp := range p.Palette.SpriteSubPalettes {
		if sp.Name == name {
			return sp, true
		}
	}
	for _, sp := range p.Palette.BGSubPalettes {
		if sp.Name == name {
			return sp, true
		}
	}
	return pixelforge_project.SubPalette{}, false
}

// collectSamplePixels returns a flat list of RGB pixels sampled
// from the image. Images smaller than the subsample threshold are
// sampled at full resolution; larger images are subsampled on a
// 16x16 grid to bound runtime.
func collectSamplePixels(img image.Image, w, h int) []color.RGBA {
	const subsampleThreshold = 64
	const gridSide = 16

	bounds := img.Bounds()
	if w <= subsampleThreshold && h <= subsampleThreshold {
		out := make([]color.RGBA, 0, w*h)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := colorAt(img, x, y)
				if c.A == 0 {
					continue
				}
				out = append(out, c)
			}
		}
		return out
	}
	out := make([]color.RGBA, 0, gridSide*gridSide)
	for gy := 0; gy < gridSide; gy++ {
		for gx := 0; gx < gridSide; gx++ {
			x := bounds.Min.X + (gx*w)/gridSide
			y := bounds.Min.Y + (gy*h)/gridSide
			c := colorAt(img, x, y)
			if c.A == 0 {
				continue
			}
			out = append(out, c)
		}
	}
	return out
}

// colorAt unpacks image.Image.At into an RGBA8 tuple. The stdlib
// returns premultiplied alpha; we keep that contract for the
// distance math (zero-alpha pixels are skipped by the caller).
func colorAt(img image.Image, x, y int) color.RGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// nearestDistance returns the smallest Euclidean distance from c to
// any color in palette. Returns sqrt of the squared distance so the
// score is reported in RGB units (the threshold UX talks in
// "noticeable RGB distance," not squared distance).
func nearestDistance(c color.RGBA, palette [4]color.RGBA) float64 {
	best := math.MaxFloat64
	for _, p := range palette {
		dr := float64(int(c.R) - int(p.R))
		dg := float64(int(c.G) - int(p.G))
		db := float64(int(c.B) - int(p.B))
		d := dr*dr + dg*dg + db*db
		if d < best {
			best = d
		}
	}
	return math.Sqrt(best)
}

// defaultNameFor returns the family's first sub-palette name when
// no project is available. Used as a safe fallback so the caller
// never receives an empty string.
func defaultNameFor(family SubPaletteFamily) string {
	if family == FamilyBG {
		return "bg_0"
	}
	return "sprite_0"
}
