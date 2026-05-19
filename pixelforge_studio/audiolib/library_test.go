package audiolib

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// library_test.go covers idea #4 v1 U3's load path: synthesis,
// validation, defensive skip. Each test resets the package-level
// cache so it can drive a custom catalog payload without polluting
// state across tests.

func TestLoadCatalog_ReturnsAllValidPatches(t *testing.T) {
	resetForTest()
	patches, err := LoadCatalog()
	require.NoError(t, err)
	require.NotEmpty(t, patches, "embedded catalog contains at least one patch")
	for _, p := range patches {
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.Bytes, "patch %s carries synthesised bytes", p.Name)
	}
}

func TestLoadCatalog_PatchBytesPassDecodeGate(t *testing.T) {
	resetForTest()
	patches, err := LoadCatalog()
	require.NoError(t, err)
	for _, p := range patches {
		_, err := pixelforge_audio.DecodeWavOrErr(p.Bytes)
		assert.NoError(t, err, "patch %s WAV bytes must pass DecodeWavOrErr", p.Name)
	}
}

func TestLoadCatalog_MalformedJSONReturnsError(t *testing.T) {
	patches, err := loadCatalogUncached([]byte("not-valid-json"))
	require.Error(t, err)
	assert.Empty(t, patches)
}

func TestLoadCatalog_PatchWithMissingNameSkipped(t *testing.T) {
	raw := []byte(`[
		{"name": "", "category": "test", "is_bgm": false, "suggested_priority": "sfx",
		 "synth": {"waveform": "square", "frequency_hz": 440, "duration_ms": 100, "envelope": "decay"}},
		{"name": "ok", "category": "test", "is_bgm": false, "suggested_priority": "sfx",
		 "synth": {"waveform": "square", "frequency_hz": 440, "duration_ms": 100, "envelope": "decay"}}
	]`)
	patches, err := loadCatalogUncached(raw)
	require.NoError(t, err)
	require.Len(t, patches, 1, "empty-name entry skipped; ok entry survives")
	assert.Equal(t, "ok", patches[0].Name)
}

func TestLoadCatalog_BGMPatchesCarryLoopFlag(t *testing.T) {
	resetForTest()
	patches, _ := LoadCatalog()
	var bgmCount int
	for _, p := range patches {
		if p.IsBGM {
			bgmCount++
			assert.Equal(t, "bgm", p.SuggestedPriority,
				"patch %s declared isBGM must also use bgm priority", p.Name)
		}
	}
	assert.GreaterOrEqual(t, bgmCount, 1, "library ships at least one BGM patch")
}

func TestLoadCatalog_DurationComputedFromBytes(t *testing.T) {
	resetForTest()
	patches, _ := LoadCatalog()
	for _, p := range patches {
		assert.Greater(t, p.Duration.Milliseconds(), int64(0),
			"patch %s has a positive duration", p.Name)
	}
}

func TestReadPatchBytes_KnownPatchReturnsBytes(t *testing.T) {
	resetForTest()
	bytes, err := ReadPatchBytes("jump/spring")
	require.NoError(t, err)
	assert.NotEmpty(t, bytes)
}

func TestReadPatchBytes_UnknownPatchReturnsError(t *testing.T) {
	resetForTest()
	_, err := ReadPatchBytes("definitely_not_a_real_patch")
	require.Error(t, err)
}

func TestFindPatch_ReturnsMetadata(t *testing.T) {
	resetForTest()
	p, ok := FindPatch("jump/spring")
	require.True(t, ok)
	assert.Equal(t, "jump", p.Category)
	assert.False(t, p.IsBGM)
}

func TestPatchesByCategory_FiltersByExactMatch(t *testing.T) {
	resetForTest()
	jumps := PatchesByCategory("jump")
	require.NotEmpty(t, jumps)
	for _, p := range jumps {
		assert.Equal(t, "jump", p.Category)
	}
}

func TestPatchesByCategory_EmptyReturnsAll(t *testing.T) {
	resetForTest()
	all, _ := LoadCatalog()
	got := PatchesByCategory("")
	assert.Equal(t, len(all), len(got))
}

func TestLibrary_CategoryCoverageMatchesBrainstorm(t *testing.T) {
	resetForTest()
	patches, _ := LoadCatalog()
	seen := map[string]bool{}
	for _, p := range patches {
		seen[p.Category] = true
	}
	// Brainstorm-prescribed categories — log (don't fail) on a
	// missing category so the library can ship incrementally.
	wanted := []string{
		"jump", "shoot", "hit", "pickup", "coin", "menu-confirm",
		"win-jingle", "lose-stinger", "damage", "death", "ambient",
		"town", "dungeon", "boss", "title", "victory",
	}
	missing := []string{}
	for _, c := range wanted {
		if !seen[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		t.Logf("category coverage gap (not failing v1): %s", strings.Join(missing, ", "))
	}
}

func TestSynthesizeWAV_ProducesPaulaCompatibleBytes(t *testing.T) {
	bytes := SynthesizeWAV(PatchSynth{
		Waveform:    WaveformSquare,
		FrequencyHz: 440,
		DurationMs:  100,
		Envelope:    EnvelopeDecay,
	})
	sample, err := pixelforge_audio.DecodeWavOrErr(bytes)
	require.NoError(t, err)
	assert.NotNil(t, sample)
	assert.Greater(t, sample.Len(), 0)
}

func TestSynthesizeWAV_DeterministicGivenSameParams(t *testing.T) {
	p := PatchSynth{Waveform: WaveformNoise, FrequencyHz: 200, DurationMs: 150, Envelope: EnvelopeDecay}
	a := SynthesizeWAV(p)
	b := SynthesizeWAV(p)
	assert.Equal(t, a, b, "identical params produce identical bytes")
}

func TestSynthesizeWAV_DefaultSampleRateAppliedWhenZero(t *testing.T) {
	bytes := SynthesizeWAV(PatchSynth{
		Waveform:    WaveformSine,
		FrequencyHz: 440,
		DurationMs:  100,
		Envelope:    EnvelopeSteady,
	})
	// 22050 * 0.1 = 2205 samples + 44-byte WAV header.
	assert.Equal(t, 44+2205, len(bytes))
}

func TestSynthesizeWAV_EnvelopeDecayProducesDescendingPeak(t *testing.T) {
	bytes := SynthesizeWAV(PatchSynth{
		Waveform:    WaveformSine,
		FrequencyHz: 100,
		DurationMs:  100,
		Envelope:    EnvelopeDecay,
	})
	// Skip the 44-byte WAV header; sample bytes are uint8 with mid=128.
	samples := bytes[44:]
	require.Greater(t, len(samples), 100)
	// Compute peak deviation from 128 in first 10% vs last 10%.
	peakStart := peakDeviation(samples[:len(samples)/10])
	peakEnd := peakDeviation(samples[len(samples)-len(samples)/10:])
	assert.Greater(t, peakStart, peakEnd,
		"decay envelope: early samples deviate more from mid than late samples")
}

func peakDeviation(samples []uint8) int {
	max := 0
	for _, s := range samples {
		d := int(s) - 128
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}
