// editor_overlays_settings.go owns idea #3 v1 U8's View-menu toggle
// helpers. Two flags live on Project.EditorOverlays; the View menu
// renders one toggleable item per flag with the Checked state read
// from the project. Toggles mutate the flag + MarkDirty so the
// designer sees the `*` in the title bar (per dirty-state-ux.md)
// and the new state persists with the next save.
//
// Persistence: handled by the existing project save/load flow —
// EditorOverlays is part of Project (idea #3 v1 U1), so no new
// save path is needed.
package editor

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"

// ScanlineOverlayEnabled returns the project's current scanline
// overlay toggle state. Nil-safe.
func (e *Editor) ScanlineOverlayEnabled() bool {
	if e == nil || e.project == nil {
		return false
	}
	return e.project.EditorOverlays.ScanlineEnabled
}

// PaletteBlockOverlayEnabled returns the project's current
// palette-block overlay toggle state. Nil-safe.
func (e *Editor) PaletteBlockOverlayEnabled() bool {
	if e == nil || e.project == nil {
		return false
	}
	return e.project.EditorOverlays.PaletteBlockEnabled
}

// ToggleScanlineOverlay flips ScanlineEnabled + marks dirty. The
// EditorOverlays.Set bit is already true (applyDefaults sets it on
// load); toggling preserves the bit so the next save round-trips
// faithfully.
func (e *Editor) ToggleScanlineOverlay() {
	if e == nil || e.project == nil {
		return
	}
	e.project.EditorOverlays.ScanlineEnabled = !e.project.EditorOverlays.ScanlineEnabled
	e.project.EditorOverlays.Set = true
	e.MarkDirty()
}

// TogglePaletteBlockOverlay flips PaletteBlockEnabled + marks dirty.
func (e *Editor) TogglePaletteBlockOverlay() {
	if e == nil || e.project == nil {
		return
	}
	e.project.EditorOverlays.PaletteBlockEnabled = !e.project.EditorOverlays.PaletteBlockEnabled
	e.project.EditorOverlays.Set = true
	e.MarkDirty()
}

// EditorOverlaysSnapshot returns a copy of the project's current
// overlay state. Exposed so tests can assert against a value
// without juggling pointers.
func (e *Editor) EditorOverlaysSnapshot() pixelforge_project.EditorOverlays {
	if e == nil || e.project == nil {
		return pixelforge_project.EditorOverlays{}
	}
	return e.project.EditorOverlays
}

// BuildOnSaveEnabled returns whether build-on-save is currently
// active. Idea #7 v1 U10's View-menu toggle reads this for its
// Checked state.
func (e *Editor) BuildOnSaveEnabled() bool {
	if e == nil || e.settings == nil {
		return false
	}
	return !e.settings.BuildOnSaveDisabled
}

// ToggleBuildOnSave flips the BuildOnSaveDisabled flag + persists
// the settings file. Idea #7 v1 U10 wires this to the View menu.
func (e *Editor) ToggleBuildOnSave() {
	if e == nil || e.settings == nil {
		return
	}
	e.settings.BuildOnSaveDisabled = !e.settings.BuildOnSaveDisabled
	e.settings.MarkDirty()
}
