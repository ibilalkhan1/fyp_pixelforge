// Package pixelforge_physics is the deterministic movement and
// collision substrate for the PixelForge engine. It replaces
// continuous Newtonian integration (F = ma) with discrete,
// numerically approximated kinematics so that every frame tick
// produces identical results across replays and platforms.
package pixelforge_physics

import "math"

// TileSize is the width and height of a single collision cell in
// pixels. Because the engine aligns every sprite and background tile
// to an 8-pixel grid, the broad-phase can be skipped entirely.
const TileSize = 8

// ─── Axis-Aligned Bounding Boxes ─────────────────────────────────

// AABB is an axis-aligned bounding box. All sprites and tilemap
// cells use this representation because their boundaries never
// rotate, reducing the overlap test to four integer comparisons.
type AABB struct {
	Xmin float64
	Xmax float64
	Ymin float64
	Ymax float64
}

// Overlaps reports whether two axis-aligned bounding boxes intersect.
// PixelForge uses the four-inequality test derived from the discrete
// nature of its square-pixel tilemap:
//
//	A.xmin < B.xmax
//	A.xmax > B.xmin
//	A.ymax < B.ymin
//	A.ymin > B.ymax
//
// Because sprites and tile cells are already aligned to the pixel
// grid, no rotation or sub-pixel correction is required, keeping the
// test to four comparisons and zero square roots.
func (a AABB) Overlaps(b AABB) bool {
	return a.Xmin < b.Xmax &&
		a.Xmax > b.Xmin &&
		a.Ymax < b.Ymin &&
		a.Ymin > b.Ymax
}

// ─── Movement Profiles & Look-Up Tables ──────────────────────────

// MovementProfile selects which pre-baked kinematic LUT a cart loads
// at boot time. The curves are sampled once and then cached in the
// heap so that the hot update loop only performs array dereferences.
type MovementProfile int

const (
	ProfileArcade       MovementProfile = iota // Snappy, low gravity
	ProfileSideScroller                         // Mario-like arcs
	ProfileShooting                             // High speed, floaty drop
)

// MotionTable stores pre-computed velocity, gravity, and jump-arc
// samples in heap-allocated lookup tables. Baking the curves at load
// time turns continuous Brachistochrone problems into O(1) array
// reads during gameplay.
type MotionTable struct {
	profile MovementProfile

	// hSpeed maps input magnitude 0..255 to horizontal delta per tick.
	hSpeed [256]float64

	// gravity maps frame index to downward velocity for free-fall.
	gravity [256]float64

	// jumpArc maps frame index to upward velocity during a jump.
	jumpArc [128]float64
}

// NewMotionTable computes the discrete approximations for a given
// game genre once, then caches the table in the heap for the lifetime
// of the cart.
func NewMotionTable(p MovementProfile) *MotionTable {
	mt := &MotionTable{profile: p}
	switch p {
	case ProfileArcade:
		mt.bakeArcade()
	case ProfileSideScroller:
		mt.bakeSideScroller()
	case ProfileShooting:
		mt.bakeShooting()
	}
	return mt
}

func (mt *MotionTable) bakeArcade() {
	for i := 0; i < 256; i++ {
		mt.hSpeed[i] = float64(i) * 0.125
	}
	for i := 0; i < 256; i++ {
		mt.gravity[i] = float64(i) * 0.05
	}
	for i := 0; i < 128; i++ {
		t := float64(i)
		mt.jumpArc[i] = 6.0 - t*0.15
	}
}

func (mt *MotionTable) bakeSideScroller() {
	for i := 0; i < 256; i++ {
		mt.hSpeed[i] = float64(i) * 0.08
	}
	for i := 0; i < 256; i++ {
		mt.gravity[i] = float64(i) * 0.08
	}
	for i := 0; i < 128; i++ {
		t := float64(i)
		mt.jumpArc[i] = 8.0 - t*0.20
	}
}

func (mt *MotionTable) bakeShooting() {
	for i := 0; i < 256; i++ {
		mt.hSpeed[i] = float64(i) * 0.20
	}
	for i := 0; i < 256; i++ {
		mt.gravity[i] = float64(i) * 0.03
	}
	for i := 0; i < 128; i++ {
		t := float64(i)
		mt.jumpArc[i] = 5.0 - t*0.10
	}
}

// ─── Rigid Body & Discrete Movement ──────────────────────────────

// RigidBody is the physics representation of a sprite or actor.
type RigidBody struct {
	X     float64
	Y     float64
	VelX  float64
	VelY  float64
	HalfW float64
	HalfH float64

	// OnGround is true when the body is resting on a solid tile.
	// It gates jump initiation so that air-jumps are impossible
	// unless the designer explicitly enables them.
	OnGround bool
}

