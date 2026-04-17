package main

import (
	_ "embed"
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
)

//go:embed "gamepad.png"
var gamepadPNG []byte

func main() {
	pixelforge.SetScreenSize(85, 60)
	pixelforge.Palette = pixelforge.DecodePalette(gamepadPNG)
	gamepad := pixelforge.DecodeCanvas(gamepadPNG)

	buttonSprites := map[pixelforge_pad.Button]pixelforge.Sprite{
		pixelforge_pad.X:      pixelforge.SpriteFrom(gamepad, 48, 18, 9, 9),
		pixelforge_pad.Y:      pixelforge.SpriteFrom(gamepad, 58, 10, 9, 9),
		pixelforge_pad.B:      pixelforge.SpriteFrom(gamepad, 68, 18, 9, 9),
		pixelforge_pad.A:      pixelforge.SpriteFrom(gamepad, 58, 26, 9, 9),
		pixelforge_pad.Left:   pixelforge.SpriteFrom(gamepad, 11, 19, 8, 8),
		pixelforge_pad.Right:  pixelforge.SpriteFrom(gamepad, 25, 19, 8, 8),
		pixelforge_pad.Top:    pixelforge.SpriteFrom(gamepad, 19, 13, 6, 8),
		pixelforge_pad.Bottom: pixelforge.SpriteFrom(gamepad, 19, 25, 6, 8),
	}

	pixelforge.Draw = func() {
		pixelforge.Cls()
		pixelforge.DrawCanvas(gamepad, 0, 0)

		for button, sprite := range buttonSprites {
			if pixelforge_pad.Duration(button) > 0 { // duration is > 0 when button is pressed
				pixelforge.DrawSprite(sprite, sprite.X, sprite.Y+1) // draw pressed button
			}
		}

		pixelforge_cofont.Print("PRESS BTN ON GAMEPAD", 3, 50)
	}

	pixelforge_ebiten.Run()
}
