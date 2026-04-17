//go:build !js

package pisnap_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_snap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureOrErr(t *testing.T) {
	// when
	file, err := pisnap.CaptureOrErr()
	// then
	require.NoError(t, err)
	assert.NotEmpty(t, file)
	// and file exists
	f, err := os.ReadFile(file)
	require.NoError(t, err, "cannot read PNG file")
	// and
	img, err := png.Decode(bytes.NewReader(f))
	require.NoError(t, err, "file is not a valid PNG")
	// and
	palettedImage, ok := img.(*image.Paletted)
	require.True(t, ok, "image is not a Paletted")
	assertPalettedImage(t, palettedImage, pixelforge.Screen())
}
