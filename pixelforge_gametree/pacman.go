// Package pixelforge_gametree provides a declarative Pac-Man builder.
// The player describes colours, enemy count, controls, and rules; the
// package handles maze parsing, ghost AI, power-pellet timing, sound
// effects, and rendering.
package pixelforge_gametree

import (
	"math"
	"os"
	"strconv"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_metrics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
)

// PacmanGame is a self-contained Pac-Man clone. Configure it with
// the declarative methods and call Loop() to play.
type PacmanGame struct {
	// ── user-configurable ──────────────────────────────────────────
	pacmanColor      int
	gridColor        int
	bgColor          int
	enemyCount       int
	upKey            string
	downKey          string
	leftKey          string
	rightKey         string
	powerDurationSec int
	lives            int
	showScore        bool
	soundOn          bool
	assetPrefix      string

	// ── runtime state ──────────────────────────────────────────────
	tiles      [pMazeH][pMazeW]pTile
	pac        pPacman
	ghosts     []pGhost
	score      int
	livesLeft  int
	dotsLeft   int
	state      pState
	stateTimer int
	powerTimer int
	flashTimer int
	eatCombo   int

	// ── audio ──────────────────────────────────────────────────────
	samples      map[string]*pixelforge_audio.Sample
	sirenSample  *pixelforge_audio.Sample
	scaredSample *pixelforge_audio.Sample
	soundsLoaded bool
	bgmActive    bool
	bgmScared    bool
}

// pState represents the high-level game state.
type pState int

const (
	pStatePlay pState = iota
	pStateWin
	pStateDead
)

// pDir is a movement direction.
type pDir int

const (
	pDirNone pDir = iota
	pDirRight
	pDirLeft
	pDirUp
	pDirDown
)

func (d pDir) dx() int {
	switch d {
	case pDirRight:
		return 1
	case pDirLeft:
		return -1
	}
	return 0
}

func (d pDir) dy() int {
	switch d {
	case pDirUp:
		return -1
	case pDirDown:
		return 1
	}
	return 0
}

// pTile is one cell in the maze.
type pTile struct {
	wall  bool
	dot   bool
	power bool
}

// pPacman is the player.
type pPacman struct {
	x       float64
	y       float64
	dir     pDir
	nextDir pDir
	anim    int
}

// pGhost is an enemy.
type pGhost struct {
	x        float64
	y        float64
	dir      pDir
	color    pf.Color
	scared   bool
	scaredT  int
	dead     bool
	respawnT int
}

// ─── Declarative API ─────────────────────────────────────────────

// NewPacmanGame creates a Pac-Man game with sensible defaults.
func NewPacmanGame() *PacmanGame {
	return &PacmanGame{
		pacmanColor:      10, // yellow
		gridColor:        1,  // dark blue
		bgColor:          0,  // black
		enemyCount:       4,
		upKey:            "arrowUp",
		downKey:          "arrowDown",
		leftKey:          "arrowLeft",
		rightKey:         "arrowRight",
		powerDurationSec: 5,
		lives:            3,
		assetPrefix:      "",
	}
}

// PacmanColor sets the player sprite colour (palette index).
func (g *PacmanGame) PacmanColor(c int) { g.pacmanColor = c }

// GridColor sets the maze wall colour (palette index).
func (g *PacmanGame) GridColor(c int) { g.gridColor = c }

// Background sets the screen clear colour (palette index).
func (g *PacmanGame) Background(c int) { g.bgColor = c }

// Enemies sets how many ghosts spawn (1–8). Each gets a predefined
// classic colour.
func (g *PacmanGame) Enemies(n int) {
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	g.enemyCount = n
}

// Movement binds the four directional inputs by name.
func (g *PacmanGame) Movement(up, down, left, right string) {
	g.upKey = up
	g.downKey = down
	g.leftKey = left
	g.rightKey = right
}

