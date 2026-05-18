package editor

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

type stubWorkspace struct {
	name        string
	displayName string
	updates     int
	state       int
	renders     int
}

func (s *stubWorkspace) Name() string                              { return s.name }
func (s *stubWorkspace) DisplayName() string                       { return s.displayName }
func (s *stubWorkspace) Render(*Editor)                            { s.renders++ }
func (s *stubWorkspace) Draw(*ebiten.Image, widgets.Rect, *Editor) {}
func (s *stubWorkspace) Update(*Editor)                            { s.updates++ }

// U3 ImGui-DockSpace migration deleted the M3 placeholder workspaces
// (behavior, audio, capture, procgen) — real packages register the
// ones they own, and with DockSpace there's no tab strip that needs
// stable placeholders to fill. Default editor now registers only the
// Scene workspace; palette / capture / scripting append on top via
// their respective RegisterWith.
func TestEditor_RegistersSceneWorkspaceByDefault(t *testing.T) {
	e := New()
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
	assert.Len(t, e.Workspaces(), 1)
	assert.Equal(t, "scene", e.Workspaces()[0].Name())
}

// SetActiveWorkspaceByName routes to the named workspace.
func TestEditor_SetActiveWorkspaceByName(t *testing.T) {
	e := New()
	e.RegisterWorkspace(&stubWorkspace{name: "palette", displayName: "Palette"})

	e.SetActiveWorkspaceByName("palette")
	assert.Equal(t, "palette", e.ActiveWorkspaceName())

	e.SetActiveWorkspaceByName("scene")
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
}

// Unknown workspace name is a no-op + status-bar warning.
func TestEditor_SetActiveWorkspaceByNameUnknown(t *testing.T) {
	e := New()
	before := e.ActiveWorkspaceName()
	e.SetActiveWorkspaceByName("nope")
	assert.Equal(t, before, e.ActiveWorkspaceName())
	assert.Contains(t, e.StatusMessage(), "unknown workspace")
}

// CycleWorkspace advances through registered workspaces in
// registration order. After New() only Scene is registered; tests
// register more to exercise the cycle. SetWindowFocus is skipped
// because no live ImGui backend is attached.
func TestEditor_CycleWorkspace(t *testing.T) {
	e := New()
	e.RegisterWorkspace(&stubWorkspace{name: "palette", displayName: "Palette"})
	e.RegisterWorkspace(&stubWorkspace{name: "capture", displayName: "Capture"})

	e.SetActiveWorkspaceByName("scene")
	e.CycleWorkspace()
	assert.Equal(t, "palette", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "capture", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
}

// Re-registering by name replaces the prior entry rather than duplicating.
// After New() the editor has 1 workspace (scene); re-registering "scene"
// must not grow that list.
func TestEditor_RegisterWorkspaceIdempotent(t *testing.T) {
	e := New()
	before := len(e.Workspaces())
	a := &stubWorkspace{name: "scene", displayName: "A"}
	b := &stubWorkspace{name: "scene", displayName: "B"}
	e.RegisterWorkspace(a)
	e.RegisterWorkspace(b)
	assert.Len(t, e.Workspaces(), before)
	for _, w := range e.Workspaces() {
		if w.Name() == "scene" {
			assert.Equal(t, "B", w.DisplayName())
		}
	}
}

// Workspace state survives a switch and back (the editor doesn't
// reinitialise workspaces on activation).
func TestEditor_WorkspaceStateSurvivesSwitch(t *testing.T) {
	e := New()
	sw := &stubWorkspace{name: "palette", displayName: "Palette"}
	e.RegisterWorkspace(sw)
	e.SetActiveWorkspaceByName("palette")

	sw.state = 42
	e.SetActiveWorkspaceByName("scene")
	e.SetActiveWorkspaceByName("palette")
	assert.Equal(t, 42, sw.state, "state survives switching away and back")
}

// (TestEditor_HandleTabStripClick removed in U2 — the native tab
// strip is gone; workspace switching now flows through the ImGui View
// menu and, in U3, DockSpace tab handling.)

// Compile-time assertion that SceneWorkspace satisfies the interface.
var _ Workspace = (*SceneWorkspace)(nil)
