package audiolib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Fixtures are original 440 Hz tones; headers alone must fail decoding.
func testBGM(t *testing.T, extension string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../pixelforge_audio/internal/test/tone." + extension)
	require.NoError(t, err)
	return data
}

func writeBGMFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestImportBGMFromFile_ValidOGGAppendsToProject(t *testing.T) {
	p, ppath := tempProjectFor(t)
	bgmPath := writeBGMFile(t, t.TempDir(), "title.ogg", testBGM(t, "ogg"))
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
	bgmPath := writeBGMFile(t, t.TempDir(), "stage1.mp3", testBGM(t, "mp3"))
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
	bgmPath := writeBGMFile(t, t.TempDir(), "stage2.mp3", testBGM(t, "mp3"))
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
	bad := writeBGMFile(t, t.TempDir(), "mystery.xyz", testBGM(t, "ogg"))
	_, err := ImportBGMFromFile(bad, p, ppath)
	require.Error(t, err)
}

func TestImportBGMFromFile_EmptyPathReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	_, err := ImportBGMFromFile("", p, ppath)
	require.Error(t, err)
}

func TestImportBGMFromFile_NilProjectReturnsError(t *testing.T) {
	bgmPath := writeBGMFile(t, t.TempDir(), "nil.ogg", testBGM(t, "ogg"))
	_, err := ImportBGMFromFile(bgmPath, nil, "")
	require.Error(t, err)
}

func TestImportBGMFromFile_NameCollisionSuffixes(t *testing.T) {
	p, ppath := tempProjectFor(t)
	dir := t.TempDir()
	first := writeBGMFile(t, dir, "loop.ogg", testBGM(t, "ogg"))
	_, err := ImportBGMFromFile(first, p, ppath)
	require.NoError(t, err)
	// Re-import the same basename — must collision-suffix.
	second := writeBGMFile(t, t.TempDir(), "loop.ogg", testBGM(t, "ogg"))
	_, err = ImportBGMFromFile(second, p, ppath)
	require.NoError(t, err)
	require.Len(t, p.Audio, 2)
	assert.Equal(t, "loop", p.Audio[0].Name)
	assert.Equal(t, "loop_2", p.Audio[1].Name)
	assert.Equal(t, "audio/loop_2.ogg", p.Audio[1].RelativePath)
}

func TestImportBGMFromFile_NoProjectSourcePathSkipsCopy(t *testing.T) {
	p := pixelforge_project.NewProject("test")
	bgmPath := writeBGMFile(t, t.TempDir(), "nopath.ogg", testBGM(t, "ogg"))
	_, err := ImportBGMFromFile(bgmPath, p, "")
	require.NoError(t, err)
	require.Len(t, p.Audio, 1, "AudioSample still appended; just no file copy")
}

func TestImportBGMFromFile_NonexistentPathReturnsError(t *testing.T) {
	p, ppath := tempProjectFor(t)
	_, err := ImportBGMFromFile("/no/such/file.ogg", p, ppath)
	require.Error(t, err)
}

func TestImportBGMRejectsHeaderOnlyFiles(t *testing.T) {
	for _, fixture := range []struct {
		name string
		data []byte
	}{{"bad.ogg", []byte("OggS")}, {"bad.mp3", []byte("ID3")}} {
		p, ppath := tempProjectFor(t)
		_, err := ImportBGMFromFile(writeBGMFile(t, t.TempDir(), fixture.name, fixture.data), p, ppath)
		require.Error(t, err)
		assert.Empty(t, p.Audio)
	}
}
