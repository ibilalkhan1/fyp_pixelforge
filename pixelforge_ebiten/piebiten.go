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
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/audio"
	"strconv"

	"github.com/ibilalkhan1/fyp_pixelforge"
	piaudio "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
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

// SetNativeOverlay registers a callback drawn after the game frame is
// scaled to the window. Use it for debug overlays that should render at
// native (window) resolution rather than the upscaled canvas.
//
// Pass nil to clear a previously registered overlay.
func SetNativeOverlay(fn func(screen *ebiten.Image)) {
	internal.NativeOverlay = fn
}

// SetMetricsBorder sets the per-side black border (in device-independent
// pixels) reserved around the game canvas for the metrics overlay.
// The right border is typically wider to serve as a dedicated column for
// the overlay panels. Pass the same value for all sides if you want a
// uniform border.
func SetMetricsBorder(left, right, top, bottom int) {
	if left < 0 {
		left = 0
	}
	if right < 0 {
		right = 0
	}
	if top < 0 {
		top = 0
	}
	if bottom < 0 {
		bottom = 0
	}
	internal.MetricsBorderLeft = float64(left)
	internal.MetricsBorderRight = float64(right)
	internal.MetricsBorderTop = float64(top)
	internal.MetricsBorderBottom = float64(bottom)
}

// GameCanvasBounds returns where the game canvas was drawn in the window
// (top-left x, y, width, height), in device-independent pixels. The
// metrics overlay uses this to position its text in the black border
// area around the canvas.
func GameCanvasBounds() (x, y, w, h float64) {
	return internal.GameCanvasX, internal.GameCanvasY, internal.GameCanvasW, internal.GameCanvasH
}

// SetTickHook installs the arcade-shipping U4 single-render-path seam.
// When fn is non-nil, the Ebitengine game's per-tick update + draw
// sequence delegates to fn(tick) instead of running the legacy
// pixelforge.Update + pixelforge.Draw + event-publish dance. The hook
// is expected to advance + render game state by exactly one tick,
// typically by delegating to pixelforge_render.RenderTickAtScreen
// against the *capsuleruntime.Runtime captured at install time.
//
// Pass nil to clear a previously registered hook and restore the
// legacy update/draw path. The default behaviour (no hook) preserves
// pixel-equivalence with every pre-U4 caller of Run, so existing
// games that don't opt in to the seam are unaffected.
//
// The hook owns publishing the Update / LateUpdate / Draw / LateDraw
// events to pixelforge_loop.Target (RenderTickAtScreen does this
// internally); pixelforge_ebiten still publishes the corresponding
// DebugTarget events so engine-diagnostics overlays continue to fire
// regardless of which path is active.
func SetTickHook(fn func(tick uint64)) {
	internal.TickHook = fn
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
