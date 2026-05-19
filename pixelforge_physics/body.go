package pixelforge_physics

import "github.com/solarlune/resolv"

// Body is a moveable physics actor — a player, an enemy, a projectile. It
// owns its Fixed32 Position/Velocity (the canonical state for determinism)
// and a paired *resolv.ConvexPolygon for broad-phase collision queries against
// the World.
//
// The resolv.ConvexPolygon's float64 X/Y are kept in sync with Position by
// Body.Sync() — call this whenever Position changes outside the integrator,
// and the integrator does it automatically on every step.
type Body struct {
	// ID is a stable string identifier. Used for snapshot serialization and
	// entity-cross-reference in higher-level packages.
	ID string

	// Position is the canonical (Fixed32, deterministic) position of the
	// body's top-left corner.
	Position Vec2

	// Velocity is in units per tick. Integrators apply Velocity to Position
	// each step.
	Velocity Vec2

	// Rotation is in Fixed32 radians. The renderer reads this; physics math
	// rarely touches it (rotated colliders are out-of-scope for U6).
	Rotation Fixed32

	// Width, Height are the AABB dimensions of the body in Fixed32 units.
	Width, Height Fixed32

	// Grounded is set by the platformer integrator (and tile collision
	// resolver) when the body's bottom edge is touching a solid tile. Reset
	// each tick.
	Grounded bool

	// OnLadder is set by the ladder state machine when the body's center
	// overlaps a ladder tile. Reset each tick.
	OnLadder bool

	// LadderClimbing is the player-intent state (separate from OnLadder):
	// the player has pressed up/down WHILE on a ladder.
	LadderClimbing bool

	// Static is true if the body is immovable (a tile, a wall). Static
	// bodies are skipped by the integrator.
	Static bool

	// Tags is an optional bitfield for filtering — bit 0 = solid, bit 1 =
	// one-way platform, bit 2 = ladder, bit 3 = hazard. Higher-level code
	// may add semantics.
	Tags uint32

	// shape is the resolv.ConvexPolygon wrapping this body for broad-phase
	// queries. Created by NewBody.
	shape *resolv.ConvexPolygon
}

// Tag bits used by the substrate. Higher-level packages may add more.
const (
	TagSolid    uint32 = 1 << 0
	TagOneWay   uint32 = 1 << 1
	TagLadder   uint32 = 1 << 2
	TagHazard   uint32 = 1 << 3
	TagPlatform uint32 = 1 << 4
)

// NewBody creates a Body with an attached resolv.ConvexPolygon. The body is NOT
// automatically added to a World — caller must World.Add(body).
func NewBody(id string, pos Vec2, width, height Fixed32) *Body {
	b := &Body{
		ID:       id,
		Position: pos,
		Width:    width,
		Height:   height,
	}
	b.shape = resolv.NewRectangle(pos.X.Float(), pos.Y.Float(), width.Float(), height.Float())
	return b
}

// Shape returns the underlying resolv.ConvexPolygon for collision queries.
func (b *Body) Shape() *resolv.ConvexPolygon { return b.shape }

// Sync writes Position to the resolv shape so the next collision query sees
// the latest state. Integrators call this automatically.
func (b *Body) Sync() {
	if b.shape == nil {
		return
	}
	b.shape.SetPositionVec(resolv.NewVector(b.Position.X.Float(), b.Position.Y.Float()))
}

// HasTag reports whether the body has the given tag bit set.
func (b *Body) HasTag(tag uint32) bool { return b.Tags&tag != 0 }
