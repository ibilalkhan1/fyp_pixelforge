// audition.go owns idea #4 v1 U5's preview-playback helper. The
// library picker's Play button calls Audition.Start; clicking the
// same patch again calls Stop. BGM patches loop automatically;
// SFX play once. Click-on-another-patch stops the current and
// starts the new one.
//
// The audition path goes through the same Paula mixer the shipped
// runtime uses — per always-on-game-embedding.md, there's no
// separate preview engine. The Allocator (U2) picks the channel
// so audition behaves like the runtime: BGM lands on Chan1/Chan2,
// SFX round-robins Chan3/Chan4. What the designer hears in the
// studio is what ships.
package audiolib

import (
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// Audition tracks which sample (if any) is currently playing as a
// preview. Singleton on the audiolib package — there's only one
// Paula mixer so there's only one audition.
type Audition struct {
	mu sync.Mutex

	allocator *pixelforge_audio.Allocator

	currentSample *pixelforge_audio.Sample
	currentChan   pixelforge_audio.Chan
	currentName   string
	isLoop        bool
}

// NewAudition returns a fresh audition state bound to the supplied
// allocator. Tests pass their own allocator so they don't disturb
// the package-level DefaultAllocator.
func NewAudition(alloc *pixelforge_audio.Allocator) *Audition {
	if alloc == nil {
		alloc = pixelforge_audio.DefaultAllocator
	}
	return &Audition{allocator: alloc}
}

// DefaultAudition is the singleton instance the picker panel (U6)
// reaches for at runtime. Tests construct their own via NewAudition
// to keep state isolated.
var DefaultAudition = NewAudition(nil)

// Start kicks off playback for the supplied sample. If the same
// sample is already auditioning, Start stops it (click-to-stop
// semantics). If a DIFFERENT sample is auditioning, Start stops
// that one and begins the new one. Returns the channel the
// allocator picked (or 0 if the audition was a stop-of-current).
//
// suggestedPriority is the AudioSample.SuggestedChannelPriority
// the allocator routes on. patchName is the picker's identifier so
// IsActive can answer "is patch X playing right now."
func (a *Audition) Start(sample *pixelforge_audio.Sample, patchName, suggestedPriority string, isLoop bool) pixelforge_audio.Chan {
	if a == nil || sample == nil {
		return 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// Click-to-stop: same patch name pressed again → stop and
	// return 0 to signal "no new audition started."
	if a.currentSample != nil && a.currentName == patchName {
		a.stopLocked()
		return 0
	}

	// Different patch (or first audition this session) — stop the
	// current one if any, then start the new one.
	if a.currentSample != nil {
		a.stopLocked()
	}

	ch := a.allocator.Pick(suggestedPriority, 0)
	pixelforge_audio.LoadSample(sample)
	pixelforge_audio.Play(ch, sample, 1.0, 1.0)
	if isLoop {
		pixelforge_audio.SetLoop(ch, 0, sample.Len(), pixelforge_audio.LoopForward, 0)
	}

	a.currentSample = sample
	a.currentChan = ch
	a.currentName = patchName
	a.isLoop = isLoop
	return ch
}

// Stop cancels the active audition. No-op when nothing is playing.
func (a *Audition) Stop() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopLocked()
}

func (a *Audition) stopLocked() {
	if a.currentSample == nil {
		return
	}
	pixelforge_audio.ClearChan(a.currentChan, 0)
	pixelforge_audio.UnloadSample(a.currentSample)
	a.currentSample = nil
	a.currentChan = 0
	a.currentName = ""
	a.isLoop = false
}

// IsActive reports whether the supplied patch name is the
// currently-auditioning sample. The picker uses this to decide
// whether the Play button should render as ▶ (start) or ■ (stop).
func (a *Audition) IsActive(patchName string) bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentSample != nil && a.currentName == patchName
}

// CurrentName returns the name of the auditioning patch, or "" when
// no audition is active. Sibling to IsActive for tests that don't
// want to pass the name they're checking against.
func (a *Audition) CurrentName() string {
	if a == nil {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentName
}

// CurrentChan returns the Chan the auditioning sample is playing
// on, or 0 when nothing is active. Exposed so the picker's status
// bar can render "playing on Chan3."
func (a *Audition) CurrentChan() pixelforge_audio.Chan {
	if a == nil {
		return 0
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentChan
}
