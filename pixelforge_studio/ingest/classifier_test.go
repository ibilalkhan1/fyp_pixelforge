package ingest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
)

func TestClassify_PNGIsSprite(t *testing.T) {
	assert.Equal(t, ingest.KindSprite, ingest.Classify("/path/to/ship.png"))
}

func TestClassify_WAVIsSFX(t *testing.T) {
	assert.Equal(t, ingest.KindSFX, ingest.Classify("/path/to/blast.wav"))
}

func TestClassify_OGGAndMP3AreBGM(t *testing.T) {
	assert.Equal(t, ingest.KindBGM, ingest.Classify("/loops/level1.ogg"))
	assert.Equal(t, ingest.KindBGM, ingest.Classify("/loops/title.mp3"))
}

func TestClassify_CaseInsensitive(t *testing.T) {
	assert.Equal(t, ingest.KindSprite, ingest.Classify("/PATH/SHIP.PNG"))
	assert.Equal(t, ingest.KindSFX, ingest.Classify("/HORN.Wav"))
}

func TestClassify_UnknownReturnsKindUnknown(t *testing.T) {
	assert.Equal(t, ingest.KindUnknown, ingest.Classify("/notes.txt"))
	assert.Equal(t, ingest.KindUnknown, ingest.Classify("/no-extension"))
	assert.Equal(t, ingest.KindUnknown, ingest.Classify(""))
}

func TestIsAssetExtension(t *testing.T) {
	assert.True(t, ingest.IsAssetExtension("foo.png"))
	assert.True(t, ingest.IsAssetExtension("foo.wav"))
	assert.True(t, ingest.IsAssetExtension("foo.ogg"))
	assert.True(t, ingest.IsAssetExtension("foo.mp3"))
	assert.False(t, ingest.IsAssetExtension("foo.txt"))
	assert.False(t, ingest.IsAssetExtension(""))
}

func TestAssetKind_String(t *testing.T) {
	assert.Equal(t, "sprite", ingest.KindSprite.String())
	assert.Equal(t, "sfx", ingest.KindSFX.String())
	assert.Equal(t, "bgm", ingest.KindBGM.String())
	assert.Equal(t, "unknown", ingest.KindUnknown.String())
}
