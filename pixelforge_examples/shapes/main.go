// Example showing how to draw shapes and use a mouse.
package main

import (
	_ "embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
	"math"
)

const (
	shapeColor = 15
	textColor  = 2
)

var (
	//go:embed sprite-sheet.png
	spriteSheetPNG []byte

	drawShape func(start, stop pixelforge.Position)

	drawFunctions = []func(start, stop pixelforge.Position){
		func(start, stop pixelforge.Position) {
			pixelforge.Rect(start.X, start.Y, stop.X, stop.Y)
			command := fmt.Sprintf("pixelforge.Rect(%d,%d,%d,%d)", start.X, start.Y, stop.X, stop.Y)
			printCmd(command)
		},
		func(start, stop pixelforge.Position) {
			pixelforge.RectFill(start.X, start.Y, stop.X, stop.Y)
			command := fmt.Sprintf("pixelforge.RectFill(%d,%d,%d,%d)", start.X, start.Y, stop.X, stop.Y)
			printCmd(command)
		},
		func(start, stop pixelforge.Position) {
			pixelforge.Line(start.X, start.Y, stop.X, stop.Y)
			command := fmt.Sprintf("pixelforge.Line(%d,%d,%d,%d)", start.X, start.Y, stop.X, stop.Y)
			printCmd(command)
		},
		func(start, stop pixelforge.Position) {
			r := radius(start.X, start.Y, stop.X, stop.Y)
			pixelforge.Circ(start.X, start.Y, r)

			command := fmt.Sprintf("pixelforge.Circ(%d,%d,%d)", start.X, start.Y, r)
			printCmd(command)
		},
		func(start, stop pixelforge.Position) {
			r := radius(start.X, start.Y, stop.X, stop.Y)
			pixelforge.CircFill(start.X, start.Y, r)
			command := fmt.Sprintf("pixelforge.CircFill(%d,%d,%d)", start.X, start.Y, r)
			printCmd(command)
		},
	}

	currentShapeIdx = 0

	shapeStart pixelforge.Position

	cursorSprites []pixelforge.Sprite
)

func main() {
	pixelforge.SetScreenSize(128, 128)
	pixelforge.SetTPS(60) // pixelforge.Update and pixelforge.Draw wil be executed 60 times per second

	pixelforge.Palette = pixelforge.DecodePalette(spriteSheetPNG)
	spriteSheet := pixelforge.DecodeCanvas(spriteSheetPNG)

	// create cursors sprite array for each shape
	for i := range drawFunctions {
		cursorSprite := pixelforge.SpriteFrom(spriteSheet, i*8, 0, 8, 8)
		cursorSprites = append(cursorSprites, cursorSprite)
	}

	pixelforge.Draw = func() {
		pixelforge.Cls()

		// change the shape if right mouse button was just pressed
		if pixelforge_mouse.Duration(pixelforge_mouse.Right) == 1 {
			currentShapeIdx++
			if currentShapeIdx == len(drawFunctions) {
				currentShapeIdx = 0
			}
		}

		// set initial coordinates on start dragging
		if pixelforge_mouse.Duration(pixelforge_mouse.Left) > 0 && drawShape == nil {
			shapeStart = pixelforge_mouse.Position
			drawShape = drawFunctions[currentShapeIdx]

		}

		// set coordinates during dragging
		if drawShape != nil {
			stop := pixelforge_mouse.Position
			pixelforge.SetColor(shapeColor)
			drawShape(shapeStart, stop)
		}

		if pixelforge_mouse.Duration(pixelforge_mouse.Left) == 0 {
			drawShape = nil
		}

		drawMousePointer()
	}

	ebiten.SetCursorMode(ebiten.CursorModeHidden) // hide cursor in Ebitengine
	pixelforge_ebiten.Run()
}

func drawMousePointer() {
	pixelforge.DrawSprite(cursorSprites[currentShapeIdx], pixelforge_mouse.Position.X, pixelforge_mouse.Position.Y)
}

func radius(x0, y0, x1, y1 int) int {
	dx := math.Abs(float64(x0 - x1))
	dy := math.Abs(float64(y0 - y1))
	return int(math.Max(dx, dy))
}

func printCmd(command string) {
	pixelforge.SetColor(textColor)
	pixelforge_cofont.Print(command, 8, 128-6-6)
}
