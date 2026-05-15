package runtime

import (
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Engine owns the per-project scripting runtime. One Engine per loaded
// project; Stop() tears down every routine and unsubscribes every
// event handler so a swap-to-new-project doesn't leak.
type Engine struct {
	mu        sync.Mutex
	project   *pixelforge_project.Project
	context   *Context
	instances []*instance
	running   bool

	// debugHook is consulted before each step Resume and each rule
	// predicate evaluation. nil means "no debugger attached".
	debugHook DebugHook

	// breakpoints maps DebugEvent.Path() → enabled. When the
	// runtime wrapper sees a matching event, it sets paused = true
	// and the step/rule does not fire until Continue or Step is
	// called.
	breakpoints map[string]bool
	paused      bool
	pausedAt    DebugEvent
	singleStep  bool

	// step trace bookkeeping for the debugger time-travel slider.
	currentTick int
}

// New constructs a runtime engine for the project. Start() begins
// execution; the engine is idle until then.
func New(p *pixelforge_project.Project) *Engine {
	return &Engine{
		project:     p,
		context:     newContext(p),
		breakpoints: map[string]bool{},
	}
}

// SetBreakpoint enables or disables a breakpoint at the given path.
// Paths are produced by DebugEvent.Path() — "steps/<graphName>/<idx>"
// or "rules/<graphName>/<ruleIdx>". Setting a breakpoint while the
// engine is running takes effect at the next event matching the
// path.
func (e *Engine) SetBreakpoint(path string, on bool) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if on {
		e.breakpoints[path] = true
	} else {
		delete(e.breakpoints, path)
	}
}

// BreakpointSet reports whether the breakpoint at path is enabled.
func (e *Engine) BreakpointSet(path string) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.breakpoints[path]
}

// Paused reports whether the engine is currently paused at a
// breakpoint, returning (true, event) when paused or (false, zero)
// otherwise.
func (e *Engine) Paused() (bool, DebugEvent) {
	if e == nil {
		return false, DebugEvent{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.paused, e.pausedAt
}

// Continue resumes execution after a breakpoint pause.
func (e *Engine) Continue() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.paused = false
	e.singleStep = false
}

// Step resumes execution for exactly one step / rule, then
// re-pauses.
func (e *Engine) Step() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.paused = false
	e.singleStep = true
}

// checkBreakpoint consults the breakpoint map and (if matched) flips
// the engine into the paused state. Returns true when execution
// should NOT proceed (engine is paused).
//
// Single-step mode (set by Engine.Step) bypasses the breakpoint
// check for exactly one event; the post-step hook then re-pauses.
func (e *Engine) checkBreakpoint(ev DebugEvent) bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.paused {
		// Already paused — execution is gated.
		return true
	}
	if e.singleStep {
		// Allow this event through; onPostStep re-pauses.
		return false
	}
	if e.breakpoints[ev.Path()] {
		e.paused = true
		e.pausedAt = ev
		return true
	}
	return false
}

// onPostStep is called after the wrapped step runs to handle the
// "Step once then re-pause" mode.
func (e *Engine) onPostStep() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.singleStep {
		e.singleStep = false
		e.paused = true
	}
}

// Project returns the bound project.
func (e *Engine) Project() *pixelforge_project.Project {
	if e == nil {
		return nil
	}
	return e.project
}

// Context returns the runtime context (the catalog.Context bridge).
func (e *Engine) Context() *Context {
	if e == nil {
		return nil
	}
	return e.context
}

// Running reports whether Start has been called and Stop hasn't.
func (e *Engine) Running() bool {
	if e == nil {
		return false
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Start compiles every non-empty BehaviorGraph into a routine +
// handler tree and schedules routines on the loop's EventUpdate.
//
// Calling Start on an already-running engine is a no-op; call
// Reload(graph) to swap one graph in place, or Stop+Start to
// re-instantiate everything.
func (e *Engine) Start() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return
	}
	e.compileAll()
	e.scheduleRoutines()
	e.running = true
}

// Stop tears down every compiled instance. Idempotent.
func (e *Engine) Stop() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}
	for _, inst := range e.instances {
		e.stopInstance(inst)
	}
	e.instances = nil
	e.running = false
}

// Reload tears down and re-instantiates a single named graph. Used by
// the editor after an in-place behaviour edit. Unknown names are a
// no-op (with a debug log).
func (e *Engine) Reload(graphName string) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}

	// Find the instance to replace.
	var keep []*instance
	for _, inst := range e.instances {
		if inst.name == graphName {
			e.stopInstance(inst)
			continue
		}
		keep = append(keep, inst)
	}
	e.instances = keep

	// Find the matching graph in the project and re-instantiate.
	if e.project == nil {
		return
	}
	for _, g := range e.project.Behaviors {
		if g.Name != graphName {
			continue
		}
		inst, err := compileGraph(g, e.context, e, e.debugHook)
		if err != nil {
			return
		}
		e.scheduleInstance(inst)
		e.instances = append(e.instances, inst)
		return
	}
}

// SetDebugHook installs a debug callback used by the visual debugger
// (U12/U13). Pass nil to detach.
//
// Setting the hook while the engine is running re-compiles routines
// in place so the new hook is wired through.
func (e *Engine) SetDebugHook(hook DebugHook) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.debugHook = hook
	if !e.running {
		return
	}
	for _, inst := range e.instances {
		e.stopInstance(inst)
	}
	e.instances = nil
	e.compileAll()
	e.scheduleRoutines()
}

// Instances returns a snapshot of the running instances. Read-only;
// the slice is a defensive copy.
func (e *Engine) Instances() []*Instance {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*Instance, 0, len(e.instances))
	for _, inst := range e.instances {
		out = append(out, &Instance{
			Name:       inst.name,
			HasRoutine: inst.routine != nil,
			Routine:    inst.routine,
		})
	}
	return out
}

// Instance is the public read-only view of one compiled behaviour.
type Instance struct {
	Name       string
	HasRoutine bool
	// Routine is exposed for the debugger panel; treat as read-only.
	Routine interface{}
}

func (e *Engine) compileAll() {
	if e.project == nil {
		return
	}
	e.instances = e.instances[:0]
	for _, g := range e.project.Behaviors {
		inst, err := compileGraph(g, e.context, e, e.debugHook)
		if err != nil {
			continue
		}
		e.instances = append(e.instances, inst)
	}
}

func (e *Engine) scheduleRoutines() {
	for _, inst := range e.instances {
		e.scheduleInstance(inst)
	}
}

func (e *Engine) scheduleInstance(inst *instance) {
	if inst.routine == nil {
		return
	}
	inst.handler = inst.routine.ScheduleOn(pixelforge_loop.EventUpdate)
}

func (e *Engine) stopInstance(inst *instance) {
	if inst == nil {
		return
	}
	if inst.routine != nil {
		inst.routine.Stop()
	}
	if inst.track != nil {
		inst.track.UnsubscribeAll()
	}
	if inst.handler != 0 {
		schedulingTarget().Unsubscribe(inst.handler)
		inst.handler = 0
	}
}