// LoseCond configures what ends the game. Only "catchByEnemy" is
// supported for Pac-Man.
func (g *PacmanGame) LoseCond(cond string) {
	_ = cond // Pac-Man always loses on enemy contact
}

// WinCond configures the victory condition. Only "eatAllDots" is
// supported.
func (g *PacmanGame) WinCond(cond string) {
	_ = cond // Pac-Man always wins when all dots are eaten
}

// PowerPelletDuration sets how many seconds ghosts stay scared.
func (g *PacmanGame) PowerPelletDuration(sec int) {
	g.powerDurationSec = sec
}

// Lives sets the starting life count.
func (g *PacmanGame) Lives(n int) { g.lives = n }

// Score enables the HUD score and life icons.
func (g *PacmanGame) Score() { g.showScore = true }

// Sound toggles all sound effects and BGM.
func (g *PacmanGame) Sound(on bool) { g.soundOn = on }

// Assets sets a path prefix for audio files (e.g. "assets/").
func (g *PacmanGame) Assets(prefix string) { g.assetPrefix = prefix }

// Loop initialises the game, wires engine hooks, and enters the
// Ebitengine main loop.
func (g *PacmanGame) Loop() {
	g.init()
	pf.Update = g.update
	pf.Draw = g.draw
	pixelforge_metrics.Start()
	pixelforge_ebiten.Run()
}

// ─── Constants ───────────────────────────────────────────────────

const (
	pScreenW  = 224
	pScreenH  = 248
	pTileSize = 8
	pMazeW    = 28
	pMazeH    = 31
)

// Predefined ghost colours (max 8).
var ghostPalette = [8]pf.Color{
	8,  // 0 Blinky – red
	14, // 1 Pinky – pink
	12, // 2 Inky – cyan
	9,  // 3 Clyde – orange
	18, // 4 purple
	11, // 5 green
	10, // 6 yellow
	7,  // 7 white
}

