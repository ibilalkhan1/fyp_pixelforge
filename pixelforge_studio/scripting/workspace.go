// Package scripting hosts the M5 visual-scripting workspace —
// behaviour authoring surfaces (Step lane editor, Event sheet
// editor, topic catalog, visual debugger) wired to the runtime
// engine in scripting/runtime.
package scripting

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// Sub-pane identifiers. The Tabs widget keeps the four sub-panes
// indexable by integer; we re-export the names for tests and keymap
// handlers.
const (
	PaneLane    = 0
	PaneSheet   = 1
	PaneCatalog = 2
	PaneDebug   = 3
)

// Workspace is the M5 behaviour workspace. Implements
// editor.CanvasWorkspace so it slots into the existing tab strip and
// canvas-resident chrome path.
type Workspace struct {
	engine    *runtime.Engine
	tabs      *pguiwidgets.Tabs
	activeTab int

	statusLine string

	// Sub-pane state. Per-pane structs are populated by later units
	// (U7, U9, U11, U13). v1 ships empty placeholders so the shell
	// renders and switches tabs.
	lane     *LaneEditor
	sheet    *EventSheetEditor
	catalog  *TopicCatalog
	debugger *Debugger
}

// NewWorkspace constructs a behaviour workspace bound to (initially)
// no engine. The engine is wired by RegisterWith once the editor
// supplies a project.
func NewWorkspace() *Workspace {
	w := &Workspace{}
	w.tabs = pguiwidgets.NewTabs(0, 0, 0, 0, pguiwidgets.TabsOptions{
		Labels:   []string{"Lane", "Sheet", "Catalog", "Debug"},
		Selected: PaneLane,
		OnSelect: func(idx int) { w.activeTab = idx },
	})
	w.lane = newLaneEditor()
	w.sheet = newEventSheetEditor()
	w.catalog = newTopicCatalog()
	w.debugger = newDebugger()
	return w
}

// Name is the stable workspace identifier (matches the M3 stub).
func (w *Workspace) Name() string { return "behavior" }

// DisplayName is the tab strip label.
func (w *Workspace) DisplayName() string { return "Behavior" }

// Engine returns the runtime engine bound to the workspace, or nil
// when no project is loaded.
func (w *Workspace) Engine() *runtime.Engine { return w.engine }

// ActiveTab returns the index of the currently-selected sub-pane.
func (w *Workspace) ActiveTab() int { return w.activeTab }

// SetActiveTab switches the active sub-pane.
func (w *Workspace) SetActiveTab(idx int) {
	if idx < 0 || idx > PaneDebug {
		return
	}
	w.activeTab = idx
	if w.tabs != nil {
		w.tabs.Selected = idx
	}
}

// Status returns the workspace's status footer text.
func (w *Workspace) Status() string { return w.statusLine }

// SetStatus updates the workspace's status footer text.
func (w *Workspace) SetStatus(s string) { w.statusLine = s }

// Lane exposes the lane sub-pane.
func (w *Workspace) Lane() *LaneEditor { return w.lane }

// Sheet exposes the event-sheet sub-pane.
func (w *Workspace) Sheet() *EventSheetEditor { return w.sheet }

// Catalog exposes the topic-catalog sub-pane.
func (w *Workspace) Catalog() *TopicCatalog { return w.catalog }

// Debugger exposes the debugger sub-pane.
func (w *Workspace) Debugger() *Debugger { return w.debugger }

// OnProjectChanged is the ProjectListener hook. Tears down the prior
// engine and spins up a fresh one rooted at p.
func (w *Workspace) OnProjectChanged(p *pixelforge_project.Project) {
	if w.engine != nil {
		w.engine.Stop()
		w.engine = nil
	}
	if p == nil {
		return
	}
	w.engine = runtime.New(p)
	w.engine.Start()
	w.lane.bind(p)
	w.sheet.bind(p)
	w.debugger.bind(w.engine)
}

// Update routes input to the active sub-pane and the tabs widget.
func (w *Workspace) Update(e *editor.Editor) {
	if e == nil {
		return
	}
	if km := e.KeyMap(); km != nil {
		if km.JustPressed("behavior.tab_lane") {
			w.SetActiveTab(PaneLane)
		}
		if km.JustPressed("behavior.tab_sheet") {
			w.SetActiveTab(PaneSheet)
		}
		if km.JustPressed("behavior.tab_catalog") {
			w.SetActiveTab(PaneCatalog)
		}
		if km.JustPressed("behavior.tab_debug") {
			w.SetActiveTab(PaneDebug)
		}
		if km.JustPressed("behavior.run") {
			w.runEngine()
		}
		if km.JustPressed("behavior.stop") {
			w.stopEngine()
		}
	}

	switch w.activeTab {
	case PaneLane:
		w.lane.Update(e, w.engine)
	case PaneSheet:
		w.sheet.Update(e, w.engine)
	case PaneCatalog:
		w.catalog.Update(e, w.engine)
	case PaneDebug:
		w.debugger.Update(e, w.engine)
	}
}

