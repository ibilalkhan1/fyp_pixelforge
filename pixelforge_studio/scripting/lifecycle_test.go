package scripting_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycle_RegisterPreparesEngineUntilPlay(t *testing.T) {
	e := editor.New()
	w := scripting.RegisterWith(e)
	require.NotNil(t, w.Engine())
	assert.False(t, w.Engine().Running(), "editing prepares rules without executing the game")
}

func TestLifecycle_SwitchProject_StopsThenPrepares(t *testing.T) {
	e := editor.New()
	w := scripting.RegisterWith(e)
	first := w.Engine()

	e.SetProject(pixelforge_project.NewProject("second"))
	second := w.Engine()
	assert.NotSame(t, first, second, "switching projects should rebuild the engine")
	assert.False(t, second.Running(), "the replacement waits for Play")
	assert.False(t, first.Running(), "old engine should stop on project swap")
}

func TestLifecycle_NilProjectStopsEngine(t *testing.T) {
	e := editor.New()
	w := scripting.RegisterWith(e)
	e.SetProject(nil)
	assert.Nil(t, w.Engine())
}

func TestLifecycle_SameProjectIsNoOp(t *testing.T) {
	e := editor.New()
	w := scripting.RegisterWith(e)
	first := w.Engine()
	e.SetProject(e.Project())
	assert.Same(t, first, w.Engine(), "re-setting the same project should not restart the engine")
}
