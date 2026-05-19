package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorld_DefaultsAreReasonable(t *testing.T) {
	w := NewWorld(PhysicsConfig{})
	require.NotNil(t, w)
	require.NotNil(t, w.Space)
	cfg := w.Config()
	assert.Equal(t, 60, cfg.TPS)
	assert.Equal(t, 320, cfg.ScreenWidth)
	assert.Equal(t, 240, cfg.ScreenHeight)
}

func TestNewWorld_RespectsExplicitConfig(t *testing.T) {
	w := NewWorld(PhysicsConfig{
		ScreenWidth:  640,
		ScreenHeight: 480,
		TPS:          120,
		ScreenWrap:   true,
	})
	cfg := w.Config()
	assert.Equal(t, 640, cfg.ScreenWidth)
	assert.Equal(t, 480, cfg.ScreenHeight)
	assert.Equal(t, 120, cfg.TPS)
	assert.True(t, cfg.ScreenWrap)
}

func TestWorld_AddRemoveBody(t *testing.T) {
	w := NewWorld(PhysicsConfig{})
	b := NewBody("hero", Vec2{}, FixedFromInt(8), FixedFromInt(8))
	w.AddBody(b)
	assert.Len(t, w.Bodies(), 1)

	w.RemoveBody(b)
	assert.Len(t, w.Bodies(), 0)
}

func TestPresets_AsteroidsHasWrapNoGravity(t *testing.T) {
	c := AsteroidsConfig()
	assert.True(t, c.ScreenWrap)
	assert.Equal(t, FixedZero, c.Gravity.X)
	assert.Equal(t, FixedZero, c.Gravity.Y)
	assert.Equal(t, CollisionModeNone, c.CollisionMode)
	assert.False(t, c.LadderAware)
}

func TestPresets_MarioHasGravityNoWrap(t *testing.T) {
	c := MarioConfig()
	assert.False(t, c.ScreenWrap)
	assert.Equal(t, FixedFromFloat(980), c.Gravity.Y)
	assert.Equal(t, CollisionModeTileAABB, c.CollisionMode)
	assert.False(t, c.LadderAware)
}

func TestPresets_BombermanGridAABB(t *testing.T) {
	c := BombermanConfig()
	assert.Equal(t, CollisionModeGridAABB, c.CollisionMode)
	assert.False(t, c.ScreenWrap)
	assert.Equal(t, FixedZero, c.Gravity.Y)
}

func TestPresets_DKHasGravityAndLadders(t *testing.T) {
	c := DKConfig()
	assert.True(t, c.LadderAware)
	assert.Equal(t, FixedFromFloat(980), c.Gravity.Y)
	assert.Equal(t, CollisionModeTileAABB, c.CollisionMode)
}
