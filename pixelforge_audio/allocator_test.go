package pixelforge_audio_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// activityBackend is a stub Backend whose ChannelActive return
// values are configured per-channel for allocator tests. Embeds
// fakeBackend's parent type for the other (unused) methods.
type activityBackend struct {
	pixelforge_audio.BackendInterface

	mu     sync.Mutex
	active map[pixelforge_audio.Chan]bool
}

func newActivityBackend() *activityBackend {
	return &activityBackend{active: map[pixelforge_audio.Chan]bool{}}
}

func (b *activityBackend) ChannelActive(ch pixelforge_audio.Chan) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.active[ch]
}

func (b *activityBackend) setActive(ch pixelforge_audio.Chan, on bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.active[ch] = on
}

// withActivityBackend swaps in the activity-tracking backend for
// the duration of the test. Restored on cleanup.
func withActivityBackend(t *testing.T) *activityBackend {
	t.Helper()
	original := pixelforge_audio.Backend
	stub := newActivityBackend()
	pixelforge_audio.Backend = stub
	t.Cleanup(func() {
		pixelforge_audio.Backend = original
	})
	return stub
}

// TestAllocator_ForceChannelOverridesPriority: a non-zero
// forceChannel returns the forced channel directly, ignoring
// priority.
func TestAllocator_ForceChannelOverridesPriority(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan4, a.Pick("bgm", 4))
	assert.Equal(t, pixelforge_audio.Chan2, a.Pick("sfx", 2))
}

// TestAllocator_BGMPicksChan1WhenIdle: bgm priority with both BGM
// channels idle returns Chan1.
func TestAllocator_BGMPicksChan1WhenIdle(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan1, a.Pick("bgm", 0))
}

// TestAllocator_BGMPicksChan2WhenChan1Busy: Chan1 busy + Chan2
// idle returns Chan2.
func TestAllocator_BGMPicksChan2WhenChan1Busy(t *testing.T) {
	stub := withActivityBackend(t)
	stub.setActive(pixelforge_audio.Chan1, true)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan2, a.Pick("bgm", 0))
}

// TestAllocator_BGMStealsChan1WhenBothBusy: both BGM channels busy
// → steal Chan1 (v1 policy).
func TestAllocator_BGMStealsChan1WhenBothBusy(t *testing.T) {
	stub := withActivityBackend(t)
	stub.setActive(pixelforge_audio.Chan1, true)
	stub.setActive(pixelforge_audio.Chan2, true)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan1, a.Pick("bgm", 0))
}

// TestAllocator_SFXRoundRobinAcrossChan3Chan4: both SFX channels
// idle; successive picks alternate Chan3 → Chan4 → Chan3.
func TestAllocator_SFXRoundRobinAcrossChan3Chan4(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan3, a.Pick("sfx", 0))
	assert.Equal(t, pixelforge_audio.Chan4, a.Pick("sfx", 0))
	assert.Equal(t, pixelforge_audio.Chan3, a.Pick("sfx", 0))
}

// TestAllocator_SFXSkipsBusyChannel: when the round-robin slot is
// busy and the sibling is idle, the sibling wins.
func TestAllocator_SFXSkipsBusyChannel(t *testing.T) {
	stub := withActivityBackend(t)
	stub.setActive(pixelforge_audio.Chan3, true)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan4, a.Pick("sfx", 0))
}

// TestAllocator_SFXStealsRoundRobinSlotWhenBothBusy: both SFX
// channels busy → steal the round-robin slot.
func TestAllocator_SFXStealsRoundRobinSlotWhenBothBusy(t *testing.T) {
	stub := withActivityBackend(t)
	stub.setActive(pixelforge_audio.Chan3, true)
	stub.setActive(pixelforge_audio.Chan4, true)
	a := pixelforge_audio.NewAllocator()
	assert.Equal(t, pixelforge_audio.Chan3, a.Pick("sfx", 0),
		"first steal targets the round-robin slot (Chan3)")
	assert.Equal(t, pixelforge_audio.Chan4, a.Pick("sfx", 0),
		"second steal advances the round-robin")
}

// TestAllocator_VoiceAndAmbientBehaveLikeSFX: non-"bgm" priority
// strings all route through the SFX path.
func TestAllocator_VoiceAndAmbientBehaveLikeSFX(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	got1 := a.Pick("voice", 0)
	got2 := a.Pick("ambient", 0)
	// Each pick should land on Chan3 or Chan4.
	assert.Contains(t, []pixelforge_audio.Chan{pixelforge_audio.Chan3, pixelforge_audio.Chan4}, got1)
	assert.Contains(t, []pixelforge_audio.Chan{pixelforge_audio.Chan3, pixelforge_audio.Chan4}, got2)
}

// TestAllocator_EmptyPriorityDefaultsToSFX: empty priority string
// (designer hasn't set it) routes to SFX channels.
func TestAllocator_EmptyPriorityDefaultsToSFX(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	got := a.Pick("", 0)
	assert.Contains(t, []pixelforge_audio.Chan{pixelforge_audio.Chan3, pixelforge_audio.Chan4}, got)
}

// TestAllocator_ResetClearsRoundRobin: after a few picks, Reset
// returns the round-robin index to Chan3.
func TestAllocator_ResetClearsRoundRobin(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	_ = a.Pick("sfx", 0)
	_ = a.Pick("sfx", 0)
	_ = a.Pick("sfx", 0)
	a.Reset()
	assert.Equal(t, pixelforge_audio.Chan3, a.Pick("sfx", 0),
		"first pick after Reset returns Chan3 again")
}

// TestAllocator_InvalidForceChannelTreatedAsAuto: forceChannel
// outside 1..4 falls through to the priority-based path.
func TestAllocator_InvalidForceChannelTreatedAsAuto(t *testing.T) {
	withActivityBackend(t)
	a := pixelforge_audio.NewAllocator()
	got := a.Pick("bgm", 99)
	assert.Equal(t, pixelforge_audio.Chan1, got,
		"forceChannel=99 ignored; priority='bgm' returns Chan1")
}

// TestAllocator_PickDoesNotMutateChannelState: the allocator must
// not flip any ChannelActive value or call Play. Confirmed by
// checking activity before and after — the stub's setActive is the
// only path to flipping state.
func TestAllocator_PickDoesNotMutateChannelState(t *testing.T) {
	stub := withActivityBackend(t)
	stub.setActive(pixelforge_audio.Chan3, true)
	a := pixelforge_audio.NewAllocator()
	pre := stub.ChannelActive(pixelforge_audio.Chan4)
	_ = a.Pick("sfx", 0)
	post := stub.ChannelActive(pixelforge_audio.Chan4)
	assert.Equal(t, pre, post,
		"Pick is pure: it never flips any channel's activity flag")
}

// TestAllocator_NilSafe: Pick on a nil receiver returns Chan3 as a
// safe default rather than panicking.
func TestAllocator_NilSafe(t *testing.T) {
	withActivityBackend(t)
	var a *pixelforge_audio.Allocator
	assert.Equal(t, pixelforge_audio.Chan3, a.Pick("sfx", 0))
}

// TestAllocator_DefaultAllocatorExists: the package-level singleton
// is non-nil and usable out of the box.
func TestAllocator_DefaultAllocatorExists(t *testing.T) {
	withActivityBackend(t)
	require.NotNil(t, pixelforge_audio.DefaultAllocator)
	got := pixelforge_audio.DefaultAllocator.Pick("bgm", 0)
	assert.Equal(t, pixelforge_audio.Chan1, got)
	pixelforge_audio.DefaultAllocator.Reset()
}
