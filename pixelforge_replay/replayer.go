package pixelforge_replay

import (
	"errors"
	"image"

	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capsuleruntime"
)

// ErrNilRuntime is returned by (*Replayer).Run when the caller
// passes a nil *capsuleruntime.Runtime. Mirrors the sentinel
// pixelforge_render exports — the contract is the same.
var ErrNilRuntime = errors.New("pixelforge_replay: runtime is nil")

// ErrNilTrace is returned by (*Replayer).Run when the caller passes
// a nil *Trace. An empty Trace (zero frames) is valid and produces
// empty result slices.
var ErrNilTrace = errors.New("pixelforge_replay: trace is nil")

// CapturedEvent pairs a published *piloop.VerbEvent with the tick
// it was observed on. Tick is the trace tick currently being
// rendered when the event fires (not a wall-clock timestamp). This
// is the shape CI assertions consume — "at tick T we expected to
// see Topic X" parity.
//
// Note: the bus event itself is a *piloop.VerbEvent. CapturedEvent
// wraps the pointer (not a copy) — modifying the underlying event's
// Args after capture would be observable through CapturedEvent.
// The convention is that bus publishers do not mutate the envelope
// after publish, which today's catalog respects.
type CapturedEvent struct {
	Tick  uint64
	Event *piloop.VerbEvent
}

// Replayer iterates a Trace against a booted *capsuleruntime.Runtime,
// producing the per-tick frame sequence + the verbs.bus events
// observed during replay. Replayer holds no state between Run calls
// — a fresh instance per replay is fine, and the type is exported
// mainly so future fields (e.g. instrumentation hooks) can land
// without an API churn.
type Replayer struct{}

// NewReplayer returns a zero-value Replayer. Currently a free
// function rather than a struct literal because we expect to grow
// the type with options (frame stride, event filter, max-frame
// cap) before the package is widely consumed; callers that switch
// to a literal now would pay a migration cost later.
func NewReplayer() *Replayer { return &Replayer{} }

// Run iterates trace.Frames in order, rendering each frame via
// pixelforge_render.RenderTickAtRGBA(rt, frame.Tick, InputFrame{...})
// and accumulating the resulting *image.RGBA into frames. While the
// iteration is in flight, Run subscribes to pixelforge_loop.VerbsBus
// and captures every published *VerbEvent into events, tagging it
// with the current trace tick.
//
// Determinism contract:
//   - No goroutines are spawned by Run. Iteration is strictly
//     serial; the subscriber runs synchronously on the Publish
//     caller's goroutine (per pievent's documented semantics).
//   - Run unsubscribes its bus listener before returning, even on
//     error, so a second Run call does not double-capture events.
//
// Audio sinks are stubbed at the caller's discretion via
// capsuleruntime.Options.SinksOverride; Run itself does not touch
// the rt's Sinks (doing so would defeat the parity invariant —
// rt's behaviour during replay must match its behaviour during a
// live tick).
//
// Caller is responsible for managing the verbs.bus lifecycle. If
// they want a clean bus (no subscribers from prior tests), they
// must call pixelforge_loop.ResetVerbsBusForTest() BEFORE booting
// the runtime they pass to Run. Run cannot reset the bus on the
// caller's behalf because that would unsubscribe the runtime's own
// verb-recipe handlers installed during Boot.
//
// Returns (nil, nil, ErrNilRuntime) on nil rt, (nil, nil, ErrNilTrace)
// on nil trace. An empty trace.Frames slice returns (empty, empty,
// nil) — not an error.
func (r *Replayer) Run(rt *capsuleruntime.Runtime, trace *Trace) ([]*image.RGBA, []*CapturedEvent, error) {
	if rt == nil {
		return nil, nil, ErrNilRuntime
	}
	if trace == nil {
		return nil, nil, ErrNilTrace
	}

	frames := make([]*image.RGBA, 0, len(trace.Frames))
	events := make([]*CapturedEvent, 0, 16)

	// currentTick is captured by the subscriber closure so each
	// event lands on the right replay tick. Assigned (not just
	// read) inside the iteration loop below.
	var currentTick uint64

	bus := piloop.VerbsBus()
	handle := bus.SubscribeAll(func(ev *piloop.VerbEvent, _ pievent.Handler) {
		if ev == nil {
			return
		}
		events = append(events, &CapturedEvent{Tick: currentTick, Event: ev})
	})
	defer bus.Unsubscribe(handle)

	for _, f := range trace.Frames {
		currentTick = f.Tick
		inputs := pixelforge_render.InputFrame{
			Keys: f.Keys,
			Pad:  f.Pad,
		}
		img, err := pixelforge_render.RenderTickAtRGBA(rt, f.Tick, inputs)
		if err != nil {
			return frames, events, err
		}
		frames = append(frames, img)
	}

	return frames, events, nil
}
