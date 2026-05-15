package pixelforge_routine

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
)

// Play creates a Routine step that triggers playback of `sample` on
// `ch` once with the given pitch and volume, then advances to the
// next step on the same tick. Sample duration tracking is out of
// scope — Play is fire-and-forget.
//
// A nil sample logs nothing and advances immediately.
func Play(ch pixelforge_audio.Chan, sample *pixelforge_audio.Sample, pitch, vol float64) Step {
	return func() bool {
		if sample != nil {
			pixelforge_audio.Play(ch, sample, pitch, vol)
		}
		return true
	}
}
