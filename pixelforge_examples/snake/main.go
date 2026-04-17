// A minimal snake (worm) game implementation.
package main

import (
	_ "embed"
	"math/rand"
	"slices"
	"strconv"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
)

var snake []pixelforge.Position           // snake body segments
var fruit pixelforge.Position             // fruit location
var direction pixelforge.Position         // current snake heading
var possibleDirection pixelforge.Position // next possible snake direction

var frame = 0
var speed int
var gameOver = false

const gridSize = 8
const width = 16
const height = 16

var leftDirection = pixelforge.Position{X: -1}
var rightDirection = pixelforge.Position{X: 1}
var upDirection = pixelforge.Position{Y: -1}
var downDirection = pixelforge.Position{Y: 1}

func startNewGame() {
	gameOver = false

	speed = 5
	direction = pixelforge.Position{X: 1, Y: 0}
	possibleDirection = direction
	fruit = pixelforge.Position{X: 8, Y: 8}
	snake = []pixelforge.Position{
		{X: 4, Y: 4},
		{X: 3, Y: 4},
		{X: 2, Y: 4},
	}
}

func handleUserInput() {
	if (pixelforge_key.Duration(pixelforge_key.Left) > 0 || pixelforge_pad.Duration(pixelforge_pad.Left) > 0) && direction.X == 0 {
		possibleDirection = leftDirection
	}
	if (pixelforge_key.Duration(pixelforge_key.Right) > 0 || pixelforge_pad.Duration(pixelforge_pad.Right) > 0) && direction.X == 0 {
		possibleDirection = rightDirection
	}
	if (pixelforge_key.Duration(pixelforge_key.Up) > 0 || pixelforge_pad.Duration(pixelforge_pad.Top) > 0) && direction.Y == 0 {
		possibleDirection = upDirection
	}
	if (pixelforge_key.Duration(pixelforge_key.Down) > 0 || pixelforge_pad.Duration(pixelforge_pad.Bottom) > 0) && direction.Y == 0 {
		possibleDirection = downDirection
	}
}

func spawnFruit() {
	fruit.X = rand.Intn(width)
	fruit.Y = rand.Intn(height)
}

func update() {
	if gameOver {
		if pixelforge_key.Duration(pixelforge_key.Enter) > 0 || pixelforge_pad.Duration(pixelforge_pad.A) > 0 {
			startNewGame()
		}
		return
	}

	handleUserInput()

	frame += 1
	if frame%speed == 0 {
		direction = possibleDirection
		// create new head position
		newPos := snake[0].Add(direction)

		// collisions
		// check collision with wall
		if newPos.X < 0 || newPos.X >= width || newPos.Y < 0 || newPos.Y >= height {
			gameOver = true
			return
		}
		// check collision with the snake itself
		for i := 0; i < len(snake); i++ {
			if snake[i] == newPos {
				gameOver = true
				return
			}
		}

		// move the snake body
		snake = slices.Insert(snake, 0, newPos)
		// check if it eats the apple
		if newPos == fruit {
			spawnFruit()
			if len(snake)%10 == 0 && speed > 0 {
				speed -= 1 // increase speed
			}
		} else {
			snake = snake[:len(snake)-1] // remove tail
		}
	}
}

func draw() {
	pixelforge.Screen().Clear(0)

	drawGrid()
	drawFruit()
	drawSnake()

	if gameOver {
		score := "SCORE: " + strconv.Itoa(len(snake)-3)
		pixelforge_cofont.Sheet.PrintStroked(score, 54, 58, 7, 5)
		pixelforge.SetColor(7)
		pixelforge_cofont.Sheet.Print("HIT ENTER TO START", 33, 74)
	}
}

func drawGrid() {
	pixelforge.SetColor(1)
	for i := 0; i < width; i++ {
		pixelforge.Line(i*gridSize, 0, i*gridSize, height*gridSize)
		pixelforge.Line(0, i*gridSize, width*gridSize, i*gridSize)
	}
}

func drawFruit() {
	verticalShift := frame % 10 / 5 // simple animation
	pixelforge.DrawSprite(fruitSprite, fruit.X*gridSize, fruit.Y*gridSize+verticalShift)
}

func drawSnake() {
	var headSprite pixelforge.Sprite
	switch direction {
	case leftDirection:
		headSprite = headHorizontal.WithFlipX(true) // reuse sprite
	case rightDirection:
		headSprite = headHorizontal
	case upDirection:
		headSprite = headVertical
	case downDirection:
		headSprite = headVertical.WithFlipY(true) // reuse sprite
	}
	pixelforge.DrawSprite(headSprite, snake[0].X*gridSize, snake[0].Y*gridSize)
	for i := 1; i < len(snake); i++ {
		bodySegment := snake[i]
		pixelforge.DrawSprite(bodySprite, bodySegment.X*gridSize, bodySegment.Y*gridSize)
	}
}

//go:embed "sprites.png"
var spritesPNG []byte

var fruitSprite, headVertical, headHorizontal, bodySprite pixelforge.Sprite

func main() {
	pixelforge.Palette = pixelforge.DecodePalette(spritesPNG)
	sprites := pixelforge.DecodeCanvas(spritesPNG)
	fruitSprite = pixelforge.SpriteFrom(sprites, 0, 0, 8, 8)
	headVertical = pixelforge.SpriteFrom(sprites, 8, 0, 8, 8)
	headHorizontal = pixelforge.SpriteFrom(sprites, 16, 0, 8, 8)
	bodySprite = pixelforge.SpriteFrom(sprites, 24, 0, 8, 8)

	pixelforge.SetTPS(30) // 60 is for hardcore players!
	pixelforge.SetScreenSize(gridSize*width, gridSize*height)
	pixelforge.Update = update
	pixelforge.Draw = draw

	startNewGame()

	pixelforge_ebiten.Run()
}
