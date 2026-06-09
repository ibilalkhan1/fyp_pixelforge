// Package pixelforge_gametree provides a declarative, sentence-like API
// for building a snake game. The player describes what the game is
// (player sprite, background colour, wall size, movement keys, lose
// condition, score) and the package handles every frame of pre-order
// scene-graph updates, collision resolution, and sound playback
// behind the scenes.
package pixelforge_gametree

import (
	"math/rand"
	"os"
	"slices"
	"strconv"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_metrics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
)

// Game is the declarative game builder. Call its methods in main()
// to describe the game, then call Loop() to start playing.
type Game struct {
	// ── configuration set by the user ──────────────────────────────
	spritePath string
	bgColor    int
	gridSize   int
	upKey      string
	downKey    string
	leftKey    string
	rightKey   string
	loseCrash  bool
	loseTarget string // "wall" | "self"
	showScore  bool
	soundOn    bool
	boxes      []pixelforge.Position // optional obstacles

	// ── runtime state ──────────────────────────────────────────────
	snake             []pixelforge.Position
	fruit             pixelforge.Position
	direction         pixelforge.Position
	possibleDirection pixelforge.Position
	frame             int
	gameOver          bool

	// ── sprites ────────────────────────────────────────────────────
	fruitSprite, headVertical, headHorizontal, bodySprite pixelforge.Sprite

	// ── audio ──────────────────────────────────────────────────────
	eatSound     *pixelforge_audio.Sample
	crashSound   *pixelforge_audio.Sample
	restartSound *pixelforge_audio.Sample
	soundsLoaded bool
}

// NewGame creates an empty game description.
func NewGame() *Game {
	return &Game{}
}

// Player chooses the 32×8 sprite sheet for the snake and fruit.
func (g *Game) Player(spriteSheet string) {
	g.spritePath = spriteSheet
}

// Background sets the palette index used to clear the screen.
func (g *Game) Background(color int) {
	g.bgColor = color
}

// Wall sets the grid size (width and height in tiles). The play
// area becomes size×size.
func (g *Game) Wall(size int) {
	g.gridSize = size
}

// Boxes adds static obstacle tiles that the player must avoid.
func (g *Game) Boxes(positions []pixelforge.Position) {
	g.boxes = append(g.boxes, positions...)
}

// Movement binds the four directional inputs by name.
// Supported key names: "arrowUp", "arrowDown", "arrowLeft", "arrowRight".
func (g *Game) Movement(up, down, left, right string) {
	g.upKey = up
	g.downKey = down
	g.leftKey = left
	g.rightKey = right
}

// LoseCond configures what ends the game. crash must be "crash".
// target can be "wall", "self", or "both".
func (g *Game) LoseCond(crash, target string) {
	g.loseCrash = (crash == "crash")
	g.loseTarget = target
}

// Score enables the on-screen score display.
func (g *Game) Score() {
	g.showScore = true
}

// Sound toggles the three built-in chiptune effects.
func (g *Game) Sound(on bool) {
	g.soundOn = on
}

// Loop initialises the game from the description, wires the engine
// hooks, and enters the main loop. The loop keeps running; when the
// player loses the game auto-restarts after a key press.
func (g *Game) Loop() {
	g.init()
	pixelforge.Update = g.update
	pixelforge.Draw = g.draw
	pixelforge_metrics.Start()
	pixelforge_ebiten.Run()
}

// ─── initialisation ──────────────────────────────────────────────

func (g *Game) init() {
	if g.spritePath == "" {
		g.spritePath = "sprites.png"
	}
	if g.gridSize == 0 {
		g.gridSize = 16
	}

	data, err := os.ReadFile(g.spritePath)
	if err != nil {
		panic("snake: cannot read sprite sheet: " + err.Error())
	}
	pixelforge.Palette = pixelforge.DecodePalette(data)
	canvas := pixelforge.DecodeCanvas(data)
	g.fruitSprite = pixelforge.SpriteFrom(canvas, 0, 0, 8, 8)
	g.headVertical = pixelforge.SpriteFrom(canvas, 8, 0, 8, 8)
	g.headHorizontal = pixelforge.SpriteFrom(canvas, 16, 0, 8, 8)
	g.bodySprite = pixelforge.SpriteFrom(canvas, 24, 0, 8, 8)

	pixelforge.SetTPS(30)
	pixelforge.SetScreenSize(tileSize*g.gridSize, tileSize*g.gridSize)
	g.startNewGame()
}

