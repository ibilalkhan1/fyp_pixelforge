package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
)

// withRecompileSpy swaps the package-level recompileFn with a counter
// and returns a cleanup closure. Test helper used by both the dropdown
// and restore-defaults tests so neither one drives the global pievent
// registry (which other packages' tests would observe).
func withRecompileSpy(t *testing.T) (*int, *[]pixelforge_project.InputBinding, func()) {
	t.Helper()
	calls := 0
	var last []pixelforge_project.InputBinding
	prev := recompileFn
	recompileFn = func(im []pixelforge_project.InputBinding) {
		calls++
		last = append(last[:0:0], im...)
	}
	return &calls, &last, func() { recompileFn = prev }
}

// findBinding is a tiny helper that returns the InputBinding for the
// supplied intent or fails the test.
func findBinding(t *testing.T, p *pixelforge_project.Project, intent string) pixelforge_project.InputBinding {
	t.Helper()
	for _, b := range p.InputMap {
		if b.Intent == intent {
			return b
		}
	}
	t.Fatalf("intent %q not present in InputMap", intent)
	return pixelforge_project.InputBinding{}
}

// TestWorkspace_Naming pins the workspace identifier the rest of the
// editor (status bar, View menu, dockspace) addresses it by.
func TestWorkspace_Naming(t *testing.T) {
	w := NewWorkspace()
	assert.Equal(t, "input", w.Name())
	assert.Equal(t, "Input", w.DisplayName())
}

// TestWorkspace_RegisteredAlongsideOtherWorkspaces verifies that
// RegisterWith slots cleanly into the editor's workspace registry
// alongside the palette and capture registrations that ship today.
// Mirrors the main.go composition order.
func TestWorkspace_RegisteredAlongsideOtherWorkspaces(t *testing.T) {
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
	assert.True(t, names["input"], "input workspace should be registered")

	// The editor switches workspaces by name; confirm the new one is
	// reachable through that public seam.
	e.SetActiveWorkspaceByName("input")
	assert.Equal(t, "input", e.ActiveWorkspaceName())
}

// TestWorkspace_EditDropdownMutatesInputMap exercises the SetBinding*
// path the dropdown change handlers route through. Direct API mutation
// (rather than simulated ImGui events) keeps the test independent of
// the cimgui-go live backend, matching the convention palette and
// capture workspace_test.go follow.
func TestWorkspace_EditDropdownMutatesInputMap(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()

	require.True(t, w.SetBindingKeyboard(e, pixelforge_input.IntentJump, string(pixelforge_key.X)))

	got := findBinding(t, e.Project(), pixelforge_input.IntentJump)
	require.Len(t, got.Keyboard, 1)
	assert.Equal(t, string(pixelforge_key.X), got.Keyboard[0])

	require.True(t, w.SetBindingGamepad(e, pixelforge_input.IntentJump, string(pixelforge_pad.B)))
	got = findBinding(t, e.Project(), pixelforge_input.IntentJump)
	assert.Equal(t, string(pixelforge_pad.B), got.GamepadButton)

	require.True(t, w.SetBindingModifier(e, pixelforge_input.IntentJump, "Ctrl"))
	got = findBinding(t, e.Project(), pixelforge_input.IntentJump)
	assert.Equal(t, "Ctrl", got.Modifier)
}

// TestWorkspace_EditCallsMarkDirty pins the dirty-state contract:
// every dropdown edit must route through MarkDirty so File > Save sees
// the unsaved change (per docs/solutions/dirty-state-ux.md).
func TestWorkspace_EditCallsMarkDirty(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()
	require.False(t, e.IsDirty(), "fresh editor should start clean")

	require.True(t, w.SetBindingKeyboard(e, pixelforge_input.IntentJump, string(pixelforge_key.X)))
	assert.True(t, e.IsDirty(), "keyboard edit must mark editor dirty")

	e.ClearDirty()
	require.True(t, w.SetBindingGamepad(e, pixelforge_input.IntentAttack, string(pixelforge_pad.Y)))
	assert.True(t, e.IsDirty(), "gamepad edit must mark editor dirty")

	e.ClearDirty()
	require.True(t, w.SetBindingModifier(e, pixelforge_input.IntentMenu, "Ctrl"))
	assert.True(t, e.IsDirty(), "modifier edit must mark editor dirty")
}

