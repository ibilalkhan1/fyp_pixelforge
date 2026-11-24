package main

import (
	"encoding/json"
	"testing"
)

func fixture() Scene {
	return Scene{Version: 1, Name: "test", Spawn: [2]int{1, 1}, Speed: 1, Color: 11, Sprite: make([]int, 64), Walls: [][2]int{{2, 1}}, Gems: [][2]int{{1, 2}}}
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
	for _, mutate := range []func(*Scene){func(s *Scene) { s.Speed = 100 }, func(s *Scene) { s.Walls = append(s.Walls, s.Spawn) }, func(s *Scene) { s.Sprite = nil }, func(s *Scene) { s.Spawn = [2]int{-1, 0} }} {
		bad := s
		mutate(&bad)
		b, _ := json.Marshal(bad)
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
