package scripting

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pguiwidgets "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
)

// LaneEditor is the canvas-resident Step lane sub-pane (U7).
//
// It owns the active graph pointer, scroll state, selected-step
// index, and the kind-picker dropdown. The lane renders one
// StepCard per StepNode in a horizontal Scrollable; clicking "+"
// appends a new node with default args.
type LaneEditor struct {
	project        *pixelforge_project.Project
	activeGraphIdx int
	selectedStep   int
	scrollX        int

	kindPicker     *pguiwidgets.Dropdown
	kindPickerOpen bool

	cards []*pguiwidgets.StepCard
}

func newLaneEditor() *LaneEditor {
	l := &LaneEditor{
		activeGraphIdx: -1,
		selectedStep:   -1,
	}
	l.kindPicker = pguiwidgets.NewDropdown(0, 0, 120, 16, 0, pguiwidgets.DropdownOptions{
		Options:  catalog.AllSteps(),
		OnSelect: nil, // wired in Update so the closure sees current state
	})
	return l
}

func (l *LaneEditor) bind(p *pixelforge_project.Project) {
	l.project = p
	if p != nil && len(p.Behaviors) > 0 {
		l.activeGraphIdx = 0
	} else {
		l.activeGraphIdx = -1
	}
	l.selectedStep = -1
	l.scrollX = 0
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

// AppendStep adds a new StepNode with the given Kind and default args
// to the active graph, then reloads the runtime engine.
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

// Update routes input to the lane pane.
func (l *LaneEditor) Update(e *editor.Editor, eng *runtime.Engine) {
	if l == nil || e == nil {
		return
	}
	if l.project != e.Project() {
		l.bind(e.Project())
	}
	// Delete key removes selected step.
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) && l.selectedStep >= 0 {
		l.DeleteSelectedStep(eng)
	}
}

// DrawCanvas paints the lane pane.
func (l *LaneEditor) DrawCanvas(rel widgets.Rect, e *editor.Editor, theme *editor.EditorTheme) {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Panel background.
	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	// Header — active graph name.
	header := "no graph"
	if g := l.ActiveGraph(); g != nil {
		header = g.Name
		if header == "" {
			header = fmt.Sprintf("graph %d", l.activeGraphIdx)
		}
	}
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(header, rel.X+4, rel.Y+2)

	g := l.ActiveGraph()
	if g == nil {
		pixelforge.SetColor(theme.TextDimSlot)
		pixelforge_cofont.Print("(no behaviour graphs — add one via the editor)", rel.X+4, rel.Y+16)
		return
	}

	// Horizontal strip of StepCards.
	stripY := rel.Y + 14
	cardW := 64
	cardH := 56
	gap := 4
	cx := rel.X + 4 - l.scrollX
	l.cards = l.cards[:0]
	for idx, node := range g.Steps {
		card := pguiwidgets.NewStepCard(cx, stripY, cardW, cardH, pguiwidgets.StepCardOptions{
			Kind:     node.Kind,
			Label:    formatStepLabel(node),
			Selected: idx == l.selectedStep,
		})
		// We draw inline below; storing references lets tests inspect
		// what was rendered.
		l.cards = append(l.cards, card)
		drawStepCard(card, theme)
		cx += cardW + gap
	}

	// "+" tile.
	pixelforge.SetColor(theme.AccentSlot)
	pixelforge.RectFill(cx, stripY, cx+cardW/2-1, stripY+cardH-1)
	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print("+", cx+cardW/4-2, stripY+cardH/2-3)

	// Inspector strip below — show args of the selected step.
	inspectY := stripY + cardH + 8
	if l.selectedStep >= 0 && l.selectedStep < len(g.Steps) {
		node := g.Steps[l.selectedStep]
		pixelforge.SetColor(theme.TextSlot)
		pixelforge_cofont.Print(fmt.Sprintf("inspect: %s", node.Kind), rel.X+4, inspectY)
		ly := inspectY + 10
		for k, v := range node.Args {
			pixelforge.SetColor(theme.TextDimSlot)
			pixelforge_cofont.Print(fmt.Sprintf("%s: %v", k, v), rel.X+8, ly)
			ly += 8
		}
	}

	// Use eng silently to keep the signature stable when we wire
	// reload semantics through the UI.
	_ = e
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

// drawStepCard paints a card using the workspace theme. The
// StepCard widget's draw routine lives in the widgets package; this
// is the workspace-side dispatch.
func drawStepCard(card *pguiwidgets.StepCard, theme *editor.EditorTheme) {
	if card == nil {
		return
	}
	bg := theme.PanelHeaderSlot
	if card.Selected {
		bg = theme.AccentSlot
	}
	if card.IsActive {
		bg = theme.AccentSlot
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)
	pixelforge.SetColor(bg)
	pixelforge.RectFill(card.X, card.Y, card.X+card.W-1, card.Y+card.H-1)

	pixelforge.SetColor(theme.TextSlot)
	pixelforge_cofont.Print(card.Kind, card.X+4, card.Y+4)
	pixelforge.SetColor(theme.TextDimSlot)
	pixelforge_cofont.Print(card.Label, card.X+4, card.Y+16)
	if card.IsActive {
		pixelforge.SetColor(theme.TextSlot)
		pixelforge.RectFill(card.X, card.Y+card.H-2, card.X+card.W-1, card.Y+card.H-1)
	}
}