// TestWorkspace_EditCallsRecompile pins the live-update contract:
// every edit calls pixelforge_input.Recompile so the intent layer
// reflects the new map without an engine restart (per F4 / plan U7).
// We verify via the package-level recompileFn seam rather than
// observing the global pievent registry — both approaches prove the
// same call edge but the seam keeps the test hermetic.
func TestWorkspace_EditCallsRecompile(t *testing.T) {
	calls, last, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()

	require.True(t, w.SetBindingKeyboard(e, pixelforge_input.IntentJump, string(pixelforge_key.X)))
	assert.Equal(t, 1, *calls, "keyboard edit should trigger one recompile")
	require.NotEmpty(t, *last)

	// Verify the recompile observed the *new* InputMap state, not a
	// stale snapshot — Recompile signals the intent layer, so passing
	// pre-edit bindings would silently leave the layer out of sync.
	gotJump := pixelforge_project.InputBinding{}
	for _, b := range *last {
		if b.Intent == pixelforge_input.IntentJump {
			gotJump = b
			break
		}
	}
	require.Len(t, gotJump.Keyboard, 1)
	assert.Equal(t, string(pixelforge_key.X), gotJump.Keyboard[0])

	require.True(t, w.SetBindingGamepad(e, pixelforge_input.IntentAttack, string(pixelforge_pad.Y)))
	assert.Equal(t, 2, *calls, "gamepad edit should trigger a second recompile")
}

// TestWorkspace_RestoreDefaults_RewritesMap verifies the restore-
// defaults button (mirrored as RestoreDefaults for tests): the map is
// replaced wholesale, the editor is marked dirty, and the intent
// layer is recompiled.
func TestWorkspace_RestoreDefaults_RewritesMap(t *testing.T) {
	calls, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	// Start with a deliberately-mangled map so we can prove restoration.
	e.Project().InputMap = []pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"weird"}},
	}
	e.ClearDirty()

	w.RestoreDefaults(e)

	assert.True(t, e.IsDirty(), "restore should mark editor dirty")
	assert.Equal(t, 1, *calls, "restore should trigger a recompile")

	want := pixelforge_project.DefaultInputMap()
	require.Len(t, e.Project().InputMap, len(want), "restored map should match default cardinality")
	for i, b := range want {
		assert.Equal(t, b.Intent, e.Project().InputMap[i].Intent, "intent at index %d", i)
		assert.Equal(t, b.Keyboard, e.Project().InputMap[i].Keyboard, "keyboard at index %d", i)
		assert.Equal(t, b.GamepadButton, e.Project().InputMap[i].GamepadButton, "gamepad at index %d", i)
	}
}

// TestWorkspace_EditWithEmptyInputMap_PopulatesDefaults exercises the
// lazy-default path: editing a binding on a project that hasn't been
// through applyDefaults (e.g. editor.New() in tests) should auto-
// populate the standard 9 NES bindings rather than no-op.
func TestWorkspace_EditWithEmptyInputMap_PopulatesDefaults(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = nil

	require.True(t, w.SetBindingKeyboard(e, pixelforge_input.IntentJump, string(pixelforge_key.X)))
	assert.Len(t, e.Project().InputMap, len(pixelforge_project.DefaultInputMap()))
}

// TestWorkspace_EditUnknownIntentReturnsFalse documents the no-match
// path: an intent not present in the active map produces no mutation
// and no MarkDirty / Recompile side effects.
func TestWorkspace_EditUnknownIntentReturnsFalse(t *testing.T) {
	calls, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()
	e.ClearDirty()

	assert.False(t, w.SetBindingKeyboard(e, "input/nope", "Z"))
	assert.False(t, e.IsDirty(), "no mutation -> no dirty bit")
	assert.Equal(t, 0, *calls, "no mutation -> no recompile")
}

// TestKeyNameCanonicalization_BridgesDefaultMismatch documents the
// "friendly default" vs "raw pixelforge_key value" bridge described in
// the package doc. DefaultInputMap stores e.g. "Space" / "ArrowUp" /
// "Escape" / "GamepadA"; the label resolver must surface a friendly
// name for both the friendly literal and the underlying raw value so
// the workspace renders the same way before and after the first edit
// rewrites the binding to the compiler-matching value.
func TestKeyNameCanonicalization_BridgesDefaultMismatch(t *testing.T) {
	cases := []struct {
		stored string
		label  string
	}{
		{"Space", "Space"},
		{" ", "Space"},
		{"ArrowUp", "Up Arrow"},
		{"Up", "Up Arrow"},
		{"ArrowDown", "Down Arrow"},
		{"Down", "Down Arrow"},
		{"ArrowLeft", "Left Arrow"},
		{"Left", "Left Arrow"},
		{"ArrowRight", "Right Arrow"},
		{"Right", "Right Arrow"},
		{"Escape", "Escape"},
		{"Esc", "Escape"},
		{"Z", "Z"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.label, labelForKeyValue(tc.stored), "labelForKeyValue(%q)", tc.stored)
	}

	padCases := []struct {
		stored string
		label  string
	}{
		{"GamepadA", "A (Bottom)"},
		{"A", "A (Bottom)"},
		{"GamepadB", "B (Right)"},
		{"B", "B (Right)"},
		{"DPadUp", "D-Pad Up"},
		{"Top", "D-Pad Up"},
		{"DPadLeft", "D-Pad Left"},
		{"Left", "D-Pad Left"},
	}
	for _, tc := range padCases {
		assert.Equal(t, tc.label, labelForButtonValue(tc.stored), "labelForButtonValue(%q)", tc.stored)
	}
}

