package pixelforge_arcade

import "math"

func (g *Game) initFrogger() {
	g.Title = "River Hopper"
	g.Message = "Rescue 3 frogs. Avoid cars. Ride logs across the river."
	g.X, g.Y = 168, 168
	for row := 3; row <= 9; row++ {
		if row == 6 {
			continue
		}
		speed, width := 0.75, 24.0
		if row%2 == 0 {
			speed = -1.0
		}
		if row < 6 {
			width = 66
			speed *= 0.65
		}
		for i := 0; i < 3; i++ {
			g.Actors = append(g.Actors, actor{x: float64(i*120 + (row*17)%70), row: row, speed: speed,
				width: width, color: 8 + row%5})
		}
	}
}

func (g *Game) stepFrogger(in Input) {
	g.remaining--
	if g.remaining <= 0 {
		g.loseFrog("Time ran out.")
		return
	}
	for i := range g.Actors {
		a := &g.Actors[i]
		a.x += a.speed
		if a.x > Width+40 {
			a.x = -a.width
		}
		if a.x+a.width < -40 {
			a.x = Width + 40
		}
	}
	if g.immune > 0 {
		return
	}
	dx, dy := direction(in)
	if g.cooldown == 0 && (dx != 0 || dy != 0) {
		g.X += float64(dx * Tile)
		g.Y += float64(dy * Tile)
		g.cooldown = 7
		g.X = math.Max(7, math.Min(Width-7, g.X))
		g.Y = math.Max(8, math.Min(168, g.Y))
		row := int(g.Y) / Tile
		if row < g.bestRow {
			g.Score += (g.bestRow - row) * 5
			g.bestRow = row
		}
	}
	row := int(g.Y) / Tile
	if row == 0 {
		g.reachHome()
		return
	}
	if row >= 7 && row <= 9 {
		g.checkRoad(row)
	}
	if row >= 3 && row <= 5 {
		g.checkRiver(row)
	}
}

func (g *Game) checkRoad(row int) {
	for _, a := range g.Actors {
		if a.row == row && g.X+5 > a.x && g.X-5 < a.x+a.width {
			g.loseFrog("Traffic hit! Wait for a safe gap.")
			return
		}
	}
}

func (g *Game) checkRiver(row int) {
	for _, a := range g.Actors {
		if a.row == row && g.X-4 >= a.x && g.X+4 <= a.x+a.width {
			g.X += a.speed
			if g.X < 5 || g.X > Width-5 {
				g.loseFrog("The log drifted off the bank.")
			}
			return
		}
	}
	g.loseFrog("Splash! Land on a log to cross water.")
}

func (g *Game) reachHome() {
	for i := range g.Homes {
		if math.Abs(g.X-float64(40+i*64)) < 13 && !g.Homes[i] {
			g.Homes[i] = true
			g.Deliveries++
			g.Score += 100
			g.Message = "Frog rescued! Choose another empty home."
			if g.Deliveries >= 3 {
				g.Won = true
				g.Message = "River cleared! Three frogs safely home."
			}
			g.resetFrog()
			return
		}
	}
	g.loseFrog("That home is full. Aim for an empty lily pad.")
}

func (g *Game) resetFrog() {
	g.X, g.Y = 168, 168
	g.remaining = 900
	g.bestRow = 10
	g.cooldown = 7
	g.immune = 30
}

func (g *Game) loseFrog(message string) {
	g.Health--
	g.Message = message
	if g.Health <= 0 {
		g.Over = true
		g.Message = "No frogs left. Restart to try again."
	}
	g.resetFrog()
}
