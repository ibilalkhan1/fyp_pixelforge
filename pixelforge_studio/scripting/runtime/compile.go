package runtime

import (
	"fmt"
	"log"
	"reflect"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting/catalog"
)

// instance is one compiled BehaviorGraph at runtime.
type instance struct {
	name       string
	routine    *pixelforge_routine.Routine
	handler    pievent.Handler // routine schedule handler on piloop
	track      *trackingHandles
	// stepTrace records the most recent step transitions for the
	// debugger's time-travel scrub (M5 U13). One entry per step
	// advance: (tick, currentStep).
	stepTrace []stepTraceEntry
}

type stepTraceEntry struct {
	Tick    int
	StepIdx int
}

// trackingHandles is a non-generic wrapper around the heterogeneous
// per-target subscription handles a single behaviour might collect.
// Unsubscribe walks the list and dispatches by stored target ref.
type trackingHandles struct {
	entries []trackEntry
}

type trackEntry struct {
	target  pievent.Inspectable
	handler pievent.Handler
}

func (t *trackingHandles) Track(target pievent.Inspectable, handler pievent.Handler) {
	t.entries = append(t.entries, trackEntry{target: target, handler: handler})
}

func (t *trackingHandles) UnsubscribeAll() {
	for _, e := range t.entries {
		unsubscribeAny(e.target, e.handler)
	}
	t.entries = nil
}

// compileGraph turns a single BehaviorGraph into a running instance.
// Errors during compile are logged and best-effort substituted with
// no-op steps so a malformed graph doesn't prevent the rest of the
// project from running.
func compileGraph(graph pixelforge_project.BehaviorGraph, ctx *Context, eng *Engine, hook DebugHook) (*instance, error) {
	inst := &instance{
		name:  graph.Name,
		track: &trackingHandles{},
	}

	// Compile Steps.
	if len(graph.Steps) > 0 {
		steps := make([]pixelforge_routine.Step, 0, len(graph.Steps))
		for idx, node := range graph.Steps {
			step := buildStepNode(node, ctx, graph.Name, idx, eng, hook)
			steps = append(steps, step)
		}
		r := pixelforge_routine.New(steps...)
		r.SetName(graph.Name)
		inst.routine = r
	}

	// Compile EventSheet rules.
	for ruleIdx, rule := range graph.EventSheet {
		path := fmt.Sprintf("%d", ruleIdx)
		compileRule(rule, ctx, inst, path, eng, hook)
	}

	return inst, nil
}

func buildStepNode(node pixelforge_project.StepNode, ctx *Context, graphName string, idx int, eng *Engine, hook DebugHook) pixelforge_routine.Step {
	builder := catalog.LookupStep(node.Kind)
	if builder == nil {
		log.Printf("[scripting] unknown step kind %q in graph %q step %d; substituting no-op", node.Kind, graphName, idx)
		return func() bool { return true }
	}
	step, err := builder(node.Args, ctx)
	if err != nil {
		log.Printf("[scripting] step %q in graph %q step %d: %v; substituting no-op", node.Kind, graphName, idx, err)
		return func() bool { return true }
	}
	// Wrap with the debug hook + breakpoint gate. The wrapper is
	// always installed (cheap when no breakpoints / hooks are set).
	return func() bool {
		ev := DebugEvent{
			Kind:      DebugEventStep,
			GraphName: graphName,
			StepIdx:   idx,
		}
		if hook != nil {
			hook(ev)
		}
		if eng != nil && eng.checkBreakpoint(ev) {
			return false
		}
		done := step()
		if eng != nil {
			eng.onPostStep()
		}
		return done
	}
}

