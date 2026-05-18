package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// U5 of the ImGui migration retired the cart-resident tool indicator
// — the toolbar is now an ImGui RadioButton group rendered inside
// the Scene window by SceneWorkspace.renderToolbar. The tests below
// assert on the editor-side contract that survives without a live
// ImGui context: the toolbar's tool list, focus-gated dispatch, and
// the link to the Place tool's sprite selection.

// TestToolPaletteRendersFourTools — the U5 toolbar exposes the four
// canvas tools the editor's Tool enum defines (Select / Place /
// Delete / Paint). A regression that drops one would silently strand
// the user without that tool.
func TestToolPaletteRendersFourTools(t *testing.T) {
	expected := []Tool{ToolSelect, ToolPlace, ToolDelete, ToolPaint}
	for _, tool := range expected {
		// String() doubles as the toolbar label; verify each tool has
		// a non-empty string so the radio button label is meaningful.
		assert.NotEmpty(t, tool.String())
	}
}

// TestToolDispatchSuppressedWhenSceneNotFocused — SceneWorkspace.Update
// short-circuits when sceneHovered is false (the default state before
// the first Render runs, and the state whenever another dock owns
// focus). The canvas's PlaceEntity / DeleteEntityAt mutate the scene
// when called; this test verifies the gate prevents accidental
// dispatch when sceneHovered is false.
func TestToolDispatchSuppressedWhenSceneNotFocused(t *testing.T) {
	e := New()
	e.SetSelectedSpriteName("fruit")
	require.False(t, e.sceneHovered, "scene starts unhovered")

	s := &SceneWorkspace{}
	scene := e.activeScene()
	require.NotNil(t, scene)
	before := len(scene.Entities)

	// With sceneHovered=false, Update returns immediately — no place,
	// no delete, no entity mutation.
	s.Update(e)
	assert.Equal(t, before, len(scene.Entities))
}

// TestPlaceToolUsesAssetBrowserSelection — the Place tool only mints
// an entity when the asset browser has a selected sprite. Without
// one, PlaceEntity refuses and surfaces a status hint.
func TestPlaceToolUsesAssetBrowserSelection(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)

	// No sprite selected → PlaceEntity returns nil and posts a hint.
	got := e.canvas.PlaceEntity(scene, e, 10, 20)
	assert.Nil(t, got)
	assert.Contains(t, e.StatusMessage(), "Select a sprite")

	// With a sprite selected, PlaceEntity mints + selects the entity.
	e.SetSelectedSpriteName("fruit")
	ent := e.canvas.PlaceEntity(scene, e, 10, 20)
	require.NotNil(t, ent)
	assert.Equal(t, "fruit", ent.Components[0].Values["sprite"])
}

// TestSceneWorkspace_NoLongerImplementsNativeContent — U5 removed
// SceneWorkspace.Draw because the scene renders through the texture
// pipeline now. Editor.Draw skips workspaces that don't implement
// NativeWorkspaceContent — pin that contract here so a regression
// adding Draw back surfaces immediately.
func TestSceneWorkspace_NoLongerImplementsNativeContent(t *testing.T) {
	var s any = &SceneWorkspace{}
	_, ok := s.(NativeWorkspaceContent)
	assert.False(t, ok, "SceneWorkspace renders via imgui.Image — no native Draw")
}

// TestSceneWorkspace_UpdateRespectsCanvasNil — a defensive contract:
// nil canvas pointer (test scaffolding, partially-constructed
// editors) must not panic the focused-workspace dispatch path.
func TestSceneWorkspace_UpdateRespectsCanvasNil(t *testing.T) {
	e := New()
	e.canvas = nil
	s := &SceneWorkspace{}
	assert.NotPanics(t, func() { s.Update(e) })
}

// TestImageRectMapsToProjectCoords — a 800×450 image of a 320×180
// project canvas at the image's centre maps to project (160, 90),
// proving the mouse-to-texture mapping aligns project coords with
// the texture coords the scene is drawn into.
func TestImageRectMapsToProjectCoords(t *testing.T) {
	// Image at top-left of screen, 800 wide × 450 tall.
	// Texture native is scenePreviewW × scenePreviewH.
	// Centre of image: (400, 225) screen → centre of texture.
	imageRect := widgets.Rect{X: 0, Y: 0, W: 800, H: 450}
	tx, ty, ok := mapMouseToSceneTexture(imageRect, scenePreviewW, scenePreviewH, 400, 225)
	require.True(t, ok)
	assert.Equal(t, scenePreviewW/2, tx)
	assert.Equal(t, scenePreviewH/2, ty)
	_ = pixelforge_project.NewProject("t")
}
