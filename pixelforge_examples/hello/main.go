package main

import (
	"github.com/ibilalkhan1/fyp_pixelforge"                   // import pixelforge core package
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont" // import very small pico-8 font
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten" // import backend
)

func main() {
	pixelforge.SetScreenSize(47, 9) // set custom screen size
	pixelforge.Draw = func() {      // draw will be executed each frame
		pixelforge_cofont.Print("HELLO WORLD", 2, 2)
	}
	pixelforge_ebiten.Run() // run backend
}
