package pixelforge_physics

// WrapPosition wraps pos.X into [0, width) and pos.Y into [0, height) using
// modular arithmetic on the Fixed32 integer representation. This is what
// Asteroids needs.
//
// Width and height are pixel integers; the wrap interval is converted to
// Fixed32 internally.
//
// Behaviour at exactly the boundary: x == width wraps to x == 0, matching
// the inclusive-lower / exclusive-upper convention.
func WrapPosition(pos Vec2, width, height int) Vec2 {
	if width > 0 {
		w := FixedFromInt(width)
		pos.X = wrapAxis(pos.X, w)
	}
	if height > 0 {
		h := FixedFromInt(height)
		pos.Y = wrapAxis(pos.Y, h)
	}
	return pos
}

// wrapAxis folds v into [0, span). Uses two passes (one for negative, one
// for ≥ span) which is bounded and deterministic — no float ops.
func wrapAxis(v, span Fixed32) Fixed32 {
	// Fast path: already in range.
	if v >= 0 && v < span {
		return v
	}
	// Negative — add span until ≥ 0.
	for v < 0 {
		v = v.Add(span)
	}
	// Too large — subtract span until in range.
	for v >= span {
		v = v.Sub(span)
	}
	return v
}

// WrapBody wraps the body's Position if the World's PhysicsConfig has
// ScreenWrap enabled. No-op otherwise. Returns true if a wrap happened.
func WrapBody(body *Body, world *World) bool {
	if body == nil || world == nil {
		return false
	}
	if !world.cfg.ScreenWrap {
		return false
	}
	before := body.Position
	body.Position = WrapPosition(body.Position, world.cfg.ScreenWidth, world.cfg.ScreenHeight)
	if body.Position != before {
		body.Sync()
		return true
	}
	return false
}
