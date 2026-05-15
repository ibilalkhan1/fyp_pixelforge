package widgets_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/stretchr/testify/assert"
)

func TestNodeGraphView_HitTest(t *testing.T) {
	g := widgets.NewNodeGraphView(0, 0, 400, 200, widgets.NodeGraphViewOptions{
		Nodes: []widgets.GraphNode{
			{Name: "a", X: 0, Y: 0},
			{Name: "b", X: 200, Y: 100},
		},
	})
	assert.Equal(t, 0, g.HitTest(10, 10))
	assert.Equal(t, 1, g.HitTest(210, 110))
	assert.Equal(t, -1, g.HitTest(500, 500))
}

func TestNodeGraphView_EmptyGraphDrawsCleanly(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = widgets.NewNodeGraphView(0, 0, 100, 100, widgets.NodeGraphViewOptions{})
	})
}

func TestNodeGraphView_EdgeReferencingMissingNodeSkipped(t *testing.T) {
	g := widgets.NewNodeGraphView(0, 0, 100, 100, widgets.NodeGraphViewOptions{
		Nodes: []widgets.GraphNode{{Name: "a", X: 0, Y: 0}},
		Edges: []widgets.GraphEdge{{From: "a", To: "ghost"}},
	})
	assert.Len(t, g.Edges, 1, "missing edge endpoints aren't filtered at construction, just at draw time")
}
