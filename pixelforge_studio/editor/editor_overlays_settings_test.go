package editor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// editor_overlays_settings_test.go covers U8's View-menu toggles +
// the per-project persistence those toggles drive.

// TestEditor_ToggleScanlineOverlayFlipsFlag: ToggleScanlineOverlay
// flips the project's ScanlineEnabled bool + marks dirty.
func TestEditor_ToggleScanlineOverlayFlipsFlag(t *testing.T) {
	e := New()
	e.SetProject(pixelforge_project.NewProject("t"))
	e.project.EditorOverlays.ScanlineEnabled = true
	e.project.EditorOverlays.Set = true
	e.ClearDirty()

	e.ToggleScanlineOverlay()
	assert.False(t, e.ScanlineOverlayEnabled())
	assert.True(t, e.IsDirty(), "toggle marks dirty")

	e.ToggleScanlineOverlay()
	assert.True(t, e.ScanlineOverlayEnabled())
}

// TestEditor_TogglePaletteBlockOverlayFlipsFlag: same for the
// palette-block toggle.
func TestEditor_TogglePaletteBlockOverlayFlipsFlag(t *testing.T) {
	e := New()
	e.SetProject(pixelforge_project.NewProject("t"))
	e.project.EditorOverlays.PaletteBlockEnabled = true
	e.project.EditorOverlays.Set = true
	e.ClearDirty()

	e.TogglePaletteBlockOverlay()
	assert.False(t, e.PaletteBlockOverlayEnabled())
	assert.True(t, e.IsDirty())
}

// TestEditor_OverlayTogglesNilSafe: toggling on an editor with no
// project doesn't panic.
func TestEditor_OverlayTogglesNilSafe(t *testing.T) {
	e := &Editor{}
	assert.NotPanics(t, func() {
		e.ToggleScanlineOverlay()
		e.TogglePaletteBlockOverlay()
	})
}

// TestViewMenu_OverlayTogglesPresent: the View menu definition
// includes both overlay toggle entries.
func TestViewMenu_OverlayTogglesPresent(t *testing.T) {
	e := New()
	defs := e.buildMenuDefs()
	scanlineFound := false
	paletteFound := false
	for _, def := range defs {
		if def.Label != "View" {
			continue
		}
		for _, item := range def.Items {
			if item.Label == "Show 8-sprite-per-scanline overlay" {
				scanlineFound = true
			}
			if item.Label == "Show 2x2 BG palette-block overlay" {
				paletteFound = true
			}
		}
	}
	assert.True(t, scanlineFound, "scanline toggle present in View menu")
	assert.True(t, paletteFound, "palette-block toggle present in View menu")
}

// TestViewMenu_OverlayCheckedReflectsState: the menu's Checked
// field tracks the project's current toggle state at build time.
func TestViewMenu_OverlayCheckedReflectsState(t *testing.T) {
	e := New()
	e.SetProject(pixelforge_project.NewProject("t"))
	e.project.EditorOverlays.ScanlineEnabled = false
	e.project.EditorOverlays.PaletteBlockEnabled = true
	e.project.EditorOverlays.Set = true

	defs := e.buildMenuDefs()
	for _, def := range defs {
		if def.Label != "View" {
			continue
		}
		for _, item := range def.Items {
			switch item.Label {
			case "Show 8-sprite-per-scanline overlay":
				assert.False(t, item.Checked,
					"scanline toggle reflects ScanlineEnabled=false")
			case "Show 2x2 BG palette-block overlay":
				assert.True(t, item.Checked,
					"palette-block toggle reflects PaletteBlockEnabled=true")
			}
		}
	}
}

// TestEditorOverlays_PersistedAcrossSaveLoad: a project with both
// overlays toggled off saves + reloads with the same state. Sets
// the Set bit so omitempty doesn't discard the false/false struct.
func TestEditorOverlays_PersistedAcrossSaveLoad(t *testing.T) {
	e := New()
	e.SetProject(pixelforge_project.NewProject("t"))
	e.project.EditorOverlays.ScanlineEnabled = false
	e.project.EditorOverlays.PaletteBlockEnabled = false
	e.project.EditorOverlays.Set = true

	data, err := json.Marshal(e.project)
	require.NoError(t, err)
	var loaded pixelforge_project.Project
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.False(t, loaded.EditorOverlays.ScanlineEnabled)
	assert.False(t, loaded.EditorOverlays.PaletteBlockEnabled)
}

// TestEditorOverlaysSnapshot_ReturnsValueCopy: the snapshot
// accessor returns a copy of the project's overlay state. Mutating
// the copy doesn't mutate the project.
func TestEditorOverlaysSnapshot_ReturnsValueCopy(t *testing.T) {
	e := New()
	e.SetProject(pixelforge_project.NewProject("t"))
	e.project.EditorOverlays.ScanlineEnabled = true
	snap := e.EditorOverlaysSnapshot()
	snap.ScanlineEnabled = false
	assert.True(t, e.ScanlineOverlayEnabled(),
		"snapshot is a value copy; project is unaffected")
}
