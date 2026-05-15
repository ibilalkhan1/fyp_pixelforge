package capture

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
)

// EventEntry records one event published on any tap-eligible
// pievent.Target during the tick that produced the parent Frame.
//
// Target is the symbolic name of the target ("loop", "mouse.button",
// "key", etc.) so the replayer doesn't need access to the original
// *pievent.Target instance. Value is a string serialization of the
// event payload (enough to drive replay; full structural fidelity is
// a deferred follow-up).
type EventEntry struct {
	Target string
	Value  string
}

// InputEntry mirrors EventEntry but is reserved for input-class
// targets (mouse/key/pad). The replayer rebroadcasts these to drive
// gameplay against recorded inputs.
type InputEntry struct {
	Target string
	Value  string
}

// Frame is a captured snapshot of one game tick.
//
// Canvas is a paletted copy of the screen at the moment EventLateDraw
// fired; Palette/PaletteMapping are the engine state needed to render
// Canvas back faithfully. Events and Inputs are the publish log for the
// tick.
//
// The recorder reuses Frame instances (ring buffer of pre-allocated
// frames) — callers must not retain pointers into a Frame's slices
// across writes.
type Frame struct {
	Canvas         pixelforge.Canvas
	Palette        pixelforge.PaletteArray
	PaletteMapping pixelforge.PaletteMap
	FrameNumber    int
	TickNumber     int
	Events         []EventEntry
	Inputs         []InputEntry
}

// resetForReuse clears the per-tick slices but keeps the canvas
// allocation so the ring buffer can avoid re-allocating on each write.
func (f *Frame) resetForReuse() {
	f.Events = f.Events[:0]
	f.Inputs = f.Inputs[:0]
}
