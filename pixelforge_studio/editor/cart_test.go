package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func TestNewEditorCart_CanvasDimensions(t *testing.T) {
	c := newEditorCart()
	assert.Equal(t, EditorCanvasW, c.canvas.W())
	assert.Equal(t, EditorCanvasH, c.canvas.H())
}

func TestEditorCart_RootCoversCanvas(t *testing.T) {
	c := newEditorCart()
	require.NotNil(t, c.Root())
	assert.Equal(t, EditorCanvasW, c.Root().W)
	assert.Equal(t, EditorCanvasH, c.Root().H)
}

func TestEditorCart_FocusManagerExists(t *testing.T) {
	c := newEditorCart()
	assert.NotNil(t, c.FocusManager())
}

func TestEditorCart_SetTheme(t *testing.T) {
	c := newEditorCart()
	theme := DefaultEditorTheme()
	c.SetTheme(theme)
	assert.Same(t, theme, c.Theme())
}

func TestEditorCart_ResetWorkspaceRoot(t *testing.T) {
	c := newEditorCart()
	first := c.WorkspaceRoot()
	rel := widgets.Rect{X: 100, Y: 50, W: 800, H: 600}
	fresh := c.ResetWorkspaceRoot(rel)
	assert.NotSame(t, first, fresh)
	assert.Equal(t, rel.X, fresh.X)
	assert.Equal(t, rel.Y, fresh.Y)
	assert.Equal(t, rel.W, fresh.W)
	assert.Equal(t, rel.H, fresh.H)
}

func TestEditorCart_WindowToCanvas(t *testing.T) {
	c := newEditorCart()
	// Workspace rect is exactly the canvas size — no letterbox.
	rect := widgets.Rect{X: 0, Y: 0, W: EditorCanvasW, H: EditorCanvasH}

	cx, cy, inside := c.WindowToCanvas(rect, 640, 400)
	assert.True(t, inside)
	assert.Equal(t, 640, cx)
	assert.Equal(t, 400, cy)
}

func TestEditorCart_WindowToCanvas_OutsideReturnsInsideFalse(t *testing.T) {
	c := newEditorCart()
	rect := widgets.Rect{X: 100, Y: 100, W: 200, H: 200}
	// Point outside the workspace rect entirely.
	_, _, inside := c.WindowToCanvas(rect, 50, 50)
	assert.False(t, inside)
}

func TestEditorCart_WindowToCanvas_DegenerateRect(t *testing.T) {
	c := newEditorCart()
	rect := widgets.Rect{X: 0, Y: 0, W: 0, H: 0}
	_, _, inside := c.WindowToCanvas(rect, 10, 10)
	assert.False(t, inside)
}

func TestEditorOwnsCart(t *testing.T) {
	e := New()
	require.NotNil(t, e.Cart())
	assert.NotNil(t, e.Cart().Theme(), "cart must have a default theme loaded")
}
