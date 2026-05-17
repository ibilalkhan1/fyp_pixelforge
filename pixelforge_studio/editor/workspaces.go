package editor

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Workspace is the surface that fills the center chrome region. M1.5
// ships the Scene workspace; M2 adds the Palette workspace.
type Workspace interface {
	Name() string
	DisplayName() string
	Draw(dst *ebiten.Image, area widgets.Rect, e *Editor)
	Update(e *Editor)
}

// CanvasWorkspace is implemented by workspaces that render their chrome
// through the editor's Pixelforge cart (R1 dogfooding). The editor's
// Draw routes the cart render through these workspaces in addition to
// the native Draw path, so the workspace can choose whichever surface
// best fits the moment (M3 ships both paths side-by-side).
type CanvasWorkspace interface {
	Workspace
	// DrawCanvas renders the workspace's canvas-resident chrome into
	// the cart's logical canvas. rel is the workspace region inside
	// the canvas (top-left at 0,0; sized to the canvas's dimensions).
	DrawCanvas(rel widgets.Rect, e *Editor)
}

// RegisterWorkspace adds w to the editor's workspace registry. Idempotent
// — re-registering by name replaces the existing entry.
func (e *Editor) RegisterWorkspace(w Workspace) {
	for i := range e.workspaces {
		if e.workspaces[i].Name() == w.Name() {
			e.workspaces[i] = w
			return
		}
	}
	e.workspaces = append(e.workspaces, w)
}

// Workspaces returns the registered workspaces in registration order.
func (e *Editor) Workspaces() []Workspace { return e.workspaces }

// ActiveWorkspaceName returns the active workspace's name, or "".
func (e *Editor) ActiveWorkspaceName() string {
	if w := e.activeWorkspaceImpl(); w != nil {
		return w.Name()
	}
	return ""
}

// SetActiveWorkspaceByName switches to the named workspace. Unknown
// names are a no-op + status-bar warning.
func (e *Editor) SetActiveWorkspaceByName(name string) {
	for i, w := range e.workspaces {
		if w.Name() == name {
			e.activeWorkspace = i
			return
		}
	}
	e.SetStatusMessage("unknown workspace: " + name)
}

// CycleWorkspace advances to the next registered workspace.
func (e *Editor) CycleWorkspace() {
	if len(e.workspaces) == 0 {
		return
	}
	e.activeWorkspace = (e.activeWorkspace + 1) % len(e.workspaces)
}

func (e *Editor) activeWorkspaceImpl() Workspace {
	if e.activeWorkspace < 0 || e.activeWorkspace >= len(e.workspaces) {
		return nil
	}
	return e.workspaces[e.activeWorkspace]
}

// installDefaultWorkspaces registers M1.5's Scene workspace plus the
// M3 stub workspaces (Behavior, Audio, Capture, Procgen) so the tab
// strip is stable from M3 onwards. The Palette workspace (M2) is
// registered by the palette package; the studio main wires it after
// New() returns.
func (e *Editor) installDefaultWorkspaces() {
	e.RegisterWorkspace(&SceneWorkspace{})
	e.installStubWorkspaces()
}

// SceneWorkspace is the M1.5 viewport workspace. It defers drawing and
// input to the editor's canvas.
type SceneWorkspace struct{}

// Name is the stable workspace identifier.
func (s *SceneWorkspace) Name() string { return "scene" }

// DisplayName is the tab strip label.
func (s *SceneWorkspace) DisplayName() string { return "Scene" }

// Draw renders the scene viewport.
//
// M3 hybrid: this still paints via the native overlay path so the
// transition is smooth. The canvas-resident equivalent lives in
// DrawCanvas and is invoked by the editor cart Render phase.
func (s *SceneWorkspace) Draw(dst *ebiten.Image, area widgets.Rect, e *Editor) {
	if e.canvas != nil {
		e.canvas.Draw(dst, area, e)
	}
}

// DrawCanvas renders the scene viewport into the cart's Pixelforge
// canvas. rel is the workspace region in canvas-local coordinates.
func (s *SceneWorkspace) DrawCanvas(rel widgets.Rect, e *Editor) {
	if e.canvas != nil {
		e.canvas.DrawCanvas(rel, e)
	}
}

// Update dispatches mouse input to the canvas, using the Scene panel
// rect ImGui captured this frame.
func (s *SceneWorkspace) Update(e *Editor) {
	area := e.PanelRect(PanelScene)
	if e.canvas != nil {
		e.canvas.Update(area, e)
	}
}

// (drawTabStrip / handleTabStripClick removed in U2 — the native tab
// strip is replaced by the ImGui menu bar's View menu and, in U3, by
// DockSpace tab handling.)
