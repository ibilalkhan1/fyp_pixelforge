package blackboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_blackboard"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
)

// withHermeticBlackboard swaps the package-level blackboardFn with a
// freshly constructed instance and returns a cleanup closure. Keeps
// the workspace test from racing against (or being observed by) other
// packages' tests through the process-wide pixelforge_blackboard
// singleton.
func withHermeticBlackboard(t *testing.T) (*pixelforge_blackboard.Blackboard, func()) {
	t.Helper()
	bb := pixelforge_blackboard.New()
	prev := blackboardFn
	blackboardFn = func() *pixelforge_blackboard.Blackboard { return bb }
	return bb, func() { blackboardFn = prev }
}

// TestBlackboardWorkspace_Naming pins the workspace identifier and
// display title the dockspace + status bar address it by.
func TestBlackboardWorkspace_Naming(t *testing.T) {
	w := NewWorkspace()
	assert.Equal(t, "blackboard", w.Name())
	assert.Equal(t, "Blackboard", w.DisplayName())
}

// TestBlackboardWorkspace_RegistersOnInit verifies that RegisterWith
// slots cleanly into the editor's workspace registry alongside the
// palette + capture registrations that ship today. Mirrors the
// composition order in pixelforge_studio/main.go.
func TestBlackboardWorkspace_RegistersOnInit(t *testing.T) {
	_, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	e := editor.New()
	palette.RegisterWith(e)
	capture.RegisterWith(e)
	w := RegisterWith(e)
	require.NotNil(t, w)

	names := map[string]bool{}
	for _, ws := range e.Workspaces() {
		names[ws.Name()] = true
	}
	assert.True(t, names["scene"], "scene workspace should still be present")
	assert.True(t, names["palette"], "palette workspace should still be present")
	assert.True(t, names["capture"], "capture workspace should still be present")
	assert.True(t, names["blackboard"], "blackboard workspace should be registered")

	e.SetActiveWorkspaceByName("blackboard")
	assert.Equal(t, "blackboard", e.ActiveWorkspaceName())
}

// TestBlackboardWorkspace_RendersCanonicalKeys asserts the workspace
// surfaces each canonical key with the current stored value (or its
// declared default when the key has not been written yet). Pulled
// off the RenderedCanonicalLine seam so the assertion stays
// independent of the live cimgui-go backend.
func TestBlackboardWorkspace_RendersCanonicalKeys(t *testing.T) {
	bb, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	bb.Set(pixelforge_blackboard.KeyScore, 100)

	got := RenderedCanonicalLine(pixelforge_blackboard.KeyScore)
	assert.Contains(t, got, "score")
	assert.Contains(t, got, "100")

	// Unwritten canonical key should still render via its declared
	// default (Lives default is 3 per CanonicalKeys).
	livesLine := RenderedCanonicalLine(pixelforge_blackboard.KeyLives)
	assert.Contains(t, livesLine, "lives")
	assert.Contains(t, livesLine, "3")
}

// TestBlackboardWorkspace_RendersTypedWidgetForScore pins the
// "score renders as an int field, not a text input" contract. The
// widget kind is resolved off the canonical key's declared Type so
// future canonical-key additions automatically pick their widget.
func TestBlackboardWorkspace_RendersTypedWidgetForScore(t *testing.T) {
	cases := []struct {
		key      string
		wantKind string
	}{
		{pixelforge_blackboard.KeyScore, "int"},
		{pixelforge_blackboard.KeyLives, "int"},
		{pixelforge_blackboard.KeyHealth, "int"},
		{pixelforge_blackboard.KeyCurrentScene, "text"},
		{pixelforge_blackboard.KeyPlayerX, "float"},
		{pixelforge_blackboard.KeyPlayerY, "float"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.wantKind, WidgetKindFor(tc.key), "WidgetKindFor(%q)", tc.key)
	}

	// Non-canonical key returns "" — the inspector renders it through
	// the read-only fallback path instead of as a typed widget.
	assert.Equal(t, "", WidgetKindFor("my_custom_flag"))
}

