package runtime

import "fmt"

// DebugEventKind enumerates the runtime moments the debug hook is
// fired before.
type DebugEventKind string

const (
	// DebugEventStep fires before a behaviour routine's Step is
	// resumed for the active tick.
	DebugEventStep DebugEventKind = "step"

	// DebugEventRule fires before an event sheet rule's predicates
	// are evaluated for a published payload.
	DebugEventRule DebugEventKind = "rule"
)

// DebugEvent is the payload passed to a DebugHook.
type DebugEvent struct {
	Kind      DebugEventKind
	GraphName string
	StepIdx   int
	RuleIdx   string // path like "rules/<graphName>/<idx>/..."
	Payload   any
}

// Path returns the breakpoint key for this event:
//   - "steps/<graphName>/<stepIdx>" for DebugEventStep
//   - "rules/<graphName>/<ruleIdx>" for DebugEventRule
func (e DebugEvent) Path() string {
	switch e.Kind {
	case DebugEventStep:
		return fmt.Sprintf("steps/%s/%d", e.GraphName, e.StepIdx)
	case DebugEventRule:
		return fmt.Sprintf("rules/%s/%s", e.GraphName, e.RuleIdx)
	}
	return ""
}

// DebugHook is the optional observer callback fired before each step
// Resume and rule evaluation. The hook is a notification — pause
// semantics live on the Engine (SetBreakpoint / Paused / Continue /
// Step), so the hook can simply forward the event to a UI surface.
type DebugHook func(event DebugEvent)
