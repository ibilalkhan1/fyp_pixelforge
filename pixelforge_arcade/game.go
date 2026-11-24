// Package pixelforge_arcade implements the opt-in Studio arcade project presets.
// The same deterministic simulation and indexed-pixel renderer run in Studio,
// native exports and WebAssembly. No browser-only game logic is involved.
package pixelforge_arcade

import (
	"fmt"
	"math"
)

const (
	Width   = 320
	Height  = 208
	Tile    = 16
	Columns = 20
	Rows    = 11
)

type Input struct {
	Up, Down, Left, Right, Action, Restart, Pause bool
}

type actor struct {
	x, y, speed, width float64
	row, color         int
	alive              bool
}

// Game owns all simulation state. Step takes explicit input, never wall time.
type Game struct {
	Kind, Title                                  string
	Layout                                       []string
	X, Y                                         float64
	Score, Health, Tick                          int
	Won, Over, Paused                            bool
	Message                                      string
	HasKey, Driving, Carrying                    bool
	Deliveries, Heat                             int
	Homes                                        [5]bool
	CarX, CarY, vx, vy                           float64
	Actors                                       []actor
	coins                                        map[[2]int]bool
	key, exit                                    [2]int
	cooldown, attack, immune, remaining, bestRow int
	actionPressed                                bool
	last                                         Input
	dirX, dirY                                   int
}

// New validates the preset and its optional authored tile layout.
func New(kind string, layout []string) (*Game, error) {
	if kind != "dungeon" && kind != "frogger" && kind != "city" {
		return nil, fmt.Errorf("unknown arcade preset %q", kind)
	}
	if len(layout) == 0 {
		layout = DefaultLayout(kind)
	}
	if len(layout) != Rows {
		return nil, fmt.Errorf("arcade layout needs %d rows", Rows)
	}
	for _, row := range layout {
		if len(row) != Columns {
			return nil, fmt.Errorf("arcade layout rows need %d columns", Columns)
		}
	}
	g := &Game{Kind: kind, Layout: append([]string(nil), layout...)}
	g.Reset()
	return g, nil
}

// Reset preserves authored data and clears every transient state.
func (g *Game) Reset() {
	kind, layout := g.Kind, g.Layout
	*g = Game{Kind: kind, Layout: layout, Health: 3, dirY: -1, remaining: 900, bestRow: 10}
	switch kind {
	case "dungeon":
		g.initDungeon()
	case "frogger":
		g.initFrogger()
	case "city":
		g.initCity()
	}
}

// Step advances one 1/30-second tick. Held inputs have bounded repeat rates.
func (g *Game) Step(in Input) {
	if in.Restart && !g.last.Restart {
		g.Reset()
	}
	if in.Pause && !g.last.Pause {
		g.Paused = !g.Paused
	}
	g.actionPressed = in.Action && !g.last.Action
	g.last = in
	if g.Paused || g.Over || g.Won {
		return
	}
	g.Tick++
	if g.cooldown > 0 {
		g.cooldown--
	}
	if g.attack > 0 {
		g.attack--
	}
	if g.immune > 0 {
		g.immune--
	}
	switch g.Kind {
	case "dungeon":
		g.stepDungeon(in)
	case "frogger":
		g.stepFrogger(in)
	case "city":
		g.stepCity(in)
	}
}

func direction(in Input) (int, int) {
	if in.Up {
		return 0, -1
	}
	if in.Down {
		return 0, 1
	}
	if in.Left {
		return -1, 0
	}
	if in.Right {
		return 1, 0
	}
	return 0, 0
}

func (g *Game) tile(x, y int) byte {
	if x < 0 || x >= Columns || y < 0 || y >= Rows {
		return '#'
	}
	return g.Layout[y][x]
}

func (g *Game) clearAt(x, y, radius float64) bool {
	for _, dx := range []float64{-radius, radius} {
		for _, dy := range []float64{-radius, radius} {
			if g.tile(int(math.Floor((x+dx)/Tile)), int(math.Floor((y+dy)/Tile))) == '#' {
				return false
			}
		}
	}
	return true
}

func (g *Game) hurt(message string) {
	if g.immune != 0 {
		return
	}
	g.Health--
	g.immune = 35
	g.Message = message
	if g.Health <= 0 {
		g.Over = true
		g.Message = "Out of lives. Restart to try again."
	}
}

// Snapshot is the real visible HUD state used by the browser shell.
func (g *Game) Snapshot() map[string]any {
	return map[string]any{
		"game": g.Title, "score": g.Score, "lives": g.Health,
		"x": int(g.X), "y": int(g.Y), "gameOver": g.Over, "won": g.Won,
		"paused": g.Paused, "message": g.Message, "driving": g.Driving,
		"deliveries": g.Deliveries, "hasKey": g.HasKey, "remaining": g.remaining / 30,
	}
}

// DefaultLayout returns the authored map used by each shipped .pforge project.
func DefaultLayout(kind string) []string {
	switch kind {
	case "dungeon":
		return []string{
			"####################", "#P....#....C......E#", "#.....#............#",
			"#..C.....####......#", "#.....#............#", "###.###.....M......#",
			"#.........#........#", "#....M....#...C....#", "#.........#........#",
			"#..K...............#", "####################",
		}
	case "frogger":
		return []string{
			"..H...H...H...H...H.", "....................", "....................",
			"~~~~~~~~~~~~~~~~~~~~", "~~~~~~~~~~~~~~~~~~~~", "~~~~~~~~~~~~~~~~~~~~",
			"....................", "====================", "====================",
			"====================", "..........P.........",
		}
	default:
		return []string{
			"....................", ".####..####..####...", ".####..####..####...",
			".####..####..####...", "....................", ".####..####..####...",
			".####..####..####...", ".####..####..####...", "....................",
			"....................", "....................",
		}
	}
}
