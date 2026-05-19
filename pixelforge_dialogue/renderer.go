package pixelforge_dialogue

// TextBoxRenderer is the runtime engine for a dialogue. It tracks
// the current node, exposes the line to render, and advances on
// Next() (the engine wires Next to the "use"/"A" input intent).
// Choices fork; selecting a choice jumps to its TargetLabel.
//
// The renderer is intentionally state-only — no imgui, no ebiten
// — so tests exercise the state machine without a live frame. The
// runtime composition layer (Capsule + studio preview) maps the
// public state (CurrentLine / CurrentChoices / IsDone) into draw
// calls.
type TextBoxRenderer struct {
	tree         *Tree
	cursor       int
	lookup       func(key string) (any, bool)
	conditionEval func(expr string) bool
}

// NewTextBoxRenderer constructs a renderer over tree. lookup
// resolves {state.key} interpolations; conditionEval evaluates
// choice "if cond" predicates. Either may be nil — interpolation
// then leaves placeholders intact and all conditions evaluate as
// true.
func NewTextBoxRenderer(tree *Tree, lookup func(key string) (any, bool), conditionEval func(expr string) bool) *TextBoxRenderer {
	r := &TextBoxRenderer{
		tree:          tree,
		lookup:        lookup,
		conditionEval: conditionEval,
	}
	r.skipNonVisible()
	return r
}

// IsDone reports whether the renderer has run off the end of the
// tree (no more lines to advance to).
func (r *TextBoxRenderer) IsDone() bool {
	return r == nil || r.tree == nil || r.cursor >= len(r.tree.Nodes)
}

// CurrentNode returns the active node, or nil when IsDone.
func (r *TextBoxRenderer) CurrentNode() *Node {
	if r.IsDone() {
		return nil
	}
	return &r.tree.Nodes[r.cursor]
}

// CurrentLine returns the rendered speaker + text for the active
// line. Interpolation is applied to text. Returns ("", "") when
// the active node is not a line (e.g. it's a choice).
func (r *TextBoxRenderer) CurrentLine() (string, string) {
	n := r.CurrentNode()
	if n == nil || n.Kind != NodeLine {
		return "", ""
	}
	return n.Speaker, Interpolate(n.Text, r.lookup)
}

// CurrentChoices returns the visible choices for the active node.
// Empty when the active node isn't a choice (or every choice was
// filtered out by its condition).
func (r *TextBoxRenderer) CurrentChoices() []Choice {
	n := r.CurrentNode()
	if n == nil || n.Kind != NodeChoice {
		return nil
	}
	var visible []Choice
	for _, c := range n.Choices {
		if c.Condition == "" || r.evalCondition(c.Condition) {
			visible = append(visible, c)
		}
	}
	return visible
}

// Advance moves to the next non-label, non-stage-direction node.
// Stage-direction nodes still surface via CurrentNode (so the
// runtime can dispatch them), but Advance treats them as
// "transient" — the next Advance walks past.
//
// Choice nodes do NOT advance via Advance — call PickChoice
// instead. Returns IsDone() after the advance.
func (r *TextBoxRenderer) Advance() bool {
	if r.IsDone() {
		return false
	}
	r.cursor++
	r.skipNonVisible()
	return !r.IsDone()
}

// PickChoice jumps to the choice's TargetLabel. Returns true on
// successful jump, false when the label doesn't resolve or the
// active node isn't a choice / the index is out of range.
func (r *TextBoxRenderer) PickChoice(idx int) bool {
	choices := r.CurrentChoices()
	if idx < 0 || idx >= len(choices) {
		return false
	}
	target := r.tree.LabelIndex(choices[idx].TargetLabel)
	if target < 0 {
		return false
	}
	r.cursor = target
	r.skipNonVisible()
	return true
}

// skipNonVisible advances past label nodes (they're not rendered
// — they're targets). Stage directions are visible (the engine
// must dispatch them) so we stop on those. Nil-tree-safe.
func (r *TextBoxRenderer) skipNonVisible() {
	if r == nil || r.tree == nil {
		return
	}
	for r.cursor < len(r.tree.Nodes) && r.tree.Nodes[r.cursor].Kind == NodeLabel {
		r.cursor++
	}
}

// evalCondition returns the conditionEval result, or true when no
// evaluator was supplied. Per the plan's "missing evaluator =
// permissive" stance: tests + early prototyping run without an
// evaluator and see all branches.
func (r *TextBoxRenderer) evalCondition(expr string) bool {
	if r.conditionEval == nil {
		return true
	}
	return r.conditionEval(expr)
}

// Reset returns the cursor to the start of the tree. Used by the
// runtime when a fresh open_dialogue verb fires on the same tree.
func (r *TextBoxRenderer) Reset() {
	if r == nil {
		return
	}
	r.cursor = 0
	r.skipNonVisible()
}
