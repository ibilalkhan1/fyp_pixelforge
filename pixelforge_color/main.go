// Package pixelforge_color provides an extended NES-inspired color generation
// system for the PixelForge engine. It produces smooth 64-shade ramps by
// interpolating between light and dark archetype palettes, giving artists
// fine-grained control over atmosphere and lighting without leaving the
// constrained 8-bit aesthetic.
package pixelforge_color

// RGB represents a 24-bit color value using the standard 0xRRGGBB encoding.
// This matches the engine's native color format and can be dropped directly
// into PaletteArray slots or ColorTable banks.
type RGB uint32

// Color is the engine's native palette index type. Values 0–63 address the
// generated shade ramp; 0 is the darkest variant and 63 is the lightest.
type Color uint8

// rgb packs three 8-bit channels into a single RGB value.
// It is the canonical way to construct colors in this package so that
// artists can read and tweak values without mentally decoding hex literals.
func rgb(r, g, b uint8) RGB {
	return RGB(uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// LightNES holds the high-luminance archetype colors used on the NES
// emphasis (light) palette. These are the "daylight" variants that artists
// reach for when a scene needs clarity and warmth.
var LightNES = [8]RGB{
	rgb(0xFC, 0xFC, 0xFC), // White  — paper, clouds, highlights
	rgb(0xF4, 0xD0, 0x7E), // Yellow — gold, sunbeams, warning stripes
	rgb(0x74, 0xD0, 0x6E), // Green  — grass, health, acid
	rgb(0x5C, 0x94, 0xFC), // Blue   — sky, water, ice
	rgb(0xE8, 0x5D, 0x5D), // Red    — lava, damage, enemy accents
	rgb(0xC2, 0x8F, 0x5C), // Brown  — earth, wood, leather
	rgb(0xB4, 0xB4, 0xB4), // Gray   — stone, metal, neutral tones
	rgb(0x3C, 0x3C, 0x3C), // Black  — deep shadow, silhouette, void
}

// DarkNES holds the low-luminance archetype colors used on the NES
// emphasis (dark) palette. These are the "twilight" variants that ground
// a scene and provide contrast for readable UI and moody backgrounds.
var DarkNES = [8]RGB{
	rgb(0x7C, 0x7C, 0x7C), // White  — overcast, mist, distant snow
	rgb(0xB8, 0x8A, 0x00), // Yellow — torchlight, amber, aged parchment
	rgb(0x00, 0x78, 0x00), // Green  — forest canopy, moss, poison
	rgb(0x00, 0x44, 0xBC), // Blue   — midnight ocean, depth, melancholy
	rgb(0xB8, 0x00, 0x00), // Red    — dried blood, brick, sunset edge
	rgb(0x68, 0x38, 0x00), // Brown  — mud, bark, rust
	rgb(0x50, 0x50, 0x50), // Gray   — slate, iron, mechanical
	rgb(0x00, 0x00, 0x00), // Black  — pitch, absence, pupil
}

// GenerateShadeRamp produces a full 64-color palette by linearly
// interpolating between each matching pair in DarkNES and LightNES.
// The resulting table is organized in 8 rows of 8 shades:
//
//	[0..7]   white ramp
//	[8..15]  yellow ramp
//	[16..23] green ramp
//	[24..31] blue ramp
//	[32..39] red ramp
//	[40..47] brown ramp
//	[48..55] gray ramp
//	[56..63] black ramp
//
// Games can hot-swap this ramp into the global Palette at runtime to
// simulate time-of-day, damage flash, or underwater tinting while keeping
// every shade within the NES-compatible chromatic space.
func GenerateShadeRamp() [64]RGB {
	var ramp [64]RGB
	for i := 0; i < 8; i++ {
		dark := DarkNES[i]
		light := LightNES[i]

		// Extract channels from dark and light.
		dr, dg, db := channelSplit(dark)
		lr, lg, lb := channelSplit(light)

		for step := 0; step < 8; step++ {
			t := float64(step) / 7.0 // interpolation factor 0.0 .. 1.0
			r := lerpUint8(dr, lr, t)
			g := lerpUint8(dg, lg, t)
			b := lerpUint8(db, lb, t)
			ramp[i*8+step] = RGB(uint32(r)<<16 | uint32(g)<<8 | uint32(b))
		}
	}
	return ramp
}

// channelSplit unpacks an RGB value into its red, green, and blue channels.
func channelSplit(c RGB) (r, g, b uint8) {
	r = uint8(c >> 16)
	g = uint8(c >> 8)
	b = uint8(c)
	return
}

// lerpUint8 performs a linear interpolation between two byte values.
// The factor t is expected in the range [0.0, 1.0]; values outside this
// range are allowed but will extrapolate rather than interpolate.
func lerpUint8(a, b uint8, t float64) uint8 {
	return uint8(float64(a)*(1.0-t) + float64(b)*t)
}
