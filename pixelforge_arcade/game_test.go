package pixelforge_arcade_test

import (
	"reflect"
	"testing"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	arcade "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_arcade"
)

func newGame(t *testing.T, kind string) *arcade.Game {
	t.Helper()
	g, err := arcade.New(kind, nil)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestDungeonCompleteRoute(t *testing.T) {
	g := newGame(t, "dungeon")
	// Walk the actual authored level with attacks held. This exercises wall
	// collision, chasing enemies, collection, the key and the locked exit.
	targets := [][2]int{{3, 3}, {11, 1}, {14, 7}, {3, 9}, {18, 1}}
	for _, target := range targets {
		for tries := 0; tries < 250 && !g.Won; tries++ {
			x, y := int(g.X)/arcade.Tile, int(g.Y)/arcade.Tile
			if x == target[0] && y == target[1] {
				break
			}
			step := nextStep(t, g.Layout, [2]int{x, y}, target)
			in := arcade.Input{Up: step[1] < y, Down: step[1] > y, Left: step[0] < x, Right: step[0] > x, Action: true}
			for range 5 {
				g.Step(in)
			}
			if g.Over {
				t.Fatalf("died on route to %v: %v", target, g.Snapshot())
			}
		}
	}
	if !g.Won || !g.HasKey || g.Score < 600 {
		t.Fatalf("route did not complete: %v", g.Snapshot())
	}
	g.Step(arcade.Input{Restart: true})
	if g.Won || g.HasKey || g.Score != 0 || g.Health != 3 {
		t.Fatal("restart did not clear objective state")
	}
}

func nextStep(t *testing.T, layout []string, start, goal [2]int) [2]int {
	t.Helper()
	queue := [][2]int{start}
	seen := map[[2]int]bool{start: true}
	parents := map[[2]int][2]int{}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == goal {
			for parents[p] != start {
				p = parents[p]
			}
			return p
		}
		for _, d := range [][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			n := [2]int{p[0] + d[0], p[1] + d[1]}
			if n[0] < 0 || n[0] >= arcade.Columns || n[1] < 0 || n[1] >= arcade.Rows || seen[n] || layout[n[1]][n[0]] == '#' {
				continue
			}
			seen[n] = true
			parents[n] = p
			queue = append(queue, n)
		}
	}
	t.Fatalf("no route from %v to %v", start, goal)
	return start
}

func TestFroggerHazardsLogsAndHomes(t *testing.T) {
	g := newGame(t, "frogger")
	g.X, g.Y = 10, 56
	g.Step(arcade.Input{})
	if g.Health != 2 || g.Y != 168 {
		t.Fatal("water did not cost a life and reset spawn")
	}
	g = newGame(t, "frogger")
	g.X, g.Y = 25, 152
	g.Step(arcade.Input{})
	if g.Health != 2 {
		t.Fatal("traffic did not cost a life")
	}
	g = newGame(t, "frogger")
	g.X, g.Y = 70, 56
	g.Step(arcade.Input{})
	if g.Health != 3 || g.X <= 70 {
		t.Fatal("safe log did not carry the frog")
	}
	for _, x := range []float64{40, 104, 168} {
		for range 31 {
			g.Step(arcade.Input{})
		}
		g.X, g.Y = x, 8
		g.Step(arcade.Input{})
	}
	if !g.Won || g.Deliveries != 3 {
		t.Fatal("three different homes did not finish the game")
	}
	g.Step(arcade.Input{Restart: true})
	if g.Won || g.Deliveries != 0 || g.Homes[0] {
		t.Fatal("restart retained occupied homes")
	}
}

func TestCityVehicleAndDeliveryObjectives(t *testing.T) {
	g := newGame(t, "city")
	g.Step(arcade.Input{Action: true})
	if !g.Driving {
		t.Fatal("could not enter the nearby car")
	}
	x := g.X
	for range 20 {
		g.Step(arcade.Input{Right: true})
	}
	if g.X <= x+20 || g.CarX != g.X {
		t.Fatal("vehicle did not move with its driver")
	}
	// Interaction requires proximity. Pressing action far from a parcel cannot collect it.
	if g.Carrying {
		t.Fatal("vehicle interaction collected a distant parcel")
	}
	pickups := [][2]float64{{280, 152}, {32, 72}, {288, 8}}
	depots := [][2]float64{{168, 72}, {168, 8}, {64, 152}}
	for i := range pickups {
		for range 12 {
			g.Step(arcade.Input{})
		}
		g.X, g.Y = pickups[i][0], pickups[i][1]
		g.Step(arcade.Input{Action: true})
		if !g.Carrying {
			t.Fatalf("pickup %d failed", i)
		}
		for range 12 {
			g.Step(arcade.Input{})
		}
		g.X, g.Y = depots[i][0], depots[i][1]
		g.Step(arcade.Input{Action: true})
	}
	if !g.Won || g.Deliveries != 3 || g.Score != 600 {
		t.Fatalf("delivery loop failed: %v", g.Snapshot())
	}
}

func TestPresetValidationPauseAndDeterminism(t *testing.T) {
	if _, err := arcade.New("missing", nil); err == nil {
		t.Fatal("unknown preset accepted")
	}
	if _, err := arcade.New("dungeon", []string{"bad"}); err == nil {
		t.Fatal("bad layout accepted")
	}
	for _, kind := range []string{"dungeon", "frogger", "city"} {
		t.Run(kind, func(t *testing.T) {
			a, b := newGame(t, kind), newGame(t, kind)
			for tick := 0; tick < 400; tick++ {
				in := arcade.Input{Up: tick%80 < 20, Right: tick%80 >= 20 && tick%80 < 40,
					Down: tick%80 >= 40 && tick%80 < 60, Left: tick%80 >= 60, Action: tick%7 == 0}
				a.Step(in)
				b.Step(in)
			}
			if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
				t.Fatal("simulation is not repeatable")
			}
			a.Reset()
			a.Step(arcade.Input{Pause: true})
			tick := a.Tick
			x := a.X
			for range 30 {
				a.Step(arcade.Input{Right: true})
			}
			if a.Tick != tick || a.X != x {
				t.Fatal("pause did not freeze simulation")
			}
			pf.SetScreenSize(arcade.Width, arcade.Height)
			copy(pf.Palette[:], arcade.Colors[:])
			a.Draw()
			colors := map[pf.Color]bool{}
			for y := 0; y < arcade.Height; y++ {
				for x := 0; x < arcade.Width; x++ {
					colors[pf.Screen().Get(x, y)] = true
				}
			}
			if len(colors) < 7 {
				t.Fatal("rendered frame lacks expected scene detail")
			}
		})
	}
}
