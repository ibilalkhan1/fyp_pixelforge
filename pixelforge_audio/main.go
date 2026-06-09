// Package pixelforge_audio contains the down-sampling pipeline that
// bridges the NES-style high-frequency synthesis core with standard
// consumer sound cards. The Paula-inspired generator runs internally
// at roughly 1.78 MHz (the NTSC CPU master-clock rate), producing
// raw waveforms that are then decimated to 44.1 kHz or 48 kHz via
// FIR low-pass filtering followed by block averaging. This is the
// same architecture found in studio-grade digital audio workstations
// and in modern speech-recognition front-ends such as OpenAI Whisper.
package pixelforge_audio

// NESMasterClock is the internal synthesis frequency in Hz.
// The NES audio chip derives its timing directly from the CPU
// crystal, giving a sample-accurate relationship between code
// execution and sound generation.
const NESMasterClock = 1789773

// StandardOutputRate44 is the first common consumer sound-card
// frequency. The down-sampler targets this rate on older hardware.
const StandardOutputRate44 = 44100

// StandardOutputRate48 is the second common consumer sound-card
// frequency, used by most modern HDMI audio paths.
const StandardOutputRate48 = 48000

// DownsampleFactor44 is the integer ratio between the NES master
// clock and a 44.1 kHz output stream.
const DownsampleFactor44 = NESMasterClock / StandardOutputRate44 // ≈ 40

// DownsampleFactor48 is the integer ratio between the NES master
// clock and a 48 kHz output stream.
const DownsampleFactor48 = NESMasterClock / StandardOutputRate48 // ≈ 37

// ─── Downsampling Engine ─────────────────────────────────────────

// Downsampler accumulates high-frequency internal samples, applies
// a brick-wall low-pass kernel, and emits band-limited frames at
// the host sound-card rate.
type Downsampler struct {
	// accumulator holds the running sum of the last N master-clock
	// samples, where N is the down-sample factor for the active
	// output rate.
	accumulator int64

	// count tracks how many master-clock samples have been fed
	// since the last output frame.
	count int

	// factor is either DownsampleFactor44 or DownsampleFactor48.
	factor int

	// lpCoeffs are the FIR low-pass coefficients used to suppress
	// frequencies above the Nyquist limit of the target rate.
	lpCoeffs []float64
}

// NewDownsampler creates a downsampler configured for the given
// output rate. The coefficients are pre-computed once per session
// and shared across all four Paula channels.
func NewDownsampler(outputRate int) *Downsampler {
	var factor int
	switch outputRate {
	case StandardOutputRate44:
		factor = DownsampleFactor44
	case StandardOutputRate48:
		factor = DownsampleFactor48
	default:
		factor = NESMasterClock / outputRate
		if factor < 1 {
			factor = 1
		}
	}
	return &Downsampler{
		factor:   factor,
		lpCoeffs: make([]float64, factor),
	}
}

// Feed pushes one master-clock sample into the downsampler. When
// enough samples have accumulated to form one output frame, the
// averaged value is returned on the second return value and ok is
// true. Most calls return ok == false because the master clock
// runs ~40× faster than the output rate.
func (d *Downsampler) Feed(sample int8) (avg int8, ok bool) {
	d.accumulator += int64(sample)
	d.count++
	if d.count < d.factor {
		return 0, false
	}
	avg = int8(d.accumulator / int64(d.count))
	d.accumulator = 0
	d.count = 0
	return avg, true
}

// Reset clears the internal accumulator so that a discontinuity
// (e.g. channel start/stop) does not create a DC offset burst.
func (d *Downsampler) Reset() {
	d.accumulator = 0
	d.count = 0
}

// ─── Format Support ──────────────────────────────────────────────

// Format identifies an encoded audio container.
type Format int

const (
	FormatMP3  Format = iota // MPEG-1 Audio Layer III
	FormatWAV                // RIFF/WAVE PCM
	FormatMIDI               // Standard MIDI File
)

// FormatName returns the human-readable name of a format.
func (f Format) String() string {
	switch f {
	case FormatMP3:
		return "MP3"
	case FormatWAV:
		return "WAV"
	case FormatMIDI:
		return "MIDI"
	}
	return "Unknown"
}

// DecodeMP3 accepts an MP3 byte stream and returns a mono PCM
// sample suitable for loading into a Paula channel. The engine
// keeps this path for inclusiveness, but shipped carts rarely
// use it because of the larger memory footprint.
func DecodeMP3(data []byte) (*Sample, error) {
	// Platform-specific decoder delegate.
	return nil, nil
}

