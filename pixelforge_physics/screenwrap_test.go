package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapPosition_RightEdgeWrapsToOne(t *testing.T) {
	got := WrapPosition(Vec2{FixedFromInt(321), FixedFromInt(50)}, 320, 240)
	assert.Equal(t, FixedFromInt(1), got.X)
	assert.Equal(t, FixedFromInt(50), got.Y)
}

func TestWrapPosition_LeftEdgeWrapsToWidthMinusOne(t *testing.T) {
	got := WrapPosition(Vec2{FixedFromInt(-1), FixedFromInt(50)}, 320, 240)
	assert.Equal(t, FixedFromInt(319), got.X)
}

func TestWrapPosition_InsideUnchanged(t *testing.T) {
	got := WrapPosition(Vec2{FixedFromInt(100), FixedFromInt(100)}, 320, 240)
	assert.Equal(t, FixedFromInt(100), got.X)
	assert.Equal(t, FixedFromInt(100), got.Y)
}

func TestWrapPosition_ExactWidthWrapsToZero(t *testing.T) {
	got := WrapPosition(Vec2{FixedFromInt(320), FixedZero}, 320, 240)
	assert.Equal(t, FixedZero, got.X)
}

func TestWrapBody_GatedByConfig(t *testing.T) {
	// ScreenWrap on — body wraps.
	w1 := NewWorld(PhysicsConfig{ScreenWidth: 320, ScreenHeight: 240, ScreenWrap: true})
	b1 := NewBody("a", Vec2{FixedFromInt(321), FixedFromInt(50)},
		FixedFromInt(8), FixedFromInt(8))
	w1.AddBody(b1)
	wrapped := WrapBody(b1, w1)
	assert.True(t, wrapped)
	assert.Equal(t, FixedFromInt(1), b1.Position.X)

	// ScreenWrap off — body stays put.
	w2 := NewWorld(PhysicsConfig{ScreenWidth: 320, ScreenHeight: 240, ScreenWrap: false})
	b2 := NewBody("a", Vec2{FixedFromInt(321), FixedFromInt(50)},
		FixedFromInt(8), FixedFromInt(8))
	w2.AddBody(b2)
	wrapped = WrapBody(b2, w2)
	assert.False(t, wrapped)
	assert.Equal(t, FixedFromInt(321), b2.Position.X)
}
