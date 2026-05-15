package scripting_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace_NameMatchesStub(t *testing.T) {
	w := scripting.NewWorkspace()
	assert.Equal(t, "behavior", w.Name())
	assert.Equal(t, "Behavior", w.DisplayName())
}

func TestRegisterWith_ReplacesM3Stub(t *testing.T) {
	e := editor.New()
	preCount := len(e.Workspaces())
	w := scripting.RegisterWith(e)
	postCount := len(e.Workspaces())
	assert.Equal(t, preCount, postCount, "RegisterWith should replace the stub, not append")
	// Find the registered workspace by name and verify identity.
	var found editor.Workspace
	for _, ws := range e.Workspaces() {
		if ws.Name() == "behavior" {
			found = ws
			break
		}
	}
	assert.NotNil(t, found)
	assert.Equal(t, "behavior", found.Name())
	assert.Same(t, w, found)
}

func TestWorkspace_TabSwitching(t *testing.T) {
	w := scripting.NewWorkspace()
	assert.Equal(t, scripting.PaneLane, w.ActiveTab())
	w.SetActiveTab(scripting.PaneSheet)
	assert.Equal(t, scripting.PaneSheet, w.ActiveTab())
	w.SetActiveTab(scripting.PaneCatalog)
	assert.Equal(t, scripting.PaneCatalog, w.ActiveTab())
	w.SetActiveTab(scripting.PaneDebug)
	assert.Equal(t, scripting.PaneDebug, w.ActiveTab())
}

func TestWorkspace_OnProjectChanged_StartsEngine(t *testing.T) {
	w := scripting.NewWorkspace()
	p := pixelforge_project.NewProject("demo")
	w.OnProjectChanged(p)
	assert.NotNil(t, w.Engine(), "OnProjectChanged should start an engine")
	assert.True(t, w.Engine().Running())
	w.OnProjectChanged(nil)
	assert.Nil(t, w.Engine(), "nil project should tear down the engine")
}

func TestWorkspace_SubpanesExpose(t *testing.T) {
	w := scripting.NewWorkspace()
	assert.NotNil(t, w.Lane())
	assert.NotNil(t, w.Sheet())
	assert.NotNil(t, w.Catalog())
	assert.NotNil(t, w.Debugger())
}

func TestEditor_RegisterProjectListener_FiresImmediately(t *testing.T) {
	e := editor.New()
	var got int
	listener := &countingListener{onChange: func(_ *pixelforge_project.Project) { got++ }}
	e.RegisterProjectListener(listener)
	assert.Equal(t, 1, got, "listener should be fired once on register with current project")

	e.SetProject(pixelforge_project.NewProject("a"))
	assert.Equal(t, 2, got)

	// Re-setting the same pointer should not re-fire.
	e.SetProject(e.Project())
	assert.Equal(t, 2, got)

	e.SetProject(pixelforge_project.NewProject("b"))
	assert.Equal(t, 3, got)
}

type countingListener struct {
	onChange func(*pixelforge_project.Project)
}

func (c *countingListener) OnProjectChanged(p *pixelforge_project.Project) {
	c.onChange(p)
}
