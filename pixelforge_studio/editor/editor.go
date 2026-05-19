// Package editor is the top-level Pixelforge Studio editor. It hosts the
// editor window, owns the project model, and orchestrates the panels
// that surface every engine subsystem the studio exposes.
//
// U2 of the ImGui migration replaced the native chrome (chrome.go,
// chrome_visibility.go) with cimgui-go calls; chrome lives in
// imgui_chrome.go. Asset browser / inspector / canvas content still
// render through their native widget code, painted into rects ImGui
// carves out per frame (see Editor.panelRects).
package editor

import (
	"image/color"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"

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

// Editor implements ebiten.Game. State is intentionally narrow:
// settings, keymap, in-memory project, current selection, and the
// inspector. ImGui owns chrome geometry; panelRects is the live
// per-frame map from panel name to screen rect, used by Draw to place
// the existing native asset browser / inspector / canvas widgets.
type Editor struct {
	settings  *Settings
	keymap    *KeyMap
	project   *pixelforge_project.Project
	inspector *Inspector

	selectedEntityID string
	// selectedSceneID names the Scene whose settings inspector should
	// render when no entity is selected. Idea #1 v1 U8 introduces the
	// scene-level inspector panel (grid-size spinners, default-tile
	// picker, camera config) and needs an explicit "the scene itself is
	// the selection target" signal — distinct from "an entity inside
	// the scene is selected". Empty means no scene is selected; the
	// inspector then falls back to "(no selection)".
	selectedSceneID string
	dirty           bool

	// M1.5 surfaces.
	filePicker         *widgets.FilePicker
	confirmDialog      *ConfirmDialog
	fileMenu           *FileMenu
	currentProjectPath string
	statusMessage      string

	assetBrowser       *AssetBrowser
	canvas             *Canvas
	tool               Tool
	selectedSpriteName string
	selectedAudioName  string

	// Idea #1 v1 Paint-tool surface. The painter, undo stack, and
	// palette are co-owned by the editor so the canvas dispatch, the
	// Scene workspace's toolbar, and the Ctrl+Z/Y shortcut handlers
	// all see the same instances. Stack and palette are constructed
	// in NewWithSettings; nil-safe accessors keep tests that build a
	// bare Editor via the zero-value path from crashing.
	painter   *TilePainter
	undoStack *UndoStack
	palette   *TilePalette

	// Idea #3 v1 U3 — PNG-import orchestration. Owns the pending
	// ImportResult between File → Import PNG and the U4 diff
	// modal's Accept/Reject/Re-quantize decision.
	importHandler      *ImportHandler
	importDiffModal    *ImportDiffModal
	audioImportHandler *AudioImportHandler

	// dragDropPoller is the optional per-frame hook plan-008 U11
	// installs to drain ebiten.DroppedFiles into the ingest
	// dispatcher. nil = no hook (tests + the studio's
	// --no-asset-library flag).
	dragDropPoller func()

	// Workspaces registered via RegisterWorkspace. Every workspace's
	// Render() runs each frame inside the DockSpace (U3); activeWorkspace
	// names the currently-focused one so the status bar and tests can
	// answer "which workspace owns input right now".
	workspaces      []Workspace
	activeWorkspace string

	// dockspace bookkeeping (U3). Tracks whether the default layout has
	// been seeded so subsequent frames defer to imgui.ini.
	dockspace *dockspaceState

	// Scene preview texture (U5). The Scene workspace renders this via
	// imgui.Image inside its docked window; the backend repaints the
	// texture each frame by calling sceneGame.Draw against an
	// off-screen ebiten.Image.
	sceneGame    *sceneGame
	sceneTexture imgui.TextureRef
	sceneTexW    int
	sceneTexH    int

	// sceneHovered tracks whether the Scene image was both focused and
	// hovered this frame. SceneWorkspace.Render writes it inside the
	// image's imgui.IsItemHovered check; SceneWorkspace.Update reads
	// it to gate tool dispatch so clicks outside the image (or in
	// other dock tabs) don't fire scene tools.
	sceneHovered bool

	// theme is the editor's loaded chrome theme. Sourced from the
	// embedded editor.pforge fixture on construction; activeImguiTheme
	// converts its palette indices into ImGui Vec4 colours each frame.
	theme *EditorTheme

	// chromeHidden — Esc toggle that hides editor chrome to make the
	// Scene panel feel fullscreen. Replaces the old chromeVisibility
	// object that owned both this flag and an off-screen game canvas;
	// the game-canvas concern moves to U5 (ImGui texture preview).
	chromeHidden bool

	// terminate, when set, makes Update return ebiten.Termination
	// (clean shutdown). File → Quit flips this after a dirty-check.
	terminate bool

	// M5 project listeners — external subsystems (scripting runtime)
	// subscribe here to be notified on project replacement.
	projectListeners []ProjectListener

	// ImGui frame host. Attached by main.go after construction via
	// AttachImguiBackend. Nil-safe: the editor still runs without a
	// backend attached (existing tests rely on this).
	imgui *imguiHost

	// panelRects holds the screen-space rect for each ImGui panel
	// captured during the current frame's buildChrome. Editor.Draw
	// uses these to place native widget content inside the ImGui
	// panel bodies. See imgui_chrome.go for capture; PanelRect for
	// the lookup.
	panelRects map[string]widgets.Rect

	// Idea #2 v1 U5 — TileAtlas painter widget state lives on the
	// existing TilePainter (idea #1's plan). SelectedTile,
	// SetSelectedTile, PaintSubMode, and SetPaintSubMode are
	// convenience accessors on the editor that proxy to
	// painter.ActiveTileID / painter.SubMode so there's one source
	// of truth for the active tile no matter which UI surface
	// mutates it (toolbar palette in the Scene workspace OR the
	// inspector's tilepainter widget).

	// Idea #2 v1 U6 / U7 — AutoTile observer + toast queue.
	// autoTileObserver is the external rule synth (wired by
	// palette.RegisterWith); queuedPromotion is the most-recent
	// promotion the dispatch handed off to the toast UX. Both are
	// session-only; neither persists in the project file.
	autoTileObserver AutoTileObserver
	queuedPromotion  *AutoTilePromotion

	// Idea #2 v1 U7 — session-rule suppression. Designers who pick
	// "No" on a toast insert the rule's signature here; future
	// strokes that re-promote the same rule consult this map and
	// stay silent.
	sessionRuleSuppression map[ruleSignature]struct{}
}

// ruleSignature is the (Pattern, Output) hash the session-suppression
// map keys by. Defined in session_state.go (U7); declared here so
// the field type compiles before U7 lands.
type ruleSignature struct {
	Pattern [9]int
	Output  int
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
// instance — the editor mutates it (via recent-project pushes) and
// triggers debounced autosaves through it.
func NewWithSettings(s *Settings) *Editor {
	if s == nil {
		s = DefaultSettings()
	}
	e := &Editor{
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
	e.fileMenu = NewFileMenu(e)
	e.installDefaultWorkspaces()
	e.theme = loadEditorTheme()
	// Idea #1 v1 Paint-tool wiring. Bind painter ↔ palette ↔ stack
	// here so the canvas dispatch, the Scene workspace's toolbar, and
	// the Ctrl+Z/Ctrl+Y shortcut handlers all see the same instances.
	e.painter = NewTilePainter()
	e.undoStack = NewUndoStack()
	e.palette = NewTilePalette(e.painter)
	e.importHandler = NewImportHandler(e)
	e.importDiffModal = NewImportDiffModal(e)
	e.audioImportHandler = NewAudioImportHandler(e)
	return e
}

// ImportDiffModal returns the editor's PNG-import diff modal. Used
// by tests and (in a future UI integration unit) the per-frame
// Render path that surfaces the popup.
func (e *Editor) ImportDiffModal() *ImportDiffModal {
	if e == nil {
		return nil
	}
	return e.importDiffModal
}

// Painter returns the editor's tile painter. Nil-safe for zero-value
// Editors constructed by tests; callers should check for nil.
func (e *Editor) Painter() *TilePainter {
	if e == nil {
		return nil
	}
	return e.painter
}

// UndoStack returns the editor's per-stroke undo stack. Nil-safe.
func (e *Editor) UndoStack() *UndoStack {
	if e == nil {
		return nil
	}
	return e.undoStack
}

// Palette returns the editor's tile palette panel. Nil-safe.
func (e *Editor) Palette() *TilePalette {
	if e == nil {
		return nil
	}
	return e.palette
}

// Theme returns the editor's loaded chrome theme. Sourced from the
// embedded editor.pforge fixture on construction; reloadable by a
// future theme-switcher feature unit.
func (e *Editor) Theme() *EditorTheme {
	if e.theme == nil {
		e.theme = DefaultEditorTheme()
	}
	return e.theme
}

// ChromeHidden reports whether workspace chrome is currently hidden via
// the Esc toggle. With ImGui chrome the toggle is decorative for now —
// U3 (DockSpace) and U5 (scene-as-texture) will use it to maximise the
// Scene panel.
func (e *Editor) ChromeHidden() bool { return e.chromeHidden }

// ToggleChromeVisibility flips the chrome-hidden state. Bound to Esc;
// callers should invoke this only when no modal is currently visible
// (modal precedence is enforced by handleEscape).
func (e *Editor) ToggleChromeVisibility() { e.chromeHidden = !e.chromeHidden }

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
	e.selectedSceneID = ""
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

// SelectScene marks the named Scene as the inspector's current target.
// An empty id clears the scene selection. Setting a scene selection
// does NOT clear the entity selection — the inspector's dispatch order
// prefers the entity selection when both are set, so a designer who
// clicks an entity inside the scene still sees the entity's character
// sheet rather than scene-level settings.
//
// Idea #1 v1 U8 introduces this selection axis so the scene-settings
// panel has an explicit "the scene itself is what we're editing" hook
// without having to invent a parallel inspector window.
func (e *Editor) SelectScene(id string) {
	e.selectedSceneID = id
}

// SelectedSceneID returns the currently scene-selected ID, or "".
func (e *Editor) SelectedSceneID() string { return e.selectedSceneID }

// selectedScene resolves selectedSceneID against the live project.
// Returns nil when no scene is selected or the ID does not match any
// scene in the project. Used by the inspector to decide whether to
// render the scene-settings panel.
func (e *Editor) selectedScene() *pixelforge_project.Scene {
	if e.project == nil || e.selectedSceneID == "" {
		return nil
	}
	for si := range e.project.Scenes {
		if e.project.Scenes[si].ID == e.selectedSceneID {
			return &e.project.Scenes[si]
		}
	}
	return nil
}

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

// SelectedTile returns the tile ID the Paint tool's brush /
// bucket / rect sub-modes write into the active TileAtlas. Defaults
// to 0 (the "empty" cell — the standard clear-on-paint convention).
// Session-only state; not persisted in the project file. Proxies to
// the editor's TilePainter so the toolbar palette and the
// inspector's tilepainter widget share one source of truth.
func (e *Editor) SelectedTile() int {
	if e == nil || e.painter == nil {
		return 0
	}
	return e.painter.ActiveTileID
}

// SetSelectedTile updates the tile ID the Paint tool writes. Called
// by the tilepainter inspector widget when the designer clicks a
// tile in the palette grid. Does NOT mark the project dirty —
// selection is a transient UI state.
func (e *Editor) SetSelectedTile(id int) {
	if e == nil || e.painter == nil {
		return
	}
	e.painter.SetActiveTile(id)
}

// PaintSubMode returns the active Paint sub-mode (Brush / Bucket /
// Rect). U6's canvas dispatch reads this when LMB events arrive.
// Defaults to PaintBrush. Proxies to the TilePainter.
func (e *Editor) PaintSubMode() PaintSubMode {
	if e == nil || e.painter == nil {
		return PaintBrush
	}
	return e.painter.SubMode
}

// SetPaintSubMode updates the active Paint sub-mode. Called by the
// tilepainter inspector widget's radio buttons. Session-only state;
// no MarkDirty.
func (e *Editor) SetPaintSubMode(m PaintSubMode) {
	if e == nil || e.painter == nil {
		return
	}
	e.painter.SetSubMode(m)
}

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

// ChromeCanvasRect returns the Scene panel's screen rect. External
// workspaces (e.g. the palette package) use this to size their sub-
// panels without needing to query ImGui directly.
//
// Returns the zero rect before the first frame's buildChrome runs.
func (e *Editor) ChromeCanvasRect() widgets.Rect {
	return e.PanelRect(PanelScene)
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

// SetDragDropPoller installs an optional per-frame hook the
// editor invokes once per Update(). Plan-008 U11 uses this seam
// to drain ebiten.DroppedFiles into the ingest dispatcher; nil
// disables the hook. Replacing the poller mid-session is safe.
func (e *Editor) SetDragDropPoller(fn func()) {
	if e == nil {
		return
	}
	e.dragDropPoller = fn
}

// Update advances the editor one tick. ImGui chrome builds first (so
// panel rects and io.WantCaptureKeyboard reflect this frame's input),
// then modals + workspace input dispatch on top.
func (e *Editor) Update() error {
	if e.terminate {
		return ebiten.Termination
	}

	if e.dragDropPoller != nil {
		e.dragDropPoller()
	}

	// ImGui frame brackets the whole tick so the chrome built below
	// sits inside one Begin/EndFrame pair. defer guarantees EndFrame
	// fires even on the early-return paths modals take.
	e.imgui.beginFrame()
	defer e.imgui.endFrame()
	e.imgui.buildContent()
	e.buildChrome()

	// Suppress native shortcuts while a modal owns input or ImGui owns
	// keyboard focus (e.g. a text input has focus). Keeps the editor's
	// Ctrl+S etc. from firing when the user is typing into an ImGui
	// widget.
	modalOwnsInput := e.filePicker.Visible() || e.confirmDialog.Visible()
	if shouldDispatchShortcuts(modalOwnsInput, e.imguiCapturesKeyboard()) {
		e.handleShortcuts()
	}

	// Modals take precedence; only one is visible at a time.
	w, h := e.viewportSize()
	if e.confirmDialog.Update(w, h) {
		return nil
	}
	if e.filePicker.Update(w, h) {
		return nil
	}

	// Active workspace handles its own input — only the focused
	// workspace processes mouse/keyboard so tools don't race across
	// every docked window each frame.
	if active := e.activeWorkspaceImpl(); active != nil {
		if handler, ok := active.(WorkspaceInputHandler); ok {
			handler.Update(e)
		}
	}
	return nil
}

// Draw paints the editor onto the framebuffer Ebitengine hands us.
// Native widget content (asset browser, inspector, scene/canvas) is
// drawn into the rects ImGui captured during Update; ImGui chrome
// composes last so its menu bar / status bar / panel titles sit above.
func (e *Editor) Draw(screen *ebiten.Image) {
	screen.Fill(themeBackground)

	// Native widget content goes into the ImGui panel rects captured
	// during Update.
	if rect := e.PanelRect(PanelAssets); rect.W > 0 && rect.H > 0 && e.assetBrowser != nil {
		e.assetBrowser.Draw(screen, rect, e)
		e.assetBrowser.Update(rect, e)
	}
	// Inspector renders entirely through ImGui now (U4). No native
	// dispatch here — the inspector's Render() runs inside buildChrome
	// and flags dirty via MarkDirty().
	// Every workspace that still renders through the native draw path
	// (Scene pre-U5; palette / capture / scripting pre-U7/U8) gets its
	// captured PanelRect dispatched here. Workspaces fully ported to
	// ImGui don't satisfy NativeWorkspaceContent and are skipped.
	for _, w := range e.workspaces {
		native, ok := w.(NativeWorkspaceContent)
		if !ok {
			continue
		}
		rect := e.PanelRect(w.DisplayName())
		if rect.W <= 0 || rect.H <= 0 {
			continue
		}
		native.Draw(screen, rect, e)
	}

	// Modals render after panel content, before ImGui chrome.
	if e.filePicker.Visible() {
		e.filePicker.Draw(screen)
	}
	if e.confirmDialog.Visible() {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		e.confirmDialog.Draw(screen, w, h)
	}

	// ImGui composes last so its menu bar / status bar / panel titles
	// sit above everything else.
	e.imgui.draw(screen)
}

// Quit triggers a graceful shutdown after a dirty-check.
func (e *Editor) Quit() {
	e.terminate = true
}

// viewportSize returns the current logical viewport dimensions. Used
// by modals that size themselves to the editor's frame.
func (e *Editor) viewportSize() (int, int) {
	w, h := defaultWindowWidth, defaultWindowHeight
	if e.settings != nil {
		if e.settings.WindowWidth > 0 {
			w = e.settings.WindowWidth
		}
		if e.settings.WindowHeight > 0 {
			h = e.settings.WindowHeight
		}
	}
	return w, h
}

// Layout reports the inner game-canvas dimensions.
//
// The studio renders at "logical" pixel resolution defined as
// (windowW / LogicalScale, windowH / LogicalScale). Setting
// LogicalScale=1 keeps the default behaviour (chrome paints at full
// window resolution). Setting it to 2/3/4 lets 4K displays render the
// chrome at integer-multiple DPI without blurring.
//
// The clamp at the end guards against absurd window-to-scale combos:
// if a 4× scale on a small window would leave less than 320×200 for
// the chrome to lay out, fall back to 1×.
func (e *Editor) Layout(outsideWidth, outsideHeight int) (int, int) {
	// ImGui needs the unscaled outside dimensions so its display size
	// matches the OS window. Forward before computing the editor's own
	// logical-scale return value.
	e.imgui.layout(outsideWidth, outsideHeight)

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