// TestKeyboardOptions_CoversAllIntentDefaults pins the dropdown
// contents: every value DefaultInputMap could persist *after* a
// designer-initiated edit must appear in the canonical keyboard
// options list, otherwise the dropdown could not represent the
// binding without falling through the "forward-compat unknown"
// prepend path.
func TestKeyboardOptions_CoversNESLetters(t *testing.T) {
	opts := KeyboardOptions()
	want := []string{
		string(pixelforge_key.Z),
		string(pixelforge_key.X),
		string(pixelforge_key.W),
		string(pixelforge_key.A),
		string(pixelforge_key.S),
		string(pixelforge_key.D),
		string(pixelforge_key.Space),
		string(pixelforge_key.Enter),
		string(pixelforge_key.Esc),
		string(pixelforge_key.Up),
		string(pixelforge_key.Down),
		string(pixelforge_key.Left),
		string(pixelforge_key.Right),
	}
	have := map[string]bool{}
	for _, o := range opts {
		have[o.Value] = true
	}
	for _, w := range want {
		assert.True(t, have[w], "keyboardOptions must include value %q", w)
	}
}

// TestGamepadOptions_CoversAllSupportedButtons verifies the gamepad
// dropdown surfaces every pixelforge_pad button constant.
func TestGamepadOptions_CoversAllSupportedButtons(t *testing.T) {
	opts := GamepadOptions()
	want := []string{
		string(pixelforge_pad.A), string(pixelforge_pad.B),
		string(pixelforge_pad.X), string(pixelforge_pad.Y),
		string(pixelforge_pad.Left), string(pixelforge_pad.Right),
		string(pixelforge_pad.Top), string(pixelforge_pad.Bottom),
	}
	have := map[string]bool{}
	for _, o := range opts {
		have[o.Value] = true
	}
	for _, w := range want {
		assert.True(t, have[w], "gamepadOptions must include value %q", w)
	}
}

// TestSortedIntentNames_ListsAllNine cross-checks that the workspace's
// rendered intent ordering surfaces every IntentName the runtime
// registers.
func TestSortedIntentNames_ListsAllNine(t *testing.T) {
	got := SortedIntentNames()
	assert.Len(t, got, 9)
	want := map[string]bool{
		pixelforge_input.IntentJump:   true,
		pixelforge_input.IntentAttack: true,
		pixelforge_input.IntentUp:     true,
		pixelforge_input.IntentDown:   true,
		pixelforge_input.IntentLeft:   true,
		pixelforge_input.IntentRight:  true,
		pixelforge_input.IntentUse:    true,
		pixelforge_input.IntentMenu:   true,
		pixelforge_input.IntentStart:  true,
	}
	for _, name := range got {
		assert.True(t, want[name], "unexpected intent %q", name)
	}
}

// TestWorkspace_NilEditor_NoPanic guards against a nil Editor passed
// through RegisterWith / Render / RestoreDefaults — the workspace
// contract permits a nil editor (the dockspace tolerates "no project"
// states) and must degrade gracefully rather than panic.
func TestWorkspace_NilEditor_NoPanic(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	w := RegisterWith(nil)
	assert.Nil(t, w)

	w2 := NewWorkspace()
	w2.RestoreDefaults(nil)
	assert.False(t, w2.SetBindingKeyboard(nil, pixelforge_input.IntentJump, "Z"))
}

// TestWorkspace_BeginCapture_TransitionsToWaiting pins the workspace-
// level entry point: clicking the Capture button (mirrored as
// BeginCapture) flips CaptureActive() and exposes the intent via
// CapturingIntent().
func TestWorkspace_BeginCapture_TransitionsToWaiting(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	w := NewWorkspace()
	assert.False(t, w.CaptureActive())

	w.BeginCapture(pixelforge_input.IntentJump)
	assert.True(t, w.CaptureActive(), "BeginCapture should flip CaptureActive")
	assert.Equal(t, pixelforge_input.IntentJump, w.CapturingIntent())
}

