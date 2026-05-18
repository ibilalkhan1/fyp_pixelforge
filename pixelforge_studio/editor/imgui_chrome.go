// imgui_chrome.go owns the editor's ImGui chrome — main menu bar,
// status bar, and the Assets/Inspector/Scene panel skeletons. U2 of the
// migration plan replaces the native chrome.go path; native asset
// browser / inspector / canvas content still renders, but inside the
// rects ImGui carved out (the Editor.Draw pass uses the captured
// panelRects). Subsequent units (U4 inspector, U5 scene, etc.) port
// each panel's content to true ImGui calls.
package editor

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Stable panel identifiers. The ImGui window names use these so
// imgui.ini layout persistence (added in U6) keys against a stable
// string. Tests assert on these too via Editor.PanelRect.
const (
	PanelAssets     = "Assets"
	PanelInspector  = "Inspector"
	PanelScene      = "Scene"
	PanelStatusBar  = "##StatusBar"
)

// statusBarHeight is the pixel height ImGui reserves for the status
// strip below the main viewport. Sized to comfortably hold one line of
// text + ImGui's default frame padding.
const statusBarHeight float32 = 26

// buildChrome builds one frame of ImGui chrome — must be called inside
// the BeginFrame/EndFrame brackets. Order matters: the main menu bar
// goes first so the viewport side bar (status) sizes against the
// remaining work area; panel skeletons land last so they can be docked
// anywhere in the viewport (real DockSpace lands in U3, but the
// skeleton windows already exist).
//
// Each panel's content-region rect (screen-pixel space) is captured
// into panelRects so Editor.Draw can render the existing native
// widgets into the right location.
func (e *Editor) buildChrome() {
	// Only run when a real cimgui-go backend is attached. Test stubs
	// (AttachImguiBackendStub) leave live=false so the imgui.* C calls
	// below stay skipped — they would segfault without an ImGui
	// context. The panel rect map is still initialised so PanelRect
	// callers get the documented zero rect instead of a nil-map panic.
	if e.imgui == nil || !e.imgui.live {
		e.ensurePanelRects()
		return
	}
	e.ensurePanelRects()
	// U6: push the editor theme onto ImGui's style stack so every
	// window built this frame uses the loaded editor.pforge colours.
	// Pop the same count after the frame's windows are built so the
	// stack is balanced on early returns.
	theme := e.activeImguiTheme()
	pushed := applyImguiTheme(theme, true)
	defer popImguiTheme(pushed, true)

	e.buildMainMenuBar()
	e.buildStatusBar()
	// DockSpace must land before any windows that want to dock into
	// it — buildPanelSkeleton + workspace.Render below register their
	// windows; the DockSpace decides where they sit.
	e.buildDockSpace()
	e.buildPanelSkeleton(PanelAssets)
	// Inspector owns its own ImGui window via Render (U4) — it builds
	// imgui.SliderFloat / InputText / Combo widgets directly rather
	// than painting into a skeleton rect.
	if e.inspector != nil && e.inspector.Render(e) {
		e.MarkDirty()
	}
	// Scene + every other registered workspace renders its own ImGui
	// window via Render() — the dockspace pulls them into tabs by
	// default and the user re-arranges from there.
	for _, w := range e.workspaces {
		w.Render(e)
	}
}

// activeImguiTheme resolves the theme + project palette pair the
// chrome should push this frame. The editor owns the loaded theme
// directly (U9); the active project supplies the palette. Either may
// be nil — the build helper substitutes default greys.
func (e *Editor) activeImguiTheme() imguiTheme {
	var palette *pixelforge_project.PaletteData
	if e.project != nil {
		palette = &e.project.Palette
	}
	return buildImguiTheme(e.Theme(), palette)
}

// ensurePanelRects lazily allocates the rect map and clears any rect
// whose corresponding ImGui window did not render this frame (handled
// by overwriting in the per-panel callbacks).
func (e *Editor) ensurePanelRects() {
	if e.panelRects == nil {
		e.panelRects = make(map[string]widgets.Rect)
	}
}

// PanelRect returns the most recently captured screen-space rect for
// the named panel. Returns the zero rect when the panel has not yet
// rendered (first frame, or panel hidden via dock close).
func (e *Editor) PanelRect(name string) widgets.Rect {
	if e.panelRects == nil {
		return widgets.Rect{}
	}
	return e.panelRects[name]
}

