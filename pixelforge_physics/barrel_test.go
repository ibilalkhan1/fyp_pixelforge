package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBarrel_StepSetsHorizontalVelocity asserts that StepBarrelTick
// writes the requested horizontal velocity (preserving the existing
// vertical component, which the integrator owns).
func TestBarrel_StepSetsHorizontalVelocity(t *testing.T) {
	b := NewBody("barrel", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	// Existing vertical velocity should NOT be touched by Step.
	b.Velocity.Y = FixedFromInt(3)
	state := NewBarrelState(BarrelRight, FixedFromInt(2))

	state = StepBarrelTick(b, state)
	assert.Equal(t, FixedFromInt(2), b.Velocity.X, "rightward step should set vx=+speed")
	assert.Equal(t, FixedFromInt(3), b.Velocity.Y, "vy must not be touched by barrel step")
	assert.Equal(t, BarrelRight, state.Direction)
}

func TestBarrel_StepLeftwardSetsNegativeX(t *testing.T) {
	b := NewBody("barrel", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	state := NewBarrelState(BarrelLeft, FixedFromInt(2))

	state = StepBarrelTick(b, state)
	assert.Equal(t, FixedFromInt(-2), b.Velocity.X, "leftward step should set vx=-speed")
}

// TestBarrel_ObserveCollisionFlipsOnWallStop covers the "barrel rolls
// into a solid wall" case: ResolveTilemapCollision zeroed Velocity.X,
// so the observer detects the stop and flips direction.
func TestBarrel_ObserveCollisionFlipsOnWallStop(t *testing.T) {
	b := NewBody("barrel", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	// Simulate post-resolve state: X velocity zeroed by a wall hit.
	b.Velocity.X = FixedZero
	state := BarrelState{Direction: BarrelRight, Speed: FixedFromInt(2)}

	state = ObserveBarrelCollision(b, state)
	assert.Equal(t, BarrelLeft, state.Direction, "wall stop should flip direction")
}

// TestBarrel_ObserveCollisionNoFlipWhenMoving covers the "barrel
// rolled freely this tick" case: Velocity.X is non-zero so the
// observer leaves the direction alone.
func TestBarrel_ObserveCollisionNoFlipWhenMoving(t *testing.T) {
	b := NewBody("barrel", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity.X = FixedFromInt(2) // still rolling — no wall hit
	state := BarrelState{Direction: BarrelRight, Speed: FixedFromInt(2)}

	state = ObserveBarrelCollision(b, state)
	assert.Equal(t, BarrelRight, state.Direction, "freely-rolling barrel must not flip")
}

// TestBarrel_RollAlongPlatformWithGravity is the integration test:
// a barrel sits on a platform with one solid wall at the right edge.
// After N ticks of (StepBarrelTick + Integrate + Resolve +
// ObserveBarrelCollision) the barrel should have moved rightward,
// hit the wall, and reversed direction.
func TestBarrel_RollAlongPlatformWithGravity(t *testing.T) {
	cfg := DKConfig()
	world := NewWorld(cfg)
	dt := FixedOne.Div(FixedFromInt(60))

	b := NewBody("barrel", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	// Position on a platform with floor at row=1 (tile-top=16).
	b.Position = Vec2{FixedFromInt(0), FixedFromInt(8)}
	world.AddBody(b)

	// Build a fake grid: floor (row=1) full of solid, wall at (col=3, row=0).
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0, 0, 1, 0}, // row 0: a wall at col=3
			{1, 1, 1, 1, 1}, // row 1: floor
		},
	}

	state := NewBarrelState(BarrelRight, FixedFromInt(2))

	for i := 0; i < 10; i++ {
		state = StepBarrelTick(b, state)
		Integrate(b, world, dt)
		ResolveTilemapCollision(b, world, grid, []int{1}, 16, 16)
		state = ObserveBarrelCollision(b, state)
	}

	// After enough ticks the barrel must have bounced off the right
	// wall — we don't pin the exact tick count, but the final direction
	// should be Left.
	assert.Equal(t, BarrelLeft, state.Direction, "barrel should have bounced off the right wall")
}

// TestBarrel_FallsOffPlatformEdge covers "no tile beyond the edge":
// the integrator's gravity pulls the barrel down without any wall
// hit, so the direction stays unchanged.
func TestBarrel_FallsOffPlatformEdge(t *testing.T) {
	cfg := DKConfig()
	world := NewWorld(cfg)
	dt := FixedOne.Div(FixedFromInt(60))

	b := NewBody("barrel", Vec2{FixedFromInt(64), FixedFromInt(8)},
		FixedFromInt(8), FixedFromInt(8))
	world.AddBody(b)

	// Floor ends at col=3 — beyond that the barrel falls.
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0, 0, 0, 0}, // row 0: empty
			{1, 1, 1, 1, 0}, // row 1: floor ends at col 3
		},
	}

	state := NewBarrelState(BarrelRight, FixedFromInt(2))
	startY := b.Position.Y

	for i := 0; i < 30; i++ {
		state = StepBarrelTick(b, state)
		Integrate(b, world, dt)
		ResolveTilemapCollision(b, world, grid, []int{1}, 16, 16)
		state = ObserveBarrelCollision(b, state)
	}

	require.Greater(t, b.Position.Y, startY, "barrel should have fallen below its starting Y")
	assert.Equal(t, BarrelRight, state.Direction, "free-falling barrel direction must not flip")
}

