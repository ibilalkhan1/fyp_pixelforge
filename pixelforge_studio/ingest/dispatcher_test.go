package ingest_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
)

type recordingRunner struct {
	calls []string
	err   error
}

func (r *recordingRunner) Run(path string) error {
	r.calls = append(r.calls, path)
	return r.err
}

func TestDispatcher_IngestRoutesPNGToSpriteRunner(t *testing.T) {
	d := ingest.NewDispatcher()
	sprite := &recordingRunner{}
	d.SetSpriteRunner(sprite)
	require.NoError(t, d.Ingest("/path/ship.png"))
	assert.Equal(t, []string{"/path/ship.png"}, sprite.calls)
}

func TestDispatcher_IngestRoutesWAVToSFXRunner(t *testing.T) {
	d := ingest.NewDispatcher()
	sfx := &recordingRunner{}
	d.SetSFXRunner(sfx)
	require.NoError(t, d.Ingest("/path/blast.wav"))
	assert.Equal(t, []string{"/path/blast.wav"}, sfx.calls)
}

func TestDispatcher_IngestRoutesOGGToBGMRunner(t *testing.T) {
	d := ingest.NewDispatcher()
	bgm := &recordingRunner{}
	d.SetBGMRunner(bgm)
	require.NoError(t, d.Ingest("/loops/level.ogg"))
	require.NoError(t, d.Ingest("/loops/title.mp3"))
	assert.Equal(t, []string{"/loops/level.ogg", "/loops/title.mp3"}, bgm.calls)
}

func TestDispatcher_IngestUnknownExtensionNoOps(t *testing.T) {
	d := ingest.NewDispatcher()
	sprite := &recordingRunner{}
	d.SetSpriteRunner(sprite)
	require.NoError(t, d.Ingest("/path/notes.txt"))
	assert.Empty(t, sprite.calls)
}

func TestDispatcher_IngestNoRunnerNoOps(t *testing.T) {
	d := ingest.NewDispatcher()
	// PNG classified as sprite but no sprite runner installed.
	assert.NoError(t, d.Ingest("/path/ship.png"))
}

func TestDispatcher_IngestEmptyPathErrors(t *testing.T) {
	d := ingest.NewDispatcher()
	err := d.Ingest("")
	require.Error(t, err)
}

func TestDispatcher_RunnerErrorBubbles(t *testing.T) {
	d := ingest.NewDispatcher()
	wantErr := errors.New("decode failed")
	d.SetSpriteRunner(&recordingRunner{err: wantErr})
	err := d.Ingest("/x.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, wantErr))
}

func TestDispatcher_RunnerFuncAdaptsBareFunction(t *testing.T) {
	called := false
	d := ingest.NewDispatcher()
	d.SetSpriteRunner(ingest.RunnerFunc(func(path string) error {
		called = true
		return nil
	}))
	require.NoError(t, d.Ingest("/x.png"))
	assert.True(t, called)
}

func TestDispatcher_HasRunnerReflectsRegistration(t *testing.T) {
	d := ingest.NewDispatcher()
	assert.False(t, d.HasSpriteRunner())
	assert.False(t, d.HasSFXRunner())
	assert.False(t, d.HasBGMRunner())
	d.SetSpriteRunner(&recordingRunner{})
	d.SetSFXRunner(&recordingRunner{})
	d.SetBGMRunner(&recordingRunner{})
	assert.True(t, d.HasSpriteRunner())
	assert.True(t, d.HasSFXRunner())
	assert.True(t, d.HasBGMRunner())
}