// buildMainMenuBar emits the top-of-window menu bar. Iterates the
// menu defs the editor already builds for the native widget bank;
// each MenuDef becomes a BeginMenu/EndMenu pair, each MenuItem (or
// separator) becomes a MenuItemBoolV / Separator call that fires the
// existing OnSelect closure.
func (e *Editor) buildMainMenuBar() {
	defs := e.buildMenuDefs()
	if !imgui.BeginMainMenuBar() {
		return
	}
	defer imgui.EndMainMenuBar()
	for _, m := range defs {
		if !imgui.BeginMenu(m.Label) {
			continue
		}
		for _, item := range m.Items {
			if item.Separator {
				imgui.Separator()
				continue
			}
			if imgui.MenuItemBoolV(item.Label, item.Shortcut, false, !item.Disabled) {
				if item.OnSelect != nil {
					item.OnSelect()
				}
			}
		}
		imgui.EndMenu()
	}
}

// buildStatusBar renders the persistent status strip below the main
// viewport using ImGui's InternalBeginViewportSideBar helper (no
// public BeginViewportSideBar exists in cimgui-go's bindings). Content
// mirrors the native chrome's status text: tool, sprite, entity count,
// optional message, and the dirty marker.
func (e *Editor) buildStatusBar() {
	viewport := imgui.MainViewport()
	flags := imgui.WindowFlagsNoSavedSettings |
		imgui.WindowFlagsNoTitleBar |
		imgui.WindowFlagsNoResize |
		imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoScrollbar
	if !imgui.InternalBeginViewportSideBar(PanelStatusBar, viewport, imgui.DirDown, statusBarHeight, flags) {
		imgui.End()
		return
	}
	defer imgui.End()
	imgui.Text(e.statusBarText())
}

// statusBarText composes the left-side status string. Mirrors the
// drawStatusBar() format the native chrome used so tests asserting on
// status content stay valid.
func (e *Editor) statusBarText() string {
	entityCount := 0
	if s := e.activeScene(); s != nil {
		entityCount = len(s.Entities)
	}
	spriteLabel := e.SelectedSpriteName()
	if spriteLabel == "" {
		spriteLabel = "(none)"
	}
	out := fmt.Sprintf("tool=%s  sprite=%s  entities=%d",
		e.Tool().String(), spriteLabel, entityCount)
	if msg := e.StatusMessage(); msg != "" {
		out += "  | " + msg
	}
	if e.IsDirty() {
		out += "  | * unsaved"
	}
	if e.ChromeHidden() {
		out += "  | chrome hidden — press Esc to restore"
	}
	return out
}

// buildPanelSkeleton renders one of the dockable side panels. The
// window owns no body content for U2 — the native asset browser /
// inspector are drawn into the captured rect by Editor.Draw on the
// existing native pass. NoBackground means ImGui's window fill does
// not paint over the native widget body; the title bar still renders
// so users can drag the panel around once DockSpace lands in U3.
func (e *Editor) buildPanelSkeleton(name string) {
	flags := imgui.WindowFlagsNoBackground |
		imgui.WindowFlagsNoScrollbar |
		imgui.WindowFlagsNoCollapse
	if !imgui.BeginV(name, nil, flags) {
		imgui.End()
		// Panel collapsed or hidden — clear its rect so Editor.Draw
		// skips the native dispatch this frame.
		e.panelRects[name] = widgets.Rect{}
		return
	}
	defer imgui.End()
	// Capture the inner content rect (cursor-position + remaining
	// available size), not the outer window rect. This is the area
	// native widgets should fill — it sits below the ImGui title bar
	// and inside the window padding, so the native draws don't paint
	// over ImGui's chrome.
	pos := imgui.CursorScreenPos()
	avail := imgui.ContentRegionAvail()
	e.panelRects[name] = widgets.Rect{
		X: int(pos.X),
		Y: int(pos.Y),
		W: int(avail.X),
		H: int(avail.Y),
	}
}

// imguiCapturesKeyboard reports whether ImGui currently owns keyboard
// input. The editor's shortcut dispatch consults this to avoid firing
// Ctrl+S etc. while an ImGui text input has focus.
//
// Returns false when no live backend is attached (test stubs, or
// before main.go calls AttachImguiBackend) — there's no IO to query.
func (e *Editor) imguiCapturesKeyboard() bool {
	if e.imgui == nil || !e.imgui.live {
		return false
	}
	io := imgui.CurrentIO()
	if io == nil {
		return false
	}
	return io.WantCaptureKeyboard() || io.WantTextInput()
}

// shouldDispatchShortcuts is the pure decision function the editor's
// Update loop uses to gate handleShortcuts. Extracted so tests can
// assert the gate's truth table without needing a live ImGui IO.
//
// Shortcuts fire only when neither a native modal nor ImGui currently
// own keyboard input.
func shouldDispatchShortcuts(modalOpen, imguiKeyboardCaptured bool) bool {
	return !modalOpen && !imguiKeyboardCaptured
}
