package pixelforge_arcade

import "math"

var cityPickups = [3][2]float64{{280, 152}, {32, 72}, {288, 8}}
var cityDepots = [3][2]float64{{168, 72}, {168, 8}, {64, 152}}

func (g *Game) initCity() {
	g.Title = "Neon Courier"
	g.Health = 6
	g.remaining = 5400
	g.X, g.Y = 44, 152
	g.CarX, g.CarY = 64, 152
	g.Message = "Space near yellow car: drive. Collect and deliver 3 parcels."
	g.Actors = []actor{
		{x: 120, y: 8, speed: 1.0, color: 9, alive: true},
		{x: 240, y: 72, speed: -0.8, color: 11, alive: true},
		{x: 16, y: 136, speed: 1.2, color: 10, alive: true},
	}
}

func (g *Game) cityTarget() [2]float64 {
	index := min(g.Deliveries, 2)
	if g.Carrying {
		return cityDepots[index]
	}
	return cityPickups[index]
}

func (g *Game) stepCity(in Input) {
	g.remaining--
	if g.remaining <= 0 {
		g.Over = true
		g.Message = "Shift ended. Restart for another delivery run."
		return
	}
	if g.actionPressed && g.cooldown == 0 {
		g.cityInteract()
		g.cooldown = 12
	}
	if g.Won {
		return
	}
	g.moveCity(in)
	g.cityTraffic()
	if g.Tick%180 == 0 && g.Heat > 0 {
		g.Heat--
	}
}

func (g *Game) cityInteract() {
	target := g.cityTarget()
	if math.Hypot(g.X-target[0], g.Y-target[1]) < 19 {
		if !g.Carrying {
			g.Carrying = true
			g.Message = "Parcel collected. Follow the green delivery marker."
			return
		}
		g.Carrying = false
		g.Deliveries++
		g.Score += 200
		g.Message = "Delivery complete! Follow the gold pickup marker."
		if g.Deliveries == 3 {
			g.Won = true
			g.Message = "Shift complete! All three parcels delivered."
		}
		return
	}
	if g.Driving {
		x, y := g.X+12, g.Y
		if !g.clearAt(x, y, 4) {
			x, y = g.X, g.Y+12
		}
		if !g.clearAt(x, y, 4) {
			g.Message = "No room to exit here."
			return
		}
		g.Driving = false
		g.X, g.Y = x, y
		g.vx, g.vy = 0, 0
		g.Message = "On foot. Space beside your yellow car to enter."
		return
	}
	if math.Hypot(g.X-g.CarX, g.Y-g.CarY) < 25 {
		g.Driving = true
		g.X, g.Y = g.CarX, g.CarY
		g.Message = "Driving. Arrows steer. Space at a marker handles the parcel."
	}
}

func (g *Game) moveCity(in Input) {
	dx, dy := direction(in)
	speed := 1.25
	if g.Driving {
		speed = 2.3
	}
	wantX, wantY := float64(dx)*speed, float64(dy)*speed
	if g.Driving {
		g.vx += (wantX - g.vx) * 0.24
		g.vy += (wantY - g.vy) * 0.24
	} else {
		g.vx, g.vy = wantX, wantY
	}
	if dx != 0 || dy != 0 {
		g.dirX, g.dirY = dx, dy
	}
	radius := 4.0
	if g.Driving {
		radius = 6
	}
	if g.clearAt(g.X+g.vx, g.Y, radius) {
		g.X += g.vx
	} else {
		g.cityCrash()
		g.vx = 0
	}
	if g.clearAt(g.X, g.Y+g.vy, radius) {
		g.Y += g.vy
	} else {
		g.cityCrash()
		g.vy = 0
	}
	if g.Driving {
		g.CarX, g.CarY = g.X, g.Y
	}
}

func (g *Game) cityCrash() {
	if !g.Driving || math.Hypot(g.vx, g.vy) < 1.2 || g.immune > 0 {
		return
	}
	g.hurt("Crash! Slow down and keep to the streets.")
	g.Heat = min(3, g.Heat+1)
}

func (g *Game) cityTraffic() {
	for i := range g.Actors {
		a := &g.Actors[i]
		a.x += a.speed
		if a.x > Width+12 {
			a.x = -12
		}
		if a.x < -12 {
			a.x = Width + 12
		}
		if math.Abs(g.X-a.x) < 14 && math.Abs(g.Y-a.y) < 10 {
			g.hurt("Traffic collision. Watch the moving cars.")
			g.Heat = min(3, g.Heat+1)
		}
	}
}
