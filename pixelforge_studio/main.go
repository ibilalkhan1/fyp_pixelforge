package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

func main() {
	ebiten.SetWindowTitle("Pixelforge Studio")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := editor.NewEditor()

	// Auto-scan examples folder for sprites
	game.ScanAssetsFolder("pixelforge_examples")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
