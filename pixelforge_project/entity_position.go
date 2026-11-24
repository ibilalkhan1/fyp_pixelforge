package pixelforge_project

// PositionXY and SetPositionXY expose mutable entity position to behaviour actions.
func (e *Entity) PositionXY() (float64, float64) {
	if e == nil {
		return 0, 0
	}
	return e.Position.X, e.Position.Y
}
func (e *Entity) SetPositionXY(x, y float64) {
	if e != nil {
		e.Position.X = x
		e.Position.Y = y
	}
}
