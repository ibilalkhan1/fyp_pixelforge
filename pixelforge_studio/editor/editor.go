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

	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
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

	// M1.5 surfaces.
	menuBar            *widgets.MenuBar
	filePicker         *widgets.FilePicker
	confirmDialog      *ConfirmDialog
	fileMenu           *FileMenu
	currentProjectPath string
	statusMessage      string

	// M3.1 canvas-resident chrome (U43). Lives alongside the native
	// menuBar/statusMessage path during the migration; the cart's
	// render layer dispatches to these when canvas chrome is enabled.
	canvasMenuBar   *pguiwidgets.MenuBar
	canvasStatusBar *pguiwidgets.StatusBar

	assetBrowser      *AssetBrowser
	canvas            *Canvas
	tool              Tool
	selectedSpriteName string
	selectedAudioName  string

	// M2 workspaces.
	workspaces       []Workspace
	activeWorkspace  int
	tabStripHeight   int
	tabStripRegistered bool

	// M3 editor cart: logical Pixelforge canvas + canvas-resident chrome.
	cart *editorCart

	// M3 always-on game + Esc-toggle for workspace chrome visibility.
	chromeVis *chromeVisibility

	// terminate, when set, makes Update return ebiten.Termination
	// (clean shutdown). File → Quit flips this after a dirty-check.
	terminate bool

	// M5 project listeners — external subsystems (scripting runtime)
	// subscribe here to be notified on project replacement.
	projectListeners []ProjectListener
}

// ProjectListener is the contract the scripting runtime (and any
// future per-project subsystem) implements to react to project
// replacement. Editor.SetProject fires OnProjectChanged on every
// registered listener, in registration order.
type ProjectListener interface {
	OnProjectChanged(p *pixelforge_project.Project)
}

// RegisterProjectListener subscribes l to project-change notifications.
// Duplicate listeners (same pointer) are ignored.
func (e *Editor) RegisterProjectListener(l ProjectListener) {
	if l == nil {
		return
	}
	for _, existing := range e.projectListeners {
		if existing == l {
			return
		}
	}
	e.projectListeners = append(e.projectListeners, l)
	// Fire once on registration so a newly-attached listener sees
	// the current project immediately.
	l.OnProjectChanged(e.project)
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
	e := &Editor{
		chrome:    defaultChromeLayout(),
		settings:  s,
		keymap:    DefaultKeyMap(),
		project:   pixelforge_project.NewProject("untitled"),
		inspector: NewInspector(),

		filePicker:    widgets.NewFilePicker(),
		confirmDialog: NewConfirmDialog(),
		assetBrowser:  NewAssetBrowser(),
		canvas:        NewCanvas(),
		tool:          ToolSelect,
	}
	// Apply persisted panel widths from settings (U48) before the
	// first recompute so the chrome opens at the user's last layout.
	if s.LeftPanelW > 0 {
		e.chrome.LeftPanelW = s.LeftPanelW
	}
	if s.RightPanelW > 0 {
		e.chrome.RightPanelW = s.RightPanelW
	}
	e.fileMenu = NewFileMenu(e)
	defs := e.buildMenuDefs()
	e.menuBar = widgets.NewMenuBar(defs)
	e.canvasMenuBar = pguiwidgets.NewMenuBar(translateMenuDefs(defs), pguiwidgets.MenuBarOptions{})
	e.canvasStatusBar = pguiwidgets.NewStatusBar()
	e.installDefaultWorkspaces()
	e.cart = newEditorCart()
	e.cart.SetTheme(loadEditorTheme())
	e.chromeVis = newChromeVisibility()
	return e
}

// CanvasMenuBar exposes the canvas-resident menu bar that mirrors the
// native bank's content. Lives in parallel with menuBar during the M3.1
// migration; the cart's render layer dispatches here when canvas chrome
// is enabled.
func (e *Editor) CanvasMenuBar() *pguiwidgets.MenuBar { return e.canvasMenuBar }

// CanvasStatusBar exposes the canvas-resident status bar (U43).
func (e *Editor) CanvasStatusBar() *pguiwidgets.StatusBar { return e.canvasStatusBar }

