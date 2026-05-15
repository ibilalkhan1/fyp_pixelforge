package scripting_test

import (
	"strings"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmit_WaitStep(t *testing.T) {
	graph := pixelforge_project.BehaviorGraph{
		Name: "demo",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Wait", Args: map[string]any{"ticks": float64(30)}},
		},
	}
	out, err := scripting.Emit(graph)
	require.NoError(t, err)
	assert.Contains(t, out, "piroutine.Wait(30)")
	assert.Contains(t, out, "package generated")
}

func TestEmit_EventSheetRuleSubscribes(t *testing.T) {
	graph := pixelforge_project.BehaviorGraph{
		Name: "demo",
		EventSheet: []pixelforge_project.EventSheetRule{{
			Conditions: []pixelforge_project.Condition{{
				Kind: "event_fired",
				Args: map[string]any{"event": "Jump"},
			}},
			Actions: []pixelforge_project.Action{{
				Kind: "publish_event",
				Args: map[string]any{"target": "loop.main", "event": "echo"},
			}},
		}},
	}
	out, err := scripting.Emit(graph)
	require.NoError(t, err)
	assert.Contains(t, out, "SubscribeAll")
	assert.Contains(t, out, `payload == "Jump"`)
	assert.Contains(t, out, `targetFor("loop.main").Publish("echo")`)
}

func TestEmit_EmptyGraph(t *testing.T) {
	out, err := scripting.Emit(pixelforge_project.BehaviorGraph{Name: "empty"})
	require.NoError(t, err)
	assert.Contains(t, out, "empty graph")
}

func TestEmit_UnknownKindAsTodo(t *testing.T) {
	graph := pixelforge_project.BehaviorGraph{
		Name: "demo",
		Steps: []pixelforge_project.StepNode{
			{Kind: "FrobnicateTheWidgets", Args: map[string]any{}},
		},
	}
	out, err := scripting.Emit(graph)
	require.NoError(t, err)
	assert.Contains(t, out, "TODO: unknown kind")
}

func TestEmit_BranchStepRendersAsBranch(t *testing.T) {
	graph := pixelforge_project.BehaviorGraph{
		Name: "demo",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Branch", Args: map[string]any{"predicate": false}},
		},
	}
	out, err := scripting.Emit(graph)
	require.NoError(t, err)
	assert.Contains(t, out, "piroutine.Branch")
}

func TestEmit_HeaderIncludesPackageAndImports(t *testing.T) {
	out, err := scripting.Emit(pixelforge_project.BehaviorGraph{Name: "demo"})
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "import"))
}

func TestEmit_GraphNameWithInvalidCharsSanitised(t *testing.T) {
	out, err := scripting.Emit(pixelforge_project.BehaviorGraph{Name: "my-graph!"})
	require.NoError(t, err)
	// The hyphen and exclamation become underscores.
	assert.Contains(t, out, "my_graph_")
}