func (g *Game) loadSounds() {
	if g.soundsLoaded || !g.soundOn {
		return
	}
	if data, err := os.ReadFile("eat.wav"); err == nil {
		g.eatSound = pixelforge_audio.DecodeWav(data)
		pixelforge_audio.LoadSample(g.eatSound)
	}
	if data, err := os.ReadFile("crash.wav"); err == nil {
		g.crashSound = pixelforge_audio.DecodeWav(data)
		pixelforge_audio.LoadSample(g.crashSound)
	}
	if data, err := os.ReadFile("restart.wav"); err == nil {
		g.restartSound = pixelforge_audio.DecodeWav(data)
		pixelforge_audio.LoadSample(g.restartSound)
	}
	g.soundsLoaded = true
}

func (g *Game) playEat() {
	if g.soundOn && g.eatSound != nil {
		pixelforge_audio.Play(pixelforge_audio.Chan1, g.eatSound, 1.0, 1.0)
	}
}

func (g *Game) playCrash() {
	if g.soundOn && g.crashSound != nil {
		pixelforge_audio.Play(pixelforge_audio.Chan2, g.crashSound, 1.0, 1.0)
	}
}

func (g *Game) playRestart() {
	if g.soundOn && g.restartSound != nil {
		pixelforge_audio.Play(pixelforge_audio.Chan3, g.restartSound, 1.0, 1.0)
	}
}

// ─── game constants ──────────────────────────────────────────────

const tileSize = 8

var (
	leftDir  = pixelforge.Position{X: -1}
	rightDir = pixelforge.Position{X: 1}
	upDir    = pixelforge.Position{Y: -1}
	downDir  = pixelforge.Position{Y: 1}
)

// ─── game logic ──────────────────────────────────────────────────

func (g *Game) startNewGame() {
	g.gameOver = false
	g.direction = pixelforge.Position{X: 1, Y: 0}
	g.possibleDirection = g.direction
	g.fruit = pixelforge.Position{X: g.gridSize / 2, Y: g.gridSize / 2}
	g.snake = []pixelforge.Position{
		{X: g.gridSize / 4, Y: g.gridSize / 2},
		{X: g.gridSize/4 - 1, Y: g.gridSize / 2},
		{X: g.gridSize/4 - 2, Y: g.gridSize / 2},
	}
}

func (g *Game) handleInput() {
	if (g.keyPressed(g.leftKey)) && g.direction.X == 0 {
		g.possibleDirection = leftDir
	}
	if (g.keyPressed(g.rightKey)) && g.direction.X == 0 {
		g.possibleDirection = rightDir
	}
	if (g.keyPressed(g.upKey)) && g.direction.Y == 0 {
		g.possibleDirection = upDir
	}
	if (g.keyPressed(g.downKey)) && g.direction.Y == 0 {
		g.possibleDirection = downDir
	}
}

func (g *Game) keyPressed(name string) bool {
	// Arrow aliases also check the gamepad d-pad.
	switch name {
	case "arrowUp":
		return pixelforge_key.Duration(pixelforge_key.Up) > 0 || pixelforge_pad.Duration(pixelforge_pad.Top) > 0
	case "arrowDown":
		return pixelforge_key.Duration(pixelforge_key.Down) > 0 || pixelforge_pad.Duration(pixelforge_pad.Bottom) > 0
	case "arrowLeft":
		return pixelforge_key.Duration(pixelforge_key.Left) > 0 || pixelforge_pad.Duration(pixelforge_pad.Left) > 0
	case "arrowRight":
		return pixelforge_key.Duration(pixelforge_key.Right) > 0 || pixelforge_pad.Duration(pixelforge_pad.Right) > 0
	}
	// Any other key name is looked up in the keyboard map.
	k := resolveKey(name)
	if k != "" {
		return pixelforge_key.Duration(k) > 0
	}
	return false
}

