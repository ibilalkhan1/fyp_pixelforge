package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// viewBox preserves the project's aspect ratio inside the available rect.
func TestCanvas_ViewBoxAspectRatio(t *testing.T) {
	c := NewCanvas()
	p := pixelforge_project.NewProject("t")
	p.ScreenWidth = 320
	p.ScreenHeight = 180 // 16:9

	// Wider area → pillarbox.
	vb := c.viewBox(widgets.Rect{W: 1600, H: 400}, p)
	assert.Equal(t, 400, vb.H)
	// Aspect 16:9 → w = 400 * 16/9 ≈ 711
	assert.InDelta(t, 711, vb.W, 2)

	// Taller area → letterbox.
	vb2 := c.viewBox(widgets.Rect{W: 400, H: 1600}, p)
	assert.Equal(t, 400, vb2.W)
	assert.InDelta(t, 225, vb2.H, 2) // 400 * 9/16
}

// PlaceEntity refuses to mint when no sprite is selected.
func TestCanvas_PlaceEntityRequiresSprite(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	got := e.canvas.PlaceEntity(scene, e, 10, 20)
	assert.Nil(t, got)
	assert.Empty(t, scene.Entities)
	assert.Contains(t, e.StatusMessage(), "Select a sprite")
}

// PlaceEntity adds an entity at the given scene-space coordinates.
func TestCanvas_PlaceEntityAddsEntity(t *testing.T) {
	e := New()
	e.SetSelectedSpriteName("fruit")
	scene := e.activeScene()
	require.NotNil(t, scene)

	ent := e.canvas.PlaceEntity(scene, e, 40, 60)
	require.NotNil(t, ent)
	assert.Len(t, scene.Entities, 1)
	assert.Equal(t, float64(40), ent.Position.X)
	assert.Equal(t, float64(60), ent.Position.Y)
	require.Len(t, ent.Components, 1)
	assert.Equal(t, "Sprite", ent.Components[0].Type)
	assert.Equal(t, "fruit", ent.Components[0].Values["sprite"])
	assert.True(t, e.IsDirty())
	assert.Equal(t, ent.ID, e.SelectedEntityID())
}

// DeleteEntityAt removes the topmost entity at the hit point.
func TestCanvas_DeleteEntityAtRemovesTopmost(t *testing.T) {
	e := New()
	e.SetSelectedSpriteName("fruit")
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "a", Position: pixelforge_project.EntityPosition{X: 50, Y: 50}},
		{ID: "b", Position: pixelforge_project.EntityPosition{X: 50, Y: 50}},
	}
	// In a 320×180 scene rendered into a 320×180 view-box, position (50,50)
	// projects to window pixel (50,50).
	viewBox := widgets.Rect{X: 0, Y: 0, W: 320, H: 180}
	deleted := e.canvas.DeleteEntityAt(scene, viewBox, e.Project(), 50, 50)
	assert.Equal(t, "b", deleted, "topmost (last in array) is deleted")
	assert.Len(t, scene.Entities, 1)
	assert.Equal(t, "a", scene.Entities[0].ID)
}

// PickEntityAt returns the ID of the entity under the cursor.
func TestCanvas_PickEntityAt(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "head", Position: pixelforge_project.EntityPosition{X: 30, Y: 40}},
	}
	viewBox := widgets.Rect{X: 0, Y: 0, W: 320, H: 180}
	got := e.canvas.PickEntityAt(scene, viewBox, e.Project(), 30, 40)
	assert.Equal(t, "head", got)

	miss := e.canvas.PickEntityAt(scene, viewBox, e.Project(), 200, 200)
	assert.Equal(t, "", miss)
}

// windowToScene clamps to scene bounds.
func TestCanvas_WindowToSceneClamps(t *testing.T) {
	c := NewCanvas()
	p := pixelforge_project.NewProject("t")
	p.ScreenWidth = 100
	p.ScreenHeight = 50
	viewBox := widgets.Rect{X: 0, Y: 0, W: 100, H: 50}
	x, y := c.windowToScene(viewBox, p, -5, -10)
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)
	x, y = c.windowToScene(viewBox, p, 999, 999)
	assert.Equal(t, p.ScreenWidth, x)
	assert.Equal(t, p.ScreenHeight, y)
}

// DeleteEntityAt with no hit returns "".
func TestCanvas_DeleteEntityAtMiss(t *testing.T) {
	e := New()
	scene := e.activeScene()
	scene.Entities = []pixelforge_project.Entity{
		{ID: "a", Position: pixelforge_project.EntityPosition{X: 10, Y: 10}},
	}
	viewBox := widgets.Rect{X: 0, Y: 0, W: 320, H: 180}
	got := e.canvas.DeleteEntityAt(scene, viewBox, e.Project(), 200, 200)
	assert.Equal(t, "", got)
	assert.Len(t, scene.Entities, 1)
}
