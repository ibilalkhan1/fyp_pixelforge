package scripting

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// EventSheetEditor is the canvas-resident event sheet sub-pane (U9).
//
// It owns the active graph pointer, the selected rule path (a
// breadcrumb-style index list), and the col/idx of the
// currently-edited condition or action.
type EventSheetEditor struct {
	project        *pixelforge_project.Project
	activeGraphIdx int

	// selectedRulePath addresses one rule in the tree. Each entry
	// is an index into the parent's Children list (or graph's
	// EventSheet at the top level).
	selectedRulePath []int

	// selectedCol is 0 for the Conditions column, 1 for Actions, -1
	// when only the rule itself (not a condition/action) is selected.
	selectedCol int
	selectedIdx int

	scrollY int
}

func newEventSheetEditor() *EventSheetEditor {
	return &EventSheetEditor{
		activeGraphIdx: -1,
		selectedCol:    -1,
		selectedIdx:    -1,
	}
}

func (s *EventSheetEditor) bind(p *pixelforge_project.Project) {
	s.project = p
	if p != nil && len(p.Behaviors) > 0 {
		s.activeGraphIdx = 0
	} else {
		s.activeGraphIdx = -1
	}
	s.selectedRulePath = nil
	s.selectedCol = -1
	s.selectedIdx = -1
}

// ActiveGraph returns the graph currently being edited, or nil.
func (s *EventSheetEditor) ActiveGraph() *pixelforge_project.BehaviorGraph {
	if s == nil || s.project == nil {
		return nil
	}
	if s.activeGraphIdx < 0 || s.activeGraphIdx >= len(s.project.Behaviors) {
		return nil
	}
	return &s.project.Behaviors[s.activeGraphIdx]
}

// AddRule appends an empty rule at the top level of the active graph.
func (s *EventSheetEditor) AddRule(eng *runtime.Engine) {
	g := s.ActiveGraph()
	if g == nil {
		return
	}
	g.EventSheet = append(g.EventSheet, pixelforge_project.EventSheetRule{})
	if eng != nil {
		eng.Reload(g.Name)
	}
}

// AddChildRule adds a nested rule under the rule at path.
func (s *EventSheetEditor) AddChildRule(path []int, eng *runtime.Engine) {
	g := s.ActiveGraph()
	if g == nil {
		return
	}
	rule := ruleAtPath(g, path)
	if rule == nil {
		return
	}
	rule.Children = append(rule.Children, pixelforge_project.EventSheetRule{})
	if eng != nil {
		eng.Reload(g.Name)
	}
}

// Update routes input.
func (s *EventSheetEditor) Update(e *editor.Editor, _ *runtime.Engine) {
	if s == nil || e == nil {
		return
	}
	if s.project != e.Project() {
		s.bind(e.Project())
	}
}

// DrawCanvas renders the event sheet pane.
func (s *EventSheetEditor) DrawCanvas(rel widgets.Rect, e *editor.Editor, theme *editor.EditorTheme) {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	g := s.ActiveGraph()
	if g == nil {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(no graph)", rel.X+4, rel.Y+4)
		return
	}

	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(fmt.Sprintf("%s — %d rules", g.Name, len(g.EventSheet)), rel.X+4, rel.Y+2)

	// Vertical list — render each rule with indent.
	y := rel.Y + 14
	for idx, rule := range g.EventSheet {
		y = s.drawRule(rel, rule, []int{idx}, 0, y, theme)
		if y > rel.Y+rel.H-10 {
			break
		}
	}

	// "+" tile at the bottom.
	pixelforge.SetColor(theme.AccentSlot)
	pixelforge.RectFill(rel.X+4, y, rel.X+16, y+10)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("+", rel.X+8, y+2)

	_ = e
}

func (s *EventSheetEditor) drawRule(rel widgets.Rect, rule pixelforge_project.EventSheetRule, path []int, indent, y int, theme *editor.EditorTheme) int {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	rowH := 12
	indentPx := indent * 16
	rect := widgets.Rect{X: rel.X + 4 + indentPx, Y: y, W: rel.W - 8 - indentPx, H: rowH}

	// Background highlight if this rule's path matches the selected.
	if pathsEqual(s.selectedRulePath, path) {
		pixelforge.SetColor(theme.AccentSlot)
		pixelforge.RectFill(rect.X, rect.Y, rect.X+rect.W-1, rect.Y+rect.H-1)
	}

	// Two columns: conditions on the left half, actions on the right.
	col := rect.W / 2
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(formatConditions(rule.Conditions), rect.X+2, rect.Y+2)
	pixelforge_cofont.Print(formatActions(rule.Actions), rect.X+col, rect.Y+2)

	y += rowH + 1
	for childIdx, child := range rule.Children {
		childPath := append([]int(nil), path...)
		childPath = append(childPath, childIdx)
		y = s.drawRule(rel, child, childPath, indent+1, y, theme)
	}
	return y
}

func formatConditions(cs []pixelforge_project.Condition) string {
	if len(cs) == 0 {
		return "(always)"
	}
	out := ""
	for i, c := range cs {
		if i > 0 {
			out += " ∧ "
		}
		out += c.Kind
	}
	return out
}

func formatActions(as []pixelforge_project.Action) string {
	if len(as) == 0 {
		return "—"
	}
	out := ""
	for i, a := range as {
		if i > 0 {
			out += ", "
		}
		out += a.Kind
	}
	return out
}

func ruleAtPath(g *pixelforge_project.BehaviorGraph, path []int) *pixelforge_project.EventSheetRule {
	if g == nil || len(path) == 0 {
		return nil
	}
	if path[0] < 0 || path[0] >= len(g.EventSheet) {
		return nil
	}
	r := &g.EventSheet[path[0]]
	for _, idx := range path[1:] {
		if idx < 0 || idx >= len(r.Children) {
			return nil
		}
		r = &r.Children[idx]
	}
	return r
}

func pathsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