func compileRule(rule pixelforge_project.EventSheetRule, ctx *Context, inst *instance, path string, eng *Engine, hook DebugHook) {
	// Build predicates.
	predicates := make([]catalog.Predicate, 0, len(rule.Conditions))
	// Determine which target to subscribe on. v1 convention: if any
	// condition declares args["target"], we subscribe to that
	// target; otherwise we subscribe to loop.main so the rule fires
	// every update.
	targetName := "loop.main"
	for _, c := range rule.Conditions {
		if t := condTarget(c); t != "" {
			targetName = t
			break
		}
		builder := catalog.LookupCondition(c.Kind)
		if builder == nil {
			log.Printf("[scripting] unknown condition kind %q at %s; skipping", c.Kind, path)
			continue
		}
		pred, err := builder(c.Args, ctx)
		if err != nil {
			log.Printf("[scripting] condition %q at %s: %v; skipping", c.Kind, path, err)
			continue
		}
		predicates = append(predicates, pred)
	}
	// Re-walk to assemble predicates (so we don't miss any while
	// hunting targetName).
	predicates = predicates[:0]
	for _, c := range rule.Conditions {
		builder := catalog.LookupCondition(c.Kind)
		if builder == nil {
			continue
		}
		pred, err := builder(c.Args, ctx)
		if err != nil {
			continue
		}
		predicates = append(predicates, pred)
	}

	// Build effects.
	effects := make([]catalog.Effect, 0, len(rule.Actions))
	for _, a := range rule.Actions {
		builder := catalog.LookupAction(a.Kind)
		if builder == nil {
			log.Printf("[scripting] unknown action kind %q at %s; skipping", a.Kind, path)
			continue
		}
		eff, err := builder(a.Args, ctx)
		if err != nil {
			log.Printf("[scripting] action %q at %s: %v; skipping", a.Kind, path, err)
			continue
		}
		effects = append(effects, eff)
	}

	// Subscribe on the chosen target. v1 uses reflection so any
	// pievent.Target[T] works regardless of T.
	inspectable := pievent.LookupTarget(targetName)
	if inspectable == nil {
		log.Printf("[scripting] unknown target %q for %s; rule will never fire", targetName, path)
		return
	}
	handler, ok := subscribeAny(inspectable, func(payload any) {
		if hook != nil {
			hook(DebugEvent{
				Kind:      DebugEventRule,
				GraphName: inst.name,
				RuleIdx:   path,
				Payload:   payload,
			})
		}
		for _, pred := range predicates {
			if !pred(payload) {
				return
			}
		}
		for _, eff := range effects {
			eff(payload)
		}
	})
	if !ok {
		log.Printf("[scripting] subscribe failed for target %q at %s", targetName, path)
		return
	}
	inst.track.Track(inspectable, handler)

	// Recurse into children — they re-subscribe on their own
	// declared target (or inherit loop.main).
	for childIdx, child := range rule.Children {
		compileRule(child, ctx, inst, fmt.Sprintf("%s/%d", path, childIdx), eng, hook)
	}
}

// condTarget returns the args["target"] of a Condition, if any. v1's
// event_fired uses "target" to declare which pievent.Target to
// subscribe on.
func condTarget(c pixelforge_project.Condition) string {
	if c.Args == nil {
		return ""
	}
	if s, ok := c.Args["target"].(string); ok {
		return s
	}
	return ""
}

// subscribeAny is a reflection-driven SubscribeAll that works on any
// pievent.Target[T]. Returns (handler, true) on success; (_, false)
// when the underlying value doesn't expose a Subscribe-shaped method.
func subscribeAny(target pievent.Inspectable, listener func(any)) (pievent.Handler, bool) {
	v := reflect.ValueOf(target)
	if !v.IsValid() {
		return 0, false
	}
	m := v.MethodByName("SubscribeAll")
	if !m.IsValid() {
		return 0, false
	}
	mt := m.Type()
	if mt.NumIn() != 1 {
		return 0, false
	}
	fnType := mt.In(0)
	wrapped := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		var payload any
		if len(args) > 0 {
			payload = args[0].Interface()
		}
		listener(payload)
		return nil
	})
	out := m.Call([]reflect.Value{wrapped})
	if len(out) != 1 {
		return 0, false
	}
	h, ok := out[0].Interface().(pievent.Handler)
	if !ok {
		return 0, false
	}
	return h, true
}

// unsubscribeAny calls Unsubscribe(handler) on target via reflection.
func unsubscribeAny(target pievent.Inspectable, handler pievent.Handler) {
	v := reflect.ValueOf(target)
	if !v.IsValid() {
		return
	}
	m := v.MethodByName("Unsubscribe")
	if !m.IsValid() {
		return
	}
	m.Call([]reflect.Value{reflect.ValueOf(handler)})
}

// schedulingTarget returns the loop target the runtime's routines
// resume on. Centralised so tests can stub it if needed.
func schedulingTarget() pievent.Target[pixelforge_loop.Event] {
	return pixelforge_loop.Target()
}