// translateMenuDefs maps the editor's native widgets.MenuDef to the
// canvas widget bank's pguiwidgets.MenuDef. Field-for-field copy; the
// canvas bank intentionally mirrors the native shape so this stays
// trivial.
func translateMenuDefs(in []widgets.MenuDef) []pguiwidgets.MenuDef {
	out := make([]pguiwidgets.MenuDef, len(in))
	for i, m := range in {
		items := make([]pguiwidgets.MenuItem, len(m.Items))
		for j, it := range m.Items {
			items[j] = pguiwidgets.MenuItem{
				Label:     it.Label,
				Shortcut:  it.Shortcut,
				OnSelect:  it.OnSelect,
				Separator: it.Separator,
				Disabled:  it.Disabled,
			}
		}
		out[i] = pguiwidgets.MenuDef{Label: m.Label, Items: items}
	}
	return out
}

// Cart exposes the editor's logical Pixelforge canvas wrapper for
// canvas-resident workspace code (U28-U33).
func (e *Editor) Cart() *editorCart { return e.cart }

// ChromeHidden reports whether workspace chrome is currently hidden via
// the Esc toggle.
func (e *Editor) ChromeHidden() bool {
	if e.chromeVis == nil {
		return false
	}
	return e.chromeVis.Hidden()
}

// ToggleChromeVisibility flips the chrome-hidden state. Bound to Esc;
// callers should invoke this only when no modal is currently visible
// (modal precedence is enforced by handleEscape).
func (e *Editor) ToggleChromeVisibility() {
	if e.chromeVis == nil {
		return
	}
	e.chromeVis.Toggle()
}