// Classic Pac-Man maze template.
var mazeTemplate = [pMazeH]string{
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
	"######.## #      # ##.######",
	"      .   #      #   .      ",
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

// ─── Initialisation ──────────────────────────────────────────────

func (g *PacmanGame) init() {
	pf.SetTPS(30)
	pf.SetScreenSize(pScreenW, pScreenH)
	g.resetLevel()
}

func (g *PacmanGame) resetLevel() {
	g.score = 0
	g.livesLeft = g.lives
	g.dotsLeft = 0
	g.state = pStatePlay
	g.stateTimer = 0
	g.powerTimer = 0
	g.flashTimer = 0
	g.eatCombo = 0
	g.bgmActive = false
	g.bgmScared = false

	for row := 0; row < pMazeH; row++ {
		line := mazeTemplate[row]
		for col := 0; col < pMazeW; col++ {
			ch := byte(' ')
			if col < len(line) {
				ch = line[col]
			}
			t := &g.tiles[row][col]
			t.wall = ch == '#'
			t.dot = ch == '.'
			t.power = ch == 'o'
			if t.dot || t.power {
				g.dotsLeft++
			}
		}
	}

	g.pac = pPacman{
		x:   14 * pTileSize,
		y:   23 * pTileSize,
		dir: pDirLeft,
	}

	g.ghosts = make([]pGhost, g.enemyCount)
	for i := 0; i < g.enemyCount; i++ {
		g.ghosts[i] = pGhost{
			x:     float64(ghostStartCol(i) * pTileSize),
			y:     float64(ghostStartRow(i) * pTileSize),
			dir:   pDirLeft,
			color: ghostPalette[i],
		}
	}
}

func ghostStartCol(i int) int {
	cols := [8]int{13, 14, 13, 15, 13, 14, 13, 15}
	return cols[i%8]
}

func ghostStartRow(i int) int {
	rows := [8]int{13, 13, 14, 14, 13, 13, 14, 14}
	return rows[i%8]
}

// ─── Audio ───────────────────────────────────────────────────────

func (g *PacmanGame) assetPath(name string) string {
	return g.assetPrefix + name
}

func (g *PacmanGame) loadSound(name string) *pixelforge_audio.Sample {
	if s, ok := g.samples[name]; ok {
		return s
	}
	data, err := os.ReadFile(g.assetPath(name))
	if err != nil {
		return nil
	}
	s, err := pixelforge_audio.DecodeWavOrErr(data)
	if err != nil {
		return nil
	}
	pixelforge_audio.LoadSample(s)
	if g.samples == nil {
		g.samples = make(map[string]*pixelforge_audio.Sample)
	}
	g.samples[name] = s
	return s
}

func (g *PacmanGame) loadSounds() {
	if g.soundsLoaded {
		return
	}
	g.soundsLoaded = true

	g.loadSound("pacman_chomp.wav")
	g.loadSound("pacman_eatfruit.wav")
	g.loadSound("pacman_eatghost.wav")
	g.loadSound("pacman_death.wav")
	g.loadSound("pacman_beginning.wav")
	g.loadSound("pacman_intermission.wav")
	g.loadSound("pacman_extrapac.wav")

	// Generate siren BGM sample (1-second oscillating tone)
	g.sirenSample = g.generateSiren(200, 280, 1.0)
	pixelforge_audio.LoadSample(g.sirenSample)

	// Generate scared BGM sample (faster, higher pitch)
	g.scaredSample = g.generateSiren(350, 450, 0.6)
	pixelforge_audio.LoadSample(g.scaredSample)
}

func (g *PacmanGame) generateSiren(lowFreq, highFreq float64, durationSec float64) *pixelforge_audio.Sample {
	sr := 11025
	n := int(float64(sr) * durationSec)
	data := make([]int8, n)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(sr)
		// Triangle-wave frequency sweep
		cycle := t / durationSec
		phase := cycle - math.Floor(cycle)
		var freq float64
		if phase < 0.5 {
			freq = lowFreq + (highFreq-lowFreq)*(phase*2)
		} else {
			freq = highFreq - (highFreq-lowFreq)*((phase-0.5)*2)
		}
		v := math.Sin(2 * math.Pi * freq * t)
		if v > 0 {
			v = 1
		} else {
			v = -1
		}
		data[i] = int8(v * 100)
	}
	return pixelforge_audio.NewSample(data, uint16(sr))
}

func (g *PacmanGame) playSample(ch pixelforge_audio.Chan, name string, pitch, vol float64) {
	s := g.loadSound(name)
	if s == nil {
		return
	}
	pixelforge_audio.Play(ch, s, pitch, vol)
}

func (g *PacmanGame) playBGM(scared bool) {
	if !g.soundOn {
		return
	}
	if scared == g.bgmScared && g.bgmActive {
		return
	}
	g.bgmScared = scared
	g.bgmActive = true

	var s *pixelforge_audio.Sample
	if scared {
		s = g.scaredSample
	} else {
		s = g.sirenSample
	}

	delay := 0.0
	pixelforge_audio.ClearChan(pixelforge_audio.Chan4, delay)
	pixelforge_audio.SetSample(pixelforge_audio.Chan4, s, 0, delay)
	pixelforge_audio.SetLoop(pixelforge_audio.Chan4, 0, s.Len(), pixelforge_audio.LoopForward, delay)
	pixelforge_audio.SetPitch(pixelforge_audio.Chan4, 1.0, delay)
	pixelforge_audio.SetVolume(pixelforge_audio.Chan4, 0.35, delay)
}

func (g *PacmanGame) stopBGM() {
	if !g.bgmActive {
		return
	}
	g.bgmActive = false
	pixelforge_audio.ClearChan(pixelforge_audio.Chan4, 0)
}

func (g *PacmanGame) sfxEatDot() {
	if !g.soundOn {
		return
	}
	g.playSample(pixelforge_audio.Chan1, "pacman_chomp.wav", 1.0, 0.6)
}