// TestWorkspace_SubmitCaptureKey_RecordsBinding is the AE8 happy-path
// integration: open workspace → click Capture next to jump → press
// Space → binding records as " " in workspace.
func TestWorkspace_SubmitCaptureKey_RecordsBinding(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()
	e.ClearDirty()

	w.BeginCapture(pixelforge_input.IntentJump)
	require.True(t, w.CaptureActive())

	ok := w.SubmitCaptureKey(e, ebiten.KeySpace)
	assert.True(t, ok, "SubmitCaptureKey on a non-modifier key should return true")
	assert.False(t, w.CaptureActive(), "capture clears after binding")
	assert.True(t, e.IsDirty(), "capture-completed binding flips dirty bit (AE8)")

	got := findBinding(t, e.Project(), pixelforge_input.IntentJump)
	require.Len(t, got.Keyboard, 1)
	assert.Equal(t, " ", got.Keyboard[0], "AE8: Space press records as the U3 \" \" value")
}

// TestWorkspace_SubmitCaptureKey_EscCancels covers the Esc-cancel
// integration: capture is dismissed without mutating the project.
func TestWorkspace_SubmitCaptureKey_EscCancels(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()
	original := findBinding(t, e.Project(), pixelforge_input.IntentJump)
	e.ClearDirty()

	w.BeginCapture(pixelforge_input.IntentJump)
	ok := w.SubmitCaptureKey(e, ebiten.KeyEscape)
	assert.False(t, ok, "Esc should not capture")
	assert.False(t, w.CaptureActive(), "Esc transitions out of Waiting")
	assert.False(t, e.IsDirty(), "Esc cancel should not flip dirty bit")

	after := findBinding(t, e.Project(), pixelforge_input.IntentJump)
	assert.Equal(t, original.Keyboard, after.Keyboard, "Esc must not mutate the binding")
}

// TestWorkspace_SubmitCaptureKey_ModifierStaysActive enforces the
// design-lens finding via the workspace seam.
func TestWorkspace_SubmitCaptureKey_ModifierStaysActive(t *testing.T) {
	_, _, cleanup := withRecompileSpy(t)
	defer cleanup()

	e := editor.New()
	w := RegisterWith(e)
	e.Project().InputMap = pixelforge_project.DefaultInputMap()
	e.ClearDirty()

	w.BeginCapture(pixelforge_input.IntentJump)
	ok := w.SubmitCaptureKey(e, ebiten.KeyShiftLeft)
	assert.False(t, ok, "modifier-only key should not capture")
	assert.True(t, w.CaptureActive(), "capture should remain Waiting after a modifier")
	assert.Equal(t, pixelforge_input.IntentJump, w.CapturingIntent())
	assert.False(t, e.IsDirty(), "modifier-only press should not flip dirty bit")
}

// TestWorkspace_BeginCapture_SameIntentTogglesOff documents the
// "click Capture twice to cancel" affordance at the workspace seam.
func TestWorkspace_BeginCapture_SameIntentTogglesOff(t *testing.T) {
	w := NewWorkspace()
	w.BeginCapture(pixelforge_input.IntentJump)
	require.True(t, w.CaptureActive())
	w.BeginCapture(pixelforge_input.IntentJump)
	assert.False(t, w.CaptureActive(), "second click on same intent should cancel")
}

// TestWorkspace_CancelCapture_ResetsState gives the workspace's
// explicit cancel hook a smoke test.
func TestWorkspace_CancelCapture_ResetsState(t *testing.T) {
	w := NewWorkspace()
	w.BeginCapture(pixelforge_input.IntentJump)
	w.CancelCapture()
	assert.False(t, w.CaptureActive())
	assert.Empty(t, w.CapturingIntent())
}

// TestWorkspace_NilSafe_CaptureMethods extends the existing nil-editor
// graceful degradation to the new capture methods.
func TestWorkspace_NilSafe_CaptureMethods(t *testing.T) {
	var w *Workspace
	w.BeginCapture("x") // must not panic
	w.CancelCapture()   // must not panic
	assert.False(t, w.CaptureActive())
	assert.Empty(t, w.CapturingIntent())
	assert.False(t, w.SubmitCaptureKey(nil, ebiten.KeySpace))
}

// TestWorkspace_F4Flow_RemapTakesEffectWithoutRestart is the AE-level
// scenario asserting that a designer edit propagates through the
// intent layer with no engine restart. The full end-to-end check
// (publish a key event on the production key.main target, observe an
// IntentEvent on the production intent target) lives in U10's
// integration_test package; this unit test pins only the seam this
// workspace owns. See plan U7's test-scenarios section, "Skip ...
// better covered in U10".
func TestWorkspace_F4Flow_RemapTakesEffectWithoutRestart(t *testing.T) {
	t.Skip("Full F4 end-to-end (key.main publish -> intent observed) is U10's responsibility; this unit-tested the workspace edit -> Recompile contract above (TestWorkspace_EditCallsRecompile).")
}
