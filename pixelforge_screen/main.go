// Package pixelforge_screen handles the translation between the engine's
// tiny fixed NES resolution and the unpredictable sizes of modern displays.
// It locks rendering to a 256×240 pixel grid and then scales that grid
// using integer multiples so that every physical pixel stays a crisp
// rectangle rather than a blurry interpolation.
package pixelforge_screen

// ScreenWidth is the horizontal resolution of the hypothetical NES screen.
// 256 is the classic span used by the NES PPU and gives designers a
// predictable 8-pixel tile boundary.
const ScreenWidth = 256

// ScreenHeight is the vertical resolution of the hypothetical NES screen.
// 240 lines matches the NTSC-visible area and pairs cleanly with
// 8×8 tiles (30 rows).
const ScreenHeight = 240

// AspectNumerator and AspectDenominator describe the 8:7 pixel aspect
// ratio that the engine targets. Modern flat panels use square pixels,
// so a 4:3 stretch would turn circles into ovals. By respecting 8:7
// we keep sprites geometrically faithful to how they were authored.
const AspectNumerator = 8
const AspectDenominator = 7

// Scale holds the integer scaling multipliers for the X and Y axes.
// Each value is a whole number (≥1) so that nearest-neighbor upsampling
// never introduces fractional pixel boundaries.
type Scale struct {
	X int
	Y int
}

// Viewport describes where the scaled canvas should be centered on the
// physical display, together with the final drawable size in pixels.
type Viewport struct {
	// CanvasW and CanvasH are the dimensions of the virtual back-buffer.
	CanvasW int
	CanvasH int

	// ScreenW and ScreenH are the final scaled dimensions on the monitor.
	ScreenW int
	ScreenH int

	// OffsetX and OffsetY are the top-left corner of the centered image.
	OffsetX int
	OffsetY int

	// Scale carries the integer multipliers used to reach ScreenW/H.
	Scale Scale
}

// CalculateScale determines the largest whole-number multiplier that
// fits the NES screen inside the given physical display bounds.
//
// The formula used is:
//
//	scaleX = floor(baseWidth  / ScreenWidth)
//	scaleY = floor(baseHeight / ScreenHeight)
//
// If the physical monitor is smaller than the NES resolution in either
// axis, the multiplier is clamped to 1 so the canvas is never down-sampled.
func CalculateScale(baseWidth, baseHeight int) Scale {
	sx := baseWidth / ScreenWidth
	if sx < 1 {
		sx = 1
	}
	sy := baseHeight / ScreenHeight
	if sy < 1 {
		sy = 1
	}
	return Scale{X: sx, Y: sy}
}

// FitViewport computes a centered integer-scaled viewport for a
// physical monitor of the given size.
//
// It first applies CalculateScale, then multiplies the NES resolution,
// and finally centers the resulting rectangle with letter-box or
// pillar-box offsets as needed.
func FitViewport(baseWidth, baseHeight int) Viewport {
	s := CalculateScale(baseWidth, baseHeight)

	scaledW := ScreenWidth * s.X
	scaledH := ScreenHeight * s.Y

	// Center the canvas, leaving black bars if the monitor aspect
	// does not match the 8:7 target.
	offX := (baseWidth - scaledW) / 2
	if offX < 0 {
		offX = 0
	}
	offY := (baseHeight - scaledH) / 2
	if offY < 0 {
		offY = 0
	}

	return Viewport{
		CanvasW: ScreenWidth,
		CanvasH: ScreenHeight,
		ScreenW: scaledW,
		ScreenH: scaledH,
		OffsetX: offX,
		OffsetY: offY,
		Scale:   s,
	}
}

// ApplyAspectAdjust returns the display width that preserves the 8:7
// pixel aspect ratio for a given height. Use this when the renderer
// needs to know how wide the output rectangle should look on a
// square-pixel monitor so that circles stay round.
func ApplyAspectAdjust(height int) int {
	return height * AspectNumerator / AspectDenominator
}
