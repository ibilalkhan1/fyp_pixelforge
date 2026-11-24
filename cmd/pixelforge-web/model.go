package main

import (
	"encoding/json"
	"fmt"
	pf "github.com/ibilalkhan1/fyp_pixelforge"
	ph "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
)

const width, height, tile = 320, 180, 10

// Scene is the browser demonstrator format, separate from Studio's .pforge schema.
type Scene struct {
	Version int      `json:"version"`
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
}

func parseScene(data []byte) (Scene, error) {
	var s Scene
	if len(data) > 100000 {
		return s, fmt.Errorf("scene exceeds 100 KB")
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	if s.Version != 1 {
		return s, fmt.Errorf("unsupported scene version")
	}
	if s.Speed < 0.5 || s.Speed > 4 {
		return s, fmt.Errorf("speed must be between 0.5 and 4")
	}
	if s.Color < 0 || s.Color > 63 {
		return s, fmt.Errorf("colour must be between 0 and 63")
	}
	if len(s.Name) > 80 || len(s.Walls) > 576 || len(s.Gems) > 100 {
		return s, fmt.Errorf("scene is too large")
	}
	for _, cells := range [][][2]int{s.Walls, s.Gems, [][2]int{s.Spawn}} {
		for _, c := range cells {
			if c[0] < 0 || c[0] >= 32 || c[1] < 0 || c[1] >= 18 {
				return s, fmt.Errorf("tile outside the 32 x 18 scene")
			}
		}
	}
	if len(s.Sprite) != 64 {
		return s, fmt.Errorf("sprite must have 64 pixels")
	}
	for _, p := range s.Sprite {
		if p < 0 || p > 63 {
			return s, fmt.Errorf("invalid sprite colour")
		}
	}
	for _, c := range s.Walls {
		if c == s.Spawn {
			return s, fmt.Errorf("player spawn overlaps a wall")
		}
	}
	return s, nil
}
func (g *Game) Reset(s Scene) {
	g.Scene = s
	g.X = ph.FixedFromInt(s.Spawn[0]*tile + 1)
	g.Y = ph.FixedFromInt(s.Spawn[1]*tile + 1)
	g.Collected = map[int]bool{}
	g.Tick = 0
	g.Won = false
}
func overlap(x, y, w, h, a, b, c, d int) bool { return x < a+c && x+w > a && y < b+d && y+h > b }
func (g *Game) blocked(x, y ph.Fixed32) bool {
	a, b := x.Int(), y.Int()
	if a < 0 || b < 0 || a+8 > width || b+8 > height {
		return true
	}
	for _, v := range g.Scene.Walls {
		if overlap(a, b, 8, 8, v[0]*tile, v[1]*tile, tile, tile) {
			return true
		}
	}
	return false
}
func (g *Game) Step(dx, dy int) {
	if g.Won {
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
		if overlap(g.X.Int(), g.Y.Int(), 8, 8, c[0]*tile+2, c[1]*tile+2, 6, 6) {
			g.Collected[i] = true
		}
	}
	g.Won = len(g.Scene.Gems) > 0 && len(g.Collected) == len(g.Scene.Gems)
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
	src := pf.NewCanvas(8, 8)
	for i, c := range g.Scene.Sprite {
		src.Set(i%8, i/8, uint8(c))
	}
	pf.DrawSprite(pf.CanvasSprite(src), g.X.Int(), g.Y.Int())
	if g.Won {
		pf.SetColor(11)
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
