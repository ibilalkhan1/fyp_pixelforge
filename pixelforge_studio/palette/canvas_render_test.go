package palette

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func TestPaletteWorkspace_ImplementsCanvasWorkspace(t *testing.T) {
	var _ editor.CanvasWorkspace = (*Workspace)(nil)
}

func TestPaletteWorkspace_DrawCanvas_PaintsSwatches(t *testing.T) {
	e := editor.New()
	w := NewWorkspace()

	cart := e.Cart()
	require.NotNil(t, cart)
	prev := pixelforge.SetDrawTarget(cart.Canvas())
	defer pixelforge.SetDrawTarget(prev)
	cart.Canvas().Clear(0)

	w.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 800, H: 600}, e)

	// Any non-zero pixels in the swatch grid region (16..208 wide, 28..220 tall)?
	any := false
	for x := 16; x < 210 && !any; x++ {
		for y := 28; y < 220 && !any; y++ {
			if cart.Canvas().Get(x, y) != 0 {
				any = true
			}
		}
	}
	assert.True(t, any, "palette grid should populate canvas pixels")
}

func TestPaletteWorkspace_DrawCanvas_NoProjectIsNoOp(t *testing.T) {
	w := NewWorkspace()
	// Use an editor that's been zeroed of project; SetProject(nil) is
	// nominally unsupported, but we want to ensure no panic.
	e := editor.New()
	e.SetProject(nil)

	cart := e.Cart()
	prev := pixelforge.SetDrawTarget(cart.Canvas())
	defer pixelforge.SetDrawTarget(prev)
	cart.Canvas().Clear(0)

	assert.NotPanics(t, func() {
		w.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 800, H: 600}, e)
	})
}
