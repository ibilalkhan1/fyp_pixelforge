package audiolib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// bgm_runner_test.go covers plan-009 U19's BGM-import seam: magic-
// byte validation + project-side AudioSample append with Loop=true
// + collision suffixing. The decoder is stubbed (see file header in
// bgm_runner.go for the deferral note); these tests cover the
// wiring, not the still-pending audio-v2 decode pipeline.

// minimalOGG returns a 4-byte stub carrying the OggS magic. The
// runner only validates the magic at the head — full-OGG decode
// lands later — so this is enough to exercise the success path.
func minimalOGG() []byte {
	return []byte("OggS")
}

// minimalMP3 returns 3 bytes carrying the ID3 magic. Same reason
// as minimalOGG.
func minimalMP3() []byte {
	return []byte("ID3")
}

// minimalMP3Sync returns 2 bytes that satisfy the MPEG sync-word
// detector (0xFF + 0xE0..0xFF in the second byte).
func minimalMP3Sync() []byte {
	return []byte{0xFF, 0xFB}
}

func writeBGMFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestImportBGMFromFile_ValidOGGAppendsToProject(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bgmPath := writeBGMFile(t, t.TempDir(), "title.ogg", minimalOGG())
	res, err := ImportBGMFromFile(bgmPath, p, ppath)
	require.NoError(t, err)
	assert.Equal(t, "title", res.SampleName)
	assert.Equal(t, BGMContainerOGG, res.Container)
	assert.Equal(t, "audio/title.ogg", res.RelativePath)
	require.Len(t, p.Audio, 1)
	assert.Equal(t, "bgm", p.Audio[0].SuggestedChannelPriority,
		"BGM imports default to bgm priority")
	assert.True(t, p.Audio[0].Loop, "BGM imports default to Loop=true")
	dst := filepath.Join(pixelforge_project.AssetsDir(ppath), "audio", "title.ogg")
	_, err = os.Stat(dst)
	assert.NoError(t, err, "file copied to project's audio assets dir")
}

func TestImportBGMFromFile_ValidMP3WithID3AppendsToProject(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bgmPath := writeBGMFile(t, t.TempDir(), "stage1.mp3", minimalMP3())
	res, err := ImportBGMFromFile(bgmPath, p, ppath)
	require.NoError(t, err)
	assert.Equal(t, "stage1", res.SampleName)
	assert.Equal(t, BGMContainerMP3, res.Container)
	assert.Equal(t, "audio/stage1.mp3", res.RelativePath)
	require.Len(t, p.Audio, 1)
	assert.True(t, p.Audio[0].Loop)
}

func TestImportBGMFromFile_ValidMP3WithSyncWordAppendsToProject(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bgmPath := writeBGMFile(t, t.TempDir(), "stage2.mp3", minimalMP3Sync())
	res, err := ImportBGMFromFile(bgmPath, p, ppath)
	require.NoError(t, err)
	assert.Equal(t, BGMContainerMP3, res.Container)
	require.Len(t, p.Audio, 1)
}

func TestImportBGMFromFile_BadOGGMagicReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bad := writeBGMFile(t, t.TempDir(), "fake.ogg", []byte("not really ogg"))
	_, err := ImportBGMFromFile(bad, p, ppath)
	require.Error(t, err)
	assert.Empty(t, p.Audio, "failed import leaves project untouched")
}

func TestImportBGMFromFile_BadMP3MagicReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bad := writeBGMFile(t, t.TempDir(), "fake.mp3", []byte{0x00, 0x00, 0x00, 0x00})
	_, err := ImportBGMFromFile(bad, p, ppath)
	require.Error(t, err)
}

func TestImportBGMFromFile_UnrecognisedExtensionReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bad := writeBGMFile(t, t.TempDir(), "mystery.xyz", minimalOGG())
	_, err := ImportBGMFromFile(bad, p, ppath)
	require.Error(t, err)
}

func TestImportBGMFromFile_EmptyPathReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	_, err := ImportBGMFromFile("", p, ppath)
	require.Error(t, err)
}

func TestImportBGMFromFile_NilProjectReturnsError(t *testing.T) {
	bgmPath := writeBGMFile(t, t.TempDir(), "nil.ogg", minimalOGG())
	_, err := ImportBGMFromFile(bgmPath, nil, "")
	require.Error(t, err)
}

func TestImportBGMFromFile_NameCollisionSuffixes(t *testing.T) {
	p, ppath := tempProjectFor(t)
	dir := t.TempDir()
	first := writeBGMFile(t, dir, "loop.ogg", minimalOGG())
	_, err := ImportBGMFromFile(first, p, ppath)
	require.NoError(t, err)
	// Re-import the same basename — must collision-suffix.
	second := writeBGMFile(t, t.TempDir(), "loop.ogg", minimalOGG())
	_, err = ImportBGMFromFile(second, p, ppath)
	require.NoError(t, err)
	require.Len(t, p.Audio, 2)
	assert.Equal(t, "loop", p.Audio[0].Name)
	assert.Equal(t, "loop_2", p.Audio[1].Name)
	assert.Equal(t, "audio/loop_2.ogg", p.Audio[1].RelativePath)
}

func TestImportBGMFromFile_NoProjectSourcePathSkipsCopy(t *testing.T) {
	p := pixelforge_project.NewProject("test")
	bgmPath := writeBGMFile(t, t.TempDir(), "nopath.ogg", minimalOGG())
	_, err := ImportBGMFromFile(bgmPath, p, "")
	require.NoError(t, err)
	require.Len(t, p.Audio, 1, "AudioSample still appended; just no file copy")
}

func TestImportBGMFromFile_NonexistentPathReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	_, err := ImportBGMFromFile("/no/such/file.ogg", p, ppath)
	require.Error(t, err)
}
