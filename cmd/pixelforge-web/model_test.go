package main

import (
	"encoding/json"
	"testing"
)

func fixture() Scene {
	return Scene{
		Version: 1, Name: "test", Spawn: [2]int{1, 1}, Speed: 1, Color: 11,
		Sprite: make([]int, 64), Walls: [][2]int{{2, 1}}, Gems: [][2]int{{1, 2}},
	}
}
func TestCollisionAndWin(t *testing.T) {
	var g Game
	g.Reset(fixture())
	for i := 0; i < 20; i++ {
		g.Step(1, 0)
	}
	if g.X.Int()+8 > 20 {
		t.Fatal("walked through wall")
	}
	g.Reset(fixture())
	for i := 0; i < 15; i++ {
		g.Step(0, 1)
	}
	if !g.Won || len(g.Collected) != 1 {
		t.Fatal("collectible did not end level")
	}
	tick := g.Tick
	g.Step(0, 1)
	if g.Tick != tick {
		t.Fatal("completed game advanced")
	}
}
func TestSceneValidation(t *testing.T) {
	s := fixture()
	mutations := []func(*Scene){
		func(s *Scene) { s.Speed = 100 },
		func(s *Scene) { s.Walls = append(s.Walls, s.Spawn) },
		func(s *Scene) { s.Sprite = nil },
		func(s *Scene) { s.Spawn = [2]int{-1, 0} },
	}
	for _, mutate := range mutations {
		bad := s
		mutate(&bad)
		b, err := json.Marshal(bad)
		if err != nil {
			t.Fatal(err)
		}
		if _, e := parseScene(b); e == nil {
			t.Fatal("accepted invalid scene")
		}
	}
}
func TestDeterministicReset(t *testing.T) {
	s := fixture()
	var a, b Game
	a.Reset(s)
	b.Reset(s)
	for i := 0; i < 5; i++ {
		a.Step(0, 1)
		b.Step(0, 1)
	}
	if a.X != b.X || a.Y != b.Y || a.Tick != b.Tick {
		t.Fatal("non-deterministic movement")
	}
	a.Reset(s)
	if a.Tick != 0 || len(a.Collected) != 0 || a.Won {
		t.Fatal("reset retained state")
	}
}

func TestCreatorRulesAndLegacyScenes(t *testing.T) {
	s := fixture()
	b, marshalErr := json.Marshal(s)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	legacy, err := parseScene(b)
	if err != nil || legacy.Mode != "collect" {
		t.Fatalf("legacy scene: %v, mode %q", err, legacy.Mode)
	}
	s.Mode = "escape"
	goal := [2]int{1, 3}
	s.Goal = &goal
	var g Game
	g.Reset(s)
	for i := 0; i < 10; i++ {
		g.Step(0, 1)
	}
	if len(g.Collected) != 1 || g.Won {
		t.Fatal("escape game won before reaching finish")
	}
	for i := 0; i < 20; i++ {
		g.Step(0, 1)
	}
	if !g.Won || g.Lost {
		t.Fatal("finish did not win escape game")
	}
}
func TestHazardsStopAndResetGame(t *testing.T) {
	s := fixture()
	s.Hazards = [][2]int{{1, 2}}
	var g Game
	g.Reset(s)
	for i := 0; i < 20; i++ {
		g.Step(0, 1)
	}
	if !g.Lost || g.Won {
		t.Fatal("hazard must take priority over last crystal")
	}
	tick, y := g.Tick, g.Y
	g.Step(0, 1)
	if tick != g.Tick || y != g.Y {
		t.Fatal("lost game advanced")
	}
	g.Reset(s)
	if g.Lost || g.Won || g.Tick != 0 {
		t.Fatal("reset retained terminal state")
	}
}
func TestCreatorValidation(t *testing.T) {
	cases := []func(*Scene){
		func(s *Scene) { s.Mode = "unknown" },
		func(s *Scene) { s.Mode = "escape" },
		func(s *Scene) { c := [2]int{32, 1}; s.Goal = &c },
		func(s *Scene) { c := s.Spawn; s.Goal = &c },
		func(s *Scene) { c := s.Walls[0]; s.Goal = &c },
		func(s *Scene) { s.Hazards = [][2]int{{-1, 1}} },
		func(s *Scene) { s.Hazards = [][2]int{s.Spawn} },
		func(s *Scene) { c := [2]int{3, 3}; s.Goal = &c; s.Hazards = [][2]int{c} },
	}
	for i, mutate := range cases {
		s := fixture()
		mutate(&s)
		b, marshalErr := json.Marshal(s)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, err := parseScene(b); err == nil {
			t.Errorf("accepted invalid creator scene %d", i)
		}
	}
}
