package main

import (
	"encoding/json"
	"errors"
	"fmt"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	ph "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
)

const width, height, tile = 320, 180, 10

// Scene is the browser creator format, separate from Studio's .pforge schema.
type Scene struct {
	Version int      `json:"version"`
	Mode    string   `json:"mode,omitempty"`
	Goal    *[2]int  `json:"goal,omitempty"`
	Hazards [][2]int `json:"hazards,omitempty"`
	Name    string   `json:"name"`
	Walls   [][2]int `json:"walls"`
	Gems    [][2]int `json:"gems"`
	Spawn   [2]int   `json:"spawn"`
	Color   int      `json:"color"`
	Speed   float64  `json:"speed"`
	Sprite  []int    `json:"sprite"`
}
type Game struct {
	Scene     Scene
	X, Y      ph.Fixed32
	Collected map[int]bool
	Tick      int
	Won       bool
	Lost      bool
}

func parseScene(data []byte) (Scene, error) {
	var s Scene
	if len(data) > 100000 {
		return s, errors.New("scene exceeds 100 KB")
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse scene: %w", err)
	}
	if s.Mode == "" {
		s.Mode = "collect"
	}
	if s.Mode != "collect" && s.Mode != "escape" {
		return s, errors.New("choose collect or escape rules")
	}
	if s.Mode == "escape" && s.Goal == nil {
		return s, errors.New("escape games need a finish tile")
	}
	if err := validateSceneLimits(s); err != nil {
		return s, err
	}
	if err := validateSceneCells(s); err != nil {
		return s, err
	}

	if err := validateScenePlacements(s); err != nil {
		return s, err
	}
	return s, nil
}

func validateSceneLimits(s Scene) error {
	if s.Version != 1 {
		return errors.New("unsupported scene version")
	}
	if s.Speed < 0.5 || s.Speed > 4 {
		return errors.New("speed must be between 0.5 and 4")
	}
	if s.Color < 0 || s.Color > 63 {
		return errors.New("colour must be between 0 and 63")
	}
	if len(s.Name) > 80 || len(s.Walls) > 576 || len(s.Gems) > 100 || len(s.Hazards) > 100 {
		return errors.New("scene is too large")
	}
	if len(s.Sprite) != 64 {
		return errors.New("sprite must have 64 pixels")
	}
	for _, p := range s.Sprite {
		if p < 0 || p > 63 {
			return errors.New("invalid sprite colour")
		}
	}
	return nil
}

func validateScenePlacements(s Scene) error {
	for _, c := range s.Walls {
		if c == s.Spawn {
			return errors.New("player spawn overlaps a wall")
		}
	}
	for _, c := range s.Hazards {
		if c == s.Spawn {
			return errors.New("player spawn overlaps a hazard")
		}
		if s.Goal != nil && c == *s.Goal {
			return errors.New("finish overlaps a hazard")
		}
	}
	if s.Goal != nil {
		if *s.Goal == s.Spawn {
			return errors.New("finish must be away from the player")
		}
		for _, c := range s.Walls {
			if c == *s.Goal {
				return errors.New("finish overlaps a wall")
			}
		}
	}
	return nil
}

