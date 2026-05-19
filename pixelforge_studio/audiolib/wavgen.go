// wavgen.go produces deterministic 8-bit mono PCM WAV bytes from a
// compact set of synthesis parameters. Used by LoadCatalog (U3) to
// build the bundled audio library at startup without checking in
// binary WAVs — designers ship-and-iterate by editing catalog.json
// rather than juggling binary files in the repo.
//
// Synthesis vocabulary is intentionally small: a waveform (square,
// triangle, sine, noise), a base frequency, a duration, an
// envelope (linear-decay or steady). Combined with an Allocator-
// friendly priority and a loop flag, this covers the NES-class
// arcade vocabulary: laser zaps, coin pickups, jumps, hits, BGM
// loops. The output passes pixelforge_audio.DecodeWavOrErr's
// strict 8-bit-mono PCM gate by construction.
package audiolib

import (
	"encoding/binary"
	"math"
	"math/rand"
)

// Waveform names the four basic oscillator shapes the synthesizer
// supports. Choosing a waveform per patch is the dominant control —
// "square at 600 Hz with 80 ms decay" is the classic NES jump; "noise
// burst with 100 ms decay" is the explosion; "triangle sweep" is
// the menu confirm.
type Waveform string

const (
	WaveformSquare   Waveform = "square"
	WaveformTriangle Waveform = "triangle"
	WaveformSine     Waveform = "sine"
	WaveformNoise    Waveform = "noise"
)

// Envelope describes how the patch's amplitude evolves over its
// duration. Two shapes cover the v1 vocabulary; v2 may add ADSR.
type Envelope string

const (
	// EnvelopeDecay starts at full volume and ramps linearly to
	// silence over the patch's duration. The arcade default.
	EnvelopeDecay Envelope = "decay"

	// EnvelopeSteady holds at full volume the whole patch. BGM
	// loops use this so the loop point isn't a fade.
	EnvelopeSteady Envelope = "steady"
)

// PatchSynth is the declarative description LoadCatalog feeds into
// the synthesizer. Embedded in CatalogEntry (catalog.json) so the
// library is "code-free data."
type PatchSynth struct {
	Waveform     Waveform `json:"waveform"`
	FrequencyHz  float64  `json:"frequency_hz"`
	DurationMs   int      `json:"duration_ms"`
	Envelope     Envelope `json:"envelope"`
	SampleRateHz uint16   `json:"sample_rate_hz,omitempty"` // default 22050
}

// defaultSampleRate is the synth's default sample rate. 22050 Hz
// balances Paula authenticity with bundle size; the decoder
// accepts up to 48000 Hz (DecodeWavOrErr's gate).
const defaultSampleRate uint16 = 22050

// SynthesizeWAV produces an 8-bit mono PCM WAV byte slice for the
// supplied PatchSynth. Returns Paula-compatible bytes that pass
// DecodeWavOrErr's strict gate. Deterministic given identical
// input — same parameters always produce identical bytes (the noise
// waveform seeds its RNG from the duration + frequency hash so a
// "boss explosion" patch always sounds the same).
func SynthesizeWAV(p PatchSynth) []byte {
	rate := p.SampleRateHz
	if rate == 0 {
		rate = defaultSampleRate
	}
	if p.DurationMs <= 0 {
		p.DurationMs = 200
	}

	totalSamples := int(rate) * p.DurationMs / 1000
	if totalSamples < 1 {
		totalSamples = 1
	}

	samples := make([]uint8, totalSamples)
	rng := newDeterministicRNG(p)

	for i := 0; i < totalSamples; i++ {
		t := float64(i) / float64(rate)
		amp := envelopeAtTime(p.Envelope, i, totalSamples)
		raw := waveformAtTime(p.Waveform, p.FrequencyHz, t, rng)
		// raw is in [-1, 1]; map to uint8 [0, 255] with mid = 128.
		v := int(128.0 + amp*raw*127.0)
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		samples[i] = uint8(v)
	}

	return encodeWAV(samples, rate)
}

