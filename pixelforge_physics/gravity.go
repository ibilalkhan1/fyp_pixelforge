package pixelforge_physics

// Integrate advances a body one tick. It applies the World's gravity,
// applies drag, integrates velocity into position, and syncs the body's
// resolv shape.
//
// dt is the fraction of a second this tick represents — for a 60 TPS loop,
// pass FixedFromFloat(1.0/60.0) (or, equivalently, use the World's TPS to
// scale gravity yourself).
//
// Integrate does NOT resolve collisions — call ResolveTilemapCollision
// afterward for tile-based games, or rely on the resolv Intersection helper
// for arbitrary-shape games.
//
// Deterministic: all math is Fixed32, no math.* calls, no time.Now() reads.
//
// Ladder-aware: when body.LadderClimbing is true the gravity contribution is
// suppressed for this tick. The Donkey-Kong-class ladder climb (U17) drives
// vertical motion authoritatively via ApplyLadderMovement and would fight a
// gravity term on the same axis otherwise. Drag + integration of the
// existing Velocity still run so a horizontal push while climbing is honoured.
func Integrate(body *Body, world *World, dt Fixed32) {
	if body == nil || body.Static {
		return
	}
	cfg := world.cfg

	if !body.LadderClimbing {
		// gravity acceleration per tick (units / tick²)
		// stored gravity is per-second² — multiply by dt to get per-tick velocity delta
		gx := cfg.Gravity.X.Mul(dt)
		gy := cfg.Gravity.Y.Mul(dt)
		body.Velocity.X = body.Velocity.X.Add(gx)
		body.Velocity.Y = body.Velocity.Y.Add(gy)
	}

	// Drag: multiply velocity by (1 - drag).
	if cfg.Drag != 0 {
		oneMinusDrag := FixedOne.Sub(cfg.Drag)
		body.Velocity.X = body.Velocity.X.Mul(oneMinusDrag)
		body.Velocity.Y = body.Velocity.Y.Mul(oneMinusDrag)
	}

	// Integrate velocity into position.
	body.Position.X = body.Position.X.Add(body.Velocity.X)
	body.Position.Y = body.Position.Y.Add(body.Velocity.Y)

	// Sync resolv shape with new Fixed32 position.
	body.Sync()
}

// ApplyJump sets the body's Y velocity to -strength (upward in screen
// coordinates) and clears the Grounded flag. Caller is responsible for
// gating on Grounded — the substrate does not enforce "no double jump".
func ApplyJump(body *Body, strength Fixed32) {
	if body == nil {
		return
	}
	body.Velocity.Y = strength.Neg()
	body.Grounded = false
}

// ApplyImpulse adds (vx, vy) to the body's velocity. Used by motion sinks
// for thrust and similar verbs.
func ApplyImpulse(body *Body, vx, vy Fixed32) {
	if body == nil {
		return
	}
	body.Velocity.X = body.Velocity.X.Add(vx)
	body.Velocity.Y = body.Velocity.Y.Add(vy)
}
