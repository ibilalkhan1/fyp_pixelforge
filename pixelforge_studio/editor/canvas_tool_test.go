package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Sanity-check that the tool indicator writes some pixels in the
// top-left of the workspace region when a project is loaded and the
// Place tool is active.
func TestCanvas_DrawToolIndicator_PaintsTopLeft(t *testing.T) {
	e := New()
	e.SetTool(ToolPlace)
	e.SetSelectedSpriteName("fruit")

	cart := e.Cart()
	require.NotNil(t, cart)
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	ws := &SceneWorkspace{}
	ws.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 320, H: 180}, e)

	// Look for any non-background pixel in the top-left text area
	// (x=8..80, y=8..16).
	found := false
	for x := 8; x < 80 && !found; x++ {
		for y := 8; y < 16 && !found; y++ {
			if cart.canvas.Get(x, y) != 0 {
				// Could be background or text. Background is the
				// theme's BackgroundSlot; we want anything that
				// isn't background OR is the text colour. Since the
				// canvas was Clear(0), anything non-zero is something
				// the renderer wrote.
				found = true
			}
		}
	}
	assert.True(t, found, "expected tool indicator pixels in workspace top-left")
}

func TestCanvas_DrawToolIndicator_NoProjectNoCrash(t *testing.T) {
	e := New()
	// Clear scene/project to simulate headless mode.
	e.project = nil

	cart := e.Cart()
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	ws := &SceneWorkspace{}
	assert.NotPanics(t, func() {
		ws.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 320, H: 180}, e)
	})
}