// envelopeAtTime returns the per-sample amplitude multiplier in
// [0, 1]. EnvelopeDecay ramps from 1.0 to 0.0 linearly across the
// patch; EnvelopeSteady stays at 1.0.
func envelopeAtTime(env Envelope, i, total int) float64 {
	switch env {
	case EnvelopeSteady:
		return 1.0
	default:
		// EnvelopeDecay (also the fallback for unknown envelopes).
		if total <= 1 {
			return 1.0
		}
		return 1.0 - float64(i)/float64(total-1)
	}
}

// waveformAtTime returns one sample in [-1, 1] for the supplied
// waveform at time t.
func waveformAtTime(w Waveform, freq, t float64, rng *rand.Rand) float64 {
	switch w {
	case WaveformTriangle:
		// Triangle wave: linear ramp up and down across one period.
		period := 1.0 / freq
		phase := math.Mod(t, period) / period // 0..1
		if phase < 0.5 {
			return 4*phase - 1
		}
		return 3 - 4*phase
	case WaveformSine:
		return math.Sin(2 * math.Pi * freq * t)
	case WaveformNoise:
		// White noise — seeded RNG so the patch is deterministic.
		return rng.Float64()*2 - 1
	default:
		// WaveformSquare (also the fallback for unknown waveforms).
		// Square wave: +1 for first half of period, -1 for second.
		period := 1.0 / freq
		if math.Mod(t, period) < period/2 {
			return 1
		}
		return -1
	}
}

// encodeWAV wraps the supplied uint8 PCM samples in a Paula-
// compatible RIFF/WAVE container. Chunk layout follows the canonical
// WAV format the existing DecodeWavOrErr accepts:
//
//	RIFF<size>WAVE
//	fmt <16> (PCM, mono, sampleRate, ..., 8-bit)
//	data<size><samples>
func encodeWAV(samples []uint8, sampleRate uint16) []byte {
	const (
		// fmt sub-chunk size for PCM.
		fmtChunkSize = 16
		// PCM audio format constant.
		pcmFormat     uint16 = 1
		monoChannels  uint16 = 1
		bitsPerSample uint16 = 8
	)
	dataSize := uint32(len(samples))
	// 4 ("WAVE") + 8 (fmt header) + 16 (fmt body) + 8 (data header) + dataSize
	riffSize := uint32(4 + 8 + fmtChunkSize + 8 + int(dataSize))
	byteRate := uint32(sampleRate) * uint32(monoChannels) * uint32(bitsPerSample) / 8
	blockAlign := monoChannels * bitsPerSample / 8

	out := make([]byte, 0, 44+len(samples))
	out = append(out, []byte("RIFF")...)
	out = appendU32LE(out, riffSize)
	out = append(out, []byte("WAVE")...)
	out = append(out, []byte("fmt ")...)
	out = appendU32LE(out, fmtChunkSize)
	out = appendU16LE(out, pcmFormat)
	out = appendU16LE(out, monoChannels)
	out = appendU32LE(out, uint32(sampleRate))
	out = appendU32LE(out, byteRate)
	out = appendU16LE(out, blockAlign)
	out = appendU16LE(out, bitsPerSample)
	out = append(out, []byte("data")...)
	out = appendU32LE(out, dataSize)
	out = append(out, samples...)
	return out
}

func appendU16LE(b []byte, v uint16) []byte {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	return append(b, buf[:]...)
}

func appendU32LE(b []byte, v uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return append(b, buf[:]...)
}

// newDeterministicRNG seeds a per-patch RNG so noise-waveform
// patches synthesise the same bytes on every run. Hashed from the
// waveform/freq/duration tuple so distinct patches sound distinct
// while staying reproducible.
func newDeterministicRNG(p PatchSynth) *rand.Rand {
	seed := int64(p.DurationMs) ^ int64(p.FrequencyHz*1000) ^ int64(len(p.Waveform))*131
	return rand.New(rand.NewSource(seed))
}
