package pixelforge_input_test

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// recorder captures IntentEvent publishes for a set of named intents
// in declaration order. Test helper; not concurrency-safe (tests
// drive events synchronously).
type recorder struct {
	mu       sync.Mutex
	events   []pixelforge_input.IntentEvent
	targets  map[string]pievent.Target[pixelforge_input.IntentEvent]
}

func newRecorder(intents ...string) *recorder {
	r := &recorder{targets: map[string]pievent.Target[pixelforge_input.IntentEvent]{}}
	for _, name := range intents {
		t := pievent.NewTarget[pixelforge_input.IntentEvent]()
		r.targets[name] = t
		t.SubscribeAll(func(ev pixelforge_input.IntentEvent, _ pievent.Handler) {
			r.mu.Lock()
			r.events = append(r.events, ev)
			r.mu.Unlock()
		})
	}
	return r
}

func (r *recorder) resolver(intent string) pievent.Target[pixelforge_input.IntentEvent] {
	return r.targets[intent]
}

func (r *recorder) snapshot() []pixelforge_input.IntentEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]pixelforge_input.IntentEvent, len(r.events))
	copy(out, r.events)
	return out
}

// newTestRig wires an isolated Compiler against fresh key and pad
// targets so test publishes don't bleed into the production
// pievent.RegisterTarget("key.main") / ("pad.button") instances used
// by other packages' tests in the same process.
func newTestRig(intents ...string) (*pixelforge_input.Compiler, pievent.Target[pixelforge_key.Event], pievent.Target[pixelforge_pad.EventButton], *recorder) {
	keyT := pievent.NewTarget[pixelforge_key.Event]()
	padT := pievent.NewTarget[pixelforge_pad.EventButton]()
	rec := newRecorder(intents...)
	c := pixelforge_input.NewCompiler(keyT, padT)
	c.SetIntentResolver(rec.resolver)
	return c, keyT, padT, rec
}

// A keyboard binding republishes on the intent target when the named
// key is pressed.
func TestCompiler_KeyboardEventPublishesIntent(t *testing.T) {
	c, keyT, _, rec := newTestRig(pixelforge_input.IntentJump)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"Z"}},
	})

	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Z})

	events := rec.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, pixelforge_input.IntentEvent{
		Type:   pixelforge_input.IntentEventDown,
		Intent: pixelforge_input.IntentJump,
	}, events[0])

	// Key release republishes as IntentEventUp.
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventUp, Key: pixelforge_key.Z})
	events = rec.snapshot()
	require.Len(t, events, 2)
	assert.Equal(t, pixelforge_input.IntentEventUp, events[1].Type)
}

// A gamepad-only binding fires the intent on the corresponding
// EventButton.
func TestCompiler_GamepadEventPublishesIntent(t *testing.T) {
	c, _, padT, rec := newTestRig(pixelforge_input.IntentJump)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, GamepadButton: "A"},
	})

	padT.Publish(pixelforge_pad.EventButton{Type: pixelforge_pad.EventDown, Button: pixelforge_pad.A})

	events := rec.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, pixelforge_input.IntentEvent{
		Type:   pixelforge_input.IntentEventDown,
		Intent: pixelforge_input.IntentJump,
	}, events[0])
}

// A binding with multiple keyboard keys plus a gamepad button fires
// the intent independently from each source.
func TestCompiler_MultiBindingFiresOnAny(t *testing.T) {
	c, keyT, padT, rec := newTestRig(pixelforge_input.IntentJump)

	// Note: pixelforge_key.Space is the literal " " string; bindings
	// reference physical key names verbatim. The U2 default map
	// canonicalises a separate "Space" alias — out of scope for the
	// intent layer, which matches keys by string equality only.
	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{string(pixelforge_key.Z), string(pixelforge_key.Space)}, GamepadButton: "A"},
	})

	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Z})
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Space})
	padT.Publish(pixelforge_pad.EventButton{Type: pixelforge_pad.EventDown, Button: pixelforge_pad.A})

	events := rec.snapshot()
	require.Len(t, events, 3, "all three independent sources should fire the intent")
	for _, ev := range events {
		assert.Equal(t, pixelforge_input.IntentJump, ev.Intent)
		assert.Equal(t, pixelforge_input.IntentEventDown, ev.Type)
	}
}

// A modifier-gated binding only fires when the modifier is held.
func TestCompiler_ModifierGate(t *testing.T) {
	c, keyT, _, rec := newTestRig(pixelforge_input.IntentMenu)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentMenu, Keyboard: []string{"S"}, Modifier: "Ctrl"},
	})

	// Plain S — no intent.
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.S})
	assert.Empty(t, rec.snapshot(), "S without Ctrl must not fire IntentMenu")

	// Ctrl down, then S — intent fires.
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.CtrlLeft})
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.S})
	events := rec.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, pixelforge_input.IntentMenu, events[0].Intent)

	// Release Ctrl, press S — no further intent.
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventUp, Key: pixelforge_key.CtrlLeft})
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.S})
	assert.Len(t, rec.snapshot(), 1, "S after Ctrl release must not refire")
}

// Recompile must unsubscribe the prior bridge so the previously-bound
// physical source no longer fires the intent.
func TestCompiler_Recompile_UnsubscribesOldBindings(t *testing.T) {
	c, keyT, _, rec := newTestRig(pixelforge_input.IntentJump)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"Z"}},
	})
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Z})
	require.Len(t, rec.snapshot(), 1, "first Z press should fire after Compile")

	// Swap to X-only. Z should no longer fire.
	c.Recompile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"X"}},
	})

	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Z})
	assert.Len(t, rec.snapshot(), 1,
		"Z press after Recompile to X must not fire intent")

	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.X})
	events := rec.snapshot()
	require.Len(t, events, 2, "X should fire after Recompile to X")
	assert.Equal(t, pixelforge_input.IntentJump, events[1].Intent)

	// The underlying key target must only have a single live bridge
	// subscriber (not two stacked after Recompile).
	assert.Equal(t, 1, keyT.SubscriberCount(),
		"Recompile must leave exactly one bridge subscriber on the key target")
}

// A binding whose Intent name isn't a registered intent logs a
// warning but does not crash — the binding is preserved (so a later
// re-register can pick it up) and other bindings continue to work.
func TestCompiler_UnknownIntentInMap_LogsWarning(t *testing.T) {
	c, keyT, _, rec := newTestRig(pixelforge_input.IntentJump)

	var buf bytes.Buffer
	origOut := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(origOut)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: "input/nonexistent", Keyboard: []string{"Q"}},
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"Z"}},
	})

	assert.True(t,
		strings.Contains(buf.String(), "input/nonexistent"),
		"warning must mention the unknown intent name; got: %s", buf.String())

	// The valid binding continues to fire.
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Z})
	events := rec.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, pixelforge_input.IntentJump, events[0].Intent)

	// The unknown intent's key press is silently dropped (no
	// resolver target exists).
	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Q})
	assert.Len(t, rec.snapshot(), 1, "unknown intent must not produce events")
}

// An unmapped physical key produces no intent event.
func TestCompiler_NoMatchingBinding_NoEvent(t *testing.T) {
	c, keyT, padT, rec := newTestRig(pixelforge_input.IntentJump)

	c.Compile([]pixelforge_project.InputBinding{
		{Intent: pixelforge_input.IntentJump, Keyboard: []string{"Z"}},
	})

	keyT.Publish(pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Q})
	padT.Publish(pixelforge_pad.EventButton{Type: pixelforge_pad.EventDown, Button: pixelforge_pad.B})

	assert.Empty(t, rec.snapshot(), "unmapped sources must not publish any intent event")
}
