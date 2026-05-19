package assetlibrary

import (
	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// editorAdapter wraps *editor.Editor so the workspace's imgui-free
// state surface (workspace.go) calls back into the editor through
// the small EditorBinding interface. Keeps workspace.go testable
// without importing the editor package.
type editorAdapter struct{ e *editor.Editor }

func (a *editorAdapter) Project() *pixelforge_project.Project { return a.e.Project() }
func (a *editorAdapter) CurrentProjectPath() string           { return a.e.CurrentProjectPath() }
func (a *editorAdapter) MarkDirty()                           { a.e.MarkDirty() }

// RegisterWith installs a Library workspace on editor bound to
// lib. Returns the workspace so callers can hold the handle for
// further wiring (per-pack download progress, Custom-tab refresh).
//
// Plan-008 U12: the workspace appears under the View menu as
// "Library" with Ctrl+9 (continuing the Ctrl+5/6/7/8 series).
// Keymap registration lives in editor/file_menu.go; this helper
// only owns the workspace registration.
func RegisterWith(e *editor.Editor, lib *Library) *Workspace {
	w := NewWorkspace(&editorAdapter{e: e}, lib)
	e.RegisterWorkspace(w)
	return w
}

// Render satisfies editor.Workspace. The UI is intentionally
// minimal for U12 — tab strip + asset rows + Add buttons —
// because the heavy lifting (state mutations, asset copy, project
// updates) all lives in the imgui-free methods on workspace.go.
// Designers see what packs are installed + can click Add to
// project; richer affordances (audition, thumbnails, drag-into-
// scene) layer on top in follow-on plans.
func (w *Workspace) Render(_ *editor.Editor) {
	if w == nil {
		return
	}
	if !imgui.Begin(w.DisplayName()) {
		imgui.End()
		return
	}
	defer imgui.End()

	for _, tab := range CanonicalGameTabs {
		selected := w.ActiveTab() == tab
		label := TabDisplayName(tab)
		if selected {
			label = "[" + label + "]"
		}
		if imgui.Button(label) {
			w.SetActiveTab(tab)
		}
		imgui.SameLine()
	}
	imgui.NewLine()
	imgui.Separator()

	if w.ActiveTab() == "custom" {
		assets := w.CustomAssets()
		if len(assets) == 0 {
			imgui.TextUnformatted("(no user-library assets — drop files into the user-library/ folder or onto this window)")
			return
		}
		for _, a := range assets {
			imgui.TextUnformatted(a.Path + " [" + LicenseBadge(a.License) + "]")
		}
		return
	}

	packs := w.PacksForActiveTab()
	if len(packs) == 0 {
		imgui.TextUnformatted("(no packs installed — first-launch download may still be running)")
		return
	}
	for _, pack := range packs {
		imgui.SeparatorText(pack.Title)
		for _, a := range pack.Assets {
			line := a.Path + "  [" + LicenseBadge(a.License) + "]"
			if a.Author != "" {
				line += "  by " + a.Author
			}
			imgui.TextUnformatted(line)
			imgui.SameLine()
			if imgui.Button("Add##" + pack.ID + "/" + a.Path) {
				_ = w.AddToProject(pack.ID, a.Path)
			}
		}
	}
}
