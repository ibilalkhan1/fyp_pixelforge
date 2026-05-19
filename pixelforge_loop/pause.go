// pause.go owns idea #6 v1 U3's scene-pause primitive. The Pause /
// Resume pair toggles a package-level gate that the per-frame
// publish path consults: when paused, EventUpdate + EventLateUpdate
// dispatches are suppressed (so behavior graphs, event-sheet rules,
// and entity ticks stop), while EventDraw, EventLateDraw, input,
// and frame_start continue (so overlays render + respond and the
// scene preview stays live).
//
// Studio-side menu code (idea #6 v1 U7's MenuStack pause-on-overlay
// hook) calls Pause when an overlay opens and Resume when the last
// overlay closes. The shipped runtime's MenuStack does the same;
// because the gate is package-scoped on pixelforge_loop (which both
// the studio and the runtime import), the behavior is identical
// on both sides.
package pixelforge_loop

import "sync/atomic"

// paused is the package-level gate. atomic uint32 so the per-frame
// publish path doesn't need a mutex round trip.
var paused atomic.Bool

// Pause freezes EventUpdate / EventLateUpdate dispatch through
// Target() (and its debug sibling). Idempotent. Concurrency-safe.
func Pause() {
	paused.Store(true)
}

// Resume releases the gate. Idempotent. Concurrency-safe.
func Resume() {
	paused.Store(false)
}

// IsPaused reports the current gate state. Tests + the menu stack
// consult this to decide whether an open overlay should call Pause
// or whether it's already in pause state.
func IsPaused() bool {
	return paused.Load()
}

// SuppressedWhilePaused reports whether the supplied event is
// blocked by the current pause gate. EventUpdate + EventLateUpdate
// suppress when paused; everything else passes through. Exposed so
// the publish-site callers (pixelforge_ebiten/internal/ebitengame
// and the studio's sceneGame loop) can early-return without
// constructing the full event payload.
func SuppressedWhilePaused(e Event) bool {
	if !paused.Load() {
		return false
	}
	switch e {
	case EventUpdate, EventLateUpdate:
		return true
	}
	return false
}
