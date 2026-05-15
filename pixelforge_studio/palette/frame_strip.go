package palette

import (
	"image"
)

// DetectFrames examines img for fully-transparent vertical gutters
// that separate frames in a horizontal strip. Returns (frameW, frameH)
// per the detection. When no gutters are found, the result is
// (img.Width, img.Height) — treat as a single frame.
//
// The algorithm: find columns that are fully transparent AND have a
// non-empty column to their immediate left and right. Each such column
// is a gutter; frame width = (image_width - gutter_count) / frame_count.
// Verticality is detected similarly on rows for sprites strip vertically.
func DetectFrames(img image.Image) (int, int) {
	if img == nil {
		return 0, 0
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return 0, 0
	}

	// Build column and row "emptiness" predicates.
	colEmpty := make([]bool, w)
	rowEmpty := make([]bool, h)
	for x := 0; x < w; x++ {
		colEmpty[x] = true
		for y := 0; y < h; y++ {
			_, _, _, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a > 0 {
				colEmpty[x] = false
				rowEmpty[y] = false
				break
			}
		}
	}
	for y := 0; y < h; y++ {
		if rowEmpty[y] {
			// Already proved non-empty above? rowEmpty only flipped
			// when we found pixels — so an unset rowEmpty is still
			// "unknown". Re-scan to set it accurately for non-empty
			// rows that had only their first column hit.
			rowEmpty[y] = true
			for x := 0; x < w; x++ {
				_, _, _, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				if a > 0 {
					rowEmpty[y] = false
					break
				}
			}
		}
	}

	// Horizontal frame strip: find vertical gutters with non-empty
	// neighbors on both sides.
	frameW := w
	if hasInteriorGutter(colEmpty) {
		frameW = framesFromGutters(colEmpty)
		if frameW <= 0 {
			frameW = w
		}
	}
	frameH := h
	if hasInteriorGutter(rowEmpty) {
		frameH = framesFromGutters(rowEmpty)
		if frameH <= 0 {
			frameH = h
		}
	}
	return frameW, frameH
}

// hasInteriorGutter reports whether the flags contain a gutter column
// (true) that is NOT at the boundary.
func hasInteriorGutter(flags []bool) bool {
	for i, e := range flags {
		if !e {
			continue
		}
		if i == 0 || i == len(flags)-1 {
			continue
		}
		if !flags[i-1] && !flags[i+1] {
			return true
		}
	}
	return false
}

// framesFromGutters returns the per-frame size by counting gutters that
// have non-empty neighbors on both sides.
func framesFromGutters(flags []bool) int {
	gutters := 0
	for i := 1; i < len(flags)-1; i++ {
		if flags[i] && !flags[i-1] && !flags[i+1] {
			gutters++
		}
	}
	if gutters == 0 {
		return len(flags)
	}
	// Frames = gutters + 1.
	frames := gutters + 1
	return (len(flags) - gutters) / frames
}
