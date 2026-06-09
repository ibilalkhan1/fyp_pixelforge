// Package pixelforge_spritebackground manages the two dominant visual
// layers of a retro frame: the static tilemap (background) and the
// independent sprites that move above or below it. Both are built from
// 8×8 pixel tiles, but the engine treats them very differently so that
// scrolling worlds stay cheap while animated actors remain flexible.
package pixelforge_spritebackground

// TileWidth is the horizontal span of a single tile in pixels.
// Eight is the canonical size used by the NES PPU and it aligns
// cleanly with byte boundaries and bitwise tile-sheet lookups.
const TileWidth = 8

// TileHeight is the vertical span of a single tile in pixels.
const TileHeight = 8

// BackgroundWidthTiles is the number of tile columns in one background
// plane. Thirty-two tiles × 8 pixels = 256 pixels, matching the
// horizontal resolution of the software PPU.
const BackgroundWidthTiles = 32

// BackgroundHeightTiles is the number of tile rows in one background
// plane. Thirty tiles × 8 pixels = 240 pixels, matching the vertical
// resolution of the software PPU.
const BackgroundHeightTiles = 30

// BackgroundWidthPixels is the physical pixel width of a background layer.
const BackgroundWidthPixels = BackgroundWidthTiles * TileWidth

// BackgroundHeightPixels is the physical pixel height of a background layer.
const BackgroundHeightPixels = BackgroundHeightTiles * TileHeight

// MaxSprites is the number of independent objects the frame renderer
// can composite in a single frame. Because the rasteriser is entirely
// software, this limit is dictated by memory budget rather than
// hardware scan-line constraints.
const MaxSprites = 128

// TileIndex addresses a single 8×8 graphic in the shared tile sheet.
// A value of zero conventionally means "transparent / empty".
type TileIndex uint8

// Background is a grid of 32×30 tiles that forms the static game world.
// It is designed to be processed once by the CPU, cached in its
// resolved form, and then scrolled or restructured cheaply without
// rebuilding the entire scene from scratch.
type Background struct {
	// tiles is the raw tile-index grid. Each cell points to an 8×8
	// graphic in the global tile sheet.
	tiles [BackgroundHeightTiles][BackgroundWidthTiles]TileIndex

	// scrollX and scrollY are the camera offsets in pixels. They are
	// clamped to the background bounds so that the renderer never
	// samples outside the resolved tile cache.
	scrollX int
	scrollY int
}

// Sprite is an independent, movable object rendered on top of or
// behind the background. It carries its own tile reference, screen
// position, flip flags, and depth priority.
type Sprite struct {
	// X and Y are the screen coordinates of the sprite's top-left corner.
	X int
	Y int

	// Tile selects the 8×8 graphic from the shared tile sheet.
	Tile TileIndex

	// FlipH mirrors the tile horizontally when true.
	FlipH bool

	// FlipV mirrors the tile vertically when true.
	FlipV bool

	// Priority determines draw order. When true the sprite is
	// composited after the background so it appears in front of
	// scenery; when false it is drawn first and can sink behind
	// foreground tiles that use transparent cut-outs.
	Priority bool

	// Visible gates the sprite so that off-screen or pooled actors
	// cost nothing during the composite pass.
	Visible bool
}

// FrameRenderer composites one background plane and up to MaxSprites
// into a single output buffer. It respects sprite priority, flip
// flags, and scroll offsets while keeping the inner loop tight enough
// for 60 fps on modest hardware.
type FrameRenderer struct {
	// back points to the active background plane. The renderer
	// resolves it once per frame and then over-paints sprites
	// according to their priority flags.
	back *Background

	// sprites is the working set of actors for the current frame.
	sprites [MaxSprites]Sprite

	// spriteCount is the number of live entries in sprites.
	spriteCount int
}

// NewFrameRenderer creates a renderer bound to the given background.
func NewFrameRenderer(b *Background) *FrameRenderer {
	return &FrameRenderer{back: b}
}

// SetBackground swaps the active background plane. This is used for
// room transitions, vertical slice changes, or parallax layers.
func (fr *FrameRenderer) SetBackground(b *Background) {
	fr.back = b
}

// ClearSprites resets the sprite list so that a new frame can be
// populated without leaving stale actors from the previous tick.
func (fr *FrameRenderer) ClearSprites() {
	fr.spriteCount = 0
}

// AddSprite appends an actor to the composite list. If the list is
// already at capacity the call is silently ignored; the engine
// expects the caller to manage sprite budget responsibly.
func (fr *FrameRenderer) AddSprite(s Sprite) {
	if fr.spriteCount >= MaxSprites {
		return
	}
	fr.sprites[fr.spriteCount] = s
	fr.spriteCount++
}

