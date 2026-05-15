package palette

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ebitenutilPrint is a single chokepoint for debug text so future
// themed font work can swap implementations without churning callers.
func ebitenutilPrint(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, s, x, y)
}
