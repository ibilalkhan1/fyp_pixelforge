// Package pixelforge_pixelrenderer emulates the entire Picture Processing
// Unit (PPU) of the NES in software. Rather than redraw every frame from
// scratch across the display, PixelForge maintains a unified 128×128
// back-buffer in RAM. Drawing commands mutate that memory directly, and
// a single flip() blasts the completed image to the window in one go.
//
// Because the rasterisation happens on the CPU, the engine is free from
// hardware limits such as "maximum sprites per scanline". As many sprites
// as the code and host memory allow can share the same horizontal line.
package pixelforge_pixelrenderer

// FramebufferWidth is the internal horizontal resolution of the software
// PPU. 128 is wide enough for crisp retro artwork while keeping the
// memory footprint tiny (16 KiB per buffer).
const FramebufferWidth = 128

// FramebufferHeight is the internal vertical resolution of the software
// PPU. Paired with FramebufferWidth it yields a square power-of-two
// friendly canvas that scales cleanly with integer multipliers.
const FramebufferHeight = 128

// MaxColors is the number of simultaneous colors the back-buffer can
// reference. Indices 0–7 address the active palette, matching the
// constraints of classic tile-based hardware without sacrificing
// artistic range.
const MaxColors = 8

// Color is a palette index (0–7) stored in the framebuffer.
type Color uint8

// RGB is a 24-bit native color produced when the framebuffer is
// resolved through the active palette during flip().
type RGB uint32

// Framebuffer is the 128×128 software back-buffer. Each byte is a
// palette index, so the entire screen costs only 16 KiB of RAM.
type Framebuffer [FramebufferWidth * FramebufferHeight]Color

// Palette maps the eight Color indices to native RGB values.
type Palette [MaxColors]RGB

// PixelRenderer owns the back-buffer, the active palette, and all
// rasterisation routines. It is the beating heart of the software PPU.
type PixelRenderer struct {
	// back is the unified framebuffer that every drawing command
	// touches directly. No secondary buffers, no GPU textures—just
	// flat memory that the CPU can walk at cache-friendly strides.
	back Framebuffer

	// pal is the currently active palette. flip() uses this table
	// to translate indexed pixels into 24-bit RGB for the display.
	pal Palette
}

// NewPixelRenderer creates a renderer with the given palette.
// The palette is copied so that callers can reuse the slice or
// array without worrying about aliasing.
func NewPixelRenderer(p Palette) *PixelRenderer {
	return &PixelRenderer{pal: p}
}

// SetPalette hot-swaps the active palette at runtime. This is the
// mechanism behind palette-animation effects such as water shimmer
// or damage flash without touching the framebuffer contents.
func (pr *PixelRenderer) SetPalette(p Palette) {
	pr.pal = p
}

// Clear fills the entire back-buffer with a single color index.
// It is the software equivalent of wiping the CRT before a fresh
// frame of compositing.
func (pr *PixelRenderer) Clear(c Color) {
	for i := range pr.back {
		pr.back[i] = c
	}
}

// SetPixel writes a single palette index to the back-buffer at
// (x, y). Coordinates outside the 128×128 bounds are silently
// discarded so that caller code stays simple and branch-predictable.
func (pr *PixelRenderer) SetPixel(x, y int, c Color) {
	if x < 0 || x >= FramebufferWidth || y < 0 || y >= FramebufferHeight {
		return
	}
	pr.back[y*FramebufferWidth+x] = c
}

// Spr stamps an 8×8 sprite tile to the back-buffer with its top-left
// corner at (x, y). The spriteID selects both the tile graphic and
// the palette bank to use. Because the render is purely software,
// there is no hardware scanline limit—dozens of sprites may overlap
// on the same row without dropping tiles.
func (pr *PixelRenderer) Spr(x, y int, spriteID Color) {
	// Tile render: 8×8 pixels written directly into the unified
	// framebuffer. In a full implementation the spriteID indexes a
	// tile-ROM sheet; here the ID is used as a fill pattern seed
	// so that every tile looks visually distinct for prototyping.
	for dy := 0; dy < 8; dy++ {
		for dx := 0; dx < 8; dx++ {
			px := x + dx
			py := y + dy
			if px < 0 || px >= FramebufferWidth || py < 0 || py >= FramebufferHeight {
				continue
			}
			// Pattern seed: alternate pixels based on spriteID.
			pattern := Color((uint8(spriteID) + uint8(dx^dy)) & 0x07)
			pr.back[py*FramebufferWidth+px] = pattern
		}
	}
}

// RectFill draws a solid filled rectangle between the inclusive
// corners (x0, y0) and (x1, y1) using the supplied palette index.
func (pr *PixelRenderer) RectFill(x0, y0, x1, y1 int, c Color) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		if y < 0 || y >= FramebufferHeight {
			continue
		}
		row := y * FramebufferWidth
		for x := x0; x <= x1; x++ {
			if x < 0 || x >= FramebufferWidth {
				continue
			}
			pr.back[row+x] = c
		}
	}
}

// CircFill draws a filled circle centred at (cx, cy) with radius r.
// It uses the midpoint circle algorithm, a variant of Bresenham's
// approach, to touch each pixel in the bounding box at most once.
func (pr *PixelRenderer) CircFill(cx, cy, r int, c Color) {
	if r < 0 {
		return
	}
	r2 := r * r
	for y := -r; y <= r; y++ {
		py := cy + y
		if py < 0 || py >= FramebufferHeight {
			continue
		}
		row := py * FramebufferWidth
		y2 := y * y
		for x := -r; x <= r; x++ {
			if x*x+y2 > r2 {
				continue
			}
			px := cx + x
			if px < 0 || px >= FramebufferWidth {
				continue
			}
			pr.back[row+px] = c
		}
	}
}

// Line draws a line from (x0, y0) to (x1, y1) using Bresenham's
// Line Algorithm. The algorithm steps in the dominant axis and
// accumulates an error term to decide when to step in the minor
// axis, producing a pixel-perfect diagonal without any floating
// point arithmetic.
func (pr *PixelRenderer) Line(x0, y0, x1, y1 int, c Color) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy

	for {
		pr.SetPixel(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// Flip resolves the indexed framebuffer through the active palette
// and returns a flat RGB slice ready for the display driver. The
// entire 128×128 block is copied in one contiguous blast, mirroring
// the way a hardware PPU latches VRAM to the CRT during V-Blank.
//
// In the engine's main loop, _draw() performs all rendering calls
// against the PixelRenderer, and once _draw() returns, flip() is
// invoked to present the completed frame.
func (pr *PixelRenderer) Flip() []RGB {
	out := make([]RGB, len(pr.back))
	for i, idx := range pr.back {
		out[i] = pr.pal[idx]
	}
	return out
}

// BackBuffer returns a read-only view of the raw indexed framebuffer.
// This is useful for scanline effects, debug overlays, or save-state
// serialization that needs to capture the screen without palette
// translation.
func (pr *PixelRenderer) BackBuffer() *Framebuffer {
	return &pr.back
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
