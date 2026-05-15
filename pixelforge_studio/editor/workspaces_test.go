package editor

import (
	"image/color"
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
}

func (s *stubWorkspace) Name() string                                 { return s.name }
func (s *stubWorkspace) DisplayName() string                          { return s.displayName }
func (s *stubWorkspace) Draw(*ebiten.Image, widgets.Rect, *Editor)    {}
func (s *stubWorkspace) Update(*Editor)                               { s.updates++ }

// Default editor registers the Scene workspace as the active one and
// installs M3 stub workspaces (Behavior, Audio, Capture, Procgen) so
// the tab strip is stable from M3 onwards. Palette is registered by
// the palette package; the studio main wires it after New() returns.
func TestEditor_RegistersSceneWorkspaceByDefault(t *testing.T) {
	e := New()
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
	assert.Len(t, e.Workspaces(), 5)
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
// registration order. After New() the order is:
//
//	scene → behavior → audio → capture → procgen → (scene)
func TestEditor_CycleWorkspace(t *testing.T) {
	e := New()

	e.SetActiveWorkspaceByName("scene")
	e.CycleWorkspace()
	assert.Equal(t, "behavior", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "audio", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "capture", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "procgen", e.ActiveWorkspaceName())
	e.CycleWorkspace()
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
}

// Re-registering by name replaces the prior entry rather than duplicating.
// After New() the editor has 5 workspaces (scene + 4 M3 stubs); re-
// registering "scene" must not grow that list.
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

// handleTabStripClick maps click X to workspace index. After New(),
// tab order is scene, behavior, audio, capture, procgen.
func TestEditor_HandleTabStripClick(t *testing.T) {
	e := New()
	area := widgets.Rect{X: 0, Y: 0, W: 800, H: tabStripH}
	// First tab spans roughly X=[8, 108) → scene.
	e.handleTabStripClick(area, 20, 5)
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
	// Second tab spans roughly X=[108, 208) → behavior.
	e.handleTabStripClick(area, 120, 5)
	assert.Equal(t, "behavior", e.ActiveWorkspaceName())
}

// Compile-time assertion that SceneWorkspace satisfies the interface.
var _ Workspace = (*SceneWorkspace)(nil)

// Suppress unused-import warning when color is imported elsewhere only.
var _ = color.RGBA{}
