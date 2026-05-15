package scripting_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSynthesiseFromInputLog_HappyPath(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 0, Inputs: []capture.InputEntry{{Target: "key", Value: "Down"}}},
		{TickNumber: 1, Inputs: []capture.InputEntry{{Target: "key", Value: "Left"}}},
		{TickNumber: 3, Inputs: []capture.InputEntry{{Target: "key", Value: "Right"}}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 2)
	// Expected: Publish, Wait(1), Publish, Wait(2), Publish (no leading wait since first tick is 0)
	require.Len(t, graph.Steps, 5)
	assert.Equal(t, "Publish", graph.Steps[0].Kind)
	assert.Equal(t, "Wait", graph.Steps[1].Kind)
	assert.Equal(t, float64(1), graph.Steps[1].Args["ticks"])
	assert.Equal(t, "Publish", graph.Steps[2].Kind)
	assert.Equal(t, "Wait", graph.Steps[3].Kind)
	assert.Equal(t, float64(2), graph.Steps[3].Args["ticks"])
	assert.Equal(t, "Publish", graph.Steps[4].Kind)
}

func TestSynthesiseFromInputLog_LeadingWait(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 5, Inputs: []capture.InputEntry{{Target: "key", Value: "A"}}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 0)
	// Wait(5), Publish
	require.Len(t, graph.Steps, 2)
	assert.Equal(t, "Wait", graph.Steps[0].Kind)
	assert.Equal(t, float64(5), graph.Steps[0].Args["ticks"])
	assert.Equal(t, "Publish", graph.Steps[1].Kind)
}

func TestSynthesiseFromInputLog_NoInputsOnlyWaits(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 0, Inputs: nil},
		{TickNumber: 5, Inputs: nil},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 1)
	// Just a Wait between tick 0 and 5.
	require.Len(t, graph.Steps, 1)
	assert.Equal(t, "Wait", graph.Steps[0].Kind)
	assert.Equal(t, float64(5), graph.Steps[0].Args["ticks"])
}

func TestSynthesiseFromInputLog_MultipleInputsOnSameTick(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 0, Inputs: []capture.InputEntry{
			{Target: "key", Value: "A"},
			{Target: "key", Value: "B"},
		}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 0)
	// Two Publishes, no intervening Wait.
	require.Len(t, graph.Steps, 2)
	assert.Equal(t, "Publish", graph.Steps[0].Kind)
	assert.Equal(t, "Publish", graph.Steps[1].Kind)
	assert.Equal(t, "A", graph.Steps[0].Args["event"])
	assert.Equal(t, "B", graph.Steps[1].Args["event"])
}

func TestSynthesiseFromInputLog_EmptyRange(t *testing.T) {
	graph := scripting.SynthesiseFromInputLog(nil, 0, 0)
	assert.Empty(t, graph.Steps)
}

func TestSynthesiseFromInputLog_SwapsOutOfOrderIndices(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 0, Inputs: []capture.InputEntry{{Target: "k", Value: "x"}}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 2, 0)
	assert.NotEmpty(t, graph.Steps)
}

func TestSynthesiseFromInputLog_ClampsOverRange(t *testing.T) {
	frames := []*capture.Frame{
		{TickNumber: 0, Inputs: []capture.InputEntry{{Target: "k", Value: "x"}}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 999)
	assert.NotEmpty(t, graph.Steps)
}

func TestSynthesiseFromInputLog_NilFramesSkipped(t *testing.T) {
	frames := []*capture.Frame{
		nil,
		{TickNumber: 1, Inputs: []capture.InputEntry{{Target: "k", Value: "x"}}},
	}
	graph := scripting.SynthesiseFromInputLog(frames, 0, 1)
	require.NotEmpty(t, graph.Steps)
}
