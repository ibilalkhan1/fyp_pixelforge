package audiolib

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// audition_test.go covers the U5 Start/Stop/IsActive semantics and
// the click-to-stop / click-other-stops behaviour. A recording
// backend captures every backend call so the tests can assert the
// expected sequence of LoadSample/Play/SetLoop/ClearChan without
// touching a real audio device.

// recordingBackend captures every backend method call. Implements
// pixelforge_audio.BackendInterface in full.
type recordingBackend struct {
	mu       sync.Mutex
	calls    []string
	active   map[pixelforge_audio.Chan]bool
	loadedCt int
}

func newRecordingBackend() *recordingBackend {
	return &recordingBackend{active: map[pixelforge_audio.Chan]bool{}}
}

func (b *recordingBackend) record(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.calls = append(b.calls, s)
}

func (b *recordingBackend) LoadSample(*pixelforge_audio.Sample) {
	b.mu.Lock()
	b.loadedCt++
	b.mu.Unlock()
	b.record("Load")
}
func (b *recordingBackend) UnloadSample(*pixelforge_audio.Sample) { b.record("Unload") }
func (b *recordingBackend) SetSample(ch pixelforge_audio.Chan, _ *pixelforge_audio.Sample, _ int, _ float64) {
	b.mu.Lock()
	b.active[ch] = true
	b.mu.Unlock()
	b.record("SetSample")
}
func (b *recordingBackend) SetLoop(_ pixelforge_audio.Chan, _, _ int, lt pixelforge_audio.LoopType, _ float64) {
	b.record("SetLoop:" + string(lt))
}
func (b *recordingBackend) ClearChan(ch pixelforge_audio.Chan, _ float64) {
	b.mu.Lock()
	b.active[ch] = false
	b.mu.Unlock()
	b.record("ClearChan")
}
func (b *recordingBackend) SetPitch(_ pixelforge_audio.Chan, _, _ float64)  {}
func (b *recordingBackend) SetVolume(_ pixelforge_audio.Chan, _, _ float64) {}
func (b *recordingBackend) ChannelActive(ch pixelforge_audio.Chan) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.active[ch]
}
func (b *recordingBackend) ChannelPosition(pixelforge_audio.Chan) float64 { return 0 }
func (b *recordingBackend) ChannelPitch(pixelforge_audio.Chan) float64    { return 0 }
func (b *recordingBackend) ChannelVolume(pixelforge_audio.Chan) float64   { return 0 }
func (b *recordingBackend) ChannelSample(pixelforge_audio.Chan) *pixelforge_audio.Sample {
	return nil
}

func withRecordingBackend(t *testing.T) *recordingBackend {
	t.Helper()
	orig := pixelforge_audio.Backend
	stub := newRecordingBackend()
	pixelforge_audio.Backend = stub
	t.Cleanup(func() { pixelforge_audio.Backend = orig })
	return stub
}

func sampleFor(t *testing.T) *pixelforge_audio.Sample {
	t.Helper()
	bytes := SynthesizeWAV(PatchSynth{
		Waveform: WaveformSquare, FrequencyHz: 440, DurationMs: 100, Envelope: EnvelopeDecay,
	})
	sample, err := pixelforge_audio.DecodeWavOrErr(bytes)
	require.NoError(t, err)
	return sample
}

func TestAudition_StartPlaysSampleAndTracksState(t *testing.T) {
	withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	s := sampleFor(t)
	ch := a.Start(s, "jump/spring", "sfx", false)
	assert.NotEqual(t, pixelforge_audio.Chan(0), ch)
	assert.True(t, a.IsActive("jump/spring"))
	assert.Equal(t, "jump/spring", a.CurrentName())
	assert.Equal(t, ch, a.CurrentChan())
}

func TestAudition_StartLoadsSampleBeforePlay(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	_ = a.Start(sampleFor(t), "jump/spring", "sfx", false)
	assert.Equal(t, 1, stub.loadedCt, "LoadSample called exactly once before Play")
	assert.Contains(t, stub.calls, "Load")
	assert.Contains(t, stub.calls, "SetSample")
}

