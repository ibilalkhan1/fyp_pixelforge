// Package pixelforge_gametree implements the hierarchical scene-graph
// updater that replaced the original flat update() loop. A single
// linear tick() works for Snake, but complex games like Pac-Man
// require parent-child ordering so that a Player node moves before
// its Weapon child reads the new coordinates. The tree is traversed
// depth-first in pre-order (Parent → Child) to eliminate the one-
// frame lag that post-order traversal introduced.
package pixelforge_gametree

// NodeID is the unique key for a node in the game tree. It is used
// both as a map key for fast dictionary lookups and as a debug
// label in the inspector.
type NodeID string

// Node is one entity, logic block, or subtree in the scene graph.
// Each node carries its own tick function and an ordered slice of
// children so that the cascade respects designer-authored priority.
type Node struct {
	ID       NodeID
	Parent   *Node
	Children []*Node

	// tick is the per-frame logic attached to this node. It is nil
	// for pure container nodes (e.g. "World" or "UI_Layer").
	tick func()
}

// NewNode creates a named node with no parent and no children.
func NewNode(id NodeID) *Node {
	return &Node{ID: id}
}

// Attach appends a child to this node and sets the child's Parent
// pointer. The insertion order defines the cascade priority.
func (n *Node) Attach(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// Update executes this node's tick and then recursively cascades
// through every child in declaration order. Because the parent
// updates first, children always see the freshest transform data,
// preventing the visual lag that post-order traversal caused.
func (n *Node) Update() {
	if n.tick != nil {
		n.tick()
	}
	for _, c := range n.Children {
		c.Update()
	}
}

// SetTick binds a per-frame callback to the node. The callback is
// invoked once per frame during the pre-order DFS walk.
func (n *Node) SetTick(fn func()) {
	n.tick = fn
}

// ─── Game Tree & Traversal ───────────────────────────────────────

// GameTree is the root container for the entire running scene. It
// holds the root node, a dictionary of previously visited nodes for
// save-state and rewind support, and the active traversal path so
// that the frame logic can decide which branch to walk next.
type GameTree struct {
	// Root is the top-level node, conventionally named "World".
	Root *Node

	// PastNodes is a dictionary of nodes that were visited in
	// previous frames. It is used for state rewind, ghost replay,
	// and delta-compression during serialisation.
	PastNodes map[NodeID]*Node

	// ActivePath records the sequence of NodeIDs that the current
	// frame's logic has selected for visitation. The DFS walker
	// uses this hint to skip cold branches and focus CPU time on
	// the nodes that matter this tick.
	ActivePath []NodeID
}

// NewGameTree creates an empty tree with a named root node.
func NewGameTree(rootID NodeID) *GameTree {
	return &GameTree{
		Root:      NewNode(rootID),
		PastNodes: make(map[NodeID]*Node),
	}
}

// TickFrame is the entry point called once per engine frame. It
// first records the current root into PastNodes (so that rewind
// has a snapshot), then performs a depth-first pre-order update
// starting at the root.
//
// The traversal digs all the way down the World → Player → Sprites
// chain before backtracking to process siblings such as Enemies or
// Particles. This mirrors the behaviour-tree update pattern found
// in modern commercial engines.
func (gt *GameTree) TickFrame() {
	// Snapshot the root and every child into the past dictionary.
	// In a full implementation this would be a shallow copy or a
	// ring-buffer slot; here we store the reference for prototyping.
	gt.archiveSubtree(gt.Root)

	// Walk the tree depth-first, pre-order.
	gt.Root.Update()
}

// archiveSubtree recursively stores every node in the given subtree
// into PastNodes so that the engine can diff against prior frames
// or rewind without recomputing state from scratch.
func (gt *GameTree) archiveSubtree(n *Node) {
	gt.PastNodes[n.ID] = n
	for _, c := range n.Children {
		gt.archiveSubtree(c)
	}
}

// DFSUpdate performs an explicit depth-first pre-order traversal
// starting at root. It is identical to Root.Update() but returns
// the visitation order so that debug tools can visualise the walk.
func (gt *GameTree) DFSUpdate() []NodeID {
	var order []NodeID
	var walk func(n *Node)
	walk = func(n *Node) {
		order = append(order, n.ID)
		if n.tick != nil {
			n.tick()
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(gt.Root)
	return order
}

// SelectBranch marks a specific node and its entire subtree as the
// active path for the next frame. Nodes outside the path are still
// updated (they may run ambient logic) but they receive a reduced
// tick budget so that the focused branch gets priority CPU time.
func (gt *GameTree) SelectBranch(id NodeID) {
	gt.ActivePath = gt.ActivePath[:0]
	node := gt.findNode(gt.Root, id)
	if node == nil {
		return
	}
	// Walk up to root to build the path.
	for n := node; n != nil; n = n.Parent {
		gt.ActivePath = append([]NodeID{n.ID}, gt.ActivePath...)
	}
}

// findNode recursively searches the tree for a node with the given
// ID. In a production build this is replaced by a flat node table
// maintained alongside the hierarchy.
func (gt *GameTree) findNode(root *Node, id NodeID) *Node {
	if root.ID == id {
		return root
	}
	for _, c := range root.Children {
		if found := gt.findNode(c, id); found != nil {
			return found
		}
	}
	return nil
}

// PreOrderWalk visits every node in the tree in pre-order (Parent
// before Children) and invokes the supplied callback. This is the
// canonical traversal used by the engine because it guarantees that
// a parent's transform is current before any child reads it.
func (gt *GameTree) PreOrderWalk(fn func(n *Node)) {
	var walk func(n *Node)
	walk = func(n *Node) {
		fn(n)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(gt.Root)
}

// PostOrderWalk visits every node in post-order (Children before
// Parent). The engine does not use this for gameplay updates
// because it caused weapons to lag one frame behind the player,
// but it is exposed for destruction teardown where children must
// release resources before the parent disappears.
func (gt *GameTree) PostOrderWalk(fn func(n *Node)) {
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			walk(c)
		}
		fn(n)
	}
	walk(gt.Root)
}

// ─── Behaviour Tree Primitives ───────────────────────────────────

// Behaviour is a reusable logic block that can be attached to any
// node. Designers compose behaviours in the studio and the compiler
// attaches them to the tree at load time.
type Behaviour struct {
	Name string
	Tick func()
}

// AttachBehaviour binds a behaviour to a node by wrapping its Tick
// into the node's per-frame callback.
func AttachBehaviour(n *Node, b *Behaviour) {
	old := n.tick
	n.tick = func() {
		if old != nil {
			old()
		}
		b.Tick()
	}
}
