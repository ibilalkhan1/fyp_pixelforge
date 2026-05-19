package editor

import (
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Workspace is the contract for a dockable editor surface. U3 of the
// ImGui migration replaced the tab-strip switcher with an ImGui
// DockSpace; a workspace is now "a set of registered ImGui windows"
// rather than a panel that owns the canvas. Each frame, every
// registered workspace's Render is called inside the dockspace; ImGui
// handles tab grouping when multiple windows share a dock node, and
// the user's docking choices persist via imgui.ini.
//
// Workspaces whose body content still renders through native
// ebitenutil/vector calls (the M2 palette, the M4 capture pre-U7
// rewrite, the M5 scripting pre-U8 rewrite, and the M1.5 scene pre-U5
// rewrite) use Render() purely to register the ImGui window and
// capture its inner content rect; Editor.Draw then dispatches the
// native widget code into that rect.
type Workspace interface {
	Name() string
	DisplayName() string
	Render(e *Editor)
}

// NativeWorkspaceContent is the optional contract pre-rewrite
// workspaces use to dispatch their existing ebitenutil/vector drawing
// into the rect Render() captured. Editor.Draw type-asserts each
// registered workspace against this interface and routes accordingly.
// Workspaces fully ported to ImGui (post-U5/U7/U8) won't implement it.
type NativeWorkspaceContent interface {
	Draw(dst *ebiten.Image, area widgets.Rect, e *Editor)
}

// WorkspaceInputHandler is the optional contract a workspace
// implements when it consumes input outside of its own ImGui Render
// call. The editor dispatches to the focused workspace's Update()
// each frame; SceneWorkspace uses this to route mouse coords through
// the texture-mapped canvas tool dispatch (U5).
type WorkspaceInputHandler interface {
	Update(e *Editor)
}

// RegisterWorkspace adds w to the editor's workspace registry. Idempotent
// — re-registering by name replaces the existing entry. Replacement
// (vs. append) is load-bearing: capture.RegisterWith and
// scripting.RegisterWith call this with their real workspace, which
// supersedes any earlier registration of the same name.
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

// ActiveWorkspaceName returns the workspace whose window currently
// owns ImGui focus, or the last name SetActiveWorkspaceByName routed
// to. With DockSpace there is no single "active" workspace — every
// docked window renders every frame — but the M2 surface that used
// activeWorkspace for status bar text still asks via this accessor.
func (e *Editor) ActiveWorkspaceName() string { return e.activeWorkspace }

// SetActiveWorkspaceByName focuses the matching ImGui window. Unknown
// names are a no-op + status-bar warning.
//
// With DockSpace, "active" means "currently focused" — SetWindowFocus
// brings the workspace's dock tab to front. The editor records the
// name so ActiveWorkspaceName still returns a meaningful value to
// status-bar and test callers.
func (e *Editor) SetActiveWorkspaceByName(name string) {
	var found Workspace
	for _, w := range e.workspaces {
		if w.Name() == name {
			found = w
			break
		}
	}
	if found == nil {
		e.SetStatusMessage("unknown workspace: " + name)
		return
	}
	e.activeWorkspace = name
	if e.imgui != nil && e.imgui.live {
		imgui.SetWindowFocusStr(found.DisplayName())
	}
}

// CycleWorkspace advances to the next registered workspace.
func (e *Editor) CycleWorkspace() {
	if len(e.workspaces) == 0 {
		return
	}
	idx := 0
	for i, w := range e.workspaces {
		if w.Name() == e.activeWorkspace {
			idx = (i + 1) % len(e.workspaces)
			break
		}
	}
	e.SetActiveWorkspaceByName(e.workspaces[idx].Name())
}

// activeWorkspaceImpl returns the workspace currently named "active",
// or nil. Used by Editor.Update to route input to the focused
// workspace's Update() — only the focused workspace owns input so we
// don't race tools across multiple workspaces.
func (e *Editor) activeWorkspaceImpl() Workspace {
	for _, w := range e.workspaces {
		if w.Name() == e.activeWorkspace {
			return w
		}
	}
	return nil
}

// SetPanelRect records a panel's content-region rect for the current
// frame. Workspaces call this from inside their Render() so
// Editor.Draw can dispatch native widget content into the rect ImGui
// carved out. Stable identifier is the panel name used as the ImGui
// window title.
func (e *Editor) SetPanelRect(name string, r widgets.Rect) {
	e.ensurePanelRects()
	e.panelRects[name] = r
}

// CaptureCurrentWindowRect is the helper workspaces invoke inside
// their Render() after imgui.Begin succeeds. It captures the inner
// content region (where native widget code should paint) and records
// it under name. Called between imgui.Begin and imgui.End.
func (e *Editor) CaptureCurrentWindowRect(name string) {
	pos := imgui.CursorScreenPos()
	avail := imgui.ContentRegionAvail()
	e.SetPanelRect(name, widgets.Rect{
		X: int(pos.X), Y: int(pos.Y),
		W: int(avail.X), H: int(avail.Y),
	})
}

// installDefaultWorkspaces registers the Scene workspace. M4–M7
// workspaces register themselves via their package's RegisterWith;
// with DockSpace, every registered workspace renders simultaneously
// (tabbed into the central node by default), so the M3 stub
// placeholders that filled the tab strip are no longer needed.
func (e *Editor) installDefaultWorkspaces() {
	e.RegisterWorkspace(&SceneWorkspace{})
	e.activeWorkspace = "scene"
}

// SceneWorkspace is the central viewport workspace. U3 keeps the
// native canvas drawing path (PanelRect("Scene") → canvas.Draw); U5
// replaces it with an ImGui Image rendered from CreateTextureFromGame.
type SceneWorkspace struct{}

// Name is the stable workspace identifier.
func (s *SceneWorkspace) Name() string { return "scene" }

// DisplayName is the ImGui window title (also the dock-builder key).
func (s *SceneWorkspace) DisplayName() string { return "Scene" }

// Render builds the Scene workspace's ImGui window. U5 replaced the
// native canvas overlay with an imgui.Image rendered from the
// editor's sceneTexture, which the cimgui-go backend repaints each
// frame by calling sceneGame.Draw. A small toolbar (Select / Place /
// Delete / Paint) sits above the image; the rest is the scaled
// preview.
//
// SceneWorkspace.Render captures the image's screen rect — not the
// window's content rect — into PanelRect so SceneWorkspace.Update can
// map mouse coords onto the texture without confusing toolbar clicks
// for scene clicks.
func (s *SceneWorkspace) Render(e *Editor) {
	if e == nil || e.imgui == nil || !e.imgui.live {
		return
	}
	flags := imgui.WindowFlagsNoScrollbar
	if !imgui.BeginV(s.DisplayName(), nil, flags) {
		imgui.End()
		e.SetPanelRect(s.DisplayName(), widgets.Rect{})
		return
	}
	defer imgui.End()
	s.renderToolbar(e)
	s.renderPreview(e)
}

// renderToolbar emits the four tool radio buttons (Select, Place,
// Delete, Paint) at the top of the Scene window. Clicking one sets
// the editor's active tool.
//
// Idea #1 v1 extends the toolbar with a Paint-tool sub-mode picker
// (brush/bucket/rectangle radios) and a tile palette panel, both
// rendered inline when ToolPaint is active. Keeping the painter UI
// in the Scene workspace rather than the inspector matches the
// plan's "U6 deviation note" — the painter lives where the canvas
// dispatch can reach it.
func (s *SceneWorkspace) renderToolbar(e *Editor) {
	tools := []struct {
		label string
		tool  Tool
	}{
		{"Select", ToolSelect},
		{"Place", ToolPlace},
		{"Delete", ToolDelete},
		{"Paint", ToolPaint},
	}
	current := e.Tool()
	for i, t := range tools {
		if i > 0 {
			imgui.SameLine()
		}
		if imgui.RadioButtonBool(t.label, current == t.tool) {
			e.SetTool(t.tool)
		}
	}
	imgui.Separator()

	if current == ToolPaint {
		// Idea #2 v1 U6 supersedes idea #1's sub-mode picker location:
		// the Brush / Bucket / Rectangle radio buttons live in the
		// inspector's tilepainter widget (where the painter UI is
		// authored) rather than on the Scene-workspace toolbar. The
		// top-level Paint button stays here so designers can still
		// switch into the Paint tool without opening the inspector;
		// the palette panel below also stays for designers who prefer
		// the toolbar-style tile picker.
		s.renderPaintPalette(e)
		imgui.Separator()
	}
}

// renderPaintPalette draws the tile palette panel under the sub-mode
// picker. v1 uses the fallback numbered-button palette because the
// sheet-decoder pipeline that would render real tile thumbnails
// hasn't landed yet (see tile_palette.go header).
func (s *SceneWorkspace) renderPaintPalette(e *Editor) {
	palette := e.Palette()
	if palette == nil {
		return
	}
	scene := e.activeScene()
	var layer *pixelforge_project.TileAtlas
	if scene != nil && len(scene.TileAtlases) > 0 {
		layer = &scene.TileAtlases[0]
	}
	palette.Render(layer)
}

// renderPreview emits the imgui.Image that displays the scene
// texture, sized to fill the available content region while
// preserving the texture's aspect ratio. The image's screen rect is
// captured into PanelRect("Scene") for input mapping.
func (s *SceneWorkspace) renderPreview(e *Editor) {
	tex, texW, texH, ok := e.SceneTextureRef()
	if !ok {
		// No texture yet (test stub, pre-attach). Fall back to a
		// transparent placeholder rect so the panel still claims space
		// and PanelRect callers see the right area.
		e.CaptureCurrentWindowRect(s.DisplayName())
		return
	}

	avail := imgui.ContentRegionAvail()
	if avail.X <= 0 || avail.Y <= 0 {
		e.SetPanelRect(s.DisplayName(), widgets.Rect{})
		return
	}
	// Preserve the texture's aspect ratio inside the available rect.
	aspect := float32(texW) / float32(texH)
	imgW := avail.X
	imgH := imgW / aspect
	if imgH > avail.Y {
		imgH = avail.Y
		imgW = imgH * aspect
	}

	pos := imgui.CursorScreenPos()
	imgui.ImageWithBgV(
		tex,
		imgui.NewVec2(imgW, imgH),
		imgui.NewVec2(0, 0),
		imgui.NewVec2(1, 1),
		imgui.NewVec4(0, 0, 0, 0),
		imgui.NewVec4(1, 1, 1, 1),
	)
	e.SetPanelRect(s.DisplayName(), widgets.Rect{
		X: int(pos.X), Y: int(pos.Y),
		W: int(imgW), H: int(imgH),
	})
	// Record whether the user is pointing at the scene this frame —
	// SceneWorkspace.Update reads this to gate tool dispatch.
	e.sceneHovered = imgui.IsItemHovered() && imgui.IsWindowFocused()
}

// Update dispatches mouse input to the canvas only when the Scene
// image is both focused and hovered (U5). Mouse coords are remapped
// from window-pixel space into texture-pixel space, and the canvas
// is told its working area is the texture itself — that way the
// viewBox + entity-marker math operates in the exact same coordinate
// space sceneGame.Draw paints into. Without this remap, clicks at
// display coords would miss markers drawn at texture coords whenever
// the image is shown scaled.
//
// When the Scene image isn't focused the canvas Update is skipped so
// clicks on the toolbar (or in other docked workspaces) don't
// accidentally place/delete entities.
func (s *SceneWorkspace) Update(e *Editor) {
	if e == nil || e.canvas == nil {
		return
	}
	if !e.sceneHovered {
		return
	}
	tex, texW, texH, ok := e.SceneTextureRef()
	_ = tex
	if !ok {
		return
	}
	imageRect := e.PanelRect(s.DisplayName())
	if imageRect.W <= 0 || imageRect.H <= 0 {
		return
	}
	wx, wy := ebiten.CursorPosition()
	tx, ty, inside := mapMouseToSceneTexture(imageRect, texW, texH, wx, wy)
	if !inside {
		return
	}
	textureArea := widgets.Rect{X: 0, Y: 0, W: texW, H: texH}
	e.canvas.UpdateAt(textureArea, tx, ty, e)
}
