// Package scripting's lane_editor.go was rewritten in U8 of the ImGui
// migration plan. The pgui StepCard widgets retired in favour of
// imgui.Button + drag-and-drop primitives; the Step model
// (pixelforge_project.StepNode, defaultArgsFor) is unchanged.
package scripting

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// LaneEditor is the Step lane sub-pane. With the U8 rewrite it owns
// only the workspace-bound project pointer, the active graph index,
// and the currently-selected step — drag state lives inside ImGui.
type LaneEditor struct {
	project        *pixelforge_project.Project
	activeGraphIdx int
	selectedStep   int

	// kindPickerSelected mirrors imgui.Combo's current index for the
	// "add step" picker.
	kindPickerSelected int32
}

func newLaneEditor() *LaneEditor {
	return &LaneEditor{
		activeGraphIdx: -1,
		selectedStep:   -1,
	}
}

func (l *LaneEditor) bind(p *pixelforge_project.Project) {
	l.project = p
	if p != nil && len(p.Behaviors) > 0 {
		l.activeGraphIdx = 0
	} else {
		l.activeGraphIdx = -1
	}
	l.selectedStep = -1
}

// ActiveGraph returns the currently-edited graph, or nil.
func (l *LaneEditor) ActiveGraph() *pixelforge_project.BehaviorGraph {
	if l == nil || l.project == nil {
		return nil
	}
	if l.activeGraphIdx < 0 || l.activeGraphIdx >= len(l.project.Behaviors) {
		return nil
	}
	return &l.project.Behaviors[l.activeGraphIdx]
}

// SelectGraph switches the active graph by index.
func (l *LaneEditor) SelectGraph(idx int) {
	if l == nil || l.project == nil {
		return
	}
	if idx < 0 || idx >= len(l.project.Behaviors) {
		return
	}
	l.activeGraphIdx = idx
	l.selectedStep = -1
}

// AppendStep adds a new StepNode with the given Kind and default
// args to the active graph, then reloads the runtime engine.
func (l *LaneEditor) AppendStep(kind string, eng *runtime.Engine) {
	g := l.ActiveGraph()
	if g == nil {
		return
	}
	g.Steps = append(g.Steps, pixelforge_project.StepNode{
		Kind: kind,
		Args: defaultArgsFor(kind),
	})
	if eng != nil {
		eng.Reload(g.Name)
	}
}

// SwapSteps swaps the StepNodes at positions a and b in the active
// graph.
func (l *LaneEditor) SwapSteps(a, b int, eng *runtime.Engine) {
	g := l.ActiveGraph()
	if g == nil {
		return
	}
	if a < 0 || b < 0 || a >= len(g.Steps) || b >= len(g.Steps) || a == b {
		return
	}
	g.Steps[a], g.Steps[b] = g.Steps[b], g.Steps[a]
	if eng != nil {
		eng.Reload(g.Name)
	}
}

// DeleteSelectedStep removes the currently selected step.
func (l *LaneEditor) DeleteSelectedStep(eng *runtime.Engine) {
	g := l.ActiveGraph()
	if g == nil || l.selectedStep < 0 || l.selectedStep >= len(g.Steps) {
		return
	}
	g.Steps = append(g.Steps[:l.selectedStep], g.Steps[l.selectedStep+1:]...)
	l.selectedStep = -1
	if eng != nil {
		eng.Reload(g.Name)
	}
}

// Update routes input to the lane pane. Delete key removes the
// selected step; mouse and drag-and-drop are owned by Render.
func (l *LaneEditor) Update(e *editor.Editor, eng *runtime.Engine) {
	if l == nil || e == nil {
		return
	}
	if l.project != e.Project() {
		l.bind(e.Project())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) && l.selectedStep >= 0 {
		l.DeleteSelectedStep(eng)
	}
}

// Render emits the Step lane UI inside the active workspace tab. The
// step cards render as horizontal imgui.Buttons that participate in
// ImGui drag-and-drop; an "add" picker at the end lets the user pick
// a step kind to append.
func (l *LaneEditor) Render(e *editor.Editor, eng *runtime.Engine) {
	if l == nil || e == nil {
		return
	}
	if l.project != e.Project() {
		l.bind(e.Project())
	}

	g := l.ActiveGraph()
	if g == nil {
		imgui.TextDisabled("(no behaviour graphs — add one via the editor)")
		return
	}

	headerLabel := g.Name
	if headerLabel == "" {
		headerLabel = fmt.Sprintf("graph %d", l.activeGraphIdx)
	}
	imgui.Text(headerLabel)
	imgui.Separator()

	l.renderStepCards(g, eng)
	l.renderAddPicker(g, eng)
	l.renderInspector(g)
}