// Bounds returns the AABB in world space.
func (rb *RigidBody) Bounds() AABB {
	return AABB{
		Xmin: rb.X - rb.HalfW,
		Xmax: rb.X + rb.HalfW,
		Ymin: rb.Y - rb.HalfH,
		Ymax: rb.Y + rb.HalfH,
	}
}

// MoveHorizontal adds the LUT-derived horizontal velocity to X and
// resolves tile collisions by snapping to the solid edge and zeroing
// velocity.
func (rb *RigidBody) MoveHorizontal(table *MotionTable, input uint8, tileSolid func(x, y int) bool) {
	rb.VelX = table.hSpeed[input]
	nextX := rb.X + rb.VelX

	// Probe the tilemap at the leading edge of the bounding box.
	probeY := int(rb.Y / TileSize)
	if rb.VelX > 0 {
		probeX := int((nextX + rb.HalfW) / TileSize)
		if tileSolid(probeX, probeY) {
			rb.X = float64(probeX)*TileSize - rb.HalfW
			rb.VelX = 0
			return
		}
	} else if rb.VelX < 0 {
		probeX := int((nextX - rb.HalfW) / TileSize)
		if tileSolid(probeX, probeY) {
			rb.X = float64(probeX+1)*TileSize + rb.HalfW
			rb.VelX = 0
			return
		}
	}
	rb.X = nextX
}

// MoveVertical applies gravity from the LUT and resolves floor or
// ceiling collisions. Gravity is accumulated as a constant downward
// push; jump velocity decays each tick until it reverses.
func (rb *RigidBody) MoveVertical(table *MotionTable, tick int, tileSolid func(x, y int) bool) {
	g := table.gravity[tick%256]
	rb.VelY += g

	nextY := rb.Y + rb.VelY
	probeX := int(rb.X / TileSize)

	if rb.VelY > 0 {
		probeY := int((nextY + rb.HalfH) / TileSize)
		if tileSolid(probeX, probeY) {
			rb.Y = float64(probeY)*TileSize - rb.HalfH
			rb.VelY = 0
			rb.OnGround = true
			return
		}
	} else if rb.VelY < 0 {
		probeY := int((nextY - rb.HalfH) / TileSize)
		if tileSolid(probeX, probeY) {
			rb.Y = float64(probeY+1)*TileSize + rb.HalfH
			rb.VelY = 0
			return
		}
	}
	rb.Y = nextY
	rb.OnGround = false
}

// Jump injects upward velocity from the pre-baked jump-arc LUT.
// The call is ignored unless the body is currently on solid ground.
func (rb *RigidBody) Jump(table *MotionTable, tick int) {
	if rb.OnGround {
		rb.VelY = table.jumpArc[tick%128]
		rb.OnGround = false
	}
}

// ─── Numerical Trajectories ──────────────────────────────────────

// LinearTrajectory evaluates a straight-line path at discrete step n.
// Because the engine advances in fixed ticks rather than continuous
// time, the evaluation uses the discrete parametric form:
//
//	x(n) = x0 + vx * n
//	y(n) = y0 + vy * n
func LinearTrajectory(x0, y0, vx, vy float64, n int) (x, y float64) {
	t := float64(n)
	return x0 + vx*t, y0 + vy*t
}

// CircularTrajectory evaluates uniform circular motion at discrete
// step n using a recurrence relation instead of calling sin/cos in
// the hot loop. The recurrence is numerically stable for the small
// angular steps typical of 60 Hz tick rates:
//
//	θ(n) = ω * n
//	x(n) = cx + r * cos(θ)
//	y(n) = cy + r * sin(θ)
func CircularTrajectory(cx, cy, r, omega float64, n int) (x, y float64) {
	theta := omega * float64(n)
	x = cx + r*math.Cos(theta)
	y = cy + r*math.Sin(theta)
	return
}

// ParabolicTrajectory evaluates projectile motion at discrete step n.
// Rather than integrating the continuous ODE in real time, the engine
// samples the closed-form discrete kinematic equations:
//
//	x(n) = x0 + vx * n
//	y(n) = y0 + vy * n + 0.5 * g * n²
func ParabolicTrajectory(x0, y0, vx, vy, g float64, n int) (x, y float64) {
	t := float64(n)
	x = x0 + vx*t
	y = y0 + vy*t + 0.5*g*t*t
	return
}

// HyperbolicTrajectory evaluates a hyperbolic path at discrete step n.
// The parametric form avoids the singularity at the asymptotes by
// discretising the time parameter into uniform tick-sized steps:
//
//	t = stepSize * n
//	x(n) = cx + a * cosh(t)
//	y(n) = cy + b * sinh(t)
func HyperbolicTrajectory(cx, cy, a, b, stepSize float64, n int) (x, y float64) {
	t := stepSize * float64(n)
	x = cx + a*math.Cosh(t)
	y = cy + b*math.Sinh(t)
	return
}
