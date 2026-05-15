package pixelforge_routine

import "github.com/ibilalkhan1/fyp_pixelforge"

// Move creates a Routine step that advances pos by (dx, dy) over
// `ticks` ticks. Internally a pair of Tween-like linear interpolations
// over integer Position coordinates.
//
// `ticks == 0` or `dx == dy == 0` returns true on the first tick.
func Move(pos *pixelforge.Position, dx, dy int, ticks int) Step {
	if pos == nil {
		return func() bool { return true }
	}
	if ticks <= 0 || (dx == 0 && dy == 0) {
		return func() bool {
			pos.X += dx
			pos.Y += dy
			return true
		}
	}
	startX, startY := pos.X, pos.Y
	elapsed := 0
	return func() bool {
		elapsed++
		if elapsed >= ticks {
			pos.X = startX + dx
			pos.Y = startY + dy
			return true
		}
		t := float64(elapsed) / float64(ticks)
		pos.X = startX + int(float64(dx)*t)
		pos.Y = startY + int(float64(dy)*t)
		return false
	}
}