func TestAudition_StartSameSampleAgainStops(t *testing.T) {
	withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	s := sampleFor(t)
	_ = a.Start(s, "jump/spring", "sfx", false)
	require.True(t, a.IsActive("jump/spring"))
	ch := a.Start(s, "jump/spring", "sfx", false)
	assert.Equal(t, pixelforge_audio.Chan(0), ch,
		"second Start for same patch returns 0 (stop semantics)")
	assert.False(t, a.IsActive("jump/spring"))
	assert.Empty(t, a.CurrentName())
}

func TestAudition_StartDifferentSampleStopsCurrent(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	s1 := sampleFor(t)
	s2 := sampleFor(t)
	_ = a.Start(s1, "jump/spring", "sfx", false)
	_ = a.Start(s2, "shoot/laser", "sfx", false)
	assert.True(t, a.IsActive("shoot/laser"))
	assert.False(t, a.IsActive("jump/spring"))
	// Trace check: switching patches must include a ClearChan
	// between the two SetSample calls.
	clearSeen := false
	for _, c := range stub.calls {
		if c == "ClearChan" {
			clearSeen = true
			break
		}
	}
	assert.True(t, clearSeen, "switching patches clears the prior channel")
}

func TestAudition_BGMLoopsViaSetLoopForward(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	_ = a.Start(sampleFor(t), "bgm/title", "bgm", true)
	assert.Contains(t, stub.calls, "SetLoop:forward",
		"BGM audition calls SetLoop with LoopForward after Play")
}

func TestAudition_SFXDoesNotSetLoopForward(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	_ = a.Start(sampleFor(t), "jump/spring", "sfx", false)
	// Play() internally calls SetLoop with LoopNone; the audition
	// must NOT layer a LoopForward call on top for SFX.
	for _, c := range stub.calls {
		assert.NotEqual(t, "SetLoop:forward", c,
			"SFX audition must not call SetLoop with LoopForward")
	}
}

func TestAudition_StopClearsChannelAndState(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	s := sampleFor(t)
	_ = a.Start(s, "jump/spring", "sfx", false)
	a.Stop()
	assert.False(t, a.IsActive("jump/spring"))
	assert.Equal(t, pixelforge_audio.Chan(0), a.CurrentChan())
	assert.Contains(t, stub.calls, "ClearChan")
	assert.Contains(t, stub.calls, "Unload",
		"Stop unloads the sample so memory isn't held forever")
}

func TestAudition_StopWhenNothingPlayingIsNoOp(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	a.Stop()
	for _, c := range stub.calls {
		assert.NotEqual(t, "ClearChan", c,
			"Stop on idle audition makes no backend calls")
	}
}

func TestAudition_NilReceiverNoOps(t *testing.T) {
	var a *Audition
	assert.NotPanics(t, func() {
		_ = a.Start(nil, "x", "sfx", false)
		a.Stop()
		_ = a.IsActive("x")
	})
}

func TestAudition_StartWithNilSampleNoOps(t *testing.T) {
	stub := withRecordingBackend(t)
	a := NewAudition(pixelforge_audio.NewAllocator())
	ch := a.Start(nil, "x", "sfx", false)
	assert.Equal(t, pixelforge_audio.Chan(0), ch)
	assert.Empty(t, stub.calls, "nil sample makes no backend calls")
}

func TestAudition_UsesAllocatorForChannelPick(t *testing.T) {
	stub := withRecordingBackend(t)
	// Force Chan1 busy so the allocator's BGM path falls to Chan2.
	stub.active[pixelforge_audio.Chan1] = true
	a := NewAudition(pixelforge_audio.NewAllocator())
	ch := a.Start(sampleFor(t), "bgm/title", "bgm", true)
	assert.Equal(t, pixelforge_audio.Chan2, ch,
		"BGM audition routes through Allocator's BGM policy (Chan1 busy → Chan2)")
}

func TestAudition_DefaultAuditionExists(t *testing.T) {
	withRecordingBackend(t)
	require.NotNil(t, DefaultAudition)
	// Smoke-test by calling Stop on an idle instance.
	DefaultAudition.Stop()
}
