//go:build !js

//nolint:testpackage // Exercises the private host update path without opening an audio device.
package internal

import (
	"testing"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/ebitentesting"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten/internal/input"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/stretchr/testify/require"
)

type silentFrameAudio struct{}

func (silentFrameAudio) OnBeforeUpdate() {}
func (silentFrameAudio) OnAfterUpdate()  {}
func TestMain(m *testing.M)              { ebitentesting.MainWithRunLoop(m) }

func TestTickHookOwnsInitAndFrameStartEvents(t *testing.T) {
	previousInit, previousTPS := pf.Init, pf.TPS()
	previousFrame, previousTime := pf.Frame, pf.Time
	t.Cleanup(func() {
		TickHook = nil
		pf.Init = previousInit
		pf.SetTPS(previousTPS)
		pf.Frame = previousFrame
		pf.Time = previousTime
	})
	pf.SetTPS(30)
	initCalls, initEvents, frameEvents, ticks := 0, 0, 0, 0
	pf.Init = func() { initCalls++ }
	subscription := piloop.Target().SubscribeAll(func(event piloop.Event, _ pievent.Handler) {
		if event == piloop.EventInit {
			initEvents++
		}
		if event == piloop.EventFrameStart {
			frameEvents++
		}
	})
	t.Cleanup(func() { piloop.Target().Unsubscribe(subscription) })
	TickHook = func(tick uint64) {
		ticks++
		if tick == 0 {
			pf.Init()
			piloop.Target().Publish(piloop.EventInit)
		}
		piloop.Target().Publish(piloop.EventFrameStart)
	}
	game := &EbitenGame{audioBackend: silentFrameAudio{}}
	game.inputBackend = &input.Backend{
		Paused: &game.paused, LeftOffset: &game.left,
		TopOffset: &game.top, Scale: &game.scale,
	}
	for range 10 {
		require.NoError(t, game.Update())
	}
	require.Equal(t, 5, ticks)
	require.Equal(t, 1, initCalls, "host must not repeat the unified pipeline's init")
	require.Equal(t, 1, initEvents)
	require.Equal(t, ticks, frameEvents, "frame-start behaviours must execute once per game tick")
}