// resolveKey maps human-friendly key names to pixelforge_key constants.
func resolveKey(name string) pixelforge_key.Key {
	switch name {
	// directions
	case "up":
		return pixelforge_key.Up
	case "down":
		return pixelforge_key.Down
	case "left":
		return pixelforge_key.Left
	case "right":
		return pixelforge_key.Right
	// letters
	case "a", "A":
		return pixelforge_key.A
	case "b", "B":
		return pixelforge_key.B
	case "c", "C":
		return pixelforge_key.C
	case "d", "D":
		return pixelforge_key.D
	case "e", "E":
		return pixelforge_key.E
	case "f", "F":
		return pixelforge_key.F
	case "g", "G":
		return pixelforge_key.G
	case "h", "H":
		return pixelforge_key.H
	case "i", "I":
		return pixelforge_key.I
	case "j", "J":
		return pixelforge_key.J
	case "k", "K":
		return pixelforge_key.K
	case "l", "L":
		return pixelforge_key.L
	case "m", "M":
		return pixelforge_key.M
	case "n", "N":
		return pixelforge_key.N
	case "o", "O":
		return pixelforge_key.O
	case "p", "P":
		return pixelforge_key.P
	case "q", "Q":
		return pixelforge_key.Q
	case "r", "R":
		return pixelforge_key.R
	case "s", "S":
		return pixelforge_key.S
	case "t", "T":
		return pixelforge_key.T
	case "u", "U":
		return pixelforge_key.U
	case "v", "V":
		return pixelforge_key.V
	case "w", "W":
		return pixelforge_key.W
	case "x", "X":
		return pixelforge_key.X
	case "y", "Y":
		return pixelforge_key.Y
	case "z", "Z":
		return pixelforge_key.Z
	// digits
	case "0":
		return pixelforge_key.Digit0
	case "1":
		return pixelforge_key.Digit1
	case "2":
		return pixelforge_key.Digit2
	case "3":
		return pixelforge_key.Digit3
	case "4":
		return pixelforge_key.Digit4
	case "5":
		return pixelforge_key.Digit5
	case "6":
		return pixelforge_key.Digit6
	case "7":
		return pixelforge_key.Digit7
	case "8":
		return pixelforge_key.Digit8
	case "9":
		return pixelforge_key.Digit9
	// common named keys
	case "space", "Space":
		return pixelforge_key.Space
	case "enter", "Enter", "return", "Return":
		return pixelforge_key.Enter
	case "esc", "Esc", "escape", "Escape":
		return pixelforge_key.Esc
	case "tab", "Tab":
		return pixelforge_key.Tab
	case "shift", "Shift":
		return pixelforge_key.Shift
	case "ctrl", "Ctrl", "control", "Control":
		return pixelforge_key.Ctrl
	case "alt", "Alt":
		return pixelforge_key.Alt
	case "backspace", "Backspace":
		return pixelforge_key.Backspace
	case "capslock", "CapsLock":
		return pixelforge_key.CapsLock
	case "minus", "Minus", "-":
		return pixelforge_key.Minus
	case "equal", "Equal", "=":
		return pixelforge_key.Equal
	case "comma", "Comma", ",":
		return pixelforge_key.Comma
	case "period", "Period", ".":
		return pixelforge_key.Period
	case "slash", "Slash", "/":
		return pixelforge_key.Slash
	case "semicolon", "Semicolon", ";":
		return pixelforge_key.Semicolon
	case "quote", "Quote", "'":
		return pixelforge_key.Quote
	case "backslash", "Backslash", `\`:
		return pixelforge_key.Backslash
	case "backquote", "Backquote", "`":
		return pixelforge_key.Backquote
	case "bracketleft", "BracketLeft", "[":
		return pixelforge_key.BracketLeft
	case "bracketright", "BracketRight", "]":
		return pixelforge_key.BracketRight
	// function keys
	case "f1", "F1":
		return pixelforge_key.F1
	case "f2", "F2":
		return pixelforge_key.F2
	case "f3", "F3":
		return pixelforge_key.F3
	case "f4", "F4":
		return pixelforge_key.F4
	case "f5", "F5":
		return pixelforge_key.F5
	case "f6", "F6":
		return pixelforge_key.F6
	case "f7", "F7":
		return pixelforge_key.F7
	case "f8", "F8":
		return pixelforge_key.F8
	case "f9", "F9":
		return pixelforge_key.F9
	case "f10", "F10":
		return pixelforge_key.F10
	case "f11", "F11":
		return pixelforge_key.F11
	case "f12", "F12":
		return pixelforge_key.F12
	}
	return ""
}

func (g *Game) spawnFruit() {
	g.fruit.X = rand.Intn(g.gridSize)
	g.fruit.Y = rand.Intn(g.gridSize)
}

