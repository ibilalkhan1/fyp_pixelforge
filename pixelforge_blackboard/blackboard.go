package pixelforge_blackboard

import (
	"sort"
	"sync"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
)

// TargetName is the well-known pievent registry name under which the
// process-wide blackboard registers itself at init time. Studio
// surfaces and runtime code locate the blackboard via
// pievent.LookupTarget(TargetName) and type-assert to *Blackboard.
const TargetName = "state/*"

// BlackboardChange is the event published on every successful Set
// call. Subscribers receive both the previous value (nil when the key
// is being introduced) and the freshly-written one, so inspector
// panels can render transitions without an extra lookup.
type BlackboardChange struct {
	Key string
	Old any
	New any
}

// Blackboard is a flat, mutex-guarded key/value store published as a
// single pievent.Target. Construct via New; the package-level
// instance returned by Default is registered automatically at init.
type Blackboard struct {
	mu        sync.RWMutex
	values    map[string]any
	publishMu sync.Mutex
	target    pievent.Target[BlackboardChange]
}

// New constructs an empty Blackboard with its own internal change
// target. Tests and embedding code may create additional instances;
// production code typically uses Default.
func New() *Blackboard {
	return &Blackboard{
		values: map[string]any{},
		target: pievent.NewTarget[BlackboardChange](),
	}
}

// Set writes value under key and publishes a BlackboardChange event
// carrying the previous value (or nil when the key is new).
//
// Set is safe to call from multiple goroutines. publishMu serialises
// the underlying pievent.Target.Publish call (which is not itself
// concurrency-safe) without holding the value mutex, so subscribers
// may freely call Get from inside their handler.
func (b *Blackboard) Set(key string, value any) {
	b.mu.Lock()
	old, existed := b.values[key]
	b.values[key] = value
	b.mu.Unlock()

	change := BlackboardChange{Key: key, New: value}
	if existed {
		change.Old = old
	}

	b.publishMu.Lock()
	b.target.Publish(change)
	b.publishMu.Unlock()
}

// Get returns the value stored under key and whether the key is
// present. Missing keys return (nil, false).
func (b *Blackboard) Get(key string) (any, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	v, ok := b.values[key]
	return v, ok
}

// Keys returns a snapshot of every key currently held by the
// blackboard, sorted alphabetically. The returned slice is a copy;
// mutating it does not affect internal state.
func (b *Blackboard) Keys() []string {
	return b.KeysSnapshot()
}

// KeysSnapshot is the studio-facing alias for Keys. The blackboard
// inspector workspace (U9) calls KeysSnapshot for clarity at the
// call site.
func (b *Blackboard) KeysSnapshot() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.values))
	for k := range b.values {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Lookup is an alias for Get used by inspector code that prefers the
// "lookup" verb when displaying read-only widgets.
func (b *Blackboard) Lookup(key string) (any, bool) {
	return b.Get(key)
}

// Changes returns the internal pievent.Target that publishes
// BlackboardChange events. Callers subscribe via SubscribeAll to
// observe every mutation.
func (b *Blackboard) Changes() pievent.Target[BlackboardChange] {
	return b.target
}

// SubscriberCount reports the number of currently-registered
// listeners on the internal change target. Satisfies the
// pievent.Inspectable contract so the blackboard can be enumerated
// alongside other named targets in the topic catalog.
func (b *Blackboard) SubscriberCount() int {
	return b.target.SubscriberCount()
}

// PublishCount reports the total number of BlackboardChange events
// published since this Blackboard was created. Satisfies
// pievent.Inspectable.
func (b *Blackboard) PublishCount() uint64 {
	return b.target.PublishCount()
}

var defaultBlackboard = New()

// Default returns the process-wide *Blackboard registered with the
// pievent target registry at init time.
func Default() *Blackboard {
	return defaultBlackboard
}

func init() {
	pievent.RegisterTarget(TargetName, defaultBlackboard)
}
