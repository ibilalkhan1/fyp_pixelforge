// Package capture implements the continuous-capture spine described in
// the M4 plan: a ring buffer of paletted screen snapshots, the
// SubscribeAll input/event taps that feed it, and the six user-facing
// tools (timeline scrub, animation cliplets, regression replay, GIF
// and MP4 export, bug-repro zip) that consume the recorded stream.
//
// The recorder lives behind the Capture workspace (U36). When a project
// is loaded, the recorder is running. Older frames evict ring-buffer
// style so the working set stays bounded at CaptureBudgetFrames.
package capture

import (
	"fmt"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	pikey "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	pimouse "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
	pipad "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	pirand "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_rand"
	piring "github.com/ibilalkhan1/fyp_pixelforge/internal/pixelforge_ring"
)

// DefaultBudgetFrames is the recorder's default ring size.
//
// 300 frames × 30 TPS = 10 seconds of history; at 320×180 paletted
// that's ~17 MB worst-case. Configurable via the
// CaptureBudgetFrames setting.
const DefaultBudgetFrames = 300

// Recorder owns the ring buffer of captured Frames and the
// subscriptions that feed it.
//
// Lifecycle:
//   - New(budget) constructs an empty Recorder.
//   - Start() subscribes to piloop + input targets.
//   - SaveFrame() is called from the EventLateDraw subscriber to
//     commit the current tick into the ring.
//   - Stop() unsubscribes; the ring contents are kept (you can still
//     scrub, export, and promote without the recorder running).
//   - Reset() empties the ring but keeps subscriptions alive.
type Recorder struct {
	frames *piring.Buffer[Frame]
	budget int

	tick        int
	frameNumber int

	pendingEvents []EventEntry
	pendingInputs []InputEntry

	loopHandler  pievent.Handler
	mouseBtnH    pievent.Handler
	mouseMoveH   pievent.Handler
	keyH         pievent.Handler
	padBtnH      pievent.Handler
	padConnH     pievent.Handler
	running      bool

	initialSeed uint64
}

// New constructs a Recorder with a ring of `budget` frame slots.
// Panics if budget <= 0 (a zero-size ring has no useful behaviour).
func New(budget int) *Recorder {
	if budget <= 0 {
		panic(fmt.Sprintf("pixelforge capture: budget must be > 0 (got %d)", budget))
	}
	return &Recorder{
		frames: piring.NewBuffer[Frame](budget),
		budget: budget,
	}
}

// Budget returns the current ring capacity in frames.
func (r *Recorder) Budget() int { return r.budget }

// SetBudget rebuilds the ring with a new size, dropping the existing
// contents. Rolling history forward into the new ring is a deferred
// follow-up — the plan calls this out explicitly.
func (r *Recorder) SetBudget(n int) {
	if n <= 0 {
		panic(fmt.Sprintf("pixelforge capture: budget must be > 0 (got %d)", n))
	}
	r.frames = piring.NewBuffer[Frame](n)
	r.budget = n
}

// Start subscribes to the loop + input targets and begins capturing.
// Calling Start twice is a no-op (idempotent).
//
// The recorder snapshots the active pirand seed so the regression
// replayer can rehydrate determinism without needing a separate hook.
func (r *Recorder) Start() {
	if r.running {
		return
	}
	r.initialSeed = pirand.CurrentSeed()
	r.loopHandler = piloop.Target().Subscribe(piloop.EventLateDraw, func(piloop.Event, pievent.Handler) {
		r.SaveFrame()
	})
	r.mouseBtnH = pimouse.ButtonTarget().SubscribeAll(func(e pimouse.EventButton, _ pievent.Handler) {
		r.recordInput("mouse.button", fmt.Sprintf("%s:%v", e.Type, e.Button))
	})
	r.mouseMoveH = pimouse.MoveTarget().SubscribeAll(func(e pimouse.EventMove, _ pievent.Handler) {
		r.recordInput("mouse.move", fmt.Sprintf("%d,%d", e.Position.X, e.Position.Y))
	})
	r.keyH = pikey.Target().SubscribeAll(func(e pikey.Event, _ pievent.Handler) {
		r.recordInput("key", fmt.Sprintf("%s:%v", e.Type, e.Key))
	})
	r.padBtnH = pipad.ButtonTarget().SubscribeAll(func(e pipad.EventButton, _ pievent.Handler) {
		r.recordInput("pad.button", fmt.Sprintf("%s:%v:%d", e.Type, e.Button, e.Player))
	})
	r.padConnH = pipad.ConnectionTarget().SubscribeAll(func(e pipad.EventConnection, _ pievent.Handler) {
		r.recordEvent("pad.connection", fmt.Sprintf("%s:%d", e.Type, e.Player))
	})
	r.running = true
}