// handleEscape routes Esc with modal precedence: open modals dismiss
// first; otherwise the chrome-visibility toggle fires.
func (e *Editor) handleEscape() {
	if e.confirmDialog != nil && e.confirmDialog.Visible() {
		// Confirm modal owns Esc via its own Update.
		return
	}
	if e.filePicker != nil && e.filePicker.Visible() {
		// File picker owns Esc via its own Update.
		return
	}
	e.ToggleChromeVisibility()
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
// selection, dirty flag, and selected sprite/audio. Fires every
// registered ProjectListener so per-project subsystems (the M5
// scripting runtime in particular) can teardown and re-instantiate.
func (e *Editor) SetProject(p *pixelforge_project.Project) {
	if e.project == p && p != nil {
		// Same pointer — listeners don't need to restart.
		return
	}
	e.project = p
	e.selectedEntityID = ""
	e.selectedSpriteName = ""
	e.selectedAudioName = ""
	e.dirty = false
	if e.assetBrowser != nil {
		e.assetBrowser.InvalidateCache()
	}
	for _, l := range e.projectListeners {
		l.OnProjectChanged(p)
	}
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

// ClearDirty drops the dirty flag. Called by Save handlers.
func (e *Editor) ClearDirty() { e.dirty = false }

// CurrentProjectPath returns the on-disk path the loaded project came
// from, or "" for an untitled project.
func (e *Editor) CurrentProjectPath() string { return e.currentProjectPath }

// SetCurrentProjectPath records the path the next Save should write to.
func (e *Editor) SetCurrentProjectPath(path string) { e.currentProjectPath = path }

// SelectedSpriteName returns the name of the sprite the Place tool will
// instantiate. Empty when nothing is selected.
func (e *Editor) SelectedSpriteName() string { return e.selectedSpriteName }

// SetSelectedSpriteName updates the active sprite for the Place tool.
func (e *Editor) SetSelectedSpriteName(name string) { e.selectedSpriteName = name }

// SelectedAudioName returns the currently-highlighted audio sample name.
func (e *Editor) SelectedAudioName() string { return e.selectedAudioName }

// SetSelectedAudioName updates the highlighted audio sample.
func (e *Editor) SetSelectedAudioName(name string) { e.selectedAudioName = name }

// Tool returns the active canvas tool.
func (e *Editor) Tool() Tool { return e.tool }

// SetTool switches the active canvas tool.
func (e *Editor) SetTool(t Tool) { e.tool = t }

// StatusMessage returns the current status-bar message.
func (e *Editor) StatusMessage() string { return e.statusMessage }

// SetStatusMessage updates the status-bar message.
func (e *Editor) SetStatusMessage(s string) { e.statusMessage = s }

// FilePicker exposes the editor's file picker so menu handlers can drive it.
func (e *Editor) FilePicker() *widgets.FilePicker { return e.filePicker }

// ConfirmDialog exposes the editor's confirm modal.
func (e *Editor) ConfirmDialog() *ConfirmDialog { return e.confirmDialog }

// FileMenu exposes the editor's file-menu actions for testing.
func (e *Editor) FileMenu() *FileMenu { return e.fileMenu }

// ChromeCanvasRect returns the central canvas region in window-pixel
// space. External workspaces (e.g. the palette package) use this to
// size their sub-panels without recomputing chrome themselves.
func (e *Editor) ChromeCanvasRect() widgets.Rect {
	return e.chrome.canvasRectWidgets()
}

// AssetBrowserComponent exposes the editor's asset browser so external
// packages (palette import pipeline) can invalidate its cache.
func (e *Editor) AssetBrowserComponent() *AssetBrowser { return e.assetBrowser }

// CanvasComponent exposes the editor's canvas for tools that mutate
// scene state from outside the editor package (palette painter).
func (e *Editor) CanvasComponent() *Canvas { return e.canvas }

// PromptIfDirty runs action immediately if the project is clean,
// otherwise raises a confirm modal that runs action only on confirm.
func (e *Editor) PromptIfDirty(title, message string, action func()) {
	if !e.dirty {
		action()
		return
	}
	e.confirmDialog.Show(title, message, action)
}

// Update advances the editor one tick. Menu bar, modals, and active
// workspace all receive input here.
func (e *Editor) Update() error {
	if e.terminate {
		return ebiten.Termination
	}

	// U33: keep the always-on game canvas allocated to match the
	// project's screen size. The game canvas updates at TPS regardless
	// of editor activity; this Ensure call is cheap when sizes match.
	if e.chromeVis != nil && e.project != nil {
		e.chromeVis.EnsureGameCanvas(e.project.ScreenWidth, e.project.ScreenHeight)
	}

	// Process keyboard shortcuts. Shortcuts are skipped when a modal
	// owns input.
	modalOwnsInput := e.filePicker.Visible() || e.confirmDialog.Visible()

	if !modalOwnsInput {
		e.handleShortcuts()
	}

	// Modals take precedence; only one is visible at a time in M1.5.
	w, h := e.chrome.WindowW, e.chrome.WindowH
	if w == 0 {
		w = defaultWindowWidth
	}
	if h == 0 {
		h = defaultWindowHeight
	}
	if e.confirmDialog.Update(w, h) {
		return nil
	}
	if e.filePicker.Update(w, h) {
		return nil
	}

	// Menu bar grabs input when its label or dropdown is hovered.
	menuRect := widgets.Rect{X: 0, Y: 0, W: w, H: widgets.MenuBarHeight}
	if e.menuBar.Update(menuRect) {
		return nil
	}

	// Active workspace handles its own input.
	if e.activeWorkspaceImpl() != nil {
		e.activeWorkspaceImpl().Update(e)
	}
	return nil
}

// Draw paints the chrome and panel content onto the framebuffer
// Ebitengine hands us. The chrome layout is recomputed each frame so
// window resizes apply without an explicit hook.
func (e *Editor) Draw(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	e.chrome.recompute(w, h)
	screen.Fill(themeBackground)

	// Render the canvas-resident workspace chrome into the editor
	// cart's Pixelforge canvas. The active workspace decides whether
	// to drive its content through the cart (R1 dogfooding) or stay
	// on the native path. The cart's contents are blitted into the
	// chrome's canvas region before the rest of the native chrome
	// paints on top.
	if e.cart != nil {
		ws := e.activeWorkspaceImpl()
		if cw, ok := ws.(CanvasWorkspace); ok && cw != nil {
			e.cart.renderInto(e)
			e.cart.blitToScreen(screen, e.chrome.canvasRectWidgets())
			_ = cw
		}
	}

	e.chrome.draw(screen, e)

	// Menu bar sits on top of the chrome's title region.
	menuRect := widgets.Rect{X: 0, Y: 0, W: w, H: widgets.MenuBarHeight}
	e.menuBar.Draw(screen, menuRect)

	// Modals render after everything else.
	if e.filePicker.Visible() {
		e.filePicker.Draw(screen)
	}
	if e.confirmDialog.Visible() {
		e.confirmDialog.Draw(screen, w, h)
	}
}

// Quit triggers a graceful shutdown after a dirty-check.
func (e *Editor) Quit() {
	e.terminate = true
}

// Layout reports the inner game-canvas dimensions.
//
// The studio renders chrome at "logical" pixel resolution defined as
// (windowW / LogicalScale, windowH / LogicalScale). Setting
// LogicalScale=1 keeps M0-M3 behaviour (chrome paints at full window
// resolution). Setting it to 2/3/4 lets 4K displays render the chrome
// at integer-multiple DPI without blurring the cofont glyphs.
//
// The clamp at the end guards against absurd window-to-scale combos:
// if a 4× scale on a small window would leave less than 320×200 for
// the chrome to lay out, fall back to 1×.
func (e *Editor) Layout(outsideWidth, outsideHeight int) (int, int) {
	scale := 1
	if e.settings != nil && e.settings.LogicalScale > 0 {
		scale = e.settings.LogicalScale
	}
	if scale < 1 {
		scale = 1
	}
	if scale > 4 {
		scale = 4
	}
	logicalW := outsideWidth / scale
	logicalH := outsideHeight / scale
	// Hard floor on minimum logical canvas area so chrome doesn't
	// collapse to unusable dimensions at high scale + small window.
	minLogicalW, minLogicalH := 320, 200
	if logicalW < minLogicalW || logicalH < minLogicalH {
		return outsideWidth, outsideHeight
	}
	return logicalW, logicalH
}

// EffectiveLogicalScale returns the LogicalScale value actually in use
// after clamping. Exposed for tests and the View → Scale menu.
func (e *Editor) EffectiveLogicalScale() int {
	if e.settings == nil {
		return 1
	}
	scale := e.settings.LogicalScale
	if scale < 1 {
		return 1
	}
	if scale > 4 {
		return 4
	}
	return scale
}

// ResizeLeftPanel applies a horizontal delta to the left panel and
// persists the new width via settings.
func (e *Editor) ResizeLeftPanel(delta int) {
	if e.chrome == nil {
		return
	}
	e.chrome.ApplyLeftPanelDelta(delta)
	if e.settings != nil {
		e.settings.LeftPanelW = e.chrome.LeftPanelW
		e.settings.MarkDirty()
	}
}

// ResizeRightPanel mirrors ResizeLeftPanel for the right gutter.
func (e *Editor) ResizeRightPanel(delta int) {
	if e.chrome == nil {
		return
	}
	e.chrome.ApplyRightPanelDelta(delta)
	if e.settings != nil {
		e.settings.RightPanelW = e.chrome.RightPanelW
		e.settings.MarkDirty()
	}
}

// SetLogicalScale updates the chrome's render scale and persists it
// via the debounced settings writer.
func (e *Editor) SetLogicalScale(n int) {
	if e.settings == nil {
		return
	}
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	if e.settings.LogicalScale == n {
		return
	}
	e.settings.LogicalScale = n
	e.settings.MarkDirty()
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

// activeScene returns the active scene pointer. M1.5 uses Scene[0]; M3
// adds explicit scene switching.
func (e *Editor) activeScene() *pixelforge_project.Scene {
	if e.project == nil || len(e.project.Scenes) == 0 {
		return nil
	}
	return &e.project.Scenes[0]
}

// themeBackground is the editor's base fill.
var themeBackground = color.RGBA{R: 0x10, G: 0x10, B: 0x16, A: 0xff}
