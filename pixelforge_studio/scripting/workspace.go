// Package scripting hosts the M5 visual-scripting workspace. U8 of
// the ImGui migration plan rewrote workspace.go on ImGui primitives:
// the pgui Tabs widget retired in favour of imgui.BeginTabBar; the
// canvas-resident render path retired in favour of imgui.Begin/End.
// The visual-scripting *model* (BehaviorGraph, StepNode, EventSheet)
// is unchanged — only the editor surface ports.
package scripting

import (
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// Sub-pane identifiers. With the ImGui rewrite, the tab system is
// driven by imgui.BeginTabBar; the constants survive for tests and
// keymap actions that still address sub-panes by index.
const (
	PaneLane    = 0
	PaneSheet   = 1
	PaneCatalog = 2
	PaneDebug   = 3
)

// Workspace is the M5 behaviour workspace. Implements editor.Workspace
// (Name / DisplayName / Render) so it slots into the dockspace.
type Workspace struct {
	engine    *runtime.Engine
	activeTab int

	statusLine string

	// Sub-pane state — Lane is fully ImGui-driven by U8; the
	// remaining three sub-panes carry their existing state but
	// render placeholders until their feature units catch up.
	lane     *LaneEditor
	sheet    *EventSheetEditor
	catalog  *TopicCatalog
	debugger *Debugger
}

// NewWorkspace constructs a behaviour workspace bound to (initially)
// no engine. The engine is wired by RegisterWith once the editor
// supplies a project.
func NewWorkspace() *Workspace {
	return &Workspace{
		lane:     newLaneEditor(),
		sheet:    newEventSheetEditor(),
		catalog:  newTopicCatalog(),
		debugger: newDebugger(),
	}
}

// Name is the stable workspace identifier.
func (w *Workspace) Name() string { return "behavior" }

// DisplayName is the dock window title.
func (w *Workspace) DisplayName() string { return "Behavior" }

// Engine returns the runtime engine bound to the workspace, or nil.
func (w *Workspace) Engine() *runtime.Engine { return w.engine }

// ActiveTab returns the index of the currently-selected sub-pane.
func (w *Workspace) ActiveTab() int { return w.activeTab }

// SetActiveTab switches the active sub-pane.
func (w *Workspace) SetActiveTab(idx int) {
	if idx < 0 || idx > PaneDebug {
		return
	}
	w.activeTab = idx
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

// Render builds the Behavior ImGui window each frame. A tab bar
// switches between Lane / Sheet / Catalog / Debug sub-panes; Lane
// fully ports to ImGui under U8, the rest carry placeholders until
// their feature units rebuild on ImGui.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	flags := imgui.WindowFlagsNoCollapse
	if !imgui.BeginV(w.DisplayName(), nil, flags) {
		imgui.End()
		e.SetPanelRect(w.DisplayName(), widgets.Rect{})
		return
	}
	defer imgui.End()
	e.CaptureCurrentWindowRect(w.DisplayName())

	if e.Project() == nil {
		imgui.TextDisabled("(no project)")
		return
	}

	w.renderEngineHeader()

	if imgui.BeginTabBar("##scripting-tabs") {
		w.renderTab("Lane", PaneLane, func() { w.lane.Render(e, w.engine) })
		w.renderTab("Sheet", PaneSheet, func() { imgui.TextDisabled("event sheet — pending ImGui rewrite") })
		w.renderTab("Catalog", PaneCatalog, func() { imgui.TextDisabled("topic catalog — pending ImGui rewrite") })
		w.renderTab("Debug", PaneDebug, func() { imgui.TextDisabled("debugger — pending ImGui rewrite") })
		imgui.EndTabBar()
	}

	if w.statusLine != "" {
		imgui.Separator()
		imgui.TextDisabled(w.statusLine)
	}
}

// renderTab emits one tab in the workspace's tab bar. When the tab
// is active, body fires inside the tab item; the workspace's
// activeTab field is updated to keep the keymap-addressable index
// in sync with the user's pick.
func (w *Workspace) renderTab(label string, idx int, body func()) {
	if !imgui.BeginTabItem(label) {
		return
	}
	w.activeTab = idx
	body()
	imgui.EndTabItem()
}

// renderEngineHeader emits the "engine running / stopped" indicator
// + Run / Stop buttons at the top of the workspace.
func (w *Workspace) renderEngineHeader() {
	running := w.engine != nil && w.engine.Running()
	if running {
		imgui.Text("engine: running")
	} else {
		imgui.TextDisabled("engine: stopped")
	}
	imgui.SameLine()
	if imgui.Button("Run") {
		w.runEngine()
	}
	imgui.SameLine()
	if imgui.Button("Stop") {
		w.stopEngine()
	}
	imgui.Separator()
}

// Update routes keymap input to the active sub-pane plus the engine
// run/stop shortcuts.
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

// RegisterWith installs the behaviour workspace on the editor and
// subscribes it as a ProjectListener.
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
