package palette

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// RGBEuclidean is symmetric and 0 for identical inputs.
func TestRGBEuclidean(t *testing.T) {
	c := color.RGBA{R: 100, G: 50, B: 25}
	assert.Equal(t, 0.0, RGBEuclidean(c, c))
	assert.Equal(t, RGBEuclidean(c, color.RGBA{}), RGBEuclidean(color.RGBA{}, c))
}

// Quantizer snaps a near-red pixel to a near-red palette slot.
func TestQuantizer_SnapsToNearestPaletteSlot(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.Base[1] = "#ff0000"
	p.Palette.Base[2] = "#00ff00"
	p.Palette.Base[3] = "#0000ff"
	q := NewQuantizer()
	q.Bind(p)

	got := q.QuantizePixel(color.RGBA{R: 250, G: 5, B: 5, A: 0xff})
	assert.Equal(t, byte(1), got)
}

// Slot 0 absorbs transparent pixels regardless of RGB.
func TestQuantizer_TransparentMapsToSlot0(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	q := NewQuantizer()
	q.Bind(p)
	got := q.QuantizePixel(color.RGBA{R: 255, G: 0, B: 0, A: 0})
	assert.Equal(t, byte(0), got)
}

// Identical inputs produce byte-identical outputs (determinism contract).
func TestQuantizer_DeterministicAcrossRuns(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	for i := 1; i < pixelforge_project.MaxColors; i++ {
		p.Palette.Base[i] = "#ff0000"
	}
	q := NewQuantizer()
	q.Bind(p)

	rgba := []byte{200, 100, 50, 0xff, 10, 220, 10, 0xff}
	out1 := q.QuantizeImage(rgba, 2, 1)
	out2 := q.QuantizeImage(rgba, 2, 1)
	require.Equal(t, out1, out2)
}

// bitsSet counts the 1-bits in a mask.
func TestBitsSet(t *testing.T) {
	assert.Equal(t, 0, bitsSet(nil))
	assert.Equal(t, 0, bitsSet([]uint8{0}))
	assert.Equal(t, 8, bitsSet([]uint8{0xff, 0}))
	assert.Equal(t, 16, bitsSet([]uint8{0xff, 0xff}))
}
