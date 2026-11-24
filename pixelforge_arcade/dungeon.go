package pixelforge_arcade

import "math"

func (g *Game) initDungeon() {
	g.Title = "Vault of Ember"
	g.Message = "Collect 3 gems, find the key, reach the door. Space: attack."
	g.coins = make(map[[2]int]bool)
	for y, row := range g.Layout {
		for x, cell := range row {
			switch cell {
			case 'P':
				g.X, g.Y = float64(x*Tile+8), float64(y*Tile+8)
			case 'C':
				g.coins[[2]int{x, y}] = true
			case 'K':
				g.key = [2]int{x, y}
			case 'E':
				g.exit = [2]int{x, y}
			case 'M':
				g.Actors = append(g.Actors, actor{x: float64(x*Tile + 8), y: float64(y*Tile + 8), alive: true})
			}
		}
	}
}

func (g *Game) stepDungeon(in Input) {
	dx, dy := direction(in)
	if g.cooldown == 0 && (dx != 0 || dy != 0) {
		g.dirX, g.dirY = dx, dy
		g.moveDungeon(dx, dy)
		g.cooldown = 5
	}
	if in.Action && g.attack == 0 {
		g.swing()
		g.attack = 9
	}
	g.collectDungeon()
	if g.Tick%24 == 0 {
		g.chaseDungeon()
	}
}

func (g *Game) moveDungeon(dx, dy int) {
	x, y := g.X+float64(dx*Tile), g.Y+float64(dy*Tile)
	if g.tile(int(x)/Tile, int(y)/Tile) == '#' {
		return
	}
	for _, enemy := range g.Actors {
		if enemy.alive && enemy.x == x && enemy.y == y {
			return
		}
	}
	g.X, g.Y = x, y
}

func (g *Game) swing() {
	for i := range g.Actors {
		e := &g.Actors[i]
		if e.alive && math.Abs(e.x-g.X)+math.Abs(e.y-g.Y) <= Tile {
			e.alive = false
			g.Score += 100
			g.Message = "Slime defeated. Keep exploring."
		}
	}
}

func (g *Game) collectDungeon() {
	p := [2]int{int(g.X) / Tile, int(g.Y) / Tile}
	if g.coins[p] {
		delete(g.coins, p)
		g.Score += 100
		g.Message = "Gem collected. Find the key and the remaining gems."
	}
	if p == g.key && !g.HasKey {
		g.HasKey = true
		g.Score += 50
		g.Message = "You found the vault key."
	}
	if p != g.exit {
		return
	}
	if !g.HasKey || len(g.coins) != 0 {
		g.Message = "The door needs the key and every gem."
		return
	}
	g.Won = true
	g.Score += 250
	g.Message = "Vault cleared! All gems recovered."
}

func (g *Game) chaseDungeon() {
	for i := range g.Actors {
		e := &g.Actors[i]
		if !e.alive {
			continue
		}
		distance := math.Abs(e.x-g.X) + math.Abs(e.y-g.Y)
		if distance <= Tile {
			g.hurt("Slime hit! Space attacks nearby enemies.")
			continue
		}
		if distance > 8*Tile {
			continue
		}
		dx, dy := 0.0, 0.0
		if e.x < g.X {
			dx = Tile
		} else if e.x > g.X {
			dx = -Tile
		}
		if e.y < g.Y {
			dy = Tile
		} else if e.y > g.Y {
			dy = -Tile
		}
		if dx != 0 && g.enemyCanMove(i, e.x+dx, e.y) {
			e.x += dx
		} else if dy != 0 && g.enemyCanMove(i, e.x, e.y+dy) {
			e.y += dy
		}
	}
}

func (g *Game) enemyCanMove(index int, x, y float64) bool {
	if g.tile(int(x)/Tile, int(y)/Tile) == '#' || (x == g.X && y == g.Y) {
		return false
	}
	for i, e := range g.Actors {
		if i != index && e.alive && e.x == x && e.y == y {
			return false
		}
	}
	return true
}
