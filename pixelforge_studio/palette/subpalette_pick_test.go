package palette

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// fillImage returns an in-memory image of size w x h with every
// pixel set to the supplied color. Used by the auto-pick tests to
// build deterministic single-color inputs the algorithm scores
// against known sub-palettes.
func fillImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// projectWithSpriteSubPalettes builds a project whose sprite sub-
// palettes carry the supplied per-sub-palette slot lists, with the
// referenced base colors set so the auto-pick has real RGB to
// compare against.
func projectWithSpriteSubPalettes(t *testing.T, paletteColors map[int]string,
	sprites [4][4]int, names ...string) *pixelforge_project.Project {
	t.Helper()
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	for slot, hex := range paletteColors {
		p.Palette.Base[slot] = hex
	}
	for i := range sprites {
		name := ""
		if i < len(names) {
			name = names[i]
		} else {
			name = p.Palette.SpriteSubPalettes[i].Name
		}
		p.Palette.SpriteSubPalettes[i] = pixelforge_project.SubPalette{
			Name: name, Slots: sprites[i],
		}
	}
	return p
}

// TestPickBestSubPalette_RedImagePicksRedSubPalette: an all-red
// image scored against four sub-palettes — one red, three other —
// returns the red sub-palette.
func TestPickBestSubPalette_RedImagePicksRedSubPalette(t *testing.T) {
	p := projectWithSpriteSubPalettes(t,
		map[int]string{
			1: "#ff0000", 2: "#cc0000", 3: "#aa0000", 4: "#880000",
			5: "#00ff00", 6: "#00cc00", 7: "#00aa00", 8: "#008800",
			9: "#0000ff", 10: "#0000cc", 11: "#0000aa", 12: "#000088",
			13: "#888888", 14: "#666666", 15: "#444444", 16: "#222222",
		},
		[4][4]int{
			{1, 2, 3, 4},    // red
			{5, 6, 7, 8},    // green
			{9, 10, 11, 12}, // blue
			{13, 14, 15, 16}, // grey
		},
		"red", "green", "blue", "grey",
	)
	img := fillImage(8, 8, color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff})
	name, score := PickBestSubPalette(img, p, FamilySprite)
	assert.Equal(t, "red", name, "all-red image picks the red sub-palette")
	assert.InDelta(t, 0.0, score, 0.5, "exact-match score is near zero")
}

// TestPickBestSubPalette_GreyscalePicksGreyscalePalette: a
// greyscale image scored against a project with one greyscale sub-
// palette and three vibrant ones picks the greyscale.
func TestPickBestSubPalette_GreyscalePicksGreyscalePalette(t *testing.T) {
	p := projectWithSpriteSubPalettes(t,
		map[int]string{
			1: "#ff0000", 2: "#00ff00", 3: "#0000ff", 4: "#ffff00",
			5: "#ff00ff", 6: "#00ffff", 7: "#ff8800", 8: "#88ff00",
			9: "#0088ff", 10: "#8800ff", 11: "#ff0088", 12: "#88ff88",
			13: "#202020", 14: "#606060", 15: "#a0a0a0", 16: "#e0e0e0",
		},
		[4][4]int{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
			{9, 10, 11, 12},
			{13, 14, 15, 16}, // grey
		},
		"vibrant_a", "vibrant_b", "vibrant_c", "grey",
	)
	img := fillImage(8, 8, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff})
	name, _ := PickBestSubPalette(img, p, FamilySprite)
	assert.Equal(t, "grey", name)
}

// TestPickBestSubPalette_TiedSubPalettesReturnsFirst: two sub-
// palettes equidistant from the image — the lower-indexed one wins
// (deterministic).
func TestPickBestSubPalette_TiedSubPalettesReturnsFirst(t *testing.T) {
	p := projectWithSpriteSubPalettes(t,
		map[int]string{
			1: "#808080", 2: "#808080", 3: "#808080", 4: "#808080",
			5: "#808080", 6: "#808080", 7: "#808080", 8: "#808080",
		},
		[4][4]int{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
		"tie_first", "tie_second", "tie_third", "tie_fourth",
	)
	img := fillImage(4, 4, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff})
	name, _ := PickBestSubPalette(img, p, FamilySprite)
	assert.Equal(t, "tie_first", name,
		"ties resolve to the lower-indexed sub-palette")
}

