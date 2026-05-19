// allocator.go owns idea #4 v1's Paula channel allocator. The
// allocator reads AudioSample.SuggestedChannelPriority and returns
// the channel Play should target — without it, every caller would
// have to hard-code a channel choice and AudioBinding.ForceChannel
// wouldn't have a routing seam.
//
// Policy summary (matches origin R10):
//   - ForceChannel in 1..4 overrides everything.
//   - "bgm" → Chan1 preferred, Chan2 fallback, steal Chan1 when both
//     are busy (brief glitch on swap; new BGM "takes over").
//   - "sfx", "voice", "ambient", "" → round-robin Chan3/Chan4; skip
//     a busy channel; steal the next round-robin slot when both
//     busy (SFX interrupts SFX, matching NES feel).
//
// The allocator is pure: Pick returns a Chan, never calls Play or
// mutates backend state. Callers (audition, runtime Play step)
// invoke Play themselves so the side-effects stay traceable.
package pixelforge_audio

import "sync"

// Allocator carries the round-robin index for SFX channels. Single
// allocator per process is the v1 design — matches the singleton
// Paula mixer. A per-project allocator is a v2 refinement.
type Allocator struct {
	mu       sync.Mutex
	sfxNext  Chan // next SFX channel to consider in round-robin (Chan3 or Chan4)
	bgmNext  Chan // next BGM channel to steal when both are busy
}

// NewAllocator returns a fresh allocator with round-robin pointing
// at Chan3 (SFX) and Chan1 (BGM). Reset() restores this state.
func NewAllocator() *Allocator {
	return &Allocator{sfxNext: Chan3, bgmNext: Chan1}
}

// Reset clears round-robin state. Tests call this between cases so
// the SFX cycle starts fresh; production code can call it at scene
// transitions if needed.
func (a *Allocator) Reset() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sfxNext = Chan3
	a.bgmNext = Chan1
}

// Pick returns the channel Play should target for the supplied
// (priority, forceChannel) combination. priority is the
// AudioSample.SuggestedChannelPriority string ("bgm", "sfx",
// "voice", "ambient", or empty); forceChannel is the
// AudioBinding.ForceChannel value (1..4 to override, 0 for auto).
//
// Channel-activity queries route through ChannelActive(Chan), which
// hits the package-level Backend. Tests substitute a fake Backend
// to drive specific busy/idle scenarios.
func (a *Allocator) Pick(priority string, forceChannel int) Chan {
	if a == nil {
		return Chan3
	}
	// Force-channel override: when the binding pins a channel, the
	// allocator just returns it. Invalid pins (outside 1..4) fall
	// through to the priority-based path.
	if forced := forcedChannel(forceChannel); forced != 0 {
		return forced
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	switch priority {
	case "bgm":
		return a.pickBGMLocked()
	default:
		// "sfx", "voice", "ambient", empty — all SFX-class.
		return a.pickSFXLocked()
	}
}

// PickForSample is the convenience wrapper that reads
// SuggestedChannelPriority off an AudioSample. Existing callers
// that have the sample handy use this; callers with only the
// metadata string use Pick directly.
//
// AudioSample lives in pixelforge_project; we don't import that
// here (it imports us via the schema doc). The function takes the
// raw fields by name instead.
func (a *Allocator) PickForSample(suggestedPriority string, forceChannel int) Chan {
	return a.Pick(suggestedPriority, forceChannel)
}

// pickBGMLocked implements the BGM policy: prefer Chan1, fall back
// to Chan2, otherwise steal Chan1 (oldest BGM). The bgmNext field
// tracks which BGM channel was stolen last so a future "steal Chan2
// next time" policy is easy to layer on; v1 always steals Chan1.
func (a *Allocator) pickBGMLocked() Chan {
	if !ChannelActive(Chan1) {
		return Chan1
	}
	if !ChannelActive(Chan2) {
		return Chan2
	}
	// Both busy — steal Chan1. Future v2 might rotate
	// stealing between Chan1 and Chan2; the bgmNext field
	// reserves that knob without committing to it.
	a.bgmNext = Chan1
	return Chan1
}

// pickSFXLocked implements the SFX policy: round-robin Chan3/Chan4
// preferring idle channels. When both are busy, pick the next
// round-robin slot (steal-by-rotation).
func (a *Allocator) pickSFXLocked() Chan {
	first := a.sfxNext
	second := siblingSFXChannel(first)

	if !ChannelActive(first) {
		a.advanceSFXLocked()
		return first
	}
	if !ChannelActive(second) {
		// The idle slot wins the pick; the round-robin index also
		// advances past it so the next call alternates.
		a.sfxNext = first
		a.advanceSFXLocked()
		return second
	}
	// Both busy — steal the round-robin slot.
	a.advanceSFXLocked()
	return first
}

func (a *Allocator) advanceSFXLocked() {
	if a.sfxNext == Chan3 {
		a.sfxNext = Chan4
	} else {
		a.sfxNext = Chan3
	}
}

// siblingSFXChannel returns the other SFX channel — Chan4 when
// given Chan3, Chan3 when given Chan4.
func siblingSFXChannel(c Chan) Chan {
	if c == Chan3 {
		return Chan4
	}
	return Chan3
}

// forcedChannel converts an int 1..4 to the corresponding Chan
// constant. Returns 0 for any value outside that range so the
// caller falls through to priority-based picking.
func forcedChannel(n int) Chan {
	switch n {
	case 1:
		return Chan1
	case 2:
		return Chan2
	case 3:
		return Chan3
	case 4:
		return Chan4
	}
	return 0
}

// DefaultAllocator is the process-wide allocator the studio and the
// shipped runtime both use. Lazy-initialised on first access so
// tests that swap Backend can still construct their own allocator.
var DefaultAllocator = NewAllocator()