func validateSceneCells(s Scene) error {
	groups := [][][2]int{s.Walls, s.Gems, s.Hazards, {s.Spawn}}
	if s.Goal != nil {
		groups = append(groups, [][2]int{*s.Goal})
	}
	for _, cells := range groups {
		for _, c := range cells {
			if c[0] < 0 || c[0] >= 32 || c[1] < 0 || c[1] >= 18 {
				return errors.New("tile outside the 32 x 18 scene")
			}
		}
	}
	return nil
}
func (g *Game) Reset(s Scene) {
	g.Scene = s
	g.X = ph.FixedFromInt(s.Spawn[0]*tile + 1)
	g.Y = ph.FixedFromInt(s.Spawn[1]*tile + 1)
	g.Collected = map[int]bool{}
	g.Tick = 0
	g.Won = false
	g.Lost = false
}
func playerOverlaps(x, y, a, b, w, h int) bool { return x < a+w && x+8 > a && y < b+h && y+8 > b }
func (g *Game) blocked(x, y ph.Fixed32) bool {
	a, b := x.Int(), y.Int()
	if a < 0 || b < 0 || a+8 > width || b+8 > height {
		return true
	}
	for _, v := range g.Scene.Walls {
		if playerOverlaps(a, b, v[0]*tile, v[1]*tile, tile, tile) {
			return true
		}
	}
	return false
}
func (g *Game) Step(dx, dy int) {
	if g.Won || g.Lost {
		return
	}
	v := ph.FixedFromFloat(g.Scene.Speed)
	nx := g.X + ph.FixedFromInt(dx).Mul(v)
	if !g.blocked(nx, g.Y) {
		g.X = nx
	}
	ny := g.Y + ph.FixedFromInt(dy).Mul(v)
	if !g.blocked(g.X, ny) {
		g.Y = ny
	}
	g.Tick++
	for i, c := range g.Scene.Gems {
		if playerOverlaps(g.X.Int(), g.Y.Int(), c[0]*tile+2, c[1]*tile+2, 6, 6) {
			g.Collected[i] = true
		}
	}
	for _, c := range g.Scene.Hazards {
		if playerOverlaps(g.X.Int(), g.Y.Int(), c[0]*tile+2, c[1]*tile+2, 6, 6) {
			g.Lost = true
			return
		}
	}
	if g.Scene.Mode == "escape" && g.Scene.Goal != nil {
		c := *g.Scene.Goal
		g.Won = playerOverlaps(g.X.Int(), g.Y.Int(), c[0]*tile+1, c[1]*tile+1, 8, 8)
	} else {
		g.Won = len(g.Scene.Gems) > 0 && len(g.Collected) == len(g.Scene.Gems)
	}
}
func (g *Game) Render(grid bool) []byte {
	pf.Camera = pf.Position{}
	pf.SetDrawTarget(pf.Screen())
	pf.Screen().Clear(1)
	if grid {
		pf.SetColor(2)
		for x := 0; x < width; x += tile {
			pf.Line(x, 0, x, height-1)
		}
		for y := 0; y < height; y += tile {
			pf.Line(0, y, width-1, y)
		}
	}
	for _, v := range g.Scene.Walls {
		x, y := v[0]*tile, v[1]*tile
		pf.SetColor(13)
		pf.RectFill(x, y, x+9, y+9)
		pf.SetColor(6)
		pf.Line(x, y, x+9, y)
		pf.Line(x, y, x, y+9)
	}
	for i, c := range g.Scene.Gems {
		if g.Collected[i] {
			continue
		}
		pf.SetColor(10)
		pf.CircFill(c[0]*tile+5, c[1]*tile+5, 3)
		pf.SetColor(7)
		pf.SetPixel(c[0]*tile+4, c[1]*tile+4)
	}
	for _, c := range g.Scene.Hazards {
		x, y := c[0]*tile, c[1]*tile
		pf.SetColor(8)
		pf.RectFill(x+2, y+2, x+7, y+7)
		pf.SetColor(7)
		pf.Line(x+3, y+3, x+6, y+6)
		pf.Line(x+6, y+3, x+3, y+6)
	}
	if g.Scene.Goal != nil {
		x, y := g.Scene.Goal[0]*tile, g.Scene.Goal[1]*tile
		pf.SetColor(11)
		pf.Rect(x+1, y+1, x+8, y+8)
		pf.Line(x+3, y+5, x+5, y+7)
		pf.Line(x+5, y+7, x+7, y+3)
	}
	src := pf.NewCanvas(8, 8)
	for i, c := range g.Scene.Sprite {
		src.Set(i%8, i/8, uint8(c))
	}
	pf.DrawSprite(pf.CanvasSprite(src), g.X.Int(), g.Y.Int())
	if g.Won {
		pf.SetColor(11)
		pf.Rect(1, 1, 318, 178)
	}
	if g.Lost {
		pf.SetColor(8)
		pf.Rect(1, 1, 318, 178)
	}
	rgba := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, b, c := pf.Palette[pf.Screen().Get(x, y)&63].RGB()
			i := (y*width + x) * 4
			rgba[i] = r
			rgba[i+1] = b
			rgba[i+2] = c
			rgba[i+3] = 255
		}
	}
	return rgba
}
