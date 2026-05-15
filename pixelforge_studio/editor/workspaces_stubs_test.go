package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

func TestEditor_RegistersStubWorkspaces(t *testing.T) {
	e := New()
	names := []string{}
	for _, w := range e.Workspaces() {
		names = append(names, w.Name())
	}
	assert.Equal(t, []string{"scene", "behavior", "audio", "capture", "procgen"}, names)
}

func TestStubWorkspace_ChangesActiveOnSet(t *testing.T) {
	e := New()
	e.SetActiveWorkspaceByName("audio")
	assert.Equal(t, "audio", e.ActiveWorkspaceName())
}

func TestStubWorkspace_DrawCanvasDoesNotPanic(t *testing.T) {
	e := New()
	cart := e.Cart()
	require.NotNil(t, cart)
	prev := pixelforge.SetDrawTarget(cart.Canvas())
	defer pixelforge.SetDrawTarget(prev)
	cart.Canvas().Clear(0)

	for _, name := range []string{"behavior", "audio", "capture", "procgen"} {
		e.SetActiveWorkspaceByName(name)
		ws := e.activeWorkspaceImpl()
		cw, ok := ws.(CanvasWorkspace)
		require.True(t, ok, "stub %q should implement CanvasWorkspace", name)
		assert.NotPanics(t, func() {
			cw.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: 320, H: 240}, e)
		})
	}
}

func TestPlaceholderWorkspace_Update_IsNoOp(t *testing.T) {
	w := newPlaceholderWorkspace("foo", "Foo", "M9")
	e := New()
	assert.NotPanics(t, func() { w.Update(e) })
}
