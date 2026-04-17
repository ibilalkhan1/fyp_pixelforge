// Package pisnap provides functions for taking screenshots.
package pixelforge_snap

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

// CaptureOrErr captures a screenshot and saves it to the temporary directory.
//
// It returns the filename.
//
// An error is returned if there is a problem storing the file
// or if the code is run in a browser.
func CaptureOrErr() (string, error) {
	if runtime.GOOS == "js" {
		return "", errors.New("storing files does not work on js")
	}

	img := PalettedImage()

	file, err := os.CreateTemp("", "screenshot-*.png")
	if err != nil {
		return "", fmt.Errorf("error creating temp file for screenshot: %w", err)
	}

	if err = png.Encode(file, img); err != nil {
		return "", fmt.Errorf("error encoding screenshot into PNG file: %w", err)
	}

	if err = file.Close(); err != nil {
		return "", fmt.Errorf("error closing file: %w", err)
	}

	return file.Name(), nil
}

// PalettedImage captures a screenshot and returns it as an image.PalettedImage.
func PalettedImage() image.PalettedImage {
	var palette color.Palette
	for _, col := range pixelforge.PaletteMapping {
		col &= pixelforge.MaxColors - 1
		rgb := pixelforge.Palette[col]
		r, g, b := rgb.RGB()
		rgba := &color.NRGBA{R: r, G: g, B: b, A: 255}
		palette = append(palette, rgba)
	}

	screen := pixelforge.Screen()
	size := image.Rectangle{Max: image.Point{X: screen.W(), Y: screen.H()}}
	img := image.NewPaletted(size, palette)

	copy(img.Pix, pixelforge.Screen().Data())

	return img
}
