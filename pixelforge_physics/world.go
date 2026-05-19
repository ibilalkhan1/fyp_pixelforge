package pixelforge_physics

import "github.com/solarlune/resolv"

// CollisionMode strings — string-typed so reference-game presets can be set
// from JSON without a custom unmarshaler.
const (
	CollisionModeTileAABB = "tile-aabb"
	CollisionModeGridAABB = "grid-aabb"
	CollisionModeNone     = "none"
)

// PhysicsConfig is the per-cart physics preset, embedded in Project so each
// game can pick its substrate behaviour without runtime branching all over.
type PhysicsConfig struct {
	// Gravity is added to every dynamic body's Velocity each tick (in
	// per-tick units already). For a 60 TPS target with real-world units of
	// 980 px/s², store FixedFromFloat(980.0/60.0) here OR set the raw value
	// and divide in Integrate — we choose the latter so the config reads
	// naturally as "980".
	Gravity Vec2

	// Drag is a multiplicative velocity damping (per tick). 0 = none, 1.0 =
	// full stop. Stored as Fixed32 so the integrator stays in fixed-point.
	Drag Fixed32

	// ScreenWrap enables modular position wrap-around at the screen edges
	// (Asteroids).
	ScreenWrap bool

	// LadderAware enables the ladder state machine (DK).
	LadderAware bool

	// CollisionMode picks the collision recipe — tile-AABB for platformers,
	// grid-AABB for bomberman, none for free-flying. Other values are
	// treated as "none".
	CollisionMode string

	// ScreenWidth / ScreenHeight are the screen bounds in pixels — used by
	// ScreenWrap and by the broad-phase grid sizing.
	ScreenWidth, ScreenHeight int

	// TPS is the target ticks-per-second (used by Integrate to convert
	// Gravity from per-second to per-tick units). Default 60.
	TPS int
}

// World is the physics-tick context. It owns the resolv.Space (broad-phase)
// and the PhysicsConfig (per-cart preset). Bodies registered with Add are
// integrated each Step() and queryable for collisions.
type World struct {
	*resolv.Space
	cfg    PhysicsConfig
	bodies []*Body
}

// NewWorld constructs a World from a PhysicsConfig. The resolv.Space is
// sized to the screen bounds with 16×16 cells (matching the standard
// tile size for tile-based games; works fine for Asteroids too).
func NewWorld(cfg PhysicsConfig) *World {
	if cfg.TPS == 0 {
		cfg.TPS = 60
	}
	if cfg.ScreenWidth == 0 {
		cfg.ScreenWidth = 320
	}
	if cfg.ScreenHeight == 0 {
		cfg.ScreenHeight = 240
	}
	cellW, cellH := 16, 16
	space := resolv.NewSpace(cfg.ScreenWidth, cfg.ScreenHeight, cellW, cellH)
	return &World{Space: space, cfg: cfg}
}

// Config returns the World's PhysicsConfig (read-only — callers should not
// mutate it mid-run).
func (w *World) Config() PhysicsConfig { return w.cfg }

// AddBody registers a body with the world. Its shape is added to the resolv
// Space and tracked for integration.
func (w *World) AddBody(b *Body) {
	w.bodies = append(w.bodies, b)
	if b.Shape() != nil {
		w.Space.Add(b.Shape())
	}
}

// RemoveBody unregisters a body. Both the resolv shape and the tracking
// slice are updated.
func (w *World) RemoveBody(b *Body) {
	if b.Shape() != nil {
		w.Space.Remove(b.Shape())
	}
	for i, x := range w.bodies {
		if x == b {
			w.bodies = append(w.bodies[:i], w.bodies[i+1:]...)
			return
		}
	}
}

// Bodies returns the slice of bodies in registration order. The slice
// itself is owned by the world — callers should not mutate it.
func (w *World) Bodies() []*Body { return w.bodies }

// AsteroidsConfig returns the preset for Asteroids: no gravity, no drag,
// screen-wrap on, no ladders, no tile collision.
func AsteroidsConfig() PhysicsConfig {
	return PhysicsConfig{
		Gravity:       Vec2{},
		Drag:          FixedZero,
		ScreenWrap:    true,
		LadderAware:   false,
		CollisionMode: CollisionModeNone,
		ScreenWidth:   320,
		ScreenHeight:  240,
		TPS:           60,
	}
}

// MarioConfig returns the preset for a Mario-style platformer: gravity 980
// px/s² downward, tile-AABB collision, no screen wrap, no ladders.
func MarioConfig() PhysicsConfig {
	return PhysicsConfig{
		Gravity:       Vec2{FixedZero, FixedFromFloat(980)},
		Drag:          FixedZero,
		ScreenWrap:    false,
		LadderAware:   false,
		CollisionMode: CollisionModeTileAABB,
		ScreenWidth:   320,
		ScreenHeight:  240,
		TPS:           60,
	}
}

// BombermanConfig returns the preset for a Bomberman-style grid game: no
// gravity, grid-AABB collision (only cardinal axis-aligned moves between
// tile centers), no screen wrap, no ladders.
func BombermanConfig() PhysicsConfig {
	return PhysicsConfig{
		Gravity:       Vec2{},
		Drag:          FixedZero,
		ScreenWrap:    false,
		LadderAware:   false,
		CollisionMode: CollisionModeGridAABB,
		ScreenWidth:   320,
		ScreenHeight:  240,
		TPS:           60,
	}
}

// DKConfig returns the preset for a Donkey-Kong-style platformer: gravity
// like Mario plus ladder awareness.
func DKConfig() PhysicsConfig {
	cfg := MarioConfig()
	cfg.LadderAware = true
	return cfg
}