func (g *PacmanGame) sfxEatPower() {
	if !g.soundOn {
		return
	}
	g.playSample(pixelforge_audio.Chan1, "pacman_eatfruit.wav", 1.0, 0.7)
}

func (g *PacmanGame) sfxEatGhost() {
	if !g.soundOn {
		return
	}
	g.playSample(pixelforge_audio.Chan2, "pacman_eatghost.wav", 1.0, 0.7)
}

func (g *PacmanGame) sfxDie() {
	if !g.soundOn {
		return
	}
	g.stopBGM()
	g.playSample(pixelforge_audio.Chan3, "pacman_death.wav", 1.0, 0.7)
}

func (g *PacmanGame) sfxWin() {
	if !g.soundOn {
		return
	}
	g.stopBGM()
	g.playSample(pixelforge_audio.Chan3, "pacman_intermission.wav", 1.0, 0.6)
}

func (g *PacmanGame) sfxStart() {
	if !g.soundOn {
		return
	}
	g.playSample(pixelforge_audio.Chan3, "pacman_beginning.wav", 1.0, 0.7)
}

// ─── Update ──────────────────────────────────────────────────────

func (g *PacmanGame) update() {
	if !g.soundsLoaded {
		g.loadSounds()
	}

	switch g.state {
	case pStateWin, pStateDead:
		g.stateTimer++
		if g.stateTimer == 1 && g.state == pStatePlay {
			// Just transitioned - sounds already played
		}
		if g.stateTimer > 120 || g.keyPressed("R") {
			g.resetLevel()
		}
		return
	}

	if g.stateTimer == 0 {
		// First frame of play
		g.sfxStart()
	}
	g.stateTimer++

	if g.keyPressed("R") {
		g.resetLevel()
		return
	}

	// Input
	if g.keyPressed(g.rightKey) {
		g.pac.nextDir = pDirRight
	} else if g.keyPressed(g.leftKey) {
		g.pac.nextDir = pDirLeft
	} else if g.keyPressed(g.upKey) {
		g.pac.nextDir = pDirUp
	} else if g.keyPressed(g.downKey) {
		g.pac.nextDir = pDirDown
	}

	// Move Pac-Man
	if g.aligned(g.pac.x) && g.aligned(g.pac.y) {
		row := int(g.pac.y) / pTileSize
		if row == 15 {
			col := int(g.pac.x) / pTileSize
			if col == 0 && (g.pac.nextDir == pDirLeft || g.pac.dir == pDirLeft) {
				g.pac.x = float64((pMazeW - 1) * pTileSize)
				g.pac.dir = pDirLeft
			} else if col == pMazeW-1 && (g.pac.nextDir == pDirRight || g.pac.dir == pDirRight) {
				g.pac.x = 0
				g.pac.dir = pDirRight
			} else {
				if g.canMove(g.pac.x, g.pac.y, g.pac.nextDir) {
					g.pac.dir = g.pac.nextDir
				}
				if !g.canMove(g.pac.x, g.pac.y, g.pac.dir) {
					g.pac.dir = pDirNone
				}
			}
		} else {
			if g.canMove(g.pac.x, g.pac.y, g.pac.nextDir) {
				g.pac.dir = g.pac.nextDir
			}
			if !g.canMove(g.pac.x, g.pac.y, g.pac.dir) {
				g.pac.dir = pDirNone
			}
		}
	}

	g.pac.x += float64(g.pac.dir.dx())
	g.pac.y += float64(g.pac.dir.dy())

	g.pac.anim = (g.pac.anim + 1) % 16

	// Eat dots / power pellets
	t := g.tileAt(g.pac.x, g.pac.y)
	if t != nil {
		if t.dot {
			t.dot = false
			g.score += 10
			g.dotsLeft--
			if g.soundOn {
				g.sfxEatDot()
			}
		}
		if t.power {
			t.power = false
			g.score += 50
			g.dotsLeft--
			g.powerTimer = g.powerDurationSec * pf.TPS()
			g.eatCombo = 0
			for i := range g.ghosts {
				if !g.ghosts[i].dead {
					g.ghosts[i].scared = true
					g.ghosts[i].scaredT = g.powerDurationSec * pf.TPS()
				}
			}
			if g.soundOn {
				g.sfxEatPower()
			}
		}
	}

	if g.powerTimer > 0 {
		g.powerTimer--
		if g.powerTimer == 0 {
			for i := range g.ghosts {
				g.ghosts[i].scared = false
			}
		}
	}

	// Win check
	if g.dotsLeft == 0 {
		g.state = pStateWin
		g.stateTimer = 0
		if g.soundOn {
			g.sfxWin()
		}
		return
	}

	// Move ghosts
	for i := range g.ghosts {
		gh := &g.ghosts[i]
		if gh.dead {
			gh.respawnT--
			if gh.respawnT <= 0 {
				gh.dead = false
				gh.scared = false
				gh.x = float64(ghostStartCol(i) * pTileSize)
				gh.y = float64(ghostStartRow(i) * pTileSize)
				gh.dir = pDirLeft
			}
			continue
		}
		if gh.scared && gh.scaredT > 0 {
			gh.scaredT--
		}

		moveThisTick := true
		if gh.scared && pf.Frame%3 == 0 {
			moveThisTick = false
		}

		if g.aligned(gh.x) && g.aligned(gh.y) {
			row := int(gh.y) / pTileSize
			if row == 15 {
				col := int(gh.x) / pTileSize
				if col == 0 && gh.dir == pDirLeft {
					gh.x = float64((pMazeW - 1) * pTileSize)
				} else if col == pMazeW-1 && gh.dir == pDirRight {
					gh.x = 0
				} else {
					gh.dir = g.chooseGhostDir(i, gh.x, gh.y, gh.dir)
					if !g.canMove(gh.x, gh.y, gh.dir) {
						gh.dir = pDirNone
					}
				}
			} else {
				gh.dir = g.chooseGhostDir(i, gh.x, gh.y, gh.dir)
				if !g.canMove(gh.x, gh.y, gh.dir) {
					gh.dir = pDirNone
				}
			}
		}
		if moveThisTick {
			gh.x += float64(gh.dir.dx())
			gh.y += float64(gh.dir.dy())
		}
	}

	// Ghost collision
	for i := range g.ghosts {
		gh := &g.ghosts[i]
		if gh.dead {
			continue
		}
		dx := math.Abs(g.pac.x - gh.x)
		dy := math.Abs(g.pac.y - gh.y)
		if dx < float64(pTileSize)-2 && dy < float64(pTileSize)-2 {
			if gh.scared {
				gh.dead = true
				gh.scared = false
				g.eatCombo++
				g.score += 200 * (1 << g.eatCombo)
				gh.respawnT = 180
				if g.soundOn {
					g.sfxEatGhost()
				}
			} else {
				g.livesLeft--
				if g.soundOn {
					g.sfxDie()
				}
				if g.livesLeft <= 0 {
					g.state = pStateDead
				} else {
					g.pac.x = 14 * pTileSize
					g.pac.y = 23 * pTileSize
					g.pac.dir = pDirLeft
					g.pac.nextDir = pDirLeft
					for j := range g.ghosts {
						g.ghosts[j].x = float64(ghostStartCol(j) * pTileSize)
						g.ghosts[j].y = float64(ghostStartRow(j) * pTileSize)
						g.ghosts[j].scared = false
						g.ghosts[j].dead = false
					}
				}
				g.stateTimer = 0
				return
			}
		}
	}

	g.flashTimer = (g.flashTimer + 1) % 30
}