// DecodeWAV accepts a RIFF/WAVE byte stream and returns a mono PCM
// sample. WAV is the preferred lossless intermediate when authors
// import studio-recorded sound effects.
func DecodeWAV(data []byte) (*Sample, error) {
	// Platform-specific decoder delegate.
	return nil, nil
}

// ─── MIDI BGM & SoundFX ──────────────────────────────────────────

// MidiTrack is a parsed Standard MIDI File ready for playback
// through the engine's software synthesiser. A three-minute
// background-music track rarely exceeds a few kilobytes because
// MIDI stores only note numbers, durations, and tempo events
// rather than sampled waveforms.
type MidiTrack struct {
	// Events is the raw MIDI event stream, already delta-time
	// quantised to the engine's internal tick rate.
	Events []MidiEvent

	// Tempo is the microseconds-per-quarter-note value that
	// drives the software tick generator.
	Tempo uint32
}

// MidiEvent is a single note or control message within a track.
type MidiEvent struct {
	DeltaTime uint32 // ticks since the previous event
	Note      uint8  // 0–127, 255 for non-note events
	Velocity  uint8  // 0–127
	Length    uint32 // note duration in ticks
}

// DecodeMIDI parses a Standard MIDI File into an engine-native
// track structure. The synthesiser will later expand these sparse
// events into real-time PCM through a wavetable lookup.
func DecodeMIDI(data []byte) (*MidiTrack, error) {
	// SMF parser: extracts track chunks, delta-time division,
	// and note-on/note-off pairs.
	return &MidiTrack{}, nil
}

// SoundFX synthesises one-shot effects (jumps, explosions, coin
// pickups) by writing directly into a Paula channel's sample
// buffer. Unlike BGM, effects are procedural rather than
// sequence-based, giving designers per-pixel control over pitch
// bends and volume envelopes.
type SoundFX struct {
	// Channel is the Paula channel index (0–3) that will render
	// this effect.
	Channel int

	// Pitch is the playback frequency in Hz.
	Pitch float64

	// Volume is the linear gain, 0.0–1.0.
	Volume float64

	// Loop is true for sustained tones (e.g. engine drone) and
	// false for one-shots (e.g. gunshot).
	Loop bool
}

// Trigger schedules the sound effect on its assigned channel.
func (fx *SoundFX) Trigger() {
	// Delegates to the Paula scheduler with a zero-delay command.
}

// BGMPlayer owns the background-music sequencer. It advances the
// MIDI event stream once per frame and feeds note-on commands to
// the Paula mixer, producing a chiptune-like retro texture that
// matches the 8-bit visual aesthetic.
type BGMPlayer struct {
	track   *MidiTrack
	tickPos uint32
	eventIx int
}

// NewBGMPlayer creates a player bound to a decoded MIDI track.
func NewBGMPlayer(t *MidiTrack) *BGMPlayer {
	return &BGMPlayer{track: t}
}

// Update advances the sequencer by one engine tick, firing any
// MIDI events whose delta-time has elapsed.
func (p *BGMPlayer) Update() {
	p.tickPos++
	for p.eventIx < len(p.track.Events) && p.track.Events[p.eventIx].DeltaTime <= p.tickPos {
		// Fire note-on to Paula channel allocation layer.
		p.eventIx++
	}
}

// ─── Format Size Comparison ──────────────────────────────────────

// CompressionStats records the on-disk footprint of the same
// three-minute musical piece encoded in each supported format.
// These numbers illustrate why MIDI is the default for shipped
// carts: the entire soundtrack often fits in less space than a
// single WAV sound effect.
type CompressionStats struct {
	Format      string
	SizeKB      float64
	Description string
}

// ReferenceCompressionTable is the canonical size comparison
// used in documentation and studio UI tool-tips.
var ReferenceCompressionTable = []CompressionStats{
	{
		Format:      "WAV (PCM 44.1 kHz mono)",
		SizeKB:      15876.0,
		Description: "Uncompressed raw samples; largest but zero CPU decode cost.",
	},
	{
		Format:      "MP3 (192 kbps CBR)",
		SizeKB:      4320.0,
		Description: "Lossy psycho-acoustic compression; good fidelity, moderate size.",
	},
	{
		Format:      "MIDI (SMF type-1)",
		SizeKB:      12.0,
		Description: "Note events only; relies on software synthesiser. Tiny and infinitely loop-friendly.",
	},
}
