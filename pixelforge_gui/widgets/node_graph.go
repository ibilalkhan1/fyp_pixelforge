package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// GraphNode is one rectangular node in a NodeGraphView. Caller
// positions the node; the widget does no auto-layout in v1.
type GraphNode struct {
	Name string
	X, Y int
}

// GraphEdge is a directed line from From to To (both names must
// match registered GraphNodes). Flash is the 0..1 highlight that
// the host decrements over time for the "edge flashes on publish"
// effect.
type GraphEdge struct {
	From  string
	To    string
	Flash float32
}

// NodeGraphViewOptions configures a NodeGraphView widget.
type NodeGraphViewOptions struct {
	Nodes        []GraphNode
	Edges        []GraphEdge
	NodeColor    pixelforge.Color
	AccentColor  pixelforge.Color
	DimColor     pixelforge.Color
	TextColor    pixelforge.Color
	OnSelectNode func(name string)
}

// NodeGraphView is a minimal pub/sub graph viewer: rectangular
// nodes at caller-supplied positions, directed lines between them,
// click-to-select on nodes. v1 has no auto-layout — the host
// positions every node.
type NodeGraphView struct {
	*pgui.Element

	Nodes        []GraphNode
	Edges        []GraphEdge
	NodeColor    pixelforge.Color
	AccentColor  pixelforge.Color
	DimColor     pixelforge.Color
	TextColor    pixelforge.Color
	OnSelectNode func(name string)
}

// NewNodeGraphView constructs a NodeGraphView rooted at (x, y, w, h).
func NewNodeGraphView(x, y, w, h int, opts NodeGraphViewOptions) *NodeGraphView {
	if opts.NodeColor == 0 {
		opts.NodeColor = 1
	}
	if opts.AccentColor == 0 {
		opts.AccentColor = 12
	}
	if opts.DimColor == 0 {
		opts.DimColor = 6
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	g := &NodeGraphView{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Nodes:        append([]GraphNode(nil), opts.Nodes...),
		Edges:        append([]GraphEdge(nil), opts.Edges...),
		NodeColor:    opts.NodeColor,
		AccentColor:  opts.AccentColor,
		DimColor:     opts.DimColor,
		TextColor:    opts.TextColor,
		OnSelectNode: opts.OnSelectNode,
	}
	g.Element.OnDraw = func(_ pgui.DrawEvent) { g.draw() }
	g.Element.OnTap = func(_ pgui.Event) { g.dispatchTap() }
	return g
}

const (
	graphNodeW = 96
	graphNodeH = 24
)

// HitTest returns the index of the node hit by the (mx, my) point
// in element-local coordinates, or -1 if none.
func (g *NodeGraphView) HitTest(mx, my int) int {
	for i, n := range g.Nodes {
		if mx >= n.X && mx < n.X+graphNodeW && my >= n.Y && my < n.Y+graphNodeH {
			return i
		}
	}
	return -1
}

func (g *NodeGraphView) dispatchTap() {
	mx, my := pguiPointerLocal(g.Element)
	idx := g.HitTest(mx, my)
	if idx >= 0 && g.OnSelectNode != nil {
		g.OnSelectNode(g.Nodes[idx].Name)
	}
}

func (g *NodeGraphView) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Background.
	pixelforge.SetColor(g.NodeColor)
	pixelforge.RectFill(0, 0, g.W-1, g.H-1)

	// Edges first, so nodes draw on top.
	nodeIndex := map[string]GraphNode{}
	for _, n := range g.Nodes {
		nodeIndex[n.Name] = n
	}
	for _, e := range g.Edges {
		from, okF := nodeIndex[e.From]
		to, okT := nodeIndex[e.To]
		if !okF || !okT {
			continue
		}
		x0 := from.X + graphNodeW
		y0 := from.Y + graphNodeH/2
		x1 := to.X
		y1 := to.Y + graphNodeH/2
		if e.Flash > 0 {
			pixelforge.SetColor(g.AccentColor)
		} else {
			pixelforge.SetColor(g.DimColor)
		}
		pixelforge.Line(x0, y0, x1, y1)
	}

	// Nodes.
	font := pgui.DefaultFont()
	for _, n := range g.Nodes {
		if n.X+graphNodeW < 0 || n.X > g.W || n.Y+graphNodeH < 0 || n.Y > g.H {
			continue
		}
		pixelforge.SetColor(g.NodeColor)
		pixelforge.RectFill(n.X, n.Y, n.X+graphNodeW-1, n.Y+graphNodeH-1)
		pixelforge.SetColor(g.DimColor)
		pixelforge.Rect(n.X, n.Y, n.X+graphNodeW-1, n.Y+graphNodeH-1)
		pixelforge.SetColor(g.TextColor)
		_, _ = font.Print(n.Name, n.X+4, n.Y+8)
	}
}
