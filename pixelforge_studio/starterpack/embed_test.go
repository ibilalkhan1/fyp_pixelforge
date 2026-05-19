package starterpack_test

import (
	"bytes"
	"io"
	"io/fs"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/starterpack"
)

// TestStarterFS_HasExpectedSpriteCount guards against the embed
// directive silently dropping files (which has happened before
// when an `//go:embed` line was edited without re-running tests).
func TestStarterFS_HasExpectedSpriteCount(t *testing.T) {
	sprites := listDir(t, "sprites")
	assert.Len(t, sprites, 8, "starter pack ships 8 placeholder sprites")
	expected := []string{
		"coin.png", "door.png", "enemy.png", "heart.png",
		"hero.png", "key.png", "platform.png", "wall.png",
	}
	sort.Strings(sprites)
	assert.Equal(t, expected, sprites)
}

func TestStarterFS_HasExpectedSFX(t *testing.T) {
	sfx := listDir(t, "sfx")
	sort.Strings(sfx)
	assert.Equal(t, []string{"hit.wav", "jump.wav"}, sfx)
}

func TestStarterFS_HasExpectedBGM(t *testing.T) {
	bgm := listDir(t, "bgm")
	assert.Equal(t, []string{"level_theme.ogg"}, bgm)
}

// TestStarterFS_ReadHeroBytes exercises the fs.FS open path — the
// embed directive can pass listDir checks while still failing to
// hand back bytes on Open (rare, but worth the explicit assert).
func TestStarterFS_ReadHeroBytes(t *testing.T) {
	f, err := starterpack.StarterFS().Open("sprites/hero.png")
	require.NoError(t, err)
	defer f.Close()
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.NotEmpty(t, b, "hero.png must have bytes")
	// PNG signature; sanity-check the file is actually a PNG, not
	// a stray text file.
	assert.True(t, bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47}), "hero.png missing PNG signature")
}

// TestStarterFS_SFXWavSignature asserts the SFX files carry the
// RIFF/WAVE header so the audio pipeline doesn't choke on stray
// bytes.
func TestStarterFS_SFXWavSignature(t *testing.T) {
	for _, name := range []string{"sfx/jump.wav", "sfx/hit.wav"} {
		b, err := fs.ReadFile(starterpack.StarterFS(), name)
		require.NoError(t, err, name)
		require.GreaterOrEqual(t, len(b), 12, name)
		assert.Equal(t, []byte("RIFF"), b[0:4], name+" RIFF magic")
		assert.Equal(t, []byte("WAVE"), b[8:12], name+" WAVE chunk")
	}
}

func TestStarterPack_MetadataMatchesEmbeddedFiles(t *testing.T) {
	pack := starterpack.StarterPack()
	assert.Equal(t, "starter", pack.ID)
	assert.Equal(t, starterpack.StarterPackVersion, pack.Version)
	assert.Equal(t, "Starter Pack", pack.Title)
	assert.Empty(t, pack.URL, "embedded packs have no URL")
	assert.Empty(t, pack.SHA256, "embedded packs aren't hash-verified")
	require.Len(t, pack.Assets, 11, "8 sprites + 2 sfx + 1 bgm = 11 assets")

	// Every declared asset must resolve via the embedded FS — if
	// the metadata drifts from the directory layout, fail loudly.
	for _, a := range pack.Assets {
		_, err := fs.Stat(starterpack.StarterFS(), a.Path)
		assert.NoError(t, err, "starter asset %q missing from embedded FS", a.Path)
	}

	// Every embedded file must appear in the metadata — guards
	// against an orphaned file dragged into assets/ but not
	// declared. This is the inverse of the previous assertion.
	declared := map[string]bool{}
	for _, a := range pack.Assets {
		declared[a.Path] = true
	}
	require.NoError(t, fs.WalkDir(starterpack.StarterFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		assert.True(t, declared[path], "embedded file %q not declared in StarterPack().Assets", path)
		return nil
	}))
}

func TestStarterPack_SizeBytesNonZero(t *testing.T) {
	pack := starterpack.StarterPack()
	assert.Positive(t, pack.SizeBytes, "embedded pack should report >0 bytes")
}

// listDir returns the file names directly under sub inside the
// starter FS. Helper centralised so each test reads cleanly.
func listDir(t *testing.T, sub string) []string {
	t.Helper()
	entries, err := fs.ReadDir(starterpack.StarterFS(), sub)
	require.NoError(t, err)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}
