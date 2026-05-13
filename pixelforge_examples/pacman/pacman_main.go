// Pacman for Pixelforge
//
// Drop this file as pixelforge_examples/pacman/main.go
// then run: go run .
//
// Controls: Arrow keys to move
// Eats all dots to win. Ghosts kill on contact.
// Press R to restart at any time.

package main

import (
	"math"
	"time"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_font"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_metrics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
)

// ─── constants ──────────────────────────────────────────────────────────────

const (
	screenW = 224
	screenH = 248
	tileSize = 8

	// Palette indices (Pixelforge default palette – approximate PICO-8 colours)
	colBlack  pf.Color = 0
	colDkBlue pf.Color = 1
	colBlue   pf.Color = 2
	colGreen  pf.Color = 3
	colYellow pf.Color = 10
	colOrange pf.Color = 9
	colRed    pf.Color = 8
	colPink   pf.Color = 14
	colCyan   pf.Color = 12
	colWhite  pf.Color = 7
)

// ─── maze ────────────────────────────────────────────────────────────────────
// 0 = dot, 1 = wall, 2 = empty, 3 = power pellet, 4 = ghost house

const mazeW = 28
const mazeH = 31

// Classic Pac-Man maze (simplified, 28×31 tiles)
// '#' wall, '.' dot, 'o' power pellet, ' ' empty, 'G' ghost house
var mazeTemplate = [mazeH]string{
	"############################",
	"#............##............#",
	"#.####.#####.##.#####.####.#",
	"#o####.#####.##.#####.####o#",
	"#.####.#####.##.#####.####.#",
	"#..........................#",
	"#.####.##.########.##.####.#",
	"#.####.##.########.##.####.#",
	"#......##....##....##......#",
	"######.##### ## #####.######",
	"######.##### ## #####.######",
	"######.##          ##.######",
	"######.## ###--### ##.######",
	"######.## #GGGG  # ##.######",
	"      .   #GGGG  #   .      ",
	"######.## ######## ##.######",
	"######.##          ##.######",
	"######.## ######## ##.######",
	"######.## ######## ##.######",
	"#............##............#",
	"#.####.#####.##.#####.####.#",
	"#.####.#####.##.#####.####.#",
	"#o..##.......  .......##..o#",
	"###.##.##.########.##.##.###",
	"###.##.##.########.##.##.###",
	"#......##....##....##......#",
	"#.##########.##.##########.#",
	"#.##########.##.##########.#",
	"#..........................#",
	"############################",
	"                            ",
}

// ─── types ───────────────────────────────────────────────────────────────────

type Dir int

const (
	DirNone  Dir = 0
	DirRight Dir = 1
	DirLeft  Dir = 2
	DirUp    Dir = 3
	DirDown  Dir = 4
)

func (d Dir) dx() int {
	switch d {
	case DirRight:
		return 1
	case DirLeft:
		return -1
	}
	return 0
}
func (d Dir) dy() int {
	switch d {
	case DirUp:
		return -1
	case DirDown:
		return 1
	}
	return 0
}

type Pacman struct {
	// sub-pixel position (tile coords * 8)
	x, y    float64
	dir     Dir
	nextDir Dir
	anim    int // frame counter for mouth anim
}

type Ghost struct {
	x, y      float64
	dir       Dir
	color     pf.Color
	scared    bool
	scaredT   int
	dead      bool
	respawnT  int
}

type Tile struct {
	wall  bool
	dot   bool
	power bool
}

type GameState int

const (
	StatePlay    GameState = 0
	StateWin     GameState = 1
	StateDead    GameState = 2
	StateRestart GameState = 3
)

// ─── globals ──────────────────────────────────────────────────────────────────

var (
	tiles       [mazeH][mazeW]Tile
	pac         Pacman
	ghosts      [4]Ghost
	score       int
	lives       int
	dotsLeft    int
	state       GameState
	stateTimer  int
	powerTimer  int
	flashTimer  int
	eatCombo    int // ghost eat combo multiplier
)

// ─── audio helpers ───────────────────────────────────────────────────────────
func createToneSample(freq float64, duration time.Duration, isSquare bool) *pixelforge_audio.Sample {
	sampleRate := 44100
	samples := int(float64(sampleRate) * duration.Seconds())
	data := make([]int8, samples)

	for i := 0; i < samples; i++ {
		t := float64(i) / float64(sampleRate)
		var val float64
		if isSquare {
			val = math.Sin(2 * math.Pi * freq * t)
			if val > 0 {
				val = 1
			} else {
				val = -1
			}
		} else {
			val = math.Sin(2 * math.Pi * freq * t)
		}
		data[i] = int8(val * 127)
	}

	return pixelforge_audio.NewSample(data, uint16(sampleRate))
}