func (w *Workspace) runEngine() {
	if w.engine == nil {
		return
	}
	if w.engine.Running() {
		w.statusLine = "engine already running"
		return
	}
	w.engine.Start()
	w.statusLine = "engine started"
}

func (w *Workspace) stopEngine() {
	if w.engine == nil {
		return
	}
	w.engine.Stop()
	w.statusLine = "engine stopped"
}

// Draw renders the workspace via the native overlay path (used in
// non-cart code paths and headless tests).
func (w *Workspace) Draw(dst *ebiten.Image, area widgets.Rect, e *editor.Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	if e == nil || e.Project() == nil {
		ebitenutil.DebugPrintAt(dst, "Behavior - (no project)", area.X+8, area.Y+8)
		return
	}
	running := w.engine != nil && w.engine.Running()
	ebitenutil.DebugPrintAt(dst,
		fmt.Sprintf("Behavior — engine: %v, tab: %d", running, w.activeTab),
		area.X+8, area.Y+8)
}

// DrawCanvas renders the canvas-resident chrome (panel header, tab
// strip, sub-pane content, footer).
func (w *Workspace) DrawCanvas(rel widgets.Rect, e *editor.Editor) {
	if rel.W <= 0 || rel.H <= 0 || e == nil {
		return
	}
	theme := editor.DefaultEditorTheme()
	if c := e.Cart(); c != nil && c.Theme() != nil {
		theme = c.Theme()
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Background.
	pixelforge.SetColor(theme.BackgroundSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	// Header.
	pixelforge.SetColor(theme.PanelHeaderSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+15)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("BEHAVIOR", rel.X+8, rel.Y+4)

	// Engine status indicator.
	statusText := "stopped"
	if w.engine != nil && w.engine.Running() {
		statusText = "running"
	}
	pixelforge.SetColor(theme.TextDimSlot)
	pixelforge_cofont.Print(statusText, rel.X+rel.W-len(statusText)*4-8, rel.Y+4)

	if e.Project() == nil {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(no project)", rel.X+8, rel.Y+24)
		return
	}

	// Tabs strip.
	w.tabs.X = rel.X + 8
	w.tabs.Y = rel.Y + 20
	w.tabs.W = rel.W - 16
	w.tabs.H = 18
	w.tabs.Selected = w.activeTab
	w.drawTabsWidget()

	// Sub-pane content area.
	paneRect := widgets.Rect{
		X: rel.X + 8,
		Y: rel.Y + 42,
		W: rel.W - 16,
		H: rel.H - 60,
	}
	switch w.activeTab {
	case PaneLane:
		w.lane.DrawCanvas(paneRect, e, theme)
	case PaneSheet:
		w.sheet.DrawCanvas(paneRect, e, theme)
	case PaneCatalog:
		w.catalog.DrawCanvas(paneRect, e, theme)
	case PaneDebug:
		w.debugger.DrawCanvas(paneRect, e, theme)
	}

	// Footer status line.
	if w.statusLine != "" {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print(w.statusLine, rel.X+8, rel.Y+rel.H-12)
	}
}

func (w *Workspace) drawTabsWidget() {
	prevCamX, prevCamY := pixelforge.Camera.X, pixelforge.Camera.Y
	defer func() {
		pixelforge.Camera.X, pixelforge.Camera.Y = prevCamX, prevCamY
	}()
	pixelforge.Camera.X -= w.tabs.X
	pixelforge.Camera.Y -= w.tabs.Y
	if w.tabs.OnDraw != nil {
		w.tabs.OnDraw(pgui.DrawEvent{Element: w.tabs.Element})
	}
}

// RegisterWith installs the behaviour workspace on the editor in
// place of the M3 stub, registers the keymap actions, and subscribes
// the workspace as a ProjectListener so it teardowns and re-spins on
// project changes.
func RegisterWith(e *editor.Editor) *Workspace {
	if e == nil {
		return nil
	}
	w := NewWorkspace()
	e.RegisterWorkspace(w)
	registerKeymap(e)
	e.RegisterProjectListener(w)
	return w
}

func registerKeymap(e *editor.Editor) {
	km := e.KeyMap()
	if km == nil {
		return
	}
	km.Register("behavior.tab_lane", editor.Binding{Mods: editor.ModAlt, Key: ebiten.KeyDigit1})
	km.Register("behavior.tab_sheet", editor.Binding{Mods: editor.ModAlt, Key: ebiten.KeyDigit2})
	km.Register("behavior.tab_catalog", editor.Binding{Mods: editor.ModAlt, Key: ebiten.KeyDigit3})
	km.Register("behavior.tab_debug", editor.Binding{Mods: editor.ModAlt, Key: ebiten.KeyDigit4})
	km.Register("behavior.run", editor.Binding{Key: ebiten.KeyF5})
	km.Register("behavior.stop", editor.Binding{Key: ebiten.KeyF6})
	km.Register("behavior.view_as_go", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyV})
	km.Register("behavior.record_toggle", editor.Binding{Mods: editor.ModCtrl | editor.ModShift, Key: ebiten.KeyR})
}
