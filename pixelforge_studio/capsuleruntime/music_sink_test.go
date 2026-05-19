package capsuleruntime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// noopAudioBackend is a silent stand-in for the ebiten audio
// backend so the music sink can be exercised under `go test` without
// linking the audio runtime. The package's default Backend panics on
// every call; tests that exercise the real Play path must swap it
// for this no-op backend and restore it on teardown.
type noopAudioBackend struct{}

func (noopAudioBackend) LoadSample(*pixelforge_audio.Sample)                          {}
func (noopAudioBackend) UnloadSample(*pixelforge_audio.Sample)                        {}
func (noopAudioBackend) SetSample(pixelforge_audio.Chan, *pixelforge_audio.Sample, int, float64) {}
func (noopAudioBackend) SetLoop(pixelforge_audio.Chan, int, int, pixelforge_audio.LoopType, float64) {
}
func (noopAudioBackend) SetPitch(pixelforge_audio.Chan, float64, float64)  {}
func (noopAudioBackend) SetVolume(pixelforge_audio.Chan, float64, float64) {}
func (noopAudioBackend) ClearChan(pixelforge_audio.Chan, float64)          {}
func (noopAudioBackend) ChannelActive(pixelforge_audio.Chan) bool          { return false }
func (noopAudioBackend) ChannelPosition(pixelforge_audio.Chan) float64     { return 0 }
func (noopAudioBackend) ChannelPitch(pixelforge_audio.Chan) float64        { return 1 }
func (noopAudioBackend) ChannelVolume(pixelforge_audio.Chan) float64       { return 1 }
func (noopAudioBackend) ChannelSample(pixelforge_audio.Chan) *pixelforge_audio.Sample {
	return nil
}

// withNoopAudio swaps the package-global audio backend for the
// no-op backend and restores the original on test cleanup.
func withNoopAudio(t *testing.T) {
	t.Helper()
	prev := pixelforge_audio.Backend
	pixelforge_audio.Backend = noopAudioBackend{}
	t.Cleanup(func() { pixelforge_audio.Backend = prev })
}

func TestMusicSink_PlayRegistersCurrentMusic(t *testing.T) {
	withNoopAudio(t)
	rt := bootDefault(t, pixelforge_project.NewProject("music"))
	pixelforge_audio.RegisterSample("level_theme",
		pixelforge_audio.NewSample([]int8{0, 1, 2, 3}, 8000))

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_music",
		Args:  map[string]any{"name": "level_theme"},
	})

	assert.Equal(t, "level_theme", rt.CurrentMusic)
}

func TestMusicSink_StopClearsCurrentMusic(t *testing.T) {
	withNoopAudio(t)
	rt := bootDefault(t, pixelforge_project.NewProject("music"))
	pixelforge_audio.RegisterSample("level_theme",
		pixelforge_audio.NewSample([]int8{0, 1, 2, 3}, 8000))

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_music",
		Args:  map[string]any{"name": "level_theme"},
	})
	require.Equal(t, "level_theme", rt.CurrentMusic)

	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: "audio/stop_music"})
	assert.Equal(t, "", rt.CurrentMusic)
}

func TestMusicSink_UnknownTrackClearsCurrentMusic(t *testing.T) {
	withNoopAudio(t)
	rt := bootDefault(t, pixelforge_project.NewProject("music"))
	// Pre-set current music; the unknown lookup should clear it
	// so the sink stays honest about what's playing.
	rt.CurrentMusic = "stale"

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_music",
		Args:  map[string]any{"name": "missing"},
	})
	assert.Equal(t, "", rt.CurrentMusic)
}

func TestMusicSink_EmptyNameNoOp(t *testing.T) {
	withNoopAudio(t)
	rt := bootDefault(t, pixelforge_project.NewProject("music"))
	rt.CurrentMusic = "level_theme"

	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "audio/play_music",
		Args:  map[string]any{},
	})
	// No mutation when args carry no name.
	assert.Equal(t, "level_theme", rt.CurrentMusic)
}
