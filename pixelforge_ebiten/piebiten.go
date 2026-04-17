// Package pixelforge_ebiten enables running your game using the [Ebitengine] backend.
//
// Ebitengine is a cross-platform game engine that supports Windows, macOS,
// Linux, FreeBSD, web browsers, Android, iOS, and even Nintendo Switch.
//
// To launch your game, use [Run] or [RunOrErr].
//
// This package also provides advanced functions for integrating Pixelforge
// with your own Ebitengine-based game, such as [CopyCanvasToEbitenImage].
//
// [Ebitengine]: https://ebitengine.org
package pixelforge_ebiten

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
	ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/audio"
	"strconv"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal"
)

// RememberWindow determines whether the game should open
// at its last window position, size, and monitor when set to true
var RememberWindow = false

// Run starts the Ebitengine backend. It panics if something goes wrong.
//
// If you want to handle errors gracefully, use [RunOrErr] instead.
//
// This function must be called from the first goroutine (the main thread).
func Run() {
	if err := RunOrErr(); err != nil {
		panic("piebiten.Run failed: " + err.Error())
	}
}

// RunOrErr starts the Ebitengine backend and returns an error if something goes wrong.
//
// This function must be called from the first goroutine (the main thread).
func RunOrErr() error {
	if internal.CurrentGoroutineID() != 1 {
		return errors.New("must be run from main goroutine 1")
	}
	internal.RememberWindow = RememberWindow
	return internal.RunOrErr() //nolint:wrapcheck
}

// CopyCanvasToEbitenImage copies the canvas to dst using the current
// palette in pixelforge.Palette and the palette mapping in pixelforge.PaletteMapping.
func CopyCanvasToEbitenImage(canvas pixelforge.Canvas, dst *ebiten.Image) {
	internal.CopyCanvasToEbitenImage(canvas, dst)
}

// StartAudioBackend starts the audio backend with the given Ebitengine audio.Context.
// Use if you want only pixelforge_audio functionality without Pixelforge's graphics.
//
// audio.Context must have a sample rate of 48000.
func StartAudioBackend(ctx *ebitenaudio.Context) Audio {
	if ctx.SampleRate() != audio.CtxSampleRate {
		panic("piebiten.StartAudioBackend: audio.Context must have " + strconv.Itoa(audio.CtxSampleRate) + " sample rate")
	}
	return audio.StartAudioBackend(ctx)
}

type Audio interface {
	piaudio.BackendInterface
	// OnBeforeUpdate must be called at the start of Ebitengine's Update function.
	OnBeforeUpdate()
	// OnAfterUpdate must be called at the end of Ebitengine's Update function.
	OnAfterUpdate()
}
