// Package items is idea #6 v1 U9's studio Items workspace.
// Renders Project.Items as a table — ID / Name / Icon (via the
// new SpriteThumbnailWidget) / Description / Effect verb /
// Category. State mutations route through public methods.
package items

import (
	"sort"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// Workspace implements editor.Workspace.
type Workspace struct {
	editor *editor.Editor
}

func NewWorkspace(e *editor.Editor) *Workspace  { return &Workspace{editor: e} }
func (w *Workspace) Name() string                { return "items" }
func (w *Workspace) DisplayName() string         { return "Items" }
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace(e)
	e.RegisterWorkspace(w)
	return w
}

// Items returns a fresh slice of every ItemDefinition in the
// project (cap-safe — caller mutations don't bleed back).
func (w *Workspace) Items() []pixelforge_project.ItemDefinition {
	p := w.editor.Project()
	if p == nil {
		return nil
	}
	out := make([]pixelforge_project.ItemDefinition, len(p.Items))
	copy(out, p.Items)
	return out
}

// NewItem appends a fresh item with the supplied ID. Returns true
// on success; false when the ID is empty or already exists.
func (w *Workspace) NewItem(id string) bool {
	if id == "" {
		return false
	}
	p := w.editor.Project()
	if p == nil {
		return false
	}
	for _, it := range p.Items {
		if it.ID == id {
			return false
		}
	}
	p.Items = append(p.Items, pixelforge_project.ItemDefinition{ID: id, Name: id})
	w.editor.MarkDirty()
	return true
}

// DeleteItem removes the item with the supplied ID.
func (w *Workspace) DeleteItem(id string) bool {
	p := w.editor.Project()
	if p == nil {
		return false
	}
	for i, it := range p.Items {
		if it.ID == id {
			p.Items = append(p.Items[:i], p.Items[i+1:]...)
			w.editor.MarkDirty()
			return true
		}
	}
	return false
}

// SetItemField updates one field on the item identified by ID.
// field must be "name", "icon", "description", "effect_verb", or
// "category"; unknown fields return false. Returns true on change.
func (w *Workspace) SetItemField(id, field, value string) bool {
	p := w.editor.Project()
	if p == nil {
		return false
	}
	for i := range p.Items {
		if p.Items[i].ID != id {
			continue
		}
		it := &p.Items[i]
		var current *string
		switch field {
		case "name":
			current = &it.Name
		case "icon":
			current = &it.Icon
		case "description":
			current = &it.Description
		case "effect_verb":
			current = &it.EffectVerb
		case "category":
			current = &it.Category
		default:
			return false
		}
		if *current == value {
			return false
		}
		*current = value
		w.editor.MarkDirty()
		return true
	}
	return false
}

// SortedItemIDs returns every item ID sorted alphabetically.
func (w *Workspace) SortedItemIDs() []string {
	items := w.Items()
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}
	sort.Strings(out)
	return out
}

// AvailableSpriteNames returns every sprite name in the project,
// sorted. The Icon dropdown uses this.
func (w *Workspace) AvailableSpriteNames() []string {
	p := w.editor.Project()
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Sprites))
	for _, s := range p.Sprites {
		out = append(out, s.Name)
	}
	sort.Strings(out)
	return out
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
	imgui.TextUnformatted("Items")
	imgui.Separator()
	for _, it := range w.Items() {
		imgui.PushIDStr(it.ID)
		imgui.TextUnformatted(it.ID)
		imgui.SameLine()
		imgui.TextDisabled(it.Name)
		imgui.SameLine()
		imgui.TextDisabled(it.Category)
		imgui.PopID()
	}
	imgui.Separator()
	if imgui.Button("+ New Item") {
		w.NewItem(nextItemID(w))
	}
}

func nextItemID(w *Workspace) string {
	existing := map[string]bool{}
	for _, it := range w.Items() {
		existing[it.ID] = true
	}
	for i := 1; i < 1000; i++ {
		candidate := "item_" + iitoa(i)
		if !existing[candidate] {
			return candidate
		}
	}
	return "item"
}

func iitoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
