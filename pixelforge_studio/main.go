// Pixelforge Studio — no-code visual game editor.
//
// The studio main.go is intentionally tiny: it loads user settings,
// sizes the window, and asks Ebitengine to drive the editor. All editor
// logic lives in the editor package.
//
// `pf-studio test [dir]` dispatches to the regression replayer; any
// other argv runs the editor.
package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/palette"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/scripting"
)

const windowTitle = "Pixelforge Studio"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		runTestSubcommand(os.Args[2:])
		return
	}

	settings := editor.LoadSettings()

	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowSize(settings.WindowWidth, settings.WindowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	e := editor.NewWithSettings(settings)
	palette.RegisterWith(e)
	capture.RegisterWith(e)
	scripting.RegisterWith(e)

	if err := ebiten.RunGame(e); err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}
}
