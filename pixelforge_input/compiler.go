package pixelforge_input

import (
	"log"
	"sync"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Compiler bridges physical key/pad events to intent events for the
// active InputMap. It owns the subscription handles on the underlying
// pievent targets so Recompile can swap maps atomically without
// leaking subscribers.
//
// A package-level instance (defaultCompiler) is the conventional
// caller path: Compile(im) and Recompile(im) operate on it. Tests
// construct their own *Compiler via NewCompiler to keep state
// isolated.
type Compiler struct {
	mu sync.Mutex

	// keyTarget and padTarget are the underlying physical event
	// targets. Captured at construction so tests can inject
	// alternative targets if desired.
	keyTarget pievent.Target[pixelforge_key.Event]
	padTarget pievent.Target[pixelforge_pad.EventButton]

	// bindings is the currently compiled InputMap, indexed by
	// intent name. Rebuilt on each Compile/Recompile call.
	bindings []pixelforge_project.InputBinding

	// held tracks modifier-key state derived from the key event
	// stream itself. We do not poll pixelforge_key.Duration because
	// Duration uses the per-frame counter (pixelforge.Frame) which
	// is not advanced by synchronous test Publishes; an event-driven
	// held-set is portable, race-free under the package mutex, and
	// trivially testable.
	held map[string]bool

	// keyHandle / padHandle reference the bridge subscriptions
	// installed on the physical targets so Recompile (and
	// Unsubscribe in tests) can tear them down precisely.
	keyHandle pievent.Handler
	padHandle pievent.Handler
	subscribed bool

	// intentResolver returns the pievent.Target carrying IntentEvent
	// for the named intent. Defaults to package-level Target; tests
	// may override to capture publishes on isolated targets.
	intentResolver func(string) pievent.Target[IntentEvent]
}

// NewCompiler constructs an isolated Compiler. Production callers
// should prefer the package-level Compile / Recompile entry points
// which manage a shared singleton; tests use NewCompiler to keep
// subscription state out of the global pievent targets.
func NewCompiler(keyTarget pievent.Target[pixelforge_key.Event], padTarget pievent.Target[pixelforge_pad.EventButton]) *Compiler {
	return &Compiler{
		keyTarget:      keyTarget,
		padTarget:      padTarget,
		held:           map[string]bool{},
		intentResolver: Target,
	}
}

// SetIntentResolver overrides the function used to resolve an intent
// name to its pievent.Target[IntentEvent]. Test-only: lets a test
// capture intent publishes on locally-owned targets without touching
// the package-level registry.
func (c *Compiler) SetIntentResolver(resolver func(string) pievent.Target[IntentEvent]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.intentResolver = resolver
}

// Compile installs subscriptions on the configured key and pad
// targets and stores the active InputMap. Subsequent physical events
// route through the map and republish on the corresponding intent
// targets. Calling Compile after a prior Compile is equivalent to
// calling Recompile.
func (c *Compiler) Compile(im []pixelforge_project.InputBinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.installLocked(im)
}

// Recompile unsubscribes the prior bridge handlers and installs a new
// pair tied to the supplied InputMap. Intended for the Settings ->
// Input live-edit flow (U7) where a designer changes a binding and
// expects the next physical press to route per the new map without
// restarting the engine.
func (c *Compiler) Recompile(im []pixelforge_project.InputBinding) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unsubscribeLocked()
	c.installLocked(im)
}

// Unsubscribe removes the active bridge subscriptions and clears the
// active InputMap. Exposed for tests that want to assert no further
// publishes occur after teardown.
func (c *Compiler) Unsubscribe() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unsubscribeLocked()
	c.bindings = nil
}

func (c *Compiler) installLocked(im []pixelforge_project.InputBinding) {
	c.bindings = append([]pixelforge_project.InputBinding(nil), im...)
	c.warnUnknownIntents()

	if c.keyTarget != nil {
		c.keyHandle = c.keyTarget.SubscribeAll(func(ev pixelforge_key.Event, _ pievent.Handler) {
			c.onKey(ev)
		})
	}
	if c.padTarget != nil {
		c.padHandle = c.padTarget.SubscribeAll(func(ev pixelforge_pad.EventButton, _ pievent.Handler) {
			c.onPad(ev)
		})
	}
	c.subscribed = true
}

func (c *Compiler) unsubscribeLocked() {
	if !c.subscribed {
		return
	}
	if c.keyTarget != nil {
		c.keyTarget.Unsubscribe(c.keyHandle)
	}
	if c.padTarget != nil {
		c.padTarget.Unsubscribe(c.padHandle)
	}
	c.keyHandle = 0
	c.padHandle = 0
	c.subscribed = false
}

// warnUnknownIntents logs (but does not drop) bindings whose Intent
// name has no registered pievent target. The binding is still kept in
// c.bindings so a later Recompile that re-registers the target can
// observe it; this matches the schema convention of preserving
// unknown content rather than silently dropping it.
func (c *Compiler) warnUnknownIntents() {
	for _, b := range c.bindings {
		if c.intentResolver(b.Intent) == nil {
			log.Printf("[input] InputMap binding references unknown intent %q; ignoring entry", b.Intent)
		}
	}
}