func (g *Game) update() {
	g.loadSounds()

	if g.gameOver {
		if pixelforge_key.Duration(pixelforge_key.Enter) > 0 || pixelforge_pad.Duration(pixelforge_pad.A) > 0 {
			g.playRestart()
			g.startNewGame()
		}
		return
	}

	g.handleInput()
	g.frame++

	speed := 5 // default speed, could be made configurable later
	if g.frame%speed == 0 {
		g.direction = g.possibleDirection
		newPos := g.snake[0].Add(g.direction)

		// wall collision
		if g.loseTarget == "wall" || g.loseTarget == "both" {
			if newPos.X < 0 || newPos.X >= g.gridSize || newPos.Y < 0 || newPos.Y >= g.gridSize {
				g.playCrash()
				g.gameOver = true
				return
			}
		}

		// self collision
		if g.loseTarget == "self" || g.loseTarget == "both" {
			for i := 0; i < len(g.snake); i++ {
				if g.snake[i] == newPos {
					g.playCrash()
					g.gameOver = true
					return
				}
			}
		}

		// box collision
		for _, b := range g.boxes {
			if newPos == b {
				g.playCrash()
				g.gameOver = true
				return
			}
		}

		g.snake = slices.Insert(g.snake, 0, newPos)
		if newPos == g.fruit {
			g.playEat()
			g.spawnFruit()
		} else {
			g.snake = g.snake[:len(g.snake)-1]
		}
	}
}

// ─── rendering ───────────────────────────────────────────────────

func (g *Game) draw() {
	pixelforge.Screen().Clear(pixelforge.Color(g.bgColor))
	g.drawGrid()
	g.drawFruit()
	g.drawSnake()

	if g.gameOver && g.showScore {
		screenW := tileSize * g.gridSize
		screenH := tileSize * g.gridSize

		score := "SCORE: " + strconv.Itoa(len(g.snake)-3)
		scoreW := len(score) * 4 // cofont ASCII chars are 4 px wide
		scoreX := (screenW - scoreW) / 2
		scoreY := screenH/2 - 8

		msg := "HIT ENTER TO START"
		msgW := len(msg) * 4
		msgX := (screenW - msgW) / 2
		msgY := screenH/2 + 8

		// Draw a solid grey backing box so the text is readable even when
		// the snake head or body sits behind it.
		pad := 4
		boxX := msgX - pad
		boxY := scoreY - pad
		boxW := msgW + 2*pad
		boxH := (msgY + 8) - scoreY + 2*pad

		pixelforge.SetColor(5) // dark grey box fill
		pixelforge.RectFill(boxX, boxY, boxX+boxW, boxY+boxH)
		pixelforge.SetColor(7) // white border
		pixelforge.Rect(boxX, boxY, boxX+boxW, boxY+boxH)

		// White text with a black stroke for maximum contrast on grey.
		pixelforge_cofont.Sheet.PrintStroked(score, scoreX, scoreY, 7, 0)
		pixelforge_cofont.Sheet.PrintStroked(msg, msgX, msgY, 7, 0)
	}
}

func (g *Game) drawGrid() {
	pixelforge.SetColor(1)
	for i := 0; i < g.gridSize; i++ {
		pixelforge.Line(i*tileSize, 0, i*tileSize, g.gridSize*tileSize)
		pixelforge.Line(0, i*tileSize, g.gridSize*tileSize, i*tileSize)
	}
}

func (g *Game) drawFruit() {
	shift := g.frame % 10 / 5
	pixelforge.DrawSprite(g.fruitSprite, g.fruit.X*tileSize, g.fruit.Y*tileSize+shift)
}

func (g *Game) drawSnake() {
	var head pixelforge.Sprite
	switch g.direction {
	case leftDir:
		head = g.headHorizontal.WithFlipX(true)
	case rightDir:
		head = g.headHorizontal
	case upDir:
		head = g.headVertical
	case downDir:
		head = g.headVertical.WithFlipY(true)
	}
	pixelforge.DrawSprite(head, g.snake[0].X*tileSize, g.snake[0].Y*tileSize)
	for i := 1; i < len(g.snake); i++ {
		pixelforge.DrawSprite(g.bodySprite, g.snake[i].X*tileSize, g.snake[i].Y*tileSize)
	}
}
