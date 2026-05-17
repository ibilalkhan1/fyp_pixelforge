package editor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// The five tests below cover U2's plan scenarios. Several plan
// scenarios specify live ImGui assertions (the C-side IO state, ImGui
// window list, MenuItemBool return) that need a real OpenGL context
// to exercise — those are infeasible from `go test`. We assert on the
// editor model state that backs the scenario instead, and the visual
// integration is covered by the U2 smoke run (`./pf-studio`).

// TestMenuActionFiresOnMenuItemClick — the plan asks us to verify
// that wiring imgui.MenuItem("Open") invokes FileMenu.Open() exactly
// once. The wiring lives in buildMenuDefs (file_menu.go): each
// widgets.MenuItem carries the OnSelect closure imgui_chrome dispatches
// from MenuItemBoolV's true return. Invoking OnSelect directly proves
// the same end-to-end behaviour without a live ImGui frame.
func TestMenuActionFiresOnMenuItemClick(t *testing.T) {
	e := New()
	defs := e.buildMenuDefs()

	var openItem *menuItemRef
	for i := range defs {
		if defs[i].Label != "File" {
			continue
		}
		for j := range defs[i].Items {
			if strings.HasPrefix(defs[i].Items[j].Label, "Open") {
				openItem = &menuItemRef{onSelect: defs[i].Items[j].OnSelect}
				break
			}
		}
	}
	require.NotNil(t, openItem, "File → Open menu item missing from buildMenuDefs")
	require.NotNil(t, openItem.onSelect, "Open menu item must have an OnSelect callback")

	require.False(t, e.FilePicker().Visible(), "file picker starts hidden")
	openItem.onSelect()
	assert.True(t, e.FilePicker().Visible(), "OnSelect for Open must surface the file picker")
}

// menuItemRef is a tiny named accessor so the test reads cleanly.
type menuItemRef struct {
	onSelect func()
}

// TestStatusBarRendersMessage — verify that setting a status message
// reaches the rendered text. statusBarText() is the pure function the
// ImGui status bar passes to imgui.Text; asserting on its output
// covers the same contract.
func TestStatusBarRendersMessage(t *testing.T) {
	e := New()

	// Default text should not contain a custom message yet.
	assert.NotContains(t, e.statusBarText(), "saved")

	e.SetStatusMessage("saved")
	assert.Contains(t, e.statusBarText(), "saved", "status text must surface the message the editor was told to display")
}

// TestKeymapShortcutStillFiresWhenImGuiCapturesNothing — the gate
// shouldDispatchShortcuts returns true when neither a modal nor ImGui
// own keyboard focus, so handleShortcuts runs and Ctrl+S can fire.
func TestKeymapShortcutStillFiresWhenImGuiCapturesNothing(t *testing.T) {
	assert.True(t, shouldDispatchShortcuts(false, false),
		"with no modal and no ImGui capture, shortcuts must dispatch")
}

// TestKeymapShortcutSuppressedWhenTextInputFocused — when ImGui owns
// the keyboard (e.g. a text input has focus), the gate returns false
// and handleShortcuts is skipped, so Ctrl+S does not fire while the
// user is typing into an ImGui widget.
func TestKeymapShortcutSuppressedWhenTextInputFocused(t *testing.T) {
	assert.False(t, shouldDispatchShortcuts(false, true),
		"ImGui keyboard capture must suppress editor shortcuts")
	assert.False(t, shouldDispatchShortcuts(true, false),
		"open modal must suppress editor shortcuts")
	assert.False(t, shouldDispatchShortcuts(true, true),
		"either condition alone is enough to suppress")
}

// TestPanelSkeletonsAreRegistered — verify the editor knows about the
// four panels U2 introduces. The plan's wording asks for ImGui's
// runtime window list to include them; without a live ImGui context
// we assert on the editor-side stable identifiers (used by U6's
// imgui.ini persistence and by the panelRects map).
func TestPanelSkeletonsAreRegistered(t *testing.T) {
	registered := []string{PanelAssets, PanelInspector, PanelScene, PanelStatusBar}
	want := map[string]bool{
		"Assets":      false,
		"Inspector":   false,
		"Scene":       false,
		"##StatusBar": false,
	}
	for _, name := range registered {
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		assert.True(t, seen, "panel %q must be registered", name)
	}

	// PanelRect must be safe to call before buildChrome has run.
	e := New()
	for _, name := range registered {
		assert.Equal(t, 0, e.PanelRect(name).W, "rect width before first frame")
		assert.Equal(t, 0, e.PanelRect(name).H, "rect height before first frame")
	}
}

// TestPanelRectPopulatedAfterRectAssignment — direct verification
// that PanelRect surfaces what buildPanelSkeleton would have captured
// during a live frame. We seed panelRects directly because the imgui
// calls require a real context.
func TestPanelRectPopulatedAfterRectAssignment(t *testing.T) {
	e := New()
	e.ensurePanelRects()
	require.NotNil(t, e.panelRects)

	// Simulate a captured rect.
	want := widgets.Rect{X: 10, Y: 20, W: 300, H: 400}
	e.panelRects[PanelInspector] = want

	assert.Equal(t, want, e.PanelRect(PanelInspector))
}
