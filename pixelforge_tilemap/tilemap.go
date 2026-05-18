package pixelforge_tilemap

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Drawer abstracts pixelforge.DrawSprite so unit tests can inject a
// call recorder without standing up the engine's global draw target.
// pixelforge.DrawSprite satisfies the same shape; the package-private
// engineDrawer adapts it. Tests construct a recorder, pass it to
// RenderWith, and assert on the captured calls.
type Drawer interface {
	DrawSprite(sprite pixelforge.Sprite, dx, dy int)
}

// engineDrawer routes Drawer calls through the package-level
// pixelforge.DrawSprite. It carries no state; Render constructs one
// per call so the indirection has no allocation cost beyond the
// interface boxing the compiler already pays.
type engineDrawer struct{}

func (engineDrawer) DrawSprite(sprite pixelforge.Sprite, dx, dy int) {
	pixelforge.DrawSprite(sprite, dx, dy)
}

// Render draws every non-empty cell of layer into the active
// pixelforge draw target, sourcing tile graphics from sheet.
//
// layer.Grid is iterated row-major; cell ID 0 is skipped (the painter
// reserves 0 as "empty"). Each non-zero cell at (col, row) with ID N
// blits the (N-1)-th tile from sheet's horizontal strip to scene-space
// destination (col*TileW, row*TileH). pixelforge.DrawSprite applies
// pixelforge.Camera internally, so the renderer hands it scene-space
// coords and lets the engine compute screen-space.
//
// Defensive behavior: nil layer, nil sheet, zero TileW/TileH, or an ID
// past the right edge of the sheet are all no-ops for that cell — the
// renderer never panics on malformed data, matching the loader's
// "malformed entries log + fall through to defaults" discipline. A
// zero-width sheet skips every cell because no source rect can be
// computed.
func Render(layer *pixelforge_project.TilemapLayer, sheet *pixelforge.Canvas) {
	RenderWith(layer, sheet, engineDrawer{})
}

// RenderWith is Render with an explicit Drawer. Tests use this to
// capture DrawSprite calls without depending on the engine's global
// draw target. Production callers should use Render.
//
// The split keeps Render's exported signature stable while leaving the
// seam for hermetic tests. Both functions share their iteration logic
// so a test that exercises RenderWith also covers Render.
func RenderWith(layer *pixelforge_project.TilemapLayer, sheet *pixelforge.Canvas, drawer Drawer) {
	if layer == nil || sheet == nil || drawer == nil {
		return
	}
	if layer.TileW <= 0 || layer.TileH <= 0 {
		return
	}

	sheetW := sheet.W()
	sheetH := sheet.H()
	if sheetW <= 0 || sheetH <= 0 {
		return
	}

	// Maximum tile index the sheet can satisfy. ID N indexes column
	// (N-1) of a horizontal strip, so the strip holds sheetW / TileW
	// tiles; IDs beyond that range have no source rect and are
	// skipped (rather than wrapping or panicking).
	maxTileIndex := sheetW / layer.TileW

	for row, line := range layer.Grid {
		for col, id := range line {
			if id <= 0 {
				continue
			}
			tileIdx := id - 1
			if tileIdx >= maxTileIndex {
				continue
			}
			src := pixelforge.SpriteFrom(*sheet, tileIdx*layer.TileW, 0, layer.TileW, layer.TileH)
			drawer.DrawSprite(src, col*layer.TileW, row*layer.TileH)
		}
	}
}
