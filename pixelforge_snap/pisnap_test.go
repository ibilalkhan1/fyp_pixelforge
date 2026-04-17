package pisnap_test

import (
	"image"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_snap"
	"github.com/stretchr/testify/assert"
)

func TestPalettedImage(t *testing.T) {
	pixelforge.SetScreenSize(2, 3)
	pixelforge.Palette[1] = 0xffaa44
	pixelforge.Palette[2] = 0xff0000
	pixelforge.Palette[3] = 0x00ff00
	pixelforge.Palette[4] = 0x0000ff
	pixelforge.Palette[5] = 0x00ffff
	pixelforge.Palette[6] = 0xff00ff
	screen := pixelforge.Screen()
	screen.SetAll(1, 2, 3, 4, 5, 6)
	// when
	img := pisnap.PalettedImage()
	// then
	assertPalettedImage(t, img, screen)
}

func assertPalettedImage(t *testing.T, img image.PalettedImage, screen pixelforge.Canvas) {
	t.Helper()

	// then color indexes are the same
	for y := 0; y < screen.H(); y++ {
		for x := 0; x < screen.W(); x++ {
			actual := img.ColorIndexAt(x, y)
			expected := screen.Get(x, y)
			assert.Equal(t, expected, actual)
		}
	}
	// and RGBA colors match
	for y := 0; y < screen.H(); y++ {
		for x := 0; x < screen.W(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			assert.Equal(t, uint8(0xff), uint8(a))
			actual := pixelforge.FromRGB(uint8(r), uint8(g), uint8(b))
			expected := pixelforge.Palette[screen.Get(x, y)]
			assert.Equal(t, expected, actual)
		}
	}
	// and size is the same
	assert.Equal(t,
		image.Rectangle{
			Max: image.Point{X: screen.W(), Y: screen.H()},
		},
		img.Bounds(),
		"image size is not same as screen size",
	)
}