// rowMajorGrid is the test-side TileGrid for barrel + ladder tests.
type rowMajorGrid struct {
	cells [][]int
}

func (g *rowMajorGrid) GridWidth() int {
	if len(g.cells) == 0 {
		return 0
	}
	return len(g.cells[0])
}
func (g *rowMajorGrid) GridHeight() int { return len(g.cells) }
func (g *rowMajorGrid) TileID(col, row int) int {
	if row < 0 || row >= len(g.cells) {
		return 0
	}
	r := g.cells[row]
	if col < 0 || col >= len(r) {
		return 0
	}
	return r[col]
}

// TestLadder_BodyOverlapsLadderTile_True covers the canonical "hero
// is standing in a ladder column" case.
func TestLadder_BodyOverlapsLadderTile_True(t *testing.T) {
	// 16-px tiles. Body at (16, 16) with footprint 8×8 sits inside
	// tile-cell (1, 1). Paint a ladder there.
	b := NewBody("hero", Vec2{FixedFromInt(16), FixedFromInt(16)},
		FixedFromInt(8), FixedFromInt(8))
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0, 0},
			{0, 2, 0}, // ladder at (1, 1)
			{0, 0, 0},
		},
	}
	assert.True(t, BodyOverlapsLadderTile(b, grid, 2, 16, 16))
}

func TestLadder_BodyOverlapsLadderTile_False(t *testing.T) {
	b := NewBody("hero", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0},
			{0, 2}, // ladder at (1, 1) — body at (0,0) doesn't reach
		},
	}
	assert.False(t, BodyOverlapsLadderTile(b, grid, 2, 16, 16))
}

// TestLadder_TopReached_BodyClimbsAboveLadder covers the "climber
// reaches the top of the ladder" case: the body's top edge has
// risen above the topmost ladder cell in its column.
func TestLadder_TopReached_BodyClimbsAboveLadder(t *testing.T) {
	// Ladder occupies (1, 2) and (1, 3) — topmost ladder cell is at
	// row=2, top-Y = 32.
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0, 0},
			{0, 0, 0},
			{0, 2, 0},
			{0, 2, 0},
		},
	}
	// Body 8×8 with top edge at Y=24 — that's above 32-px ladder top
	// only when by <= 32. 24 <= 32, so this IS at-top.
	b := NewBody("hero", Vec2{FixedFromInt(16), FixedFromInt(24)},
		FixedFromInt(8), FixedFromInt(8))
	assert.True(t, LadderTopReached(b, grid, 2, 16, 16))
}

func TestLadder_TopReached_BodyStillBelowLadderTop(t *testing.T) {
	// Same ladder layout — climber still 1 tile below the top.
	grid := &rowMajorGrid{
		cells: [][]int{
			{0, 0, 0},
			{0, 0, 0},
			{0, 2, 0},
			{0, 2, 0},
		},
	}
	// Top edge at 48 (mid-way through the ladder) — NOT at top.
	b := NewBody("hero", Vec2{FixedFromInt(16), FixedFromInt(48)},
		FixedFromInt(8), FixedFromInt(8))
	assert.False(t, LadderTopReached(b, grid, 2, 16, 16))
}

// TestGravity_LadderClimbingSuppressesGravity is the integration
// test for the U17 gravity-disable contract: while LadderClimbing
// is set the integrator must NOT accumulate gravity into Velocity.Y.
func TestGravity_LadderClimbingSuppressesGravity(t *testing.T) {
	cfg := DKConfig()
	world := NewWorld(cfg)
	dt := FixedOne.Div(FixedFromInt(60))
	b := NewBody("hero", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	b.LadderClimbing = true
	b.Velocity = Vec2{} // start at rest

	for i := 0; i < 60; i++ {
		Integrate(b, world, dt)
	}

	assert.Equal(t, FixedZero, b.Velocity.Y,
		"LadderClimbing should suppress gravity accumulation")
}

// TestGravity_LadderClimbingClearedRestoresGravity asserts that
// once LadderClimbing flips back to false, the very next Integrate
// resumes gravity accumulation.
func TestGravity_LadderClimbingClearedRestoresGravity(t *testing.T) {
	cfg := DKConfig()
	world := NewWorld(cfg)
	dt := FixedOne.Div(FixedFromInt(60))
	b := NewBody("hero", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))

	// First: climb tick — no gravity.
	b.LadderClimbing = true
	Integrate(b, world, dt)
	require.Equal(t, FixedZero, b.Velocity.Y)

	// Release: gravity resumes.
	b.LadderClimbing = false
	Integrate(b, world, dt)
	assert.NotEqual(t, FixedZero, b.Velocity.Y,
		"clearing LadderClimbing must allow gravity to accumulate again")
}
