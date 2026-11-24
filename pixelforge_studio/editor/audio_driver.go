package editor

// AudioDriver services the mixer around each editor tick, including modal ticks.
// Queued preview commands never reach the audio device without OnAfterUpdate.
type AudioDriver interface {
	OnBeforeUpdate()
	OnAfterUpdate()
}

// SetAudioDriver connects Studio's audition controls to its live mixer.
func (e *Editor) SetAudioDriver(driver AudioDriver) { e.audioDriver = driver }
