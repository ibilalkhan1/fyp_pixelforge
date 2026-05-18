package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// U3's dockspace_test.go covers the plan's scenarios about DockSpace
// registration and layout. ImGui's docking state lives in C and needs
// a live OpenGL context to inspect, so the tests assert on the
// editor-side bookkeeping (dockspaceState) and the workspace registry
// shape that the dockspace consumes. Visual correctness is exercised
// by the smoke run (`./pf-studio`) the plan verification calls out.

// TestDockSpaceStateInitiallyUnseeded — a fresh editor has no
// dockspace state until the first buildDockSpace runs (gated by a
// live ImGui backend). PanelRect lookups are safe before then.
func TestDockSpaceStateInitiallyUnseeded(t *testing.T) {
	e := New()
	assert.Nil(t, e.dockspace, "dockspace state allocated lazily on first live buildDockSpace")
	// PanelRect must remain safe to call without a dockspace yet.
	assert.Equal(t, 0, e.PanelRect(PanelAssets).W)
}

// TestBuildDockSpace_SkippedWithoutLiveBackend — without a live ImGui
// backend the buildDockSpace path returns immediately, so the editor
// can run headless in tests without a cgo segfault.
func TestBuildDockSpace_SkippedWithoutLiveBackend(t *testing.T) {
	e := New()
	mock := &recordingBackend{}
	e.AttachImguiBackendStub(mock)

	// Force the path — buildChrome already short-circuits, this proves
	// the inner buildDockSpace also no-ops without panicking and
	// without allocating the state.
	e.buildDockSpace()
	assert.Nil(t, e.dockspace, "no dockspace state allocated when backend is a stub")
}

// TestWorkspaceRegisteredForDockBuilder — the dockspace seeds its
// default layout by docking every registered workspace into the
// central node. Verify the workspaces present at default editor
// construction time match what the dockspace will dock.
func TestWorkspaceRegisteredForDockBuilder(t *testing.T) {
	e := New()
	names := []string{}
	for _, w := range e.Workspaces() {
		names = append(names, w.DisplayName())
	}
	assert.Contains(t, names, "Scene", "Scene workspace must be docked by default")
}

// TestDefaultDockRatiosAreSensible — the hard-coded fallback ratios
// guard against a regression where someone sets one to 0 or above 1.0
// (ImGui rejects ratios outside (0, 1) and DockBuilder no-ops).
func TestDefaultDockRatiosAreSensible(t *testing.T) {
	assert.True(t, defaultLeftPanelRatio > 0 && defaultLeftPanelRatio < 1,
		"left ratio %v must be in (0, 1)", defaultLeftPanelRatio)
	assert.True(t, defaultRightPanelRatio > 0 && defaultRightPanelRatio < 1,
		"right ratio %v must be in (0, 1)", defaultRightPanelRatio)
}

// TestWorkspaceHotkeyTracksActive — Ctrl+1..6 routes to
// SetActiveWorkspaceByName, which both records the new active
// workspace name (so the status bar / tests can see it) and asks
// ImGui to focus the corresponding window. The focus call is gated
// off here because there's no live ImGui backend, but the activeName
// bookkeeping is the editor-side contract this test covers.
func TestWorkspaceHotkeyTracksActive(t *testing.T) {
	e := New()
	e.RegisterWorkspace(&stubWorkspace{name: "palette", displayName: "Palette"})

	e.SetActiveWorkspaceByName("palette")
	assert.Equal(t, "palette", e.ActiveWorkspaceName())

	e.SetActiveWorkspaceByName("scene")
	assert.Equal(t, "scene", e.ActiveWorkspaceName())
}

// TestSceneWorkspaceRendersAsRegistered — SceneWorkspace's DisplayName
// is the stable ImGui window title the dockspace docks. The PanelRect
// it captures via Render is also keyed by this name, so a regression
// in either side would silently misplace the scene viewport.
func TestSceneWorkspaceRendersAsRegistered(t *testing.T) {
	s := &SceneWorkspace{}
	assert.Equal(t, "scene", s.Name())
	assert.Equal(t, "Scene", s.DisplayName(), "DisplayName must equal the ImGui window title and PanelRect key")
}

// TestCaptureCurrentWindowRect_SafeWithoutLiveImGui — without a live
// backend, CaptureCurrentWindowRect would issue imgui.* C calls; but
// because the Render() helpers gate on imgui.live before calling it,
// the surface remains safe. This test asserts the gating contract
// at the SetPanelRect level: rect storage works without a backend.
func TestCaptureCurrentWindowRect_SafeWithoutLiveImGui(t *testing.T) {
	e := New()
	e.SetPanelRect("UnitTest", e.PanelRect("UnitTest"))
	require.NotNil(t, e.panelRects)
}
