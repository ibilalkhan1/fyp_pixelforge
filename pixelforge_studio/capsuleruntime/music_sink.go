package capsuleruntime

import (
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// musicSink is the concrete MusicSink plan-009 U7 lands. A
// dedicated streaming-audio subsystem doesn't exist yet —
// pixelforge_audio.Play is sample-shot only — so the sink loops a
// registered sample on the dedicated music channel as the v1
// approximation. The seam (PlayMusic/StopMusic + rt.CurrentMusic)
// is the right one for the eventual streaming subsystem: when
// streaming lands, swapping the loop-on-Chan path for a real
// stream is a pure-internal change.
//
// rt.CurrentMusic carries the name of the currently-playing track
// so tests + diagnostics can assert "this music started playing"
// without polling the audio backend. Empty when stopped.
type musicSink struct {
	rt *Runtime
}

// newMusicSink constructs the production musicSink.
func newMusicSink(rt *Runtime) *musicSink {
	return &musicSink{rt: rt}
}

// musicChan is the audio channel the music sink reserves for BGM.
// Chan4 is intentional: Chan1-3 are the "sound effect" channels
// the audio-play_sound verb uses by default, so reserving Chan4
// keeps SFX from clobbering music.
const musicChan = pixelforge_audio.Chan4

// PlayMusic resolves the named sample via the audio registry and
// loops it on the music channel. Unknown sample names log + clear
// rt.CurrentMusic so the sink stays honest about whether music is
// actually playing.
func (m *musicSink) PlayMusic(name string) {
	if m == nil || m.rt == nil {
		return
	}
	if name == "" {
		debugDrop("audio/play_music", "empty name", nil)
		return
	}
	sample := pixelforge_audio.LookupSample(name)
	if sample == nil {
		log.Printf("[capsuleruntime] audio/play_music: unknown sample %q", name)
		m.rt.CurrentMusic = ""
		return
	}
	pixelforge_audio.Play(musicChan, sample, 1.0, 1.0)
	m.rt.CurrentMusic = name
}

// StopMusic halts the music channel and clears rt.CurrentMusic.
// Safe to call when no music is playing — the audio backend's
// ClearChan is idempotent.
func (m *musicSink) StopMusic() {
	if m == nil || m.rt == nil {
		return
	}
	pixelforge_audio.ClearChan(musicChan, 0)
	m.rt.CurrentMusic = ""
}

// CurrentMusic returns the name of the currently-playing track
// (empty when stopped). Exposed for diagnostics + tests; package-
// level callers can also read rt.CurrentMusic directly.
func (m *musicSink) CurrentMusic() string {
	if m == nil || m.rt == nil {
		return ""
	}
	return m.rt.CurrentMusic
}
