package pixelforge_arcade

import (
	"fmt"
	"math"

	pf "github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// Colors is the shared palette stored in the native Studio projects.
var Colors = [16]pf.RGB{
	0x080e17, 0xf3f7f7, 0x101d2b, 0x243a48, 0x476472, 0x7c8298, 0x18bbaa, 0x8be2cb,
	0xf5cc62, 0xd3725b, 0xa5486e, 0x4bb9f0, 0x28599a, 0x62c64b, 0xa4e56a, 0xb5b4af,
}

func rectangle(color, x, y, w, h int) {
	pf.SetColor(pf.Color(color))
	pf.RectFill(x, y, x+w-1, y+h-1)
}

func label(text string, x, y int) { pf.SetColor(1); pixelforge_cofont.Print(text, x, y) }

// Draw renders only through PixelForge's indexed-colour canvas API.
func (g *Game) Draw() {
	pf.Camera = pf.Position{}
	pf.Screen().Clear(0)
	switch g.Kind {
	case "dungeon":
		g.drawDungeon()
	case "frogger":
		g.drawFrogger()
	case "city":
		g.drawCity()
	}
	rectangle(0, 0, 0, Width, 24)
	label(g.Title, 5, 4)
	label(fmt.Sprintf("SCORE %04d  HP %d", g.Score, g.Health), 190, 4)
	label(g.hint(), 5, 14)
	rectangle(0, 0, 200, Width, 8)
	label("ARROWS / WASD   SPACE ACTION   P PAUSE   R RESTART", 5, 201)
	if g.Paused {
		g.overlay("PAUSED", "Press P or Resume")
	}
	if g.Won {
		g.overlay("LEVEL COMPLETE", "Restart for another run")
	}
	if g.Over {
		g.overlay("GAME OVER", "Restart to try again")
	}
}

func (g *Game) hint() string {
	switch g.Kind {
	case "dungeon":
		return fmt.Sprintf("GEMS LEFT %d   KEY %t   SPACE: SWORD", len(g.coins), g.HasKey)
	case "frogger":
		return fmt.Sprintf("FROGS HOME %d/3   TIME %d   LOGS ARE SAFE", g.Deliveries, g.remaining/30)
	default:
		mode, task := "ON FOOT", "PICKUP"
		if g.Driving {
			mode = "DRIVING"
		}
		if g.Carrying {
			task = "DELIVER"
		}
		return fmt.Sprintf("%s  %s %d/3  HEAT %d  TIME %d", mode, task, g.Deliveries+1, g.Heat, g.remaining/30)
	}
}

func (g *Game) overlay(title, detail string) {
	rectangle(6, 48, 77, 224, 54)
	rectangle(2, 50, 79, 220, 50)
	label(title, (Width-len(title)*4)/2, 89)
	label(detail, (Width-len(detail)*4)/2, 108)
}

func (g *Game) drawDungeon() {
	for y, row := range g.Layout {
		for x, c := range row {
			px, py := x*Tile, y*Tile+24
			rectangle(2, px, py, Tile, Tile)
			if c == '#' {
				rectangle(3, px, py, Tile, Tile)
				rectangle(4, px, py, Tile, 3)
				rectangle(2, px+7, py+4, 1, 11)
				rectangle(2, px, py+12, 16, 1)
			} else {
				rectangle(3, px+2, py+3, 1, 1)
				rectangle(3, px+11, py+12, 2, 1)
			}
		}
	}
	for coin := range g.coins {
		x, y := coin[0]*Tile+8, coin[1]*Tile+32
		pf.SetColor(11)
		pf.CircFill(x, y, 4)
		rectangle(1, x-1, y-2, 2, 3)
	}
	if !g.HasKey {
		x, y := g.key[0]*Tile+5, g.key[1]*Tile+29
		pf.SetColor(8)
		pf.Circ(x, y, 3)
		rectangle(8, x+2, y, 7, 2)
		rectangle(8, x+7, y+1, 2, 3)
	}
	x, y := g.exit[0]*Tile, g.exit[1]*Tile+24
	rectangle(8, x+2, y+1, 12, 15)
	rectangle(9, x+4, y+3, 8, 13)
	rectangle(0, x+9, y+9, 2, 2)
	for _, enemy := range g.Actors {
		if !enemy.alive {
			continue
		}
		x, y := int(enemy.x)-6, int(enemy.y)+19
		rectangle(10, x+2, y, 8, 2)
		rectangle(10, x, y+2, 12, 7)
		rectangle(1, x+2, y+3, 2, 2)
		rectangle(1, x+8, y+3, 2, 2)
		rectangle(0, x+3, y+7, 6, 1)
	}
	if g.immune%6 < 3 {
		hero(int(g.X), int(g.Y)+24)
	}
	if g.attack > 5 {
		x, y := int(g.X)+g.dirX*10, int(g.Y)+24+g.dirY*10
		pf.SetColor(1)
		pf.Line(x-5, y-5, x+5, y+5)
		pf.Line(x+5, y-5, x-5, y+5)
	}
}

func hero(x, y int) {
	rectangle(0, x-5, y+4, 11, 3)
	rectangle(8, x-3, y-6, 7, 5)
	rectangle(11, x-4, y-1, 9, 6)
	rectangle(12, x-4, y+4, 3, 3)
	rectangle(12, x+2, y+4, 3, 3)
	rectangle(1, x-1, y-4, 2, 1)
	rectangle(1, x+2, y-4, 2, 1)
	pf.SetColor(8)
	pf.SetPixel(x-5, y+1)
	pf.SetPixel(x+5, y+1)
}

func frog(x, y int) {
	rectangle(13, x-4, y-3, 9, 8)
	rectangle(14, x-3, y-5, 3, 3)
	rectangle(14, x+2, y-5, 3, 3)
	rectangle(0, x-2, y-4, 1, 1)
	rectangle(0, x+3, y-4, 1, 1)
	rectangle(14, x-6, y+1, 2, 5)
	rectangle(14, x+5, y+1, 2, 5)
	rectangle(6, x-2, y+4, 5, 1)
}

func (g *Game) drawFrogger() {
	for row := 0; row < Rows; row++ {
		y := row*Tile + 24
		color := 6
		if row >= 3 && row <= 5 {
			color = 12
		}
		if row >= 7 && row <= 9 {
			color = 3
		}
		rectangle(color, 0, y, Width, Tile)
		if color == 3 {
			for x := 0; x < Width; x += 32 {
				rectangle(8, x, y+14, 12, 1)
			}
		}
		if color == 12 {
			for x := 0; x < Width; x += 24 {
				rectangle(11, x+(g.Tick/5)%12, y+11, 8, 1)
			}
		}
	}
	for i, filled := range g.Homes {
		x := 40 + i*64
		pf.SetColor(14)
		pf.CircFill(x, 32, 9)
		if filled {
			frog(x, 32)
		} else {
			rectangle(6, x-2, 29, 5, 5)
		}
	}
	for _, a := range g.Actors {
		x, y := int(a.x), a.row*Tile+24
		if a.row < 6 {
			rectangle(9, x, y+2, int(a.width), 12)
			rectangle(8, x+2, y+3, int(a.width)-4, 2)
			for xx := x + 7; xx < x+int(a.width); xx += 15 {
				rectangle(10, xx, y+7, 5, 1)
			}
		} else {
			car(x+int(a.width)/2, y+8, a.color, true, false)
		}
	}
	frog(int(g.X), int(g.Y)+24)
}

func car(x, y, color int, horizontal, police bool) {
	w, h := 23, 12
	if !horizontal {
		w, h = h, w
	}
	x -= w / 2
	y -= h / 2
	rectangle(0, x+1, y+1, w, h)
	rectangle(color, x, y, w, h)
	if horizontal {
		rectangle(2, x+6, y+2, 10, 8)
		rectangle(11, x+6, y+2, 3, 8)
		rectangle(0, x+3, y-1, 5, 2)
		rectangle(0, x+15, y+h-1, 5, 2)
		rectangle(1, x+w-2, y+2, 2, 2)
		rectangle(1, x+w-2, y+h-4, 2, 2)
	} else {
		rectangle(2, x+2, y+6, 8, 10)
		rectangle(11, x+2, y+6, 8, 3)
		rectangle(0, x-1, y+3, 2, 5)
		rectangle(0, x+w-1, y+15, 2, 5)
		rectangle(1, x+2, y, 2, 2)
		rectangle(1, x+w-4, y, 2, 2)
	}
	if police {
		rectangle(9, x+w/2-3, y+h/2, 3, 2)
		rectangle(11, x+w/2, y+h/2, 3, 2)
	}
}

func (g *Game) drawCity() {
	rectangle(3, 0, 24, Width, 176)
	for y, row := range g.Layout {
		for x, c := range row {
			if c != '#' {
				continue
			}
			px, py := x*Tile, y*Tile+24
			rectangle(4, px-2, py-2, 20, 20)
			rectangle(2, px, py, 16, 16)
			rectangle(5, px+1, py+1, 14, 3)
			rectangle(12, px+4, py+7, 5, 5)
			if (x+y)%3 == 0 {
				rectangle(8, px+5, py+8, 3, 3)
			}
		}
	}
	for _, row := range []int{0, 4, 8} {
		for x := 0; x < Width; x += 24 {
			rectangle(8, x, row*Tile+31, 10, 1)
		}
	}
	for y := 25; y < 200; y += 24 {
		rectangle(1, 96, y, 1, 10)
		rectangle(1, 192, y, 1, 10)
	}
	// Small pedestrians move on the broad bottom pavement, away from traffic lanes.
	for i := 0; i < 4; i++ {
		x := 24 + i*75 + (g.Tick/12+i)%8
		rectangle(8, x, 185, 3, 3)
		rectangle(6, x-1, 188, 5, 5)
		rectangle(0, x-1, 193, 2, 2)
	}
	target := g.cityTarget()
	color := 8
	if g.Carrying {
		color = 14
	}
	pf.SetColor(pf.Color(color))
	pf.Circ(int(target[0]), int(target[1])+24, 11+(g.Tick/8)%3)
	rectangle(color, int(target[0])-3, int(target[1])+21, 7, 7)
	for _, a := range g.Actors {
		car(int(a.x), int(a.y)+24, a.color, true, false)
	}
	car(int(g.CarX), int(g.CarY)+24, 8, g.dirX != 0, false)
	if !g.Driving {
		hero(int(g.X), int(g.Y)+24)
	}
	if g.Heat > 0 {
		for i := 0; i < g.Heat; i++ {
			label("!", 287+i*8, 188)
		}
	}
	if g.Driving && math.Hypot(g.vx, g.vy) > 1 {
		rectangle(5, int(g.X)-g.dirX*13, int(g.Y)+24-g.dirY*13, 2, 2)
	}
}
