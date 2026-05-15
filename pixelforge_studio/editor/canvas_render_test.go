package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func TestSceneWorkspace_DrawCanvas_PaintsViewport(t *testing.T) {
	e := New()
	// Add a project with non-trivial screen size so viewBox isn't degenerate.
	e.project = pixelforge_project.NewProject("snake")
	e.project.ScreenWidth = 320
	e.project.ScreenHeight = 180

	cart := e.Cart()
	require.NotNil(t, cart)

	// Render directly into the cart canvas via the workspace's
	// canvas-resident path. We use a small workspace rect so the test
	// can sample known pixel positions.
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	ws := &SceneWorkspace{}
	ws.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 320, H: 180}, e)

	// The viewport should leave a non-empty image (some pixels non-zero).
	any := false
	for x := 0; x < 320 && !any; x += 8 {
		for y := 0; y < 180 && !any; y += 8 {
			if cart.canvas.Get(x, y) != 0 {
				any = true
			}
		}
	}
	assert.True(t, any, "DrawCanvas should populate the workspace region with theme colours")
}

func TestSceneWorkspace_DrawCanvas_EntityMarker(t *testing.T) {
	e := New()
	e.SetSelectedSpriteName("fruit")
	scene := e.activeScene()
	require.NotNil(t, scene)
	ent := e.canvas.PlaceEntity(scene, e, 100, 90)
	require.NotNil(t, ent)
	e.SelectEntity(ent.ID)

	cart := e.Cart()
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	ws := &SceneWorkspace{}
	ws.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 640, H: 360}, e)

	// Entity (100, 90) at scale=2 paints a 12×12 marker centred on
	// canvas (200, 180). Sample inside that rect.
	accent := cart.Theme().AccentSlot
	found := false
	for x := 194; x < 206 && !found; x++ {
		for y := 174; y < 186 && !found; y++ {
			if cart.canvas.Get(x, y) == accent {
				found = true
			}
		}
	}
	assert.True(t, found, "expected an accent-coloured entity marker in the canvas")
}

func TestSceneWorkspace_ImplementsCanvasWorkspace(t *testing.T) {
	var _ CanvasWorkspace = (*SceneWorkspace)(nil)
}
