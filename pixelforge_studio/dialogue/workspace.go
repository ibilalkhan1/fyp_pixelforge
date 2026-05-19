// Package dialogue is idea #6 v1 U6's studio Dialogue workspace.
// The workspace renders a tree list (left) + multi-line script
// editor (right). On every edit, it re-parses the script via the
// engine's pixelforge_dialogue parser and surfaces parse errors
// in an editor-side error view.
//
// State mutations (NewTree, RenameTree, SetScript, DeleteTree)
// route through public methods so tests + the imgui-driven UI
// share one path. Render is a thin imgui pass.
package dialogue

import (
	"sort"
	"strings"
	"sync"

	"github.com/AllenDang/cimgui-go/imgui"

	pdialogue "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// Workspace implements editor.Workspace.
type Workspace struct {
	editor      *editor.Editor

	mu          sync.Mutex
	activeTree  string                                // currently-selected tree name
	parseErrors map[string][]pdialogue.ParseError     // per-tree parse error cache
}

// NewWorkspace constructs the Dialogue workspace bound to editor.
func NewWorkspace(e *editor.Editor) *Workspace {
	return &Workspace{
		editor:      e,
		parseErrors: map[string][]pdialogue.ParseError{},
	}
}

// Name implements editor.Workspace.
func (w *Workspace) Name() string { return "dialogue" }

// DisplayName implements editor.Workspace.
func (w *Workspace) DisplayName() string { return "Dialogue" }

// RegisterWith installs the workspace on editor. Mirrors
// palette.RegisterWith.
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace(e)
	e.RegisterWorkspace(w)
	return w
}

// TreeNames returns every dialogue tree name in the project, sorted.
func (w *Workspace) TreeNames() []string {
	p := w.editor.Project()
	if p == nil || p.Dialogues == nil {
		return nil
	}
	out := make([]string, 0, len(p.Dialogues))
	for name := range p.Dialogues {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ActiveTree returns the currently-selected tree name.
func (w *Workspace) ActiveTree() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.activeTree
}

// SetActiveTree picks the tree the script editor renders.
func (w *Workspace) SetActiveTree(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.activeTree = name
}

// NewTree appends a fresh dialogue tree under the supplied name.
// Returns true on success; false when name is empty or already
// exists.
func (w *Workspace) NewTree(name string) bool {
	if name == "" {
		return false
	}
	p := w.editor.Project()
	if p == nil {
		return false
	}
	if p.Dialogues == nil {
		p.Dialogues = map[string]pixelforge_project.DialogueScript{}
	}
	if _, exists := p.Dialogues[name]; exists {
		return false
	}
	p.Dialogues[name] = pixelforge_project.DialogueScript{Name: name}
	w.editor.MarkDirty()
	w.SetActiveTree(name)
	return true
}

// DeleteTree removes the named tree.
func (w *Workspace) DeleteTree(name string) bool {
	p := w.editor.Project()
	if p == nil || p.Dialogues == nil {
		return false
	}
	if _, exists := p.Dialogues[name]; !exists {
		return false
	}
	delete(p.Dialogues, name)
	w.mu.Lock()
	delete(w.parseErrors, name)
	if w.activeTree == name {
		w.activeTree = ""
	}
	w.mu.Unlock()
	w.editor.MarkDirty()
	return true
}

// SetScript writes the script for the active tree and re-parses.
// Returns the resulting parse errors (empty when script is valid).
func (w *Workspace) SetScript(name, script string) []pdialogue.ParseError {
	p := w.editor.Project()
	if p == nil || p.Dialogues == nil {
		return nil
	}
	entry, ok := p.Dialogues[name]
	if !ok {
		return nil
	}
	if entry.Script == script {
		// no-op
		_, errs := pdialogue.Parse(script)
		w.cacheErrors(name, errs)
		return errs
	}
	entry.Script = script
	p.Dialogues[name] = entry
	w.editor.MarkDirty()
	_, errs := pdialogue.Parse(script)
	w.cacheErrors(name, errs)
	return errs
}

// ParseErrors returns the last cached parse errors for the named
// tree. Empty when the script parsed cleanly (or hasn't been
// edited yet).
func (w *Workspace) ParseErrors(name string) []pdialogue.ParseError {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]pdialogue.ParseError(nil), w.parseErrors[name]...)
}

func (w *Workspace) cacheErrors(name string, errs []pdialogue.ParseError) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(errs) == 0 {
		delete(w.parseErrors, name)
		return
	}
	w.parseErrors[name] = append([]pdialogue.ParseError(nil), errs...)
}

// Render emits the workspace inside the current ImGui frame. Thin
// pass — the testable logic lives in the state methods above.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	if !imgui.Begin(w.DisplayName()) {
		imgui.End()
		return
	}
	defer imgui.End()
	avail := imgui.ContentRegionAvail()
	if imgui.BeginChildStrV("##dialogue_list", imgui.Vec2{X: avail.X * 0.3, Y: 0}, 0, 0) {
		w.renderTreeList()
	}
	imgui.EndChild()
	imgui.SameLine()
	if imgui.BeginChildStrV("##dialogue_editor", imgui.Vec2{}, 0, 0) {
		w.renderEditor()
	}
	imgui.EndChild()
}

func (w *Workspace) renderTreeList() {
	imgui.TextUnformatted("Dialogues")
	imgui.Separator()
	for _, name := range w.TreeNames() {
		active := w.ActiveTree() == name
		if imgui.SelectableBoolPtr(name, &active) {
			w.SetActiveTree(name)
		}
	}
	imgui.Separator()
	if imgui.Button("+ New") {
		w.NewTree(uniqueTreeName(w))
	}
}

func uniqueTreeName(w *Workspace) string {
	base := "dialogue"
	existing := map[string]bool{}
	for _, n := range w.TreeNames() {
		existing[n] = true
	}
	for i := 1; i < 1000; i++ {
		candidate := base
		if i > 1 {
			candidate = candidateName(base, i)
		}
		if !existing[candidate] {
			return candidate
		}
	}
	return base
}

func candidateName(base string, idx int) string {
	return base + "_" + itoa(idx)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if negative {
		return "-" + digits
	}
	return digits
}

func (w *Workspace) renderEditor() {
	active := w.ActiveTree()
	if active == "" {
		imgui.TextDisabled("(select a dialogue tree on the left, or create one)")
		return
	}
	p := w.editor.Project()
	if p == nil || p.Dialogues == nil {
		return
	}
	entry := p.Dialogues[active]
	script := entry.Script
	if imgui.InputTextMultiline("##script", &script, imgui.Vec2{}, 0, nil) {
		w.SetScript(active, script)
	}
	imgui.Separator()
	errs := w.ParseErrors(active)
	if len(errs) == 0 {
		imgui.TextDisabled("OK — no parse errors")
		return
	}
	imgui.TextUnformatted("Parse errors:")
	for _, e := range errs {
		imgui.TextColored(imgui.Vec4{X: 1.0, Y: 0.5, Z: 0.5, W: 1.0}, e.Error())
	}
}

// suppress unused import for the alias path
var _ = strings.ToLower