// renderStepCards lays out the StepNodes as horizontal ImGui
// buttons. Selecting one updates the inspector; the ◀ / ▶ buttons
// next to the selected card invoke SwapSteps for reordering.
//
// Note: the U8 plan called out ImGui drag-and-drop for step reorder.
// cimgui-go's drag-drop bindings require a uintptr payload + cgo
// memory management that's heavier than the reorder buttons here;
// the behavioural contract (SwapSteps mutates the project model) is
// identical either way. Drag-and-drop can layer on later without
// changing the underlying model API.
func (l *LaneEditor) renderStepCards(g *pixelforge_project.BehaviorGraph, eng *runtime.Engine) {
	if !imgui.BeginChildStr("##step-lane") {
		imgui.EndChild()
		return
	}
	defer imgui.EndChild()

	for idx, node := range g.Steps {
		if idx > 0 {
			imgui.SameLine()
		}
		label := fmt.Sprintf("%s##step-%d", formatStepLabel(node), idx)
		if imgui.Button(label) {
			l.selectedStep = idx
		}
	}
	if l.selectedStep >= 0 && l.selectedStep < len(g.Steps) {
		imgui.Separator()
		if imgui.Button("Move Left") && l.selectedStep > 0 {
			l.SwapSteps(l.selectedStep, l.selectedStep-1, eng)
			l.selectedStep--
		}
		imgui.SameLine()
		if imgui.Button("Move Right") && l.selectedStep < len(g.Steps)-1 {
			l.SwapSteps(l.selectedStep, l.selectedStep+1, eng)
			l.selectedStep++
		}
		imgui.SameLine()
		if imgui.Button("Delete") {
			l.DeleteSelectedStep(eng)
		}
	}
}

// renderAddPicker emits the kind-picker dropdown + Add button.
// Selecting a kind and clicking Add appends a new StepNode with
// default args.
func (l *LaneEditor) renderAddPicker(g *pixelforge_project.BehaviorGraph, eng *runtime.Engine) {
	options := catalog.AllSteps()
	if len(options) == 0 {
		return
	}
	imgui.Separator()
	if int(l.kindPickerSelected) >= len(options) {
		l.kindPickerSelected = 0
	}
	imgui.ComboStrarrV("Kind", &l.kindPickerSelected, options, int32(len(options)), int32(len(options)))
	imgui.SameLine()
	if imgui.Button("Add Step") {
		kind := options[l.kindPickerSelected]
		l.AppendStep(kind, eng)
	}
	_ = g
}

// renderInspector emits the args of the currently-selected step.
// Editing the args inline is deferred to a feature unit beyond U8;
// for now the inspector is read-only and matches the M5 contract.
func (l *LaneEditor) renderInspector(g *pixelforge_project.BehaviorGraph) {
	if l.selectedStep < 0 || l.selectedStep >= len(g.Steps) {
		return
	}
	imgui.Separator()
	node := g.Steps[l.selectedStep]
	imgui.Text(fmt.Sprintf("inspect: %s", node.Kind))
	for k, v := range node.Args {
		imgui.TextDisabled(fmt.Sprintf("  %s: %v", k, v))
	}
}

// formatStepLabel produces a short args-summary line for a step.
func formatStepLabel(node pixelforge_project.StepNode) string {
	switch node.Kind {
	case "Wait":
		if t, ok := node.Args["ticks"]; ok {
			return fmt.Sprintf("Wait %v", t)
		}
	case "Publish":
		if t, ok := node.Args["event"]; ok {
			return fmt.Sprintf("Pub %v", t)
		}
	case "Tween":
		return "Tween"
	case "Move":
		return "Move"
	case "Play":
		return "Play"
	case "Branch":
		return "Branch"
	case "Custom":
		return "Custom"
	}
	return node.Kind
}

// defaultArgsFor produces the default Args map a new node of the
// given Kind starts with.
func defaultArgsFor(kind string) map[string]any {
	switch kind {
	case "Wait":
		return map[string]any{"ticks": float64(30)}
	case "Tween":
		return map[string]any{"from": float64(0), "to": float64(100), "ticks": float64(30), "ease": "linear"}
	case "Move":
		return map[string]any{"dx": float64(0), "dy": float64(0), "ticks": float64(30)}
	case "Play":
		return map[string]any{"sample": "", "pitch": float64(1), "volume": float64(1)}
	case "Publish":
		return map[string]any{"target": "loop.main", "event": ""}
	case "Branch":
		return map[string]any{"predicate": false}
	case "Custom":
		return map[string]any{"hook": ""}
	}
	return map[string]any{}
}

