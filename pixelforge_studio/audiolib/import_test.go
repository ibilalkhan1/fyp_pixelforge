package audiolib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// import_test.go covers the U4 import seam: decode → copy → append.
// Tests use a tmpdir-based project source path so file-system
// side-effects are sandboxed.

func tempProjectFor(t *testing.T) (*pixelforge_project.Project, string) {
	t.Helper()
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "test.pforge")
	return pixelforge_project.NewProject("test"), projectPath
}

func writeWAV(t *testing.T, dir, name string, synth PatchSynth) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, SynthesizeWAV(synth), 0o644))
	return path
}

func TestImportFromFile_ValidWAVAppendsToProject(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	wavPath := writeWAV(t, t.TempDir(), "jump_test.wav", PatchSynth{
		Waveform: WaveformSquare, FrequencyHz: 440, DurationMs: 100, Envelope: EnvelopeDecay,
	})
	res, err := ImportFromFile(wavPath, p, ppath)
	require.NoError(t, err)
	assert.Equal(t, "jump_test", res.SampleName)
	require.Len(t, p.Audio, 1)
	assert.Equal(t, "audio/jump_test.wav", p.Audio[0].RelativePath)
	assert.Equal(t, "sfx", p.Audio[0].SuggestedChannelPriority,
		"file imports default to sfx priority")
	assert.False(t, p.Audio[0].Loop, "file imports default to non-loop")
	dstPath := filepath.Join(pixelforge_project.AssetsDir(ppath), "audio", "jump_test.wav")
	_, err = os.Stat(dstPath)
	assert.NoError(t, err, "file copied to project's audio assets dir")
}

func TestImportFromFile_InvalidWAVReturnsError(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	badPath := filepath.Join(t.TempDir(), "bad.wav")
	require.NoError(t, os.WriteFile(badPath, []byte("not a wav"), 0o644))
	_, err := ImportFromFile(badPath, p, ppath)
	require.Error(t, err)
	assert.Empty(t, p.Audio, "failed import leaves project untouched")
}

func TestImportFromFile_EmptyPathReturnsError(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	_, err := ImportFromFile("", p, ppath)
	require.Error(t, err)
}

func TestImportFromLibrary_CopiesEmbeddedBytes(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	res, err := ImportFromLibrary("jump/spring", p, ppath)
	require.NoError(t, err)
	assert.Equal(t, "jump_spring", res.SampleName, "name sanitised: '/' → '_'")
	assert.Equal(t, "audio/jump_spring.wav", res.RelativePath)
	require.Len(t, p.Audio, 1)
	assert.Equal(t, "sfx", p.Audio[0].SuggestedChannelPriority)
	assert.False(t, p.Audio[0].Loop)
}

func TestImportFromLibrary_BGMPatchSetsLoopTrue(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	res, err := ImportFromLibrary("bgm/title/intro_loop", p, ppath)
	require.NoError(t, err)
	assert.True(t, res.IsBGM)
	require.Len(t, p.Audio, 1)
	assert.True(t, p.Audio[0].Loop, "BGM library patches import with Loop=true")
	assert.Equal(t, "bgm", p.Audio[0].SuggestedChannelPriority)
}

func TestImportFromLibrary_UnknownPatchReturnsError(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	_, err := ImportFromLibrary("no/such/patch", p, ppath)
	require.Error(t, err)
}

func TestImport_NameCollisionSuffixes(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	_, err := ImportFromLibrary("jump/spring", p, ppath)
	require.NoError(t, err)
	_, err = ImportFromLibrary("jump/spring", p, ppath)
	require.NoError(t, err)
	require.Len(t, p.Audio, 2)
	assert.Equal(t, "jump_spring", p.Audio[0].Name)
	assert.Equal(t, "jump_spring_2", p.Audio[1].Name)
	assert.Equal(t, "audio/jump_spring_2.wav", p.Audio[1].RelativePath)
}

func TestImport_AssetsDirCreatedIfMissing(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	// Confirm the audio subdir doesn't exist yet.
	dst := filepath.Join(pixelforge_project.AssetsDir(ppath), "audio")
	_, err := os.Stat(dst)
	assert.True(t, os.IsNotExist(err))

	_, err = ImportFromLibrary("jump/spring", p, ppath)
	require.NoError(t, err)
	_, err = os.Stat(dst)
	assert.NoError(t, err, "audio subdir created on first import")
}

func TestImport_SampleRateHzReadFromWAVHeader(t *testing.T) {
	resetForTest()
	p, ppath := tempProjectFor(t)
	wavPath := writeWAV(t, t.TempDir(), "fixed_rate.wav", PatchSynth{
		Waveform: WaveformSquare, FrequencyHz: 440, DurationMs: 100,
		Envelope: EnvelopeDecay, SampleRateHz: 11025,
	})
	_, err := ImportFromFile(wavPath, p, ppath)
	require.NoError(t, err)
	assert.Equal(t, 11025, p.Audio[0].SampleRateHz)
}

func TestImport_NoProjectSourcePathSkipsCopy(t *testing.T) {
	resetForTest()
	p := pixelforge_project.NewProject("test")
	// Empty projectSourcePath → no file copy attempted. Useful for
	// tests + drag-drop where the project has no on-disk home yet.
	_, err := ImportFromLibrary("jump/spring", p, "")
	require.NoError(t, err)
	require.Len(t, p.Audio, 1, "AudioSample still appended; just no file copy")
}

func TestImport_NilProjectReturnsError(t *testing.T) {
	resetForTest()
	wavPath := writeWAV(t, t.TempDir(), "nil_proj.wav", PatchSynth{
		Waveform: WaveformSquare, FrequencyHz: 440, DurationMs: 100, Envelope: EnvelopeDecay,
	})
	_, err := ImportFromFile(wavPath, nil, "")
	require.Error(t, err)
}
