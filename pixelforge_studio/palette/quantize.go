package palette

import (
	"image/color"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// ColorDistance computes the perceptual distance between two RGB
// colors. v1 ships RGB Euclidean only; the pluggable shape lets a
// Lab/CIEDE2000 metric drop in later (M2.1 if user feedback flags
// quality).
type ColorDistance func(a, b color.RGBA) float64

// RGBEuclidean is the v1 default color-distance metric.
func RGBEuclidean(a, b color.RGBA) float64 {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return float64(dr*dr + dg*dg + db*db)
}

// Quantizer snaps RGB pixels to the nearest palette slot.
type Quantizer struct {
	Distance ColorDistance
	cache    [pixelforge_project.MaxColors]color.RGBA
	valid    bool
}

// NewQuantizer returns a quantizer using RGBEuclidean by default.
func NewQuantizer() *Quantizer {
	return &Quantizer{Distance: RGBEuclidean}
}

// Bind precomputes the palette's RGB table; subsequent QuantizePixel
// calls reuse the cache.
func (q *Quantizer) Bind(p *pixelforge_project.Project) {
	for i := 0; i < pixelforge_project.MaxColors; i++ {
		c, ok := parseHexColor(p.Palette.Base[i])
		if !ok {
			c = color.RGBA{}
		}
		q.cache[i] = c
	}
	q.valid = true
}

// QuantizePixel returns the palette index whose color is closest to in.
// Slot 0 (treated as transparent at the engine layer) is skipped when
// the input pixel has any alpha — transparent inputs always quantize
// to slot 0.
func (q *Quantizer) QuantizePixel(in color.RGBA) byte {
	if !q.valid {
		return 0
	}
	if in.A == 0 {
		return 0
	}
	best := 1
	bestD := q.Distance(in, q.cache[1])
	for i := 2; i < pixelforge_project.MaxColors; i++ {
		d := q.Distance(in, q.cache[i])
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return byte(best)
}

// QuantizeImage produces an indexed byte image from rgba. width is the
// number of pixels per row in rgba (interpreted as 4 bytes/pixel).
func (q *Quantizer) QuantizeImage(rgba []byte, width, height int) []byte {
	out := make([]byte, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4
			pix := color.RGBA{R: rgba[i], G: rgba[i+1], B: rgba[i+2], A: rgba[i+3]}
			out[y*width+x] = q.QuantizePixel(pix)
		}
	}
	return out
}
