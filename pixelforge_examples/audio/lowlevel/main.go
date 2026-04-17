// Example of programming audio using the low-level API.
//
// This API can be used, for example, to implement packages
// capable of playing module formats (MOD, XM, etc.) and sound effects.
package main

import (
	_ "embed"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
)

//go:embed "wave.wav"
var clickWav []byte

func main() {
	sample := pixelforge_audio.DecodeWav(clickWav)

	pixelforge_cofont.Print("PRESS LEFT MOUSE BUTTON TO PLAY SFX", 90, 80)

	pixelforge.Init = func() {
		// The sample must be loaded before use,
		// but communication with the audio backend is only possible after starting the game.
		pixelforge_audio.LoadSample(sample)
	}

	// function that schedules playing the SFX
	scheduleSFX := func() {
		// All commands are scheduled with a minimum delay of 0 seconds.
		// However, this doesn't mean the sound plays instantly.
		// On desktop, the delay is around 20 ms; in browsers, about 60 ms.
		// The backend automatically delays commands to reduce audio glitches.
		delay := 0.0

		// Use two channels at once.
		// Chan1 is sent to a left speaker, Chan2 is sent to a right speaker.
		ch := pixelforge_audio.Chan1 | pixelforge_audio.Chan2

		// remove all planned commands from channels
		pixelforge_audio.ClearChan(ch, delay)

		// set the sample to play from the beginning (offset=0)
		pixelforge_audio.SetSample(ch, sample, 0, delay)

		// the sound is very short, so we need to loop it.
		// the loop covers the entire sample.
		pixelforge_audio.SetLoop(ch, 0, sample.Len(), pixelforge_audio.LoopForward, delay)

		for i := 1.0; i > -0.01; i -= 0.01 {
			// gradually reduce the volume to 0
			pixelforge_audio.SetVolume(ch, i, delay)

			// gradually reduce the pitch down to 0
			pitch := 1.0 - delay
			pixelforge_audio.SetPitch(ch, pitch, delay)
			delay += 0.01
		}
	}

	leftDown := pixelforge_mouse.EventButton{
		Type:   pixelforge_mouse.EventButtonDown,
		Button: pixelforge_mouse.Left,
	}
	pixelforge_mouse.ButtonTarget().Subscribe(leftDown, func(pixelforge_mouse.EventButton, pixelforge_event.Handler) {
		scheduleSFX()
	})

	pixelforge_loop.Target().Subscribe(pixelforge_loop.EventUpdate, func(pixelforge_loop.Event, pixelforge_event.Handler) {
		// pixelforge_audio.Time should be used by code that wants to, for example,
		// record when track playback started.
		log.Println("TIME", pixelforge_audio.Time)
	})

	pixelforge_ebiten.Run()
}
