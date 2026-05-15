package pixelforge_project

// AudioSample describes one audio file imported into the project. The
// engine's Paula mixer is 4-channel mono; AudioSample carries hints
// (suggested channel priority, looping) that M6 uses to auto-allocate
// channels at runtime.
type AudioSample struct {
	Name string `json:"name"`

	// RelativePath points inside the project's *-assets/ directory,
	// e.g. "audio/eat.wav". WAV is the v1 format; OGG/MP3 are deferred.
	RelativePath string `json:"relative_path"`

	// SuggestedChannelPriority categorises the sample so M6's allocator
	// can lock long BGM to channels 1-2 and round-robin short SFX on
	// 3-4. Valid values: "bgm", "sfx", "voice", "ambient".
	SuggestedChannelPriority string `json:"suggested_channel_priority"`

	// Loop indicates the runtime should loop the sample on Play. The
	// loop point defaults to the sample's start.
	Loop bool `json:"loop"`

	// SampleRateHz is the sample's native rate. The runtime resamples
	// when this differs from the audio backend rate. 0 means "use the
	// default from the WAV header" (informational only).
	SampleRateHz int `json:"sample_rate_hz"`
}

// AudioBinding maps an event topic (or scene state) to an AudioSample.
// M6 populates this from the comic-strip / session-grid editors; the
// runtime listens on the bus and triggers samples via piaudio.Play.
type AudioBinding struct {
	// SampleName references AudioSample.Name in the same project.
	SampleName string `json:"sample_name"`

	// Topic is the pievent topic that triggers playback, e.g.
	// "game/PlayerHit". Empty topic means manual trigger only.
	Topic string `json:"topic"`

	// SceneID, when non-empty, restricts the binding to a single
	// scene — useful for level-specific music cues.
	SceneID string `json:"scene_id"`

	// TriggerCondition is the GDevelop-style condition that must be
	// true for the trigger to fire. Empty means unconditional.
	TriggerCondition string `json:"trigger_condition"`

	// ForceChannel, 1..4, overrides the auto-allocator. 0 means
	// "let the allocator decide".
	ForceChannel int `json:"force_channel"`
}