// onKey is the bridge handler installed on the keyboard target. It
// first updates the held-modifier state, then walks the InputMap to
// find any binding whose Keyboard list contains the key (and whose
// Modifier, if any, is currently held). Each match publishes an
// IntentEvent on the corresponding intent target.
func (c *Compiler) onKey(ev pixelforge_key.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.updateHeldLocked(ev)

	keyName := string(ev.Key)
	for _, b := range c.bindings {
		if !containsString(b.Keyboard, keyName) {
			continue
		}
		if !c.modifierOKLocked(b.Modifier) {
			continue
		}
		c.publishLocked(b.Intent, mapKeyEventType(ev.Type))
	}
}

// onPad is the bridge handler installed on the gamepad button target.
// Gamepad bindings do not currently honour the Modifier field —
// modifier semantics are keyboard-only in v1, matching standard
// retro-controller expectations.
func (c *Compiler) onPad(ev pixelforge_pad.EventButton) {
	c.mu.Lock()
	defer c.mu.Unlock()

	btnName := string(ev.Button)
	for _, b := range c.bindings {
		if b.GamepadButton == "" || b.GamepadButton != btnName {
			continue
		}
		c.publishLocked(b.Intent, mapPadEventType(ev.Type))
	}
}

// publishLocked resolves the intent's pievent target via the
// configured resolver and publishes an IntentEvent. Called with c.mu
// held; pievent.Target.Publish is synchronous and not concurrency-
// safe, so the lock serialises publishes across the keyboard and
// gamepad bridges as well as concurrent caller code.
func (c *Compiler) publishLocked(intent string, t IntentEventType) {
	target := c.intentResolver(intent)
	if target == nil {
		return
	}
	target.Publish(IntentEvent{Type: t, Intent: intent})
}

// updateHeldLocked maintains the modifier held-set from the key event
// stream. Both left/right variants and the virtual aggregate
// constants (Ctrl/Shift/Alt) are tracked so a binding written as
// {Modifier: "Ctrl"} matches whether the user pressed CtrlLeft or
// CtrlRight.
func (c *Compiler) updateHeldLocked(ev pixelforge_key.Event) {
	mod := modifierForKey(ev.Key)
	if mod == "" {
		return
	}
	switch ev.Type {
	case pixelforge_key.EventDown:
		c.held[mod] = true
	case pixelforge_key.EventUp:
		c.held[mod] = false
	}
}

// modifierOKLocked reports whether a binding's declared Modifier is
// currently satisfied. An empty Modifier always matches. Otherwise
// the named modifier must be held according to the tracked set.
func (c *Compiler) modifierOKLocked(mod string) bool {
	if mod == "" {
		return true
	}
	return c.held[mod]
}

// modifierForKey returns the canonical modifier name ("Ctrl",
// "Shift", "Alt") for any of the six concrete modifier keys plus the
// three virtual aggregates, or "" for non-modifier keys.
func modifierForKey(k pixelforge_key.Key) string {
	switch k {
	case pixelforge_key.CtrlLeft, pixelforge_key.CtrlRight, pixelforge_key.Ctrl:
		return "Ctrl"
	case pixelforge_key.ShiftLeft, pixelforge_key.ShiftRight, pixelforge_key.Shift:
		return "Shift"
	case pixelforge_key.AltLeft, pixelforge_key.AltRight, pixelforge_key.Alt:
		return "Alt"
	}
	return ""
}

// mapKeyEventType projects pixelforge_key.EventType onto the intent
// event type vocabulary. Unknown event types fall through to
// IntentEventDown for forward compatibility with new key event
// variants (none currently planned).
func mapKeyEventType(t pixelforge_key.EventType) IntentEventType {
	if t == pixelforge_key.EventUp {
		return IntentEventUp
	}
	return IntentEventDown
}

// mapPadEventType projects pixelforge_pad.EventButtonType onto the
// intent event type vocabulary.
func mapPadEventType(t pixelforge_pad.EventButtonType) IntentEventType {
	if t == pixelforge_pad.EventUp {
		return IntentEventUp
	}
	return IntentEventDown
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// defaultCompiler is the package-level compiler used by the package
// Compile / Recompile entry points. It wires the production key.main
// and pad.button targets exposed by pixelforge_key / pixelforge_pad.
var defaultCompiler = NewCompiler(pixelforge_key.Target(), pixelforge_pad.ButtonTarget())

// Compile installs the supplied InputMap on the package-level
// compiler. This is the production entry point invoked by the
// runtime engine at project load.
func Compile(im []pixelforge_project.InputBinding) {
	defaultCompiler.Compile(im)
}

// Recompile is the package-level Recompile, intended for the studio's
// Settings -> Input workspace (U7) to apply edited bindings without
// restarting the engine.
func Recompile(im []pixelforge_project.InputBinding) {
	defaultCompiler.Recompile(im)
}
