package palette

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

// solidImage returns an opaque 8×8 image — no gutters.
func solidImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	return img
}

// stripImage builds a 32×8 frame strip with 1px transparent gutters
// between four 8×8 frames: pattern (8 solid, 1 transparent) × 4 with
// total width 8*4 + 3 = 35. We size the gutters narrower so the math
// stays clean: 8+1+8+1+8+1+8 = 35.
func stripImage(frameW, frameH, frames, gutter int) image.Image {
	w := frameW*frames + gutter*(frames-1)
	img := image.NewRGBA(image.Rect(0, 0, w, frameH))
	for f := 0; f < frames; f++ {
		xStart := f * (frameW + gutter)
		for y := 0; y < frameH; y++ {
			for x := 0; x < frameW; x++ {
				img.Set(xStart+x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}
	}
	return img
}

// Solid image with no gutters → single-frame detection.
func TestDetectFrames_SingleFrameFallsThrough(t *testing.T) {
	img := solidImage(32, 16)
	fw, fh := DetectFrames(img)
	assert.Equal(t, 32, fw)
	assert.Equal(t, 16, fh)
}

// 4×8 frame strip with 1px gutters.
func TestDetectFrames_HorizontalStrip(t *testing.T) {
	img := stripImage(8, 8, 4, 1)
	fw, fh := DetectFrames(img)
	assert.Equal(t, 8, fw)
	assert.Equal(t, 8, fh)
}

// Leading transparent column does not count as a gutter (no non-empty
// neighbor to the left).
func TestDetectFrames_BorderTransparencyIgnored(t *testing.T) {
	// 9×8: column 0 is transparent; columns 1..8 are red.
	img := image.NewRGBA(image.Rect(0, 0, 9, 8))
	for y := 0; y < 8; y++ {
		for x := 1; x < 9; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	fw, _ := DetectFrames(img)
	assert.Equal(t, 9, fw, "boundary transparent column is not a gutter")
}

// Empty image returns 0×0.
func TestDetectFrames_EmptyImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	fw, fh := DetectFrames(img)
	assert.Equal(t, 0, fw)
	assert.Equal(t, 0, fh)
}
