// Pixelforge Studio — no-code visual game editor.
//
// The studio main.go is intentionally tiny: it loads user settings,
// sizes the window, and asks Ebitengine to drive the editor. All editor
// logic lives in the editor package.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

const windowTitle = "Pixelforge Studio"

func main() {
	settings := editor.LoadSettings()

	ebiten.SetWindowTitle(windowTitle)
	ebiten.SetWindowSize(settings.WindowWidth, settings.WindowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(editor.NewWithSettings(settings)); err != nil {
		log.Fatalf("pixelforge studio: %v", err)
	}
}