// Stop unsubscribes from every target. The recorded frames stay
// available for scrubbing, export, or promotion.
func (r *Recorder) Stop() {
	if !r.running {
		return
	}
	piloop.Target().Unsubscribe(r.loopHandler)
	pimouse.ButtonTarget().Unsubscribe(r.mouseBtnH)
	pimouse.MoveTarget().Unsubscribe(r.mouseMoveH)
	pikey.Target().Unsubscribe(r.keyH)
	pipad.ButtonTarget().Unsubscribe(r.padBtnH)
	pipad.ConnectionTarget().Unsubscribe(r.padConnH)
	r.running = false
}

// Running reports whether subscriptions are currently active.
func (r *Recorder) Running() bool { return r.running }

// Reset empties the ring but keeps subscriptions alive. Frame and tick
// counters reset to 0; the next captured frame starts a fresh sequence.
func (r *Recorder) Reset() {
	r.frames.Reset()
	r.tick = 0
	r.frameNumber = 0
	r.pendingEvents = r.pendingEvents[:0]
	r.pendingInputs = r.pendingInputs[:0]
}

// InitialSeed returns the pirand seed at the moment Start() was last
// called. Regression promotion writes this to seed.txt.
func (r *Recorder) InitialSeed() uint64 { return r.initialSeed }

// SaveFrame commits the current screen state plus pending events and
// inputs as a new ring entry. Public so callers without an
// EventLateDraw hook (tests, deterministic replays, hand-driven
// captures) can drive the recorder directly.
func (r *Recorder) SaveFrame() {
	slot := r.frames.NextWritePointer()
	slot.resetForReuse()

	screen := pixelforge.Screen()
	if slot.Canvas.W() == screen.W() && slot.Canvas.H() == screen.H() {
		slot.Canvas.SetData(screen.Data())
	} else {
		slot.Canvas = screen.Clone()
	}
	slot.Palette = pixelforge.Palette
	slot.PaletteMapping = pixelforge.PaletteMapping
	slot.FrameNumber = r.frameNumber
	slot.TickNumber = r.tick
	slot.Events = append(slot.Events, r.pendingEvents...)
	slot.Inputs = append(slot.Inputs, r.pendingInputs...)

	r.pendingEvents = r.pendingEvents[:0]
	r.pendingInputs = r.pendingInputs[:0]
	r.tick++
	r.frameNumber++
}

func (r *Recorder) recordEvent(target, value string) {
	r.pendingEvents = append(r.pendingEvents, EventEntry{Target: target, Value: value})
}

func (r *Recorder) recordInput(target, value string) {
	r.pendingInputs = append(r.pendingInputs, InputEntry{Target: target, Value: value})
}

// RecordEvent is an exported hook for callers (or tests) that want to
// inject a non-input event into the active tick.
func (r *Recorder) RecordEvent(target, value string) {
	r.recordEvent(target, value)
}

// RecordInput is an exported hook mirroring RecordEvent for input-class
// entries. Tests use it to simulate input replay.
func (r *Recorder) RecordInput(target, value string) {
	r.recordInput(target, value)
}

// Frames returns the currently stored frames in chronological order
// (oldest first). The returned slice is freshly allocated; mutating
// it does not affect the recorder.
func (r *Recorder) Frames() []*Frame {
	n := r.frames.Len()
	out := make([]*Frame, n)
	for i := 0; i < n; i++ {
		out[i] = r.frames.PointerTo(i)
	}
	return out
}

// FrameCount returns the number of frames currently stored.
func (r *Recorder) FrameCount() int { return r.frames.Len() }

// MostRecent returns the most recently captured frame, or nil when the
// ring is empty.
func (r *Recorder) MostRecent() *Frame {
	if r.frames.Len() == 0 {
		return nil
	}
	return r.frames.PointerTo(r.frames.Len() - 1)
}

// FrameAt returns the i-th frame in chronological order, or nil when
// the index is out of range.
func (r *Recorder) FrameAt(i int) *Frame {
	if i < 0 || i >= r.frames.Len() {
		return nil
	}
	return r.frames.PointerTo(i)
}
