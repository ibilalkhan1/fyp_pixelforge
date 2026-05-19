// Package menus is idea #6 v1 U8's studio Menus workspace.
// Walks Project.Menus and renders one row per MenuConfig with a
// template picker + parameter editor on the right.
//
// State mutations route through public methods; Render is a thin
// imgui pass.
package menus

import (
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"

	pimenus "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_menus"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// Workspace implements editor.Workspace.
type Workspace struct {
	editor     *editor.Editor
	activeMenu string
}

// NewWorkspace constructs the Menus workspace bound to editor.
func NewWorkspace(e *editor.Editor) *Workspace { return &Workspace{editor: e} }

func (w *Workspace) Name() string        { return "menus" }
func (w *Workspace) DisplayName() string { return "Menus" }

// RegisterWith installs the workspace on editor.
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace(e)
	e.RegisterWorkspace(w)
	return w
}

// MenuNames returns the sorted list of menus in the project.
func (w *Workspace) MenuNames() []string {
	p := w.editor.Project()
	if p == nil || p.Menus == nil {
		return nil
	}
	out := make([]string, 0, len(p.Menus))
	for n := range p.Menus {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ActiveMenu returns the currently-selected menu name.
func (w *Workspace) ActiveMenu() string { return w.activeMenu }

// SetActiveMenu picks the menu the parameter editor renders.
func (w *Workspace) SetActiveMenu(name string) { w.activeMenu = name }

// NewMenu appends a fresh menu under name using the supplied
// template. Returns true on success; false when name is empty or
// already exists, or when the template is unknown.
func (w *Workspace) NewMenu(name, template string) bool {
	if name == "" || template == "" {
		return false
	}
	if _, ok := pimenus.LookupTemplate(template); !ok {
		return false
	}
	p := w.editor.Project()
	if p == nil {
		return false
	}
	if p.Menus == nil {
		p.Menus = map[string]pixelforge_project.MenuConfig{}
	}
	if _, exists := p.Menus[name]; exists {
		return false
	}
	p.Menus[name] = pixelforge_project.MenuConfig{
		Name:       name,
		Template:   template,
		Parameters: map[string]any{},
	}
	w.editor.MarkDirty()
	w.SetActiveMenu(name)
	return true
}

// DeleteMenu removes the named menu.
func (w *Workspace) DeleteMenu(name string) bool {
	p := w.editor.Project()
	if p == nil || p.Menus == nil {
		return false
	}
	if _, exists := p.Menus[name]; !exists {
		return false
	}
	delete(p.Menus, name)
	if w.activeMenu == name {
		w.activeMenu = ""
	}
	w.editor.MarkDirty()
	return true
}

// SetMenuTemplate swaps the template the named menu uses. Unknown
// templates reject.
func (w *Workspace) SetMenuTemplate(name, template string) bool {
	p := w.editor.Project()
	if p == nil || p.Menus == nil {
		return false
	}
	if _, ok := pimenus.LookupTemplate(template); !ok {
		return false
	}
	entry, exists := p.Menus[name]
	if !exists {
		return false
	}
	if entry.Template == template {
		return false
	}
	entry.Template = template
	p.Menus[name] = entry
	w.editor.MarkDirty()
	return true
}

// SetMenuParameter writes a single parameter on the named menu.
// Returns true on change.
func (w *Workspace) SetMenuParameter(name, key string, value any) bool {
	p := w.editor.Project()
	if p == nil || p.Menus == nil {
		return false
	}
	entry, exists := p.Menus[name]
	if !exists {
		return false
	}
	if entry.Parameters == nil {
		entry.Parameters = map[string]any{}
	}
	if existing, ok := entry.Parameters[key]; ok && existing == value {
		return false
	}
	entry.Parameters[key] = value
	p.Menus[name] = entry
	w.editor.MarkDirty()
	return true
}

// AvailableTemplates returns the registered template names.
func (w *Workspace) AvailableTemplates() []string {
	return pimenus.AllTemplates()
}

// Render emits the workspace UI.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	if !imgui.Begin(w.DisplayName()) {
		imgui.End()
		return
	}
	defer imgui.End()
	imgui.TextUnformatted("Menus")
	imgui.Separator()
	for _, name := range w.MenuNames() {
		active := w.ActiveMenu() == name
		if imgui.SelectableBoolPtr(name, &active) {
			w.SetActiveMenu(name)
		}
	}
}
