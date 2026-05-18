package pixelforge_input

import (
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
)

// IntentEventType distinguishes intent press from release. It mirrors
// pixelforge_key.EventType / pixelforge_pad.EventButtonType so verbs
// authored against intents can reason about edge transitions the same
// way they would against raw keys.
type IntentEventType string

const (
	// IntentEventDown is published when any binding source for the
	// intent transitions from up to down (matching keyboard EventDown
	// or gamepad button EventDown).
	IntentEventDown IntentEventType = "down"

	// IntentEventUp is published when any binding source transitions
	// from down to up.
	IntentEventUp IntentEventType = "up"
)

// IntentEvent is the payload published on an intent's pievent.Target.
// It carries the intent name so a single subscriber observing several
// intent targets via the registry can route on Intent without
// needing per-target wrappers.
type IntentEvent struct {
	Type   IntentEventType
	Intent string
}

// Canonical intent names registered as pievent.Target at init. These
// constants are the source of truth for intent topic naming; the
// pixelforge_project package mirrors the same strings to avoid an
// import cycle between schema and runtime.
const (
	IntentJump   = "input/jump"
	IntentAttack = "input/attack"
	IntentUp     = "input/up"
	IntentDown   = "input/down"
	IntentLeft   = "input/left"
	IntentRight  = "input/right"
	IntentUse    = "input/use"
	IntentMenu   = "input/menu"
	IntentStart  = "input/start"
)

// AllIntents lists the 9 canonical intents in declaration order. Used
// by tests and inspector enumeration; mutating the returned slice
// does not affect package state because a fresh slice is constructed
// on every call.
func AllIntents() []string {
	return []string{
		IntentJump, IntentAttack,
		IntentUp, IntentDown, IntentLeft, IntentRight,
		IntentUse, IntentMenu, IntentStart,
	}
}

// intentTargets holds the one pievent.Target[IntentEvent] per intent.
// Populated at init; never mutated after.
var intentTargets = map[string]pievent.Target[IntentEvent]{}

// Target returns the pievent.Target carrying IntentEvent values for
// the named intent. Returns nil for unknown intents. Verbs ordinarily
// reach the same target via pievent.LookupTarget(name); this helper
// exists for typed in-package callers (the Compiler) and tests.
func Target(intent string) pievent.Target[IntentEvent] {
	return intentTargets[intent]
}

func init() {
	for _, name := range AllIntents() {
		t := pievent.NewTarget[IntentEvent]()
		intentTargets[name] = t
		pievent.RegisterTarget(name, t)
	}
}
