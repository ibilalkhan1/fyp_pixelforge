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

func TestEngine_BreakpointPausesAtStep(t *testing.T) {
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.bp", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	target.SubscribeAll(func(e string, _ pievent.Handler) { received = append(received, e) })

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "stepper",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Publish", Args: map[string]any{"target": "test.bp", "event": "first"}},
			{Kind: "Publish", Args: map[string]any{"target": "test.bp", "event": "second"}},
		},
	})
	eng := runtime.New(p)
	eng.SetBreakpoint("steps/stepper/1", true)
	eng.Start()
	defer eng.Stop()

	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)

	// First step published "first"; second should be paused.
	assert.Equal(t, []string{"first"}, received)
	paused, ev := eng.Paused()
	assert.True(t, paused)
	assert.Equal(t, "steps/stepper/1", ev.Path())
}

func TestEngine_ContinueResumes(t *testing.T) {
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.cont", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	target.SubscribeAll(func(e string, _ pievent.Handler) { received = append(received, e) })

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "cont",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Publish", Args: map[string]any{"target": "test.cont", "event": "first"}},
			{Kind: "Publish", Args: map[string]any{"target": "test.cont", "event": "second"}},
		},
	})
	eng := runtime.New(p)
	eng.SetBreakpoint("steps/cont/1", true)
	eng.Start()
	defer eng.Stop()

	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	require.True(t, mustPaused(eng))

	// Clear breakpoint and continue.
	eng.SetBreakpoint("steps/cont/1", false)
	eng.Continue()
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)

	assert.Contains(t, received, "second")
	paused, _ := eng.Paused()
	assert.False(t, paused)
}

func TestEngine_StepAdvancesOneAndRePauses(t *testing.T) {
	target := pievent.NewTarget[string]()
	pievent.RegisterTarget("test.step", target)
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	received := []string{}
	target.SubscribeAll(func(e string, _ pievent.Handler) { received = append(received, e) })

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "step",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Publish", Args: map[string]any{"target": "test.step", "event": "a"}},
			{Kind: "Publish", Args: map[string]any{"target": "test.step", "event": "b"}},
			{Kind: "Publish", Args: map[string]any{"target": "test.step", "event": "c"}},
		},
	})
	eng := runtime.New(p)
	eng.SetBreakpoint("steps/step/1", true)
	eng.Start()
	defer eng.Stop()

	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	require.True(t, mustPaused(eng))

	eng.Step()
	pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	assert.Contains(t, received, "b")
	paused, _ := eng.Paused()
	assert.True(t, paused, "single-step should re-pause after one advance")
}

func TestEngine_HookFiresPerEvent(t *testing.T) {
	defer pievent.ResetRegistryForTest()
	registerCoreTargets()

	p := newProject("demo", pixelforge_project.BehaviorGraph{
		Name: "hooked",
		Steps: []pixelforge_project.StepNode{
			{Kind: "Wait", Args: map[string]any{"ticks": float64(1)}},
			{Kind: "Wait", Args: map[string]any{"ticks": float64(1)}},
		},
	})
	eng := runtime.New(p)

	var seen []string
	eng.SetDebugHook(func(ev runtime.DebugEvent) {
		seen = append(seen, ev.Path())
	})
	eng.Start()
	defer eng.Stop()

	for i := 0; i < 4; i++ {
		pixelforge_loop.Target().Publish(pixelforge_loop.EventUpdate)
	}
	assert.GreaterOrEqual(t, len(seen), 2)
}

func TestEngine_PausedReturnsFalseOnUnpaused(t *testing.T) {
	p := newProject("demo")
	eng := runtime.New(p)
	paused, ev := eng.Paused()
	assert.False(t, paused)
	assert.Equal(t, "", ev.Path())
}

func TestEngine_BreakpointOnMissingGraphIsHarmless(t *testing.T) {
	p := newProject("demo")
	eng := runtime.New(p)
	assert.NotPanics(t, func() {
		eng.SetBreakpoint("steps/nothing/0", true)
		eng.Start()
		eng.Stop()
	})
}

func mustPaused(eng *runtime.Engine) bool {
	p, _ := eng.Paused()
	return p
}
