package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

func main() {
	ebiten.SetWindowTitle("Pixelforge Studio")
	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	game := editor.NewEditor()

	// Scan for sprites
	game.ScanAssetsFolder("pixelforge_examples")

	state := game.State()
	fmt.Printf("Pixelforge Studio starting...\n")
	fmt.Printf("Loaded %d sprites:\n", len(state.Sprites))
	for i, s := range state.Sprites {
		fmt.Printf("  %d: %s (%dx%d)\n", i, s.Name, s.Width, s.Height)
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