func sfxEatDot() {
	go func() {
		sample := createToneSample(400, 40*time.Millisecond, true)
		pixelforge_audio.LoadSample(sample)
		pixelforge_audio.Play(0, sample, 1.0, 0.5)
		time.Sleep(50 * time.Millisecond)
		pixelforge_audio.UnloadSample(sample)
	}()
}

func sfxEatPower() {
	go func() {
		for _, f := range []float64{200, 300, 400, 600} {
			sample := createToneSample(f, 60*time.Millisecond, true)
			pixelforge_audio.LoadSample(sample)
			pixelforge_audio.Play(0, sample, 1.0, 0.5)
			time.Sleep(70 * time.Millisecond)
			pixelforge_audio.UnloadSample(sample)
		}
	}()
}

func sfxEatGhost() {
	go func() {
		for _, f := range []float64{800, 600, 1000, 800} {
			sample := createToneSample(f, 50*time.Millisecond, false)
			pixelforge_audio.LoadSample(sample)
			pixelforge_audio.Play(0, sample, 1.0, 0.5)
			time.Sleep(60 * time.Millisecond)
			pixelforge_audio.UnloadSample(sample)
		}
	}()
}

func sfxDie() {
	go func() {
		freqs := []float64{600, 500, 400, 300, 250, 200, 150, 100}
		for _, f := range freqs {
			sample := createToneSample(f, 70*time.Millisecond, true)
			pixelforge_audio.LoadSample(sample)
			pixelforge_audio.Play(0, sample, 1.0, 0.5)
			time.Sleep(80 * time.Millisecond)
			pixelforge_audio.UnloadSample(sample)
		}
	}()
}

func sfxWin() {
	go func() {
		melody := []float64{523, 659, 784, 1047, 784, 1047}
		for _, f := range melody {
			sample := createToneSample(f, 120*time.Millisecond, false)
			pixelforge_audio.LoadSample(sample)
			pixelforge_audio.Play(0, sample, 1.0, 0.5)
			time.Sleep(130 * time.Millisecond)
			pixelforge_audio.UnloadSample(sample)
		}
	}()
}

func sfxSiren() {
	go func() {
		sample := createToneSample(180, 50*time.Millisecond, false)
		pixelforge_audio.LoadSample(sample)
		pixelforge_audio.Play(0, sample, 1.0, 0.3)
		time.Sleep(60 * time.Millisecond)
		pixelforge_audio.UnloadSample(sample)
	}()
}

func sfxScared() {
	go func() {
		sample := createToneSample(280, 50*time.Millisecond, true)
		pixelforge_audio.LoadSample(sample)
		pixelforge_audio.Play(0, sample, 1.0, 0.4)
		time.Sleep(60 * time.Millisecond)
		pixelforge_audio.UnloadSample(sample)
	}()
}

// ─── initialisation ──────────────────────────────────────────────────────────

