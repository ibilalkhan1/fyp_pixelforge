package capsuleruntime

import (
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

// dialogueSink is the concrete DialogueController plan-009 U7 lands.
// It tracks the active dialogue script ID on rt.DialogueScript and
// emits "ui/dialogue_opened" / "ui/dialogue_closed" verbs back onto
// verbs.bus so chained handlers (input gating, music ducking, save-
// on-dialogue-end) can subscribe without coupling to the sink
// directly.
//
// A modal stack lands in a later plan; the sink's seam (per-runtime
// active-script field + bus republish) is the right one for both
// the eventual stack-based controller and today's single-script
// implementation.
type dialogueSink struct {
	rt *Runtime
}

// newDialogueSink constructs the production dialogueSink.
func newDialogueSink(rt *Runtime) *dialogueSink {
	return &dialogueSink{rt: rt}
}

// Open marks the named script active and republishes
// "ui/dialogue_opened". Calling Open while a dialogue is already
// open replaces the active script — the recipe-surface contract is
// "the latest open wins", matching how authored screens overlay.
func (d *dialogueSink) Open(scriptID string) {
	if d == nil || d.rt == nil {
		return
	}
	if scriptID == "" {
		debugDrop("ui/open_dialogue", "empty script_id", nil)
		return
	}
	d.rt.DialogueScript = scriptID
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/dialogue_opened",
		Args:  map[string]any{"script_id": scriptID},
	})
}

// Close clears the active script and republishes
// "ui/dialogue_closed". Calling Close when no dialogue is open is a
// no-op (the bus republish is gated by the active-script presence).
func (d *dialogueSink) Close() {
	if d == nil || d.rt == nil {
		return
	}
	if d.rt.DialogueScript == "" {
		return
	}
	prev := d.rt.DialogueScript
	d.rt.DialogueScript = ""
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "ui/dialogue_closed",
		Args:  map[string]any{"script_id": prev},
	})
}

// Active returns the currently-open script ID (empty when nothing
// is open). Exposed for diagnostics + tests that inspect dialogue
// state without reaching through rt.DialogueScript directly.
func (d *dialogueSink) Active() string {
	if d == nil || d.rt == nil {
		return ""
	}
	return d.rt.DialogueScript
}