// ─── Ghost AI ────────────────────────────────────────────────────

func (g *PacmanGame) chooseGhostDir(idx int, gx, gy float64, current pDir) pDir {
	opposites := map[pDir]pDir{
		pDirRight: pDirLeft, pDirLeft: pDirRight,
		pDirUp: pDirDown, pDirDown: pDirUp, pDirNone: pDirNone,
	}
	dirs := []pDir{pDirRight, pDirLeft, pDirUp, pDirDown}

	var targetX, targetY float64
	switch idx % 4 {
	case 0: // Blinky – direct chase
		targetX, targetY = g.pac.x, g.pac.y
	case 1: // Pinky – 4 tiles ahead
		targetX = g.pac.x + float64(g.pac.dir.dx()*4*pTileSize)
		targetY = g.pac.y + float64(g.pac.dir.dy()*4*pTileSize)
	case 2: // Inky – random
		targetX = gx + float64((pf.Frame%7-3)*pTileSize)
		targetY = gy + float64((pf.Frame%5-2)*pTileSize)
	case 3: // Clyde – scatter if close
		ddx := g.pac.x - gx
		ddy := g.pac.y - gy
		if math.Sqrt(ddx*ddx+ddy*ddy) < 8*pTileSize {
			targetX = 0
			targetY = float64(pMazeH * pTileSize)
		} else {
			targetX, targetY = g.pac.x, g.pac.y
		}
	}

	if g.ghosts[idx].scared {
		targetX = 2*gx - g.pac.x
		targetY = 2*gy - g.pac.y
	}

	bestDir := pDirNone
	bestDist := math.MaxFloat64
	for _, d := range dirs {
		if d == opposites[current] {
			continue
		}
		nx := gx + float64(d.dx()*pTileSize)
		ny := gy + float64(d.dy()*pTileSize)
		if !g.canMove(gx, gy, d) {
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
	if bestDir == pDirNone {
		for _, d := range dirs {
			if g.canMove(gx, gy, d) {
				return d
			}
		}
	}
	return bestDir
}

// ─── Helpers ─────────────────────────────────────────────────────

func (g *PacmanGame) keyPressed(name string) bool {
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
	k := resolveKey(name)
	if k != "" {
		return pixelforge_key.Duration(k) > 0
	}
	return false
}

func (g *PacmanGame) tileAt(px, py float64) *pTile {
	col := int(math.Round(px)) / pTileSize
	row := int(math.Round(py)) / pTileSize
	if col < 0 || col >= pMazeW || row < 0 || row >= pMazeH {
		return nil
	}
	return &g.tiles[row][col]
}

func (g *PacmanGame) isWallTile(col, row int) bool {
	if col < 0 || col >= pMazeW || row < 0 || row >= pMazeH {
		return true
	}
	return g.tiles[row][col].wall
}

func (g *PacmanGame) aligned(v float64) bool {
	r := math.Round(v)
	if math.Abs(v-r) > 0.0001 {
		return false
	}
	return int(r)%pTileSize == 0
}

func (g *PacmanGame) canMove(x, y float64, d pDir) bool {
	col := int(math.Round(x)) / pTileSize
	row := int(math.Round(y)) / pTileSize
	return !g.isWallTile(col+d.dx(), row+d.dy())
}

// ─── Drawing ─────────────────────────────────────────────────────

func (g *PacmanGame) draw() {
	pf.SetColor(pf.Color(g.bgColor))
	pf.Cls()

	// Maze
	for row := 0; row < pMazeH; row++ {
		for col := 0; col < pMazeW; col++ {
			t := &g.tiles[row][col]
			px := col * pTileSize
			py := row * pTileSize
			if t.wall {
				pf.SetColor(pf.Color(g.gridColor))
				pf.RectFill(px, py, px+pTileSize-1, py+pTileSize-1)
			} else if t.dot {
				pf.SetColor(7)
				pf.SetPixel(px+3, py+3)
				pf.SetPixel(px+4, py+3)
				pf.SetPixel(px+3, py+4)
				pf.SetPixel(px+4, py+4)
			} else if t.power {
				if g.flashTimer < 20 {
					pf.SetColor(7)
					pf.CircFill(px+4, py+4, 3)
				}
			}
		}
	}

	// Ghosts
	for i := range g.ghosts {
		gh := &g.ghosts[i]
		if gh.dead {
			continue
		}
		col := gh.color
		if gh.scared {
			if gh.scaredT > 60 || g.flashTimer < 15 {
				col = 1 // dark blue
			} else {
				col = 7 // white flash warning
			}
		}
		g.drawGhost(int(gh.x), int(gh.y), col)
	}

	// Pac-Man
	g.drawPacman(int(g.pac.x), int(g.pac.y), g.pac.dir, g.pac.anim)

	// HUD
	if g.showScore {
		pf.SetColor(7)
		pixelforge_cofont.Print("SCORE:", 4, pScreenH-16)
		pixelforge_cofont.Print(strconv.Itoa(g.score), 52, pScreenH-16)

		pf.SetColor(pf.Color(g.pacmanColor))
		for l := 0; l < g.livesLeft; l++ {
			g.drawSmallPac(10+l*12, pScreenH-8)
		}
	}

	// State overlays
	if g.state == pStateWin {
		g.drawOverlay("YOU WIN!")
	}
	if g.state == pStateDead {
		g.drawOverlay("GAME OVER")
	}
}

func (g *PacmanGame) drawOverlay(msg string) {
	msgW := len(msg) * 4
	msgX := (pScreenW - msgW) / 2
	msgY := pScreenH/2 - 4

	pad := 4
	boxX := msgX - pad
	boxY := msgY - pad
	boxW := msgW + 2*pad
	boxH := 8 + 2*pad

	pf.SetColor(5)
	pf.RectFill(boxX, boxY, boxX+boxW, boxY+boxH)
	pf.SetColor(7)
	pf.Rect(boxX, boxY, boxX+boxW, boxY+boxH)

	pixelforge_cofont.Sheet.PrintStroked(msg, msgX, msgY, 7, 0)
}

func (g *PacmanGame) drawGhost(x, y int, col pf.Color) {
	pf.SetColor(col)
	pf.CircFill(x+3, y+3, 4)
	pf.RectFill(x-1, y+3, x+7, y+7)
	pf.SetColor(0)
	pf.SetPixel(x-1, y+7)
	pf.SetPixel(x+2, y+7)
	pf.SetPixel(x+5, y+7)
	pf.SetColor(7)
	pf.RectFill(x, y+1, x+1, y+2)
	pf.RectFill(x+4, y+1, x+5, y+2)
	pf.SetColor(12)
	pf.SetPixel(x+1, y+2)
	pf.SetPixel(x+5, y+2)
}

func (g *PacmanGame) drawPacman(x, y int, dir pDir, anim int) {
	pf.SetColor(pf.Color(g.pacmanColor))
	mouthOpen := (anim % 8 < 4)
	if !mouthOpen {
		pf.CircFill(x+3, y+3, 4)
		return
	}
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			if dx*dx+dy*dy > 16 {
				continue
			}
			inMouth := false
			switch dir {
			case pDirRight:
				inMouth = dx >= 0 && math.Abs(float64(dy)) <= float64(dx)*0.7
			case pDirLeft:
				inMouth = dx <= 0 && math.Abs(float64(dy)) <= float64(-dx)*0.7
			case pDirUp:
				inMouth = dy <= 0 && math.Abs(float64(dx)) <= float64(-dy)*0.7
			case pDirDown:
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

func (g *PacmanGame) drawSmallPac(x, y int) {
	pf.SetColor(pf.Color(g.pacmanColor))
	pf.CircFill(x+3, y+3, 3)
	pf.SetColor(0)
	pf.SetPixel(x+4, y+2)
	pf.SetPixel(x+5, y+3)
	pf.SetPixel(x+4, y+4)
}
