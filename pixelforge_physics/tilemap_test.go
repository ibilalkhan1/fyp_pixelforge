package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGrid is a tiny in-test TileGrid that satisfies the substrate
// interface without dragging in pixelforge_project.
type fakeGrid struct {
	w, h  int
	cells []int
}

func newFakeGrid(w, h int) *fakeGrid { return &fakeGrid{w: w, h: h, cells: make([]int, w*h)} }
func (g *fakeGrid) GridWidth() int   { return g.w }
func (g *fakeGrid) GridHeight() int  { return g.h }
func (g *fakeGrid) TileID(c, r int) int {
	if c < 0 || c >= g.w || r < 0 || r >= g.h {
		return 0
	}
	return g.cells[r*g.w+c]
}
func (g *fakeGrid) set(c, r, id int) { g.cells[r*g.w+c] = id }

func TestBuildTileGridCollider_EmitsRectsForSolidTiles(t *testing.T) {
	g := newFakeGrid(4, 4)
	// Put a solid tile (ID 1) at (1, 2).
	g.set(1, 2, 1)
	// And another at (3, 3).
	g.set(3, 3, 1)
	polys := BuildTileGridCollider(g, []int{1}, 16, 16)
	require.Len(t, polys, 2)
}

func TestBuildTileGridCollider_EmptyOnNoMatches(t *testing.T) {
	g := newFakeGrid(2, 2)
	polys := BuildTileGridCollider(g, []int{1}, 16, 16)
	assert.Empty(t, polys)
}

func TestBuildTileGridCollider_NilGrid_ReturnsNil(t *testing.T) {
	assert.Nil(t, BuildTileGridCollider(nil, []int{1}, 16, 16))
}

func TestResolveTilemapCollision_FallsOntoTileTop(t *testing.T) {
	// Plan's scenario: body 8x8 at (8, 8), moves down 4px; tile row at y=16
	// solid; body snaps to y=8; grounded=true.
	//
	// We use a no-gravity config so the only motion is the explicit velocity —
	// the gravity behaviour is exercised in gravity_test.go.
	g := newFakeGrid(4, 4)
	g.set(0, 1, 1)
	g.set(1, 1, 1)
	g.set(2, 1, 1)
	g.set(3, 1, 1)
	w := NewWorld(PhysicsConfig{
		ScreenWidth:   64,
		ScreenHeight:  64,
		CollisionMode: CollisionModeTileAABB,
	})
	b := NewBody("hero", Vec2{FixedFromInt(8), FixedFromInt(8)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity = Vec2{FixedZero, FixedFromInt(4)}
	w.AddBody(b)

	// Step one tick of integrate then resolve. After integrate: body at y=12.
	Integrate(b, w, FixedFromFloat(1.0/60.0))
	resolved := ResolveTilemapCollision(b, w, g, []int{1}, 16, 16)
	assert.True(t, resolved, "should have resolved collision")
	assert.True(t, b.Grounded, "body should be grounded on tile")
	// Body bottom should be at y=16 (tile top). Body height=8, so top y=8.
	assert.Equal(t, FixedFromInt(8), b.Position.Y)
}

func TestResolveTilemapCollision_NoCollisionWhenClear(t *testing.T) {
	g := newFakeGrid(4, 4)
	w := NewWorld(MarioConfig())
	b := NewBody("hero", Vec2{FixedFromInt(8), FixedFromInt(8)},
		FixedFromInt(8), FixedFromInt(8))
	w.AddBody(b)
	resolved := ResolveTilemapCollision(b, w, g, []int{1}, 16, 16)
	assert.False(t, resolved)
	assert.False(t, b.Grounded)
}

func TestResolveTilemapCollision_WallBumpStopsX(t *testing.T) {
	// Body moving right at 8 units/tick, wall in its path. No-gravity config.
	g := newFakeGrid(4, 4)
	g.set(2, 0, 1) // wall at x=32..48, y=0..16
	w := NewWorld(PhysicsConfig{
		ScreenWidth:   64,
		ScreenHeight:  64,
		CollisionMode: CollisionModeTileAABB,
	})
	b := NewBody("hero", Vec2{FixedFromInt(28), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity = Vec2{FixedFromInt(8), FixedZero}
	w.AddBody(b)
	Integrate(b, w, FixedFromFloat(1.0/60.0))
	// After integrate: body at x=36 (overlapping tile by 4 in X).
	ResolveTilemapCollision(b, w, g, []int{1}, 16, 16)
	// After resolution, body flush against tile's left edge: x=24 (tile=32, body width 8).
	assert.Equal(t, 24, b.Position.X.Int(), "body should be flush left of wall")
	assert.Equal(t, FixedZero, b.Velocity.X, "x velocity zeroed on collision")
}
