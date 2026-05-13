package pixelforge_metrics

import (
	"fmt"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	pistat "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_stat"
)

var (
	fps       int
	frameCnt  int
	fpsTimer  time.Time
	started   bool
	handler   pievent.Handler
)

func Start() {
	if started {
		return
	}
	started = true
	pistat.Start()
	fpsTimer = time.Now()
	handler = piloop.DebugTarget().Subscribe(piloop.EventLateDraw, drawMetrics)
}

func Stop() {
	if !started {
		return
	}
	piloop.DebugTarget().Unsubscribe(handler)
	started = false
}

var inputKeys = []pixelforge_key.Key{
	pixelforge_key.Left, pixelforge_key.Right, pixelforge_key.Up, pixelforge_key.Down,
	pixelforge_key.A, pixelforge_key.B, pixelforge_key.C, pixelforge_key.D,
	pixelforge_key.W, pixelforge_key.S, pixelforge_key.D, pixelforge_key.Z,
	pixelforge_key.X, pixelforge_key.Space, pixelforge_key.Enter, pixelforge_key.Esc,
	pixelforge_key.ShiftLeft, pixelforge_key.ShiftRight,
	pixelforge_key.CtrlLeft, pixelforge_key.CtrlRight,
}

var padButtons = []pixelforge_pad.Button{
	pixelforge_pad.Left, pixelforge_pad.Right, pixelforge_pad.Top, pixelforge_pad.Bottom,
	pixelforge_pad.A, pixelforge_pad.B, pixelforge_pad.X, pixelforge_pad.Y,
}

func drawMetrics(piloop.Event, pievent.Handler) {
	frameCnt++
	if time.Since(fpsTimer) >= time.Second {
		fps = frameCnt
		frameCnt = 0
		fpsTimer = time.Now()
	}

	prevColor := pixelforge.SetColor(7)
	prevTarget := pixelforge.SetDrawTarget(pixelforge.Screen())

	x, y := 2, 2

	line := func(format string, args ...interface{}) {
		pixelforge.SetColor(7)
		pixelforge_cofont.Print(fmt.Sprintf(format, args...), x, y)
		y += 9
	}

	line("FPS:%-3d TPS:%-2d CPU:%-3d%% RAM:%-3dMB", fps, pixelforge.TPS(), pistat.CPU, pistat.MemoryMB)
	line("FRM:%-5d ALLOC:%-5d TIME:%.1fs", pixelforge.Frame, pistat.Allocs, pixelforge.Time)

	heldKeys := ""
	for _, k := range inputKeys {
		if pixelforge_key.Duration(k) > 0 {
			if heldKeys != "" {
				heldKeys += " "
			}
			heldKeys += string(k)
		}
	}
	heldPad := ""
	for _, b := range padButtons {
		if pixelforge_pad.Duration(b) > 0 {
			if heldPad != "" {
				heldPad += " "
			}
			heldPad += string(b)
		}
	}
	if heldKeys != "" {
		line("KEYS: %s", heldKeys)
	}
	if heldPad != "" {
		line("PAD: %s", heldPad)
	}

	pixelforge.SetColor(prevColor)
	pixelforge.SetDrawTarget(prevTarget)
}
