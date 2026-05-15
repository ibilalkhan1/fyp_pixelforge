package pixelforge_audio_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// fakeBackend is a minimal in-memory backend used to verify the
// package-level query wrappers forward to BackendInterface correctly.
type fakeBackend struct {
	pixelforge_audio.BackendInterface

	channels map[pixelforge_audio.Chan]channelState
}

type channelState struct {
	active   bool
	pos      float64
	pitch    float64
	volume   float64
	sample   *pixelforge_audio.Sample
}

func (b *fakeBackend) ChannelActive(ch pixelforge_audio.Chan) bool {
	return b.channels[ch].active
}
func (b *fakeBackend) ChannelPosition(ch pixelforge_audio.Chan) float64 {
	return b.channels[ch].pos
}
func (b *fakeBackend) ChannelPitch(ch pixelforge_audio.Chan) float64 {
	return b.channels[ch].pitch
}
func (b *fakeBackend) ChannelVolume(ch pixelforge_audio.Chan) float64 {
	return b.channels[ch].volume
}
func (b *fakeBackend) ChannelSample(ch pixelforge_audio.Chan) *pixelforge_audio.Sample {
	return b.channels[ch].sample
}

func TestChannelQuery_Wrappers(t *testing.T) {
	original := pixelforge_audio.Backend
	defer func() { pixelforge_audio.Backend = original }()

	sample := pixelforge_audio.NewSample([]int8{1, 2, 3}, 22050)
	pixelforge_audio.Backend = &fakeBackend{
		channels: map[pixelforge_audio.Chan]channelState{
			pixelforge_audio.Chan1: {active: true, pos: 1.5, pitch: 1.0, volume: 0.75, sample: sample},
			pixelforge_audio.Chan2: {active: false},
		},
	}

	t.Run("returns set values for active channel", func(t *testing.T) {
		assert.True(t, pixelforge_audio.ChannelActive(pixelforge_audio.Chan1))
		assert.InDelta(t, 1.5, pixelforge_audio.ChannelPosition(pixelforge_audio.Chan1), 0)
		assert.InDelta(t, 1.0, pixelforge_audio.ChannelPitch(pixelforge_audio.Chan1), 0)
		assert.InDelta(t, 0.75, pixelforge_audio.ChannelVolume(pixelforge_audio.Chan1), 0)
		assert.Same(t, sample, pixelforge_audio.ChannelSample(pixelforge_audio.Chan1))
	})

	t.Run("returns zero values for inactive channel", func(t *testing.T) {
		assert.False(t, pixelforge_audio.ChannelActive(pixelforge_audio.Chan2))
		assert.Zero(t, pixelforge_audio.ChannelPosition(pixelforge_audio.Chan2))
		assert.Zero(t, pixelforge_audio.ChannelPitch(pixelforge_audio.Chan2))
		assert.Zero(t, pixelforge_audio.ChannelVolume(pixelforge_audio.Chan2))
		assert.Nil(t, pixelforge_audio.ChannelSample(pixelforge_audio.Chan2))
	})

	t.Run("returns zero values for unconfigured channel", func(t *testing.T) {
		assert.False(t, pixelforge_audio.ChannelActive(pixelforge_audio.Chan3))
		assert.Zero(t, pixelforge_audio.ChannelPosition(pixelforge_audio.Chan3))
	})
}
