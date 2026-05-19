package ingest_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
)

// bgm_test.go focuses on plan-009 U19's classifier + dispatcher
// coverage for the BGM kind. dispatcher_test.go already proves
// OGG/MP3 route through SetBGMRunner; this file rounds out the
// surface with classifier consistency + extension-mix tests so the
// per-kind wiring stays well-anchored when main.go's three-runner
// install lands.

func TestClassify_BGMExtensions(t *testing.T) {
	cases := []struct {
		path string
		want ingest.AssetKind
	}{
		{"track.ogg", ingest.KindBGM},
		{"TRACK.OGG", ingest.KindBGM},
		{"loop.mp3", ingest.KindBGM},
		{"loop.MP3", ingest.KindBGM},
		{"/abs/path/title.ogg", ingest.KindBGM},
		{"sub/dir/stage1.mp3", ingest.KindBGM},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, ingest.Classify(tc.path),
			"Classify(%q)", tc.path)
	}
}

func TestClassify_BGMVsSFXVsSprite(t *testing.T) {
	assert.Equal(t, ingest.KindSprite, ingest.Classify("img.png"))
	assert.Equal(t, ingest.KindSFX, ingest.Classify("blast.wav"))
	assert.Equal(t, ingest.KindBGM, ingest.Classify("music.ogg"))
	assert.Equal(t, ingest.KindBGM, ingest.Classify("music.mp3"))
	assert.Equal(t, ingest.KindUnknown, ingest.Classify("notes.txt"))
	assert.Equal(t, ingest.KindUnknown, ingest.Classify("mystery.xyz"))
}

func TestDispatcher_MultiKindWiringRoutesToCorrectRunner(t *testing.T) {
	// Wire all three runners; drop one path of each kind; assert
	// each runner receives only its own kind. Mirrors the
	// production main.go wiring (PNG → ImportHandler.Import; WAV →
	// AudioImportHandler.Import; OGG/MP3 → BGMImportHandler.Import).
	var spriteCalls, sfxCalls, bgmCalls []string
	d := ingest.NewDispatcher()
	d.SetSpriteRunner(ingest.RunnerFunc(func(path string) error {
		spriteCalls = append(spriteCalls, path)
		return nil
	}))
	d.SetSFXRunner(ingest.RunnerFunc(func(path string) error {
		sfxCalls = append(sfxCalls, path)
		return nil
	}))
	d.SetBGMRunner(ingest.RunnerFunc(func(path string) error {
		bgmCalls = append(bgmCalls, path)
		return nil
	}))

	require.NoError(t, d.Ingest("/drop/cat.png"))
	require.NoError(t, d.Ingest("/drop/blast.wav"))
	require.NoError(t, d.Ingest("/drop/level.ogg"))
	require.NoError(t, d.Ingest("/drop/title.mp3"))
	require.NoError(t, d.Ingest("/drop/mystery.xyz"))

	assert.Equal(t, []string{"/drop/cat.png"}, spriteCalls)
	assert.Equal(t, []string{"/drop/blast.wav"}, sfxCalls)
	assert.Equal(t, []string{"/drop/level.ogg", "/drop/title.mp3"}, bgmCalls)
}

func TestDispatcher_BGMRunnerErrorBubbles(t *testing.T) {
	d := ingest.NewDispatcher()
	wantErr := errors.New("bgm decode failed")
	d.SetBGMRunner(ingest.RunnerFunc(func(path string) error {
		return wantErr
	}))
	err := d.Ingest("/level.ogg")
	require.Error(t, err)
	assert.True(t, errors.Is(err, wantErr))
}

func TestIsAssetExtension_BGMReturnsTrue(t *testing.T) {
	assert.True(t, ingest.IsAssetExtension("loop.ogg"))
	assert.True(t, ingest.IsAssetExtension("loop.mp3"))
	assert.False(t, ingest.IsAssetExtension("readme.txt"))
}