func initGame() {
	score = 0
	lives = 3
	dotsLeft = 0
	powerTimer = 0
	eatCombo = 0
	state = StatePlay
	stateTimer = 0

	// Parse maze
	for row := 0; row < mazeH; row++ {
		line := mazeTemplate[row]
		for col := 0; col < mazeW; col++ {
			var ch byte = ' '
			if col < len(line) {
				ch = line[col]
			}
			t := &tiles[row][col]
			t.wall = ch == '#'
			t.dot = ch == '.'
			t.power = ch == 'o'
			if t.dot || t.power {
				dotsLeft++
			}
		}
	}

	// Pac-Man starts row 23 col 14 (centre, just above ghost house)
	pac = Pacman{
		x:   14 * tileSize,
		y:   23 * tileSize,
		dir: DirLeft,
	}

	ghostColors := [4]pf.Color{colRed, colPink, colCyan, colOrange}
	ghostStartCols := [4]int{13, 14, 13, 15}
	ghostStartRows := [4]int{13, 13, 14, 14}
	for i := 0; i < 4; i++ {
		ghosts[i] = Ghost{
			x:     float64(ghostStartCols[i] * tileSize),
			y:     float64(ghostStartRows[i] * tileSize),
			dir:   DirLeft,
			color: ghostColors[i],
		}
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func tileAt(px, py float64) *Tile {
	col := int(math.Round(px)) / tileSize
	row := int(math.Round(py)) / tileSize
	if col < 0 || col >= mazeW || row < 0 || row >= mazeH {
		return nil
	}
	return &tiles[row][col]
}

func isWall(px, py float64) bool {
	t := tileAt(px, py)
	if t == nil {
		return true
	}
	return t.wall
}

// Grid-aligned? Within 1 pixel of a tile centre.
func aligned(v float64) bool {
	return math.Mod(v, tileSize) < 1.5 || math.Mod(v, tileSize) > tileSize-1.5
}

func snapToGrid(v float64) float64 {
	return math.Round(v/tileSize) * tileSize
}

func canMove(x, y float64, d Dir) bool {
	nx := x + float64(d.dx())*tileSize
	ny := y + float64(d.dy())*tileSize
	// check corners of the bounding box (6px wide)
	half := float64(tileSize)/2 - 1
	for _, ox := range []float64{-half, half} {
		for _, oy := range []float64{-half, half} {
			if isWall(nx+ox, ny+oy) {
				return false
			}
		}
	}
	return true
}

const pacSpeed = 1.5
const ghostSpeed = 1.0

// ─── update ──────────────────────────────────────────────────────────────────

func update() {
	switch state {
	case StateWin, StateDead:
		stateTimer++
		if stateTimer > 120 || pixelforge_key.Duration(pixelforge_key.R) == 1 {
			initGame()
		}
		return
	}

	if pixelforge_key.Duration(pixelforge_key.R) == 1 {
		initGame()
		return
	}

	// ── Input
	if pixelforge_key.Duration(pixelforge_key.Right) > 0 {
		pac.nextDir = DirRight
	} else if pixelforge_key.Duration(pixelforge_key.Left) > 0 {
		pac.nextDir = DirLeft
	} else if pixelforge_key.Duration(pixelforge_key.Up) > 0 {
		pac.nextDir = DirUp
	} else if pixelforge_key.Duration(pixelforge_key.Down) > 0 {
		pac.nextDir = DirDown
	}

	// ── Move Pac-Man
	// Try to turn if grid-aligned
	if aligned(pac.x) && aligned(pac.y) {
		sx := snapToGrid(pac.x)
		sy := snapToGrid(pac.y)
		if canMove(sx, sy, pac.nextDir) {
			pac.dir = pac.nextDir
			pac.x = sx
			pac.y = sy
		}
	}

	if canMove(pac.x, pac.y, pac.dir) {
		pac.x += float64(pac.dir.dx()) * pacSpeed
		pac.y += float64(pac.dir.dy()) * pacSpeed
	}

	// Tunnel wrap
	if pac.x < 0 {
		pac.x = float64((mazeW-1) * tileSize)
	}
	if pac.x >= float64(mazeW*tileSize) {
		pac.x = 0
	}

	pac.anim = (pac.anim + 1) % 16

	// ── Eat dots / power pellets
	t := tileAt(pac.x, pac.y)
	if t != nil {
		if t.dot {
			t.dot = false
			score += 10
			dotsLeft--
			if pf.Frame%2 == 0 {
				sfxEatDot()
			}
		}
		if t.power {
			t.power = false
			score += 50
			dotsLeft--
			powerTimer = 300
			eatCombo = 0
			for i := range ghosts {
				if !ghosts[i].dead {
					ghosts[i].scared = true
					ghosts[i].scaredT = 300
				}
			}
			sfxEatPower()
		}
	}

	if powerTimer > 0 {
		powerTimer--
		if powerTimer == 0 {
			for i := range ghosts {
				ghosts[i].scared = false
			}
		}
	}

	// ── Win check
	if dotsLeft == 0 {
		state = StateWin
		stateTimer = 0
		sfxWin()
		return
	}

	// ── Move ghosts
	for i := range ghosts {
		g := &ghosts[i]

		if g.dead {
			g.respawnT--
			if g.respawnT <= 0 {
				g.dead = false
				g.scared = false
				g.x = float64(ghostStartCol(i) * tileSize)
				g.y = float64(ghostStartRow(i) * tileSize)
				g.dir = DirLeft
			}
			continue
		}

		if g.scared && g.scaredT > 0 {
			g.scaredT--
		}

		spd := ghostSpeed
		if g.scared {
			spd = ghostSpeed * 0.6
		}

		// At grid alignment, pick a direction
		if aligned(g.x) && aligned(g.y) {
			sx := snapToGrid(g.x)
			sy := snapToGrid(g.y)
			g.x = sx
			g.y = sy
			g.dir = chooseGhostDir(i, sx, sy, g.dir)
		}

		if canMove(g.x, g.y, g.dir) {
			g.x += float64(g.dir.dx()) * spd
			g.y += float64(g.dir.dy()) * spd
		} else {
			g.dir = DirNone
		}

		// Tunnel
		if g.x < 0 {
			g.x = float64((mazeW-1) * tileSize)
		}
		if g.x >= float64(mazeW*tileSize) {
			g.x = 0
		}
	}

	// ── Ghost collision
	for i := range ghosts {
		g := &ghosts[i]
		if g.dead {
			continue
		}
		dx := math.Abs(pac.x - g.x)
		dy := math.Abs(pac.y - g.y)
		if dx < float64(tileSize)-2 && dy < float64(tileSize)-2 {
			if g.scared {
				// Eat ghost
				g.dead = true
				g.scared = false
				eatCombo++
				score += 200 * (1 << eatCombo)
				g.respawnT = 180
				sfxEatGhost()
			} else {
				// Pac-Man dies
				lives--
				sfxDie()
				if lives <= 0 {
					state = StateDead
				} else {
					// Reset positions
					pac.x = 14 * tileSize
					pac.y = 23 * tileSize
					pac.dir = DirLeft
					pac.nextDir = DirLeft
					for j := range ghosts {
						ghosts[j].x = float64(ghostStartCol(j) * tileSize)
						ghosts[j].y = float64(ghostStartRow(j) * tileSize)
						ghosts[j].scared = false
						ghosts[j].dead = false
					}
				}
				stateTimer = 0
				return
			}
		}
	}

	// ── Siren / scared sfx (periodic)
	if pf.Frame%30 == 0 {
		anyScared := false
		for _, g := range ghosts {
			if g.scared && !g.dead {
				anyScared = true
				break
			}
		}
		if anyScared {
			sfxScared()
		} else {
			sfxSiren()
		}
	}

	flashTimer = (flashTimer + 1) % 30
}

func ghostStartCol(i int) int { return [4]int{13, 14, 13, 15}[i] }
func ghostStartRow(i int) int { return [4]int{13, 13, 14, 14}[i] }

// chooseGhostDir picks a movement direction using simple AI:
// - Blinky (0): chase Pac directly
// - Pinky (1): target 4 tiles ahead of Pac
// - Inky (2): random
// - Clyde (3): chase if far, scatter if close
func chooseGhostDir(idx int, gx, gy float64, currentDir Dir) Dir {
	opposites := map[Dir]Dir{
		DirRight: DirLeft, DirLeft: DirRight,
		DirUp: DirDown, DirDown: DirUp, DirNone: DirNone,
	}
	dirs := []Dir{DirRight, DirLeft, DirUp, DirDown}

	var targetX, targetY float64
	switch idx {
	case 0: // Blinky — direct chase
		targetX, targetY = pac.x, pac.y
	case 1: // Pinky — 4 tiles ahead of pac
		targetX = pac.x + float64(pac.dir.dx()*4*tileSize)
		targetY = pac.y + float64(pac.dir.dy()*4*tileSize)
	case 2: // Inky — random
		targetX = gx + float64((pf.Frame%7-3)*tileSize)
		targetY = gy + float64((pf.Frame%5-2)*tileSize)
	case 3: // Clyde — scatter if close
		ddx := pac.x - gx
		ddy := pac.y - gy
		dist := math.Sqrt(ddx*ddx + ddy*ddy)
		if dist < 8*tileSize {
			targetX = 0 // scatter corner
			targetY = float64(mazeH * tileSize)
		} else {
			targetX, targetY = pac.x, pac.y
		}
	}

	if ghosts[idx].scared {
		// Run away from pac
		targetX = 2*gx - pac.x
		targetY = 2*gy - pac.y
	}

	bestDir := DirNone
	bestDist := math.MaxFloat64

	for _, d := range dirs {
		if d == opposites[currentDir] {
			continue // can't reverse
		}
		nx := gx + float64(d.dx()*tileSize)
		ny := gy + float64(d.dy()*tileSize)
		if !canMove(gx, gy, d) {
			continue
		}
		ddx := nx - targetX
		ddy := ny - targetY
		dist := ddx*ddx + ddy*ddy
		if dist < bestDist {
			bestDist = dist
			bestDir = d
		}
	}

	if bestDir == DirNone {
		// Stuck — try any valid direction
		for _, d := range dirs {
			if canMove(gx, gy, d) {
				return d
			}
		}
	}
	return bestDir
}

// ─── draw ────────────────────────────────────────────────────────────────────

func draw() {
	pf.Clear(colBlack)

	// Maze offset — centre the 28×31 maze (224×248) in the screen
	const ox = 0
	const oy = 0

	// Draw maze tiles
	for row := 0; row < mazeH; row++ {
		for col := 0; col < mazeW; col++ {
			t := &tiles[row][col]
			px := ox + col*tileSize
			py := oy + row*tileSize

			if t.wall {
				pf.SetColor(colBlue)
				pf.FillRect(px, py, tileSize, tileSize)
				// Inner shadow for depth
				pf.SetColor(colDkBlue)
				pf.FillRect(px+1, py+1, tileSize-2, tileSize-2)
			} else if t.dot {
				pf.SetColor(colWhite)
				pf.SetPixel(px+3, py+3)
				pf.SetPixel(px+4, py+3)
				pf.SetPixel(px+3, py+4)
				pf.SetPixel(px+4, py+4)
			} else if t.power {
				// Flashing power pellet
				if flashTimer < 20 {
					pf.SetColor(colWhite)
					pf.FillCircle(px+4, py+4, 3)
				}
			}
		}
	}

	// Draw ghosts
	for i := range ghosts {
		g := &ghosts[i]
		if g.dead {
			continue
		}
		gx := ox + int(g.x)
		gy := oy + int(g.y)

		col := g.color
		if g.scared {
			if g.scaredT > 60 || flashTimer < 15 {
				col = colDkBlue
			} else {
				col = colWhite // flash warning
			}
		}
		drawGhost(gx, gy, col)
	}

	// Draw Pac-Man
	px := ox + int(pac.x)
	py := oy + int(pac.y)
	drawPacman(px, py, pac.dir, pac.anim)

	// HUD
	pf.SetColor(colWhite)
	pixelforge_font.Print(4, screenH-16, "SCORE:")
	pixelforge_font.PrintNum(52, screenH-16, score)

	pf.SetColor(colYellow)
	for l := 0; l < lives; l++ {
		drawSmallPac(10+l*12, screenH-8)
	}

	// State overlays
	if state == StateWin {
		pf.SetColor(colYellow)
		pixelforge_font.Print(60, screenH/2-4, "YOU WIN!")
	}
	if state == StateDead {
		pf.SetColor(colRed)
		pixelforge_font.Print(50, screenH/2-4, "GAME OVER")
	}
}

func drawGhost(x, y int, col pf.Color) {
	// Ghost body: 7×8 pixels
	pf.SetColor(col)
	// Head (dome)
	pf.FillCircle(x+3, y+3, 4)
	// Body rectangle
	pf.FillRect(x-1, y+3, 9, 5)
	// Wavy bottom (3 bumps)
	pf.SetColor(colBlack)
	pf.SetPixel(x-1, y+7)
	pf.SetPixel(x+2, y+7)
	pf.SetPixel(x+5, y+7)
	// Eyes
	pf.SetColor(colWhite)
	pf.FillRect(x, y+1, 2, 2)
	pf.FillRect(x+4, y+1, 2, 2)
	pf.SetColor(colBlue)
	pf.SetPixel(x+1, y+2)
	pf.SetPixel(x+5, y+2)
}

func drawPacman(x, y int, dir Dir, anim int) {
	pf.SetColor(colYellow)
	// Mouth open angle depends on anim
	mouthOpen := (anim%8 < 4)

	if !mouthOpen {
		pf.FillCircle(x+3, y+3, 4)
		return
	}

	// Draw circle with mouth cut out — approximate with pixels
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if dx*dx+dy*dy > 16 {
				continue
			}
			// Mouth direction
			inMouth := false
			switch dir {
			case DirRight:
				inMouth = dx >= 0 && math.Abs(float64(dy)) <= float64(dx)*0.7
			case DirLeft:
				inMouth = dx <= 0 && math.Abs(float64(dy)) <= float64(-dx)*0.7
			case DirUp:
				inMouth = dy <= 0 && math.Abs(float64(dx)) <= float64(-dy)*0.7
			case DirDown:
				inMouth = dy >= 0 && math.Abs(float64(dx)) <= float64(dy)*0.7
			default:
				inMouth = dx >= 0 && math.Abs(float64(dy)) <= float64(dx)*0.7
			}
			if !inMouth {
				pf.SetPixel(x+3+dx, y+3+dy)
			}
		}
	}
}

func drawSmallPac(x, y int) {
	pf.SetColor(colYellow)
	pf.FillCircle(x+3, y+3, 3)
	pf.SetColor(colBlack)
	pf.SetPixel(x+4, y+2)
	pf.SetPixel(x+5, y+3)
	pf.SetPixel(x+4, y+4)
}

// ─── main ────────────────────────────────────────────────────────────────────

func main() {
	pf.SetTPS(60)

	pf.Init = func() {
		initGame()
	}

	pf.Update = update
	pf.Draw = draw

	pixelforge_metrics.Start()
	pixelforge_ebiten.Run("Pac-Man", screenW, screenH, 3)
}
