package pixelforge_physics

// IsOneWayHit reports whether a one-way platform collision should be
// resolved or ignored.
//
// A one-way platform stops a body from falling down through it, but lets it
// jump up through it from below. The resolv MTV vector points OUT of the
// platform along the resolution axis. For a typical horizontal one-way
// platform, the MTV's Y component is negative when the player is on top
// (push up to resolve), positive when the player is below (push down — we
// want to ignore in that case).
//
// Rule: resolve if and only if the player is moving DOWN (velY >= 0) AND
// the MTV pushes the player UP (mtvY < 0). All other cases pass through.
//
// This matches the recipe in pixelforge_physics/doc.go.
func IsOneWayHit(mtvY, velY Fixed32) bool {
	return mtvY < 0 && velY >= 0
}

// OneWayResolve applies a one-way resolution to a body if and only if
// IsOneWayHit returns true. Returns the new (potentially unchanged) body
// position. The caller passes the MTV obtained from a resolv Intersection.
func OneWayResolve(body *Body, mtv Vec2) (resolved bool) {
	if body == nil {
		return false
	}
	if !IsOneWayHit(mtv.Y, body.Velocity.Y) {
		return false
	}
	body.Position.Y = body.Position.Y.Add(mtv.Y)
	// Clamp downward velocity so the body sits cleanly on the platform.
	if body.Velocity.Y > 0 {
		body.Velocity.Y = FixedZero
	}
	body.Grounded = true
	body.Sync()
	return true
}
