package pixelforge_dialogue

// NodeKind names the type of a parsed dialogue node. The runtime
// dispatches on this when advancing through a tree.
type NodeKind string

const (
	// NodeLine is a speaker-attributed line of text.
	NodeLine NodeKind = "line"

	// NodeChoice presents the player with branches.
	NodeChoice NodeKind = "choice"

	// NodeStageDirection invokes an engine-side action (move an
	// entity, pause for N ticks, etc.).
	NodeStageDirection NodeKind = "stage_direction"

	// NodeLabel is a Twine-style label declaration. The parser
	// uses these as branch targets; the runtime never "visits" a
	// label node — it walks past it.
	NodeLabel NodeKind = "label"
)

// Node is one entry in the parsed dialogue tree.
type Node struct {
	Kind NodeKind

	// Line fields (Kind == NodeLine).
	Speaker string
	Text    string

	// Choice fields (Kind == NodeChoice).
	Choices []Choice

	// StageDirection fields (Kind == NodeStageDirection).
	StageVerb string // walk_left / walk_right / walk_up / walk_down / pause
	StageArg  int    // numeric arg (tiles or ticks)

	// Label fields (Kind == NodeLabel).
	Label string
}

// Choice is one branch the player can pick on a NodeChoice node.
// Condition (when set) is a runtime predicate the renderer
// evaluates against the blackboard before exposing the choice; if
// the predicate returns false the choice is hidden.
type Choice struct {
	Text       string
	TargetLabel string
	Condition   string // empty = always available
}

// Tree is the parsed dialogue. Nodes are ordered; Labels maps the
// :: name declarations to their index in Nodes so jumps + choices
// can resolve targets.
type Tree struct {
	Nodes  []Node
	Labels map[string]int
}

// LabelIndex returns the node index for the named label, or -1
// when the label isn't declared. Used by the runtime when handling
// a choice's TargetLabel.
func (t *Tree) LabelIndex(name string) int {
	if t == nil {
		return -1
	}
	idx, ok := t.Labels[name]
	if !ok {
		return -1
	}
	return idx
}

// Len returns the number of nodes in the tree (for tests).
func (t *Tree) Len() int {
	if t == nil {
		return 0
	}
	return len(t.Nodes)
}
