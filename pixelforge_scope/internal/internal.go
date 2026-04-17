package internal

import (
	"fmt"
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_debug"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
)

var bgColor, fgColor *pixelforge.Color

var consoleMode, pauseOnNextFrame bool

// Start launches the developer tools.
//
// Obecnie piscope wymaga, żeby gra miała rozdzielczość conajmniej 128
// pikseli w poziomie oraz 16 pikseli w pionie. Dodatkowa paleta gry musi
// używać conajmniej 2 kolorów.
//
// Pressing Ctrl+Shift+I will activate the tools in the game
func Start(backgroundColor, foregroundColor *pixelforge.Color) {
	bgColor = backgroundColor
	fgColor = foregroundColor

	// TODO Handle screen size change event and redraw entire gui.

	smallFont := picofont.Sheet

	registerShortcuts()

	gui := pigui.New()
	attachToolbar(gui)

	piloop.DebugTarget().Subscribe(piloop.EventUpdate, func(piloop.Event, pievent.Handler) {
		if consoleMode {
			gui.Update()

			if !pidebug.Paused() {
				theScreenRecorder.Save()
			}

			if pauseOnNextFrame {
				pidebug.SetPaused(true)
				pauseOnNextFrame = false
			}

			handleInputInConsoleMode()
		}
	})

	piloop.DebugTarget().Subscribe(piloop.EventLateDraw, func(piloop.Event, pievent.Handler) {
		if consoleMode {
			gui.Draw()

			screen := pixelforge.Screen()

			y := screen.H() - smallFont.Height - 1

			prev := pixelforge.SetColor(*bgColor)
			defer pixelforge.SetColor(prev)

			pixelColor := pixelforge.GetPixel(pimouse.Position.X, pimouse.Position.Y)
			if pixelColor != *bgColor {
				pixelforge.SetColor(pixelColor)
			} else {
				pixelforge.SetColor(1)
			}
			msg := fmt.Sprintf("%d(%d,%d)", pixelColor, pimouse.Position.X, pimouse.Position.Y)
			smallFont.Print(msg, 50, y+2)
		}
	})
}
