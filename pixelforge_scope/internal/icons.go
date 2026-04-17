package internal

import (
	_ "embed"
	"github.com/ibilalkhan1/fyp_pixelforge"
)

//go:embed "icons.png"
var iconsPNG []byte

var icons = struct {
	AlignTop    pixelforge.Sprite
	AlignBottom pixelforge.Sprite
	Screen      pixelforge.Sprite
	Palette     pixelforge.Sprite
	ColorTables pixelforge.Sprite
	Variables   pixelforge.Sprite
	Paint       pixelforge.Sprite
	Separator   pixelforge.Sprite
	Snap        pixelforge.Sprite
	Prev        pixelforge.Sprite
	Pause       pixelforge.Sprite
	Play        pixelforge.Sprite
	Next        pixelforge.Sprite
	Exit        pixelforge.Sprite
}{}

func init() {
	prevPalette := pixelforge.Palette
	defer func() {
		pixelforge.Palette = prevPalette
	}()

	pixelforge.Palette = pixelforge.DecodePalette(iconsPNG)
	iconsSheet := pixelforge.DecodeCanvas(iconsPNG)

	icons.AlignTop = pixelforge.SpriteFrom(iconsSheet, 0, 0, 8, 8)
	icons.AlignBottom = pixelforge.SpriteFrom(iconsSheet, 8, 0, 8, 8)
	icons.Screen = pixelforge.SpriteFrom(iconsSheet, 16, 0, 8, 8)
	icons.Palette = pixelforge.SpriteFrom(iconsSheet, 24, 0, 8, 8)
	icons.ColorTables = pixelforge.SpriteFrom(iconsSheet, 32, 0, 8, 8)
	icons.Variables = pixelforge.SpriteFrom(iconsSheet, 40, 0, 8, 8)
	icons.Paint = pixelforge.SpriteFrom(iconsSheet, 48, 0, 8, 8)
	icons.Separator = pixelforge.SpriteFrom(iconsSheet, 58, 0, 4, 8)
	icons.Snap = pixelforge.SpriteFrom(iconsSheet, 64, 0, 8, 8)
	icons.Prev = pixelforge.SpriteFrom(iconsSheet, 74, 0, 5, 8)
	icons.Pause = pixelforge.SpriteFrom(iconsSheet, 82, 0, 5, 8)
	icons.Play = pixelforge.SpriteFrom(iconsSheet, 90, 0, 5, 8)
	icons.Next = pixelforge.SpriteFrom(iconsSheet, 98, 0, 5, 8)
	icons.Exit = pixelforge.SpriteFrom(iconsSheet, 112, 0, 8, 8)
}
