package pixelforge_physics

// BarrelDir captures the cardinal-X direction a rolling barrel is
// currently moving. +1 = rightward, -1 = leftward, 0 = inert (used
// at construction before the first tick has fired).
type BarrelDir int8

const (
	BarrelStill BarrelDir = 0
	BarrelRight BarrelDir = 1
	BarrelLeft  BarrelDir = -1
)

// BarrelState is the per-entity rolling-barrel pattern state. DK's
// barrels roll downhill, bounce off walls (reverse direction), and
// fall off platform edges with gravity carrying them.
//
// Direction is the current horizontal sweep direction (Right/Left).
// Speed is the horizontal velocity magnitude per tick — Fixed32 so
// the substrate stays deterministic across CPUs.
//
// The state is intentionally small (no platform reference, no MTV
// memory) so a save snapshot can serialise it as two ints; the
// motion sink (capsuleruntime/motion_sink.go) keeps a map of
// BarrelState keyed by entity ID and persists it via the regular
// Globals path.
type BarrelState struct {
	Direction BarrelDir
	Speed     Fixed32
}

// NewBarrelState constructs an initial BarrelState rolling rightward
// at the given speed. DK's canonical barrel speed is ~40 px/s (≈0.67
// px/tick at 60 TPS); recipe authors override via the verb arg.
func NewBarrelState(dir BarrelDir, speed Fixed32) BarrelState {
	if dir == BarrelStill {
		dir = BarrelRight
	}
	return BarrelState{Direction: dir, Speed: speed}
}

// StepBarrelTick advances a barrel one physics tick. The caller is
// expected to follow this with the standard Integrate +
// ResolveTilemapCollision pair so gravity, drag, and platform-
// collision all run through the existing pipeline. StepBarrelTick
// only writes the HORIZONTAL velocity component — the substrate's
// gravity handles vertical motion (so a barrel that rolls off the
// edge of a platform simply keeps its X-speed and falls).
//
// After the integration + collision pass, ObserveBarrelCollision
// inspects the resolved Velocity.X: if the X-velocity was zeroed by
// a wall collision, the direction flips. This pair (StepBarrelTick
// before, ObserveBarrelCollision after) is the canonical loop.
//
// Returns the updated state. Caller passes the new state back into
// the next tick.
func StepBarrelTick(body *Body, state BarrelState) BarrelState {
	if body == nil {
		return state
	}
	if state.Direction == BarrelStill {
		state.Direction = BarrelRight
	}
	sign := Fixed32(state.Direction)
	body.Velocity.X = state.Speed.Mul(FixedFromInt(int(sign)))
	return state
}

// ObserveBarrelCollision is the post-resolve half of the barrel
// step. After Integrate + ResolveTilemapCollision have run, the
// barrel's X-velocity is zero only if a horizontal collision was
// resolved this tick. In that case, flip the barrel's direction so
// the next StepBarrelTick rolls the other way (bounce off the wall).
//
// If the barrel is mid-fall (vy != 0) the X check is still
// meaningful — a barrel that walks into a wall while airborne also
// reverses, which keeps the bounce-pattern intuitive for designers
// authoring DK-style barrel cascades.
//
// Returns the updated state.
func ObserveBarrelCollision(body *Body, state BarrelState) BarrelState {
	if body == nil {
		return state
	}
	if body.Velocity.X == FixedZero && state.Direction != BarrelStill {
		// Wall stop — flip direction.
		switch state.Direction {
		case BarrelRight:
			state.Direction = BarrelLeft
		case BarrelLeft:
			state.Direction = BarrelRight
		}
	}
	return state
}
