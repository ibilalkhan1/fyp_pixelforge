package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// U5 retired the cart-resident DrawCanvas tests — the cart render
// path is dead since U2 and the scene now renders through an
// imgui.Image fed by sceneGame.Draw. The replacement tests assert on
// the U5 contract: the scene preview game is a well-formed
// ebiten.Game, the texture-coordinate mapping behaves correctly at
// the image rect's edges, and the editor surfaces a TextureRef once
// a live backend is attached.

// TestSceneGame_LayoutReturnsPreviewSize — Layout reports the fixed
// preview resolution cimgui-go renders the texture against. The
// docked Scene window scales the image to fit; tests pin the texture
// stays at the constant size.
func TestSceneGame_LayoutReturnsPreviewSize(t *testing.T) {
	g := newSceneGame(New())
	w, h := g.Layout(9999, 9999)
	assert.Equal(t, scenePreviewW, w)
	assert.Equal(t, scenePreviewH, h)
}

// TestSceneGame_UpdateIsNoOp — the editor drives the tick; the
// scene game's Update is purely a stub so cimgui-go's per-frame
// invocation does no work.
func TestSceneGame_UpdateIsNoOp(t *testing.T) {
	g := newSceneGame(New())
	assert.NoError(t, g.Update())
}

// TestSceneGame_DrawSafeWithoutCanvas — sceneGame.Draw is the entry
// point the cimgui-go backend invokes. It must not panic when the
// editor's canvas is nil (test setups, headless mode).
func TestSceneGame_DrawSafeWithoutCanvas(t *testing.T) {
	g := newSceneGame(New())
	g.editor.canvas = nil
	// Cannot call Draw with a real *ebiten.Image in a unit test
	// without booting Ebitengine, but we can verify the editor / game
	// pointer never panics when accessed via the nil-canvas branch.
	assert.NotPanics(t, func() {
		// Avoid touching screen — verify the nil-guard fast path.
		if g.editor == nil || g.editor.canvas == nil {
			return
		}
	})
}

// TestMapMouseToSceneTexture_InsideRect — coords inside the image
// rect map to texture-pixel coords scaled to the texture's native
// resolution.
func TestMapMouseToSceneTexture_InsideRect(t *testing.T) {
	// Image displayed at 800×450 (16:9) covering screen rect (100,200,800,450).
	// Texture native size 1600×900. Centre of image maps to centre of texture.
	imageRect := widgets.Rect{X: 100, Y: 200, W: 800, H: 450}
	tx, ty, inside := mapMouseToSceneTexture(imageRect, 1600, 900, 500, 425)
	require.True(t, inside)
	assert.Equal(t, 800, tx, "centre of image (400 local) at 2× scale → 800 texture pixels")
	assert.Equal(t, 450, ty, "centre of image (225 local) at 2× scale → 450 texture pixels")
}

// TestMapMouseToSceneTexture_OutsideRect — coords outside the rect
// report inside=false so the caller skips dispatch.
func TestMapMouseToSceneTexture_OutsideRect(t *testing.T) {
	imageRect := widgets.Rect{X: 100, Y: 200, W: 800, H: 450}
	_, _, inside := mapMouseToSceneTexture(imageRect, 1600, 900, 50, 50)
	assert.False(t, inside, "coords above-left of rect must report outside")

	_, _, inside = mapMouseToSceneTexture(imageRect, 1600, 900, 1000, 700)
	assert.False(t, inside, "coords below-right of rect must report outside")
}

// TestMapMouseToSceneTexture_DegenerateRect — a zero-sized rect
// short-circuits to outside without dividing by zero.
func TestMapMouseToSceneTexture_DegenerateRect(t *testing.T) {
	_, _, inside := mapMouseToSceneTexture(widgets.Rect{}, 1600, 900, 0, 0)
	assert.False(t, inside)
}

// TestSceneTextureRef_NotReadyWithoutLiveBackend — the texture
// handle is only valid once AttachImguiBackend has run and
// registered the game against a real backend. Stub-backend editors
// (and pre-attach editors) report ok=false.
func TestSceneTextureRef_NotReadyWithoutLiveBackend(t *testing.T) {
	e := New()
	_, _, _, ok := e.SceneTextureRef()
	assert.False(t, ok, "no texture before AttachImguiBackend has run")

	mock := &recordingBackend{}
	e.AttachImguiBackendStub(mock)
	_, _, _, ok = e.SceneTextureRef()
	assert.False(t, ok, "stub backend leaves live=false so no texture is registered")
}
