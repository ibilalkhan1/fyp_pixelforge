package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func TestInspector_DrawCanvas_NoSelectionRendersPlaceholder(t *testing.T) {
	insp := NewInspector()
	cart := newEditorCart()
	cart.SetTheme(DefaultEditorTheme())

	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	assert.NotPanics(t, func() {
		insp.DrawCanvas(widgets.Rect{X: 100, Y: 50, W: 260, H: 600}, nil, nil, cart.Theme())
	})
}

func TestInspector_DrawCanvas_RendersHeader(t *testing.T) {
	insp := NewInspector()
	cart := newEditorCart()
	cart.SetTheme(DefaultEditorTheme())

	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	insp.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 260, H: 600}, nil, nil, cart.Theme())

	// Panel header strip should be the PanelHeaderSlot colour.
	hdr := cart.Theme().PanelHeaderSlot
	found := false
	for x := 0; x < 260 && !found; x++ {
		if cart.canvas.Get(x, 5) == hdr {
			found = true
		}
	}
	assert.True(t, found, "expected panel header pixels at the top of the inspector")
}

func TestInspector_DrawCanvas_RendersEntityName(t *testing.T) {
	e := New()
	e.SetSelectedSpriteName("fruit")
	scene := e.activeScene()
	require.NotNil(t, scene)
	ent := e.canvas.PlaceEntity(scene, e, 10, 20)
	require.NotNil(t, ent)
	e.SelectEntity(ent.ID)

	cart := e.Cart()
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	e.inspector.DrawCanvas(
		widgets.Rect{X: 0, Y: 0, W: 260, H: 600},
		e.project, e.selectedEntity(), cart.Theme(),
	)

	// Some non-zero pixels in the row where the entity name renders.
	found := false
	for x := 8; x < 200 && !found; x++ {
		for y := 22; y < 32 && !found; y++ {
			if cart.canvas.Get(x, y) != 0 {
				found = true
			}
		}
	}
	assert.True(t, found, "expected entity name text in the inspector canvas")
}

func TestInspector_DrawCanvas_UnknownComponentRendersWarning(t *testing.T) {
	insp := NewInspector()
	cart := newEditorCart()
	cart.SetTheme(DefaultEditorTheme())

	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	ent := pixelforge_project.Entity{
		ID:   "x",
		Name: "Test",
		Components: []pixelforge_project.EntityComponent{
			{Type: "DefinitelyNotRegistered", Values: map[string]any{}},
		},
	}
	assert.NotPanics(t, func() {
		insp.DrawCanvas(
			widgets.Rect{X: 0, Y: 0, W: 260, H: 600},
			nil, &ent, cart.Theme(),
		)
	})
}

func TestAssetBrowser_DrawCanvas_DoesNotPanic(t *testing.T) {
	e := New()
	cart := e.Cart()
	prev := pixelforge.SetDrawTarget(cart.canvas)
	defer pixelforge.SetDrawTarget(prev)
	cart.canvas.Clear(0)

	assert.NotPanics(t, func() {
		e.assetBrowser.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 220, H: 600}, e)
	})
}
