package editor

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"

// pcofont is the canonical canvas-resident text rendering chokepoint
// used by Scene workspace tool indicators (U29). The wrapper makes the
// dependency explicit so future TTF swap-ins land in one place.
func pcofont(text string, x, y int) {
	pixelforge_cofont.Print(text, x, y)
}
