// Package editor is the top-level Pixelforge Studio editor. It hosts the
// editor window, owns the project model, and orchestrates the panels that
// surface every engine subsystem the studio exposes.
//
// During M0 and M1 the editor renders all chrome at native window
// resolution via Ebitengine primitives (ebitenutil.DebugPrintAt,
// vector.DrawFilledRect). M3 migrates chrome onto a Pixelforge canvas as
// part of the editor-as-cart milestone.
package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Version is stamped into the title bar so screenshots and bug reports
// always say which build produced them.
const Version = "0.1.0-m1"

// Default window dimensions used when the user has no saved preference.
// main.go uses these for window sizing; settings.go uses them as the
// fallback for malformed or first-run config files.
const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 800
)

// Editor implements ebiten.Game. State is intentionally narrow at M1:
// chrome layout, settings, keymap, in-memory project, current
// selection, and the inspector. Later milestones append fields.
type Editor struct {
	chrome    *chromeLayout
	settings  *Settings
	keymap    *KeyMap
	project   *pixelforge_project.Project
	inspector *Inspector

	selectedEntityID string
	dirty            bool
}

// New constructs a fresh Editor with default settings and an empty
// in-memory project. Use NewWithSettings to inject loaded settings.
func New() *Editor {
	return NewWithSettings(DefaultSettings())
}

// NewWithSettings constructs an Editor bound to a specific settings
// instance — the editor mutates it (via window-resize observers and
// recent-project pushes) and triggers debounced autosaves through it.
func NewWithSettings(s *Settings) *Editor {
	if s == nil {
		s = DefaultSettings()
	}
	return &Editor{
		chrome:    defaultChromeLayout(),
		settings:  s,
		keymap:    DefaultKeyMap(),
		project:   pixelforge_project.NewProject("untitled"),
		inspector: NewInspector(),
	}
}

// Settings returns the editor's bound settings instance.
func (e *Editor) Settings() *Settings { return e.settings }

// KeyMap returns the editor's keyboard shortcut registry.
func (e *Editor) KeyMap() *KeyMap { return e.keymap }

// Project returns the in-memory project. Callers should treat the
// pointer as live state — mutations made via UI handlers are reflected
// immediately.
func (e *Editor) Project() *pixelforge_project.Project { return e.project }

// SetProject replaces the in-memory project. Clears the current
// selection and resets the dirty flag.
func (e *Editor) SetProject(p *pixelforge_project.Project) {
	e.project = p
	e.selectedEntityID = ""
	e.dirty = false
}

// SelectEntity sets the inspector focus. An empty ID clears the
// selection (the inspector then renders empty).
func (e *Editor) SelectEntity(id string) {
	e.selectedEntityID = id
}

// SelectedEntityID returns the current selection, or "".
func (e *Editor) SelectedEntityID() string { return e.selectedEntityID }

// IsDirty reports whether the in-memory project has unsaved changes.
func (e *Editor) IsDirty() bool { return e.dirty }

// MarkDirty records that the project has unsaved changes. Idempotent.
func (e *Editor) MarkDirty() { e.dirty = true }

// Update advances the editor one tick. M1 wires nothing here yet —
// input flows through individual panels in Draw.
func (e *Editor) Update() error {
	return nil
}

// Draw paints the chrome and panel content onto the framebuffer
// Ebitengine hands us. The chrome layout is recomputed each frame so
// window resizes apply without an explicit hook.
func (e *Editor) Draw(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	e.chrome.recompute(w, h)
	screen.Fill(themeBackground)
	e.chrome.draw(screen, e)

	// Inspector renders inside the right panel.
	if entity := e.selectedEntity(); entity != nil {
		area := widgets.Rect{
			X: e.chrome.RightPanel.X,
			Y: e.chrome.RightPanel.Y + 20, // skip the panel header
			W: e.chrome.RightPanel.W,
			H: e.chrome.RightPanel.H - 20,
		}
		if e.inspector.Draw(screen, area, e.project, entity) {
			e.dirty = true
		}
	}

	// Bottom-right hint when the project has unsaved changes — gives
	// the user a single source of truth without crowding the chrome.
	if e.dirty {
		ebitenutil.DebugPrintAt(screen, "* unsaved", w-80, h-18)
	}
}

// Layout reports the inner game-canvas dimensions. The studio renders
// at native window resolution, so we hand back the window dimensions
// unchanged — no internal pixel scaling.
func (e *Editor) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// selectedEntity resolves the current selection against the project.
// Returns nil if the selection is empty or stale.
func (e *Editor) selectedEntity() *pixelforge_project.Entity {
	if e.project == nil || e.selectedEntityID == "" {
		return nil
	}
	for si := range e.project.Scenes {
		for ei := range e.project.Scenes[si].Entities {
			if e.project.Scenes[si].Entities[ei].ID == e.selectedEntityID {
				return &e.project.Scenes[si].Entities[ei]
			}
		}
	}
	return nil
}

// themeBackground is the editor's base fill.
var themeBackground = color.RGBA{R: 0x10, G: 0x10, B: 0x16, A: 0xff}
