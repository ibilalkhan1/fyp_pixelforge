package runtime_test

import (
	"testing"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProject(name string, behaviors ...pixelforge_project.BehaviorGraph) *pixelforge_project.Project {
	p := pixelforge_project.NewProject(name)
	p.Behaviors = behaviors
	return p
}

func TestEngine_RoutineExecutesOnUpdate(t *testing.T) {
	// Register a custom string target for the behaviour to publish on.
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.routine", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	target.SubscribeAll(func(e string, _ pievent.Handler) {
		received = append(received, e)
	})

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "tick-publisher",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Wait", Args: map[string]any{"ticks": float64(2)}},
			{Kind: "Publish", Args: map[string]any{
				"target": "test.routine",
				"event":  "ping",
			}},
		},
	})

	eng := runtime.New(p)
	eng.Start()
	defer eng.Stop()

	require.True(t, eng.Running())
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)

	assert.Equal(t, []string{"ping"}, received)
}

func TestEngine_EventSheetSubscribes(t *testing.T) {
	source := pievent.NewTarget[string]()
	sink := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.source", source)
	pievent.RegisterTarget("test.sink", sink)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	sink.SubscribeAll(func(e string, _ pievent.Handler) {
		received = append(received, e)
	})

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "echoer",
		EventSheet: []pixelforge_project.EventSheetRule{{
			Conditions: []pixelforge_project.Condition{{
				Kind: "event_fired",
				Args: map[string]any{
					"target": "test.source",
					"event":  "ping",
				},
			}},
			Actions: []pixelforge_project.Action{{
				Kind: "publish_event",
				Args: map[string]any{
					"target": "test.sink",
					"event":  "echoed",
				},
			}},
		}},
	})

	eng := runtime.New(p)
	eng.Start()
	defer eng.Stop()

	source.Publish("ping")
	assert.Equal(t, []string{"echoed"}, received)

	source.Publish("nope")
	assert.Equal(t, []string{"echoed"}, received, "non-matching event should not fire effect")
}

func TestEngine_StopUnsubscribes(t *testing.T) {
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.unsubs", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	subCountStart := target.SubscriberCount()

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "subbed",
		EventSheet: []pixelforge_project.EventSheetRule{{
			Conditions: []pixelforge_project.Condition{{
				Kind: "event_fired",
				Args: map[string]any{"target": "test.unsubs", "event": "x"},
			}},
		}},
	})

	eng := runtime.New(p)
	eng.Start()
	assert.Greater(t, target.SubscriberCount(), subCountStart)

	eng.Stop()
	assert.Equal(t, subCountStart, target.SubscriberCount())
}

func TestEngine_ReloadRebuildsNamedGraph(t *testing.T) {
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.reload", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "a",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Publish", Args: map[string]any{"target": "test.reload", "event": "from-a"}},
		},
	}, pixelforge_project.BehaviorGraph{
		Name: "b",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Publish", Args: map[string]any{"target": "test.reload", "event": "from-b"}},
		},
	})

	eng := runtime.New(p)
	eng.Start()
	defer eng.Stop()

	// Mutate b in place — change its publish event.
	for i := range p.Behaviors {
		if p.Behaviors[i].Name == "b" {
			p.Behaviors[i].Steps[0].Args["event"] = "reloaded"
		}
	}
	eng.Reload("b")

	received := map[string]int{}
	target.SubscribeAll(func(e string, _ pievent.Handler) { received[e]++ })

	for i := 0; i < 3; i++ {
		pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	}

	// b should have published "reloaded", not "from-b".
	assert.Contains(t, received, "reloaded")
}

func TestEngine_UnknownStepKindIsLoggedAndNoOp(t *testing.T) {
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "bad",
		Steps: []pixelforge_project.StepNode{
			{Kind: "DoesNotExist", Args: map[string]any{}},
			{Kind: "Wait", Args: map[string]any{"ticks": float64(1)}},
		},
	})
	eng := runtime.New(p)
	assert.NotPanics(t, func() {
		eng.Start()
		pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
		pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
		eng.Stop()
	})
}

func TestEngine_StopIdempotent(t *testing.T) {
	p := newProject("demo")
	eng := runtime.New(p)
	eng.Start()
	eng.Stop()
	assert.NotPanics(t, func() { eng.Stop() })
}

func TestEngine_NestedRulesFireOnParentCondition(t *testing.T) {
	source := pievent.NewTarget[string]()
	sink := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.nested.source", source)
	pievent.RegisterTarget("test.nested.sink", sink)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	sink.SubscribeAll(func(e string, _ pievent.Handler) { received = append(received, e) })

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "nesting",
		EventSheet: []pixelforge_project.EventSheetRule{{
			Conditions: []pixelforge_project.Condition{{
				Kind: "event_fired",
				Args: map[string]any{"target": "test.nested.source", "event": "ping"},
			}},
			Actions: []pixelforge_project.Action{{
				Kind: "publish_event",
				Args: map[string]any{"target": "test.nested.sink", "event": "parent"},
			}},
			Children: []pixelforge_project.EventSheetRule{{
				Conditions: []pixelforge_project.Condition{{
					Kind: "event_fired",
					Args: map[string]any{"target": "test.nested.source", "event": "ping"},
				}},
				Actions: []pixelforge_project.Action{{
					Kind: "publish_event",
					Args: map[string]any{"target": "test.nested.sink", "event": "child"},
				}},
			}},
		}},
	})

	eng := runtime.New(p)
	eng.Start()
	defer eng.Stop()

	source.Publish("ping")
	assert.Contains(t, received, "parent")
	assert.Contains(t, received, "child")
}

// registerCoreTargets re-adds the engine package targets after a
// ResetRegistryForTest call. Engine packages register themselves at
// init() but the reset clears them.
func registerCoreTargets() {
	pievent.RegisterTarget("loop.main", pixelforge_loop.Target())
	pievent.RegisterTarget("loop.debug", pixelforge_loop.DebugTarget())
}