// TestPickBestSubPalette_EmptyImageReturnsFirst: a zero-size image
// returns the family's first sub-palette name with score 0.
func TestPickBestSubPalette_EmptyImageReturnsFirst(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	name, score := PickBestSubPalette(img, p, FamilySprite)
	assert.Equal(t, "sprite_0", name)
	assert.Equal(t, 0.0, score)
}

// TestPickBestSubPalette_NilProjectReturnsFamilyDefault: passing a
// nil project returns the family default (no panic).
func TestPickBestSubPalette_NilProjectReturnsFamilyDefault(t *testing.T) {
	name, _ := PickBestSubPalette(nil, nil, FamilySprite)
	assert.Equal(t, "sprite_0", name)
	name, _ = PickBestSubPalette(nil, nil, FamilyBG)
	assert.Equal(t, "bg_0", name)
}

// TestPickBestSubPalette_BGvsSpriteFamily: same image; calling with
// FamilyBG returns from the BG array, FamilySprite from the sprite
// array.
func TestPickBestSubPalette_BGvsSpriteFamily(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	img := fillImage(4, 4, color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff})

	bgName, _ := PickBestSubPalette(img, p, FamilyBG)
	spriteName, _ := PickBestSubPalette(img, p, FamilySprite)
	assert.Contains(t, []string{"bg_0", "bg_1", "bg_2", "bg_3"}, bgName)
	assert.Contains(t, []string{"sprite_0", "sprite_1", "sprite_2", "sprite_3"}, spriteName)
}

// TestPickBestSubPalette_SubsamplePerfBudget: a 1024x1024 image
// completes well under 50ms via the 16x16 subsample path. Slow
// hardware may bend this; the cap is generous enough that a CPU
// fault should be loud.
func TestPickBestSubPalette_SubsamplePerfBudget(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	img := fillImage(1024, 1024, color.RGBA{R: 0xa0, G: 0x40, B: 0x40, A: 0xff})

	start := time.Now()
	name, _ := PickBestSubPalette(img, p, FamilySprite)
	elapsed := time.Since(start)

	require.NotEmpty(t, name)
	assert.Less(t, elapsed, 50*time.Millisecond,
		"subsampled auto-pick must complete under the perf budget")
}

// TestMeanDistanceToSubPalette_KnownDistance: an image of one color
// scored against a sub-palette of distance-50 colors reports the
// known distance.
func TestMeanDistanceToSubPalette_KnownDistance(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	// Slot 17 = sprite_0's first slot per defaults. Set it to a
	// known color so the distance math is checkable.
	p.Palette.Base[17] = "#646464" // (100, 100, 100)
	p.Palette.Base[18] = "#646464"
	p.Palette.Base[19] = "#646464"
	p.Palette.Base[20] = "#646464"
	img := fillImage(4, 4, color.RGBA{R: 150, G: 150, B: 150, A: 255})
	// Pixel (150,150,150) vs slot (100,100,100): dist = sqrt(50^2+50^2+50^2) ≈ 86.6
	d := MeanDistanceToSubPalette(img, p, "sprite_0")
	assert.InDelta(t, 86.6, d, 1.0)
}

// TestMeanDistanceToSubPalette_UnknownNameReturnsZero: looking up a
// non-existent sub-palette returns 0 (no panic, no warning surfaced
// by the modal).
func TestMeanDistanceToSubPalette_UnknownNameReturnsZero(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	img := fillImage(4, 4, color.RGBA{R: 0xff, A: 0xff})
	d := MeanDistanceToSubPalette(img, p, "no_such_palette")
	assert.Equal(t, 0.0, d)
}