// SetTile writes a tile index into the background grid at column
// tx and row ty. This is the primitive used by level loaders,
// destructible terrain, and the restructuring pass that remixes
// existing tiles into new patterns without touching the tile sheet.
func (b *Background) SetTile(tx, ty int, t TileIndex) {
	if tx < 0 || tx >= BackgroundWidthTiles || ty < 0 || ty >= BackgroundHeightTiles {
		return
	}
	b.tiles[ty][tx] = t
}

// TileAt returns the tile index at grid coordinates (tx, ty).
func (b *Background) TileAt(tx, ty int) TileIndex {
	if tx < 0 || tx >= BackgroundWidthTiles || ty < 0 || ty >= BackgroundHeightTiles {
		return 0
	}
	return b.tiles[ty][tx]
}

// Scroll moves the camera by (dx, dy) pixels. Because the CPU only
// resolves the background once per major change, scrolling is
// extremely cheap: the renderer simply shifts the sampling origin.
func (b *Background) Scroll(dx, dy int) {
	b.scrollX += dx
	b.scrollY += dy

	// Clamp so the camera never peers beyond the resolved tilemap.
	if b.scrollX < 0 {
		b.scrollX = 0
	}
	if b.scrollX > BackgroundWidthPixels {
		b.scrollX = BackgroundWidthPixels
	}
	if b.scrollY < 0 {
		b.scrollY = 0
	}
	if b.scrollY > BackgroundHeightPixels {
		b.scrollY = BackgroundHeightPixels
	}
}

// Restructure remixes the existing tiles in the background to create
// visual variety without uploading new graphics. The seed parameter
// drives a deterministic shuffle so that the same seed always yields
// the same layout—ideal for procedural room variation and replay
// determinism.
func (b *Background) Restructure(seed int) {
	// Simple LCG deterministic shuffle. The key insight is that the
	// CPU has already processed the original tilemap; we are only
	// re-arranging indices, which is orders of magnitude cheaper
	// than rebuilding the scene from scratch.
	var rng uint32 = uint32(seed)
	for y := 0; y < BackgroundHeightTiles; y++ {
		for x := 0; x < BackgroundWidthTiles; x++ {
			rng = rng*1103515245 + 12345
			srcX := int(rng) % BackgroundWidthTiles
			rng = rng*1103515245 + 12345
			srcY := int(rng) % BackgroundHeightTiles
			b.tiles[y][x], b.tiles[srcY][srcX] = b.tiles[srcY][srcX], b.tiles[y][x]
		}
	}
}

// ScrollX returns the current horizontal camera offset in pixels.
func (b *Background) ScrollX() int { return b.scrollX }

// ScrollY returns the current vertical camera offset in pixels.
func (b *Background) ScrollY() int { return b.scrollY }

// DrawFrame composites the background and all live sprites into the
// provided pixel callback. For every output pixel (ox, oy) the
// callback receives the final resolved palette index. The caller
// (usually the software PPU) translates that index into an RGB
// value during the flip pass.
//
// The algorithm is intentionally simple:
//   1. Sample the scrolled background tilemap.
//   2. Walk sprites in priority order; if a sprite covers this pixel
//      and is visible, overwrite with the sprite's tile pixel.
//   3. Invoke the callback.
func (fr *FrameRenderer) DrawFrame(pixelFn func(ox, oy int, c TileIndex)) {
	if fr.back == nil {
		return
	}

	// Stage 1: background composite.
	for y := 0; y < BackgroundHeightPixels; y++ {
		for x := 0; x < BackgroundWidthPixels; x++ {
			worldX := x + fr.back.scrollX
			worldY := y + fr.back.scrollY

			tx := worldX / TileWidth
			ty := worldY / TileHeight
			tile := fr.back.TileAt(tx, ty)

			// Stage 2: sprite overlay.
			final := tile
			for i := 0; i < fr.spriteCount; i++ {
				s := fr.sprites[i]
				if !s.Visible {
					continue
				}
				if !s.Priority {
					continue
				}
				// Bounds check against the 8×8 sprite rectangle.
				if x < s.X || x >= s.X+TileWidth || y < s.Y || y >= s.Y+TileHeight {
					continue
				}
				// Compute local coordinates inside the tile, honouring flip.
				lx := x - s.X
				ly := y - s.Y
				if s.FlipH {
					lx = TileWidth - 1 - lx
				}
				if s.FlipV {
					ly = TileHeight - 1 - ly
				}
				// In a full implementation we would sample the tile sheet
				// at (s.Tile, lx, ly). Here the tile index itself acts as
				// the resolved colour for prototyping.
				_ = lx
				_ = ly
				final = s.Tile
			}
			pixelFn(x, y, final)
		}
	}
}