// TestBlackboardWorkspace_EditScoreWritesToBlackboard exercises the
// SetCanonical seam the widget edit handler routes through. Direct
// API mutation (rather than simulated ImGui events) keeps the test
// independent of the cimgui-go live backend, matching the convention
// the input workspace_test.go follows.
func TestBlackboardWorkspace_EditScoreWritesToBlackboard(t *testing.T) {
	bb, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	w := NewWorkspace()
	require.True(t, w.SetCanonical(pixelforge_blackboard.KeyScore, 200))

	got, ok := bb.Get(pixelforge_blackboard.KeyScore)
	require.True(t, ok)
	assert.Equal(t, 200, got)
}

// TestBlackboardWorkspace_DesignerAddedKeyRenderedReadOnly pins the
// read-only treatment for keys outside CanonicalKeys: a verb may
// write "my_custom_flag" at runtime; the inspector surfaces it as a
// "key: value" text line below the typed widget block.
func TestBlackboardWorkspace_DesignerAddedKeyRenderedReadOnly(t *testing.T) {
	bb, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	bb.Set("my_custom_flag", true)

	// The designer-added key appears in the read-only iteration order.
	keys := RenderedDesignerKeys()
	assert.Contains(t, keys, "my_custom_flag")

	// And the formatted line includes both the key and the value.
	line := RenderedDesignerLine("my_custom_flag")
	assert.Equal(t, "my_custom_flag: true", line)

	// Canonical keys must NOT appear in the designer-added section,
	// even after a Set — they render with typed widgets above.
	bb.Set(pixelforge_blackboard.KeyScore, 50)
	for _, k := range RenderedDesignerKeys() {
		assert.NotEqual(t, pixelforge_blackboard.KeyScore, k, "score should render in the canonical section, not designer-added")
	}
}

// TestBlackboardWorkspace_LiveUpdateOnBlackboardChange verifies the
// SubscribeAll wiring set up by RegisterWith: a Set on the active
// blackboard flips the workspace's dirty flag so any future
// invalidate-on-change indicator can react without polling. The
// "next Render reflects the new value" half is implicit — Render
// reads bb.Lookup fresh each frame.
func TestBlackboardWorkspace_LiveUpdateOnBlackboardChange(t *testing.T) {
	bb, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	require.NotNil(t, w)
	// Drain any flags raised during registration so the assertion
	// below targets the mutation we are about to perform.
	w.ConsumeDirty()

	bb.Set(pixelforge_blackboard.KeyScore, 100)
	assert.True(t, w.ConsumeDirty(), "Set should flip the workspace's dirty flag via the change subscription")

	// Subsequent reads via the rendered-content seam should reflect the
	// newly-written value, proving the workspace's cached view tracks
	// the underlying blackboard without an explicit refresh.
	line := RenderedCanonicalLine(pixelforge_blackboard.KeyScore)
	assert.Contains(t, line, "100")

	// A second mutation should re-flip the flag (the subscription is
	// persistent, not one-shot).
	bb.Set(pixelforge_blackboard.KeyScore, 250)
	assert.True(t, w.ConsumeDirty())
}

// TestBlackboardWorkspace_NilEditor_NoPanic guards the nil-Editor path
// — RegisterWith returns nil rather than crashing, matching the
// input workspace's convention.
func TestBlackboardWorkspace_NilEditor_NoPanic(t *testing.T) {
	_, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	w := RegisterWith(nil)
	assert.Nil(t, w)

	w2 := NewWorkspace()
	assert.False(t, w2.SetCanonical("not_a_canonical_key", 1))
}

// TestBlackboardWorkspace_SetCanonical_RejectsNonCanonical pins the
// "designer-added keys are not editor-writable in v1" contract.
func TestBlackboardWorkspace_SetCanonical_RejectsNonCanonical(t *testing.T) {
	bb, cleanup := withHermeticBlackboard(t)
	defer cleanup()

	w := NewWorkspace()
	assert.False(t, w.SetCanonical("arbitrary_key", "value"))

	_, ok := bb.Get("arbitrary_key")
	assert.False(t, ok, "non-canonical SetCanonical should not write to the blackboard")
}
