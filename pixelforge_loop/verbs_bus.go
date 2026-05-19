package pixelforge_loop

import pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"

// VerbEvent is the envelope verb-recipe Actions publish on the
// dedicated "verbs.bus" target. The publisher is the catalog's
// "publish_event" Action; the subscribers live in
// pixelforge_studio/capsuleruntime/subscribers.go.
//
// Topic identifies the recipe's intended subsystem (e.g.
// "audio/play_sound", "inventory/give_item"). Args carries the
// recipe's authored arg map (sample name, item id, fuse ticks,
// etc.) so subscribers can decode payload-bearing verbs without a
// blackboard round-trip.
//
// The envelope is a *VerbEvent (not VerbEvent by value) so the bus
// can use pievent.Target[*VerbEvent] — Args is a map and therefore
// not comparable, but a pointer to the envelope satisfies the
// comparable constraint. Each Publish allocates one envelope; the
// allocation is paid only when verbs actually fire (recipes are
// edge-triggered by player input and collisions, not per frame),
// so it stays cheap inside the game loop.
type VerbEvent struct {
	Topic string
	Args  map[string]any
}

// VerbsBus returns the process-wide verb-recipe target subscribers
// listen on. Engine consumers of the existing "loop.main" target
// are untouched — VerbsBus is a separate Target[*VerbEvent] so the
// typed-enum loop bus stays pristine.
func VerbsBus() pievent.Target[*VerbEvent] { return verbsBus }

// VerbsBusTargetName is the well-known registry name verb-recipe
// catalog code resolves via pievent.LookupTarget. Kept as a const
// so producers and consumers reference the same symbol.
const VerbsBusTargetName = "verbs.bus"

var verbsBus = pievent.NewTarget[*VerbEvent]()

func init() {
	pievent.RegisterTarget(VerbsBusTargetName, verbsBus)
}

// ResetVerbsBusForTest swaps the package-global verbs.bus target
// for a fresh one and re-registers it in the pievent registry.
// Test-only — callers that exercise subscribe/publish semantics
// across multiple cases use this between cases so handlers from
// the prior case don't leak forward.
func ResetVerbsBusForTest() {
	verbsBus = pievent.NewTarget[*VerbEvent]()
	pievent.RegisterTarget(VerbsBusTargetName, verbsBus)
}
