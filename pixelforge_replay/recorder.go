package pixelforge_replay

import (
	"io"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_render"
)

// Recorder captures per-tick InputFrames from a running pixelforge
// host (studio preview, headless replay capture, manual test
// harness) into an in-memory TraceFrame slice that Flush serialises
// to a .trace.jsonl stream.
//
// Recorder is intentionally not goroutine-safe: a single capture
// session writes Tick calls from one goroutine (the render thread),
// then later Flushes from the same goroutine when capture stops.
// Concurrent Tick / Flush is undefined.
//
// Zero-value Recorder is NOT valid — callers must build one via
// NewRecorder so the meta header is non-empty when Flush emits it.
type Recorder struct {
	meta   TraceMeta
	frames []TraceFrame
	tick   uint64
}

// NewRecorder returns a Recorder initialised with the supplied meta
// header. The Recorder's internal tick counter starts at 0; each
// call to Tick(inputs) appends a TraceFrame at the current counter
// and increments it. Callers who need to record at a non-zero
// starting tick should call Tick the right number of times with
// empty inputs OR construct frames manually and call AppendFrame.
//
// The supplied meta.DurationTicks is informational only — Flush
// overwrites it with the recorded frame count so a Recorder that
// stops mid-session still produces a faithful header.
func NewRecorder(meta TraceMeta) *Recorder {
	return &Recorder{meta: meta}
}

// Tick appends a TraceFrame for the current tick (using the
// Recorder's internal counter) and advances the counter by one.
// The InputFrame's Keys and Pad are deep-copied into the appended
// frame so subsequent mutation by the caller does not leak into
// the recording.
//
// Empty InputFrame{} (nil Keys, nil Pad) is valid: it records a
// no-input tick that the encoder serialises as `"keys":[],"pad":null`.
func (r *Recorder) Tick(inputs pixelforge_render.InputFrame) {
	frame := TraceFrame{
		Tick: r.tick,
		Keys: cloneInputKeys(inputs.Keys),
		Pad:  clonePad(inputs.Pad),
	}
	r.frames = append(r.frames, frame)
	r.tick++
}

// AppendFrame inserts a fully-formed TraceFrame into the recording
// at its embedded Tick. Used by tests or by a recorder that wants
// to bypass the internal counter for non-monotonic captures (e.g.
// resuming a paused session). The Recorder's internal counter is
// NOT advanced by AppendFrame — the caller owns the next tick
// number entirely.
func (r *Recorder) AppendFrame(f TraceFrame) {
	r.frames = append(r.frames, TraceFrame{
		Tick: f.Tick,
		Keys: cloneKeys(f.Keys),
		Pad:  clonePad(f.Pad),
	})
}

// Frames returns a defensive copy of the recorded frame slice. The
// returned slice is owned by the caller; mutation does not affect
// future Tick calls. Used by tests to inspect the in-flight
// recording without going through Flush + LoadTrace.
func (r *Recorder) Frames() []TraceFrame {
	out := make([]TraceFrame, len(r.frames))
	for i, f := range r.frames {
		out[i] = TraceFrame{
			Tick: f.Tick,
			Keys: cloneKeys(f.Keys),
			Pad:  clonePad(f.Pad),
		}
	}
	return out
}

// Trace returns the in-progress recording as a *Trace, useful when
// the caller wants to drive a Replayer directly without round-
// tripping through the .trace.jsonl encoding. The returned Trace's
// Meta.DurationTicks is set to len(frames) so it matches what Flush
// would emit.
func (r *Recorder) Trace() *Trace {
	meta := r.meta
	meta.DurationTicks = uint64(len(r.frames))
	return &Trace{Meta: meta, Frames: r.Frames()}
}

// Flush serialises the recorded session to w as a .trace.jsonl
// stream using the same encoding as (*Trace).Encode. The meta
// header's DurationTicks is updated to the actual recorded count
// before emission.
//
// Flush does not close w (the caller owns lifetime of the writer).
// It does not reset the Recorder either — call NewRecorder for a
// fresh session.
func (r *Recorder) Flush(w io.Writer) error {
	meta := r.meta
	meta.DurationTicks = uint64(len(r.frames))
	t := &Trace{Meta: meta, Frames: r.frames}
	return t.Encode(w)
}

// cloneInputKeys is the InputFrame-side analogue of cloneKeys in
// trace.go. Defined separately because InputFrame.Keys is the
// ebiten.Key slice type the render package uses; the helper keeps
// the imports tidy.
func cloneInputKeys(in []ebiten.Key) []ebiten.Key {
	if in == nil {
		return nil
	}
	out := make([]ebiten.Key, len(in))
	copy(out, in)
	return out
}
