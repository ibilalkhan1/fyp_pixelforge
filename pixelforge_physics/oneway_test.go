package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOneWayHit_PlayerFalling_HitsPlatform(t *testing.T) {
	// Player moving down (velY >= 0), MTV pushes up (mtvY < 0) — should resolve.
	assert.True(t, IsOneWayHit(FixedFromInt(-4), FixedFromInt(2)))
	assert.True(t, IsOneWayHit(FixedFromInt(-1), FixedZero))
}

func TestIsOneWayHit_PlayerJumping_PassesThrough(t *testing.T) {
	// Player moving up (velY < 0) — should NOT resolve, even if MTV is "up".
	assert.False(t, IsOneWayHit(FixedFromInt(-4), FixedFromInt(-3)))
}

func TestIsOneWayHit_MtvDownward_NeverResolves(t *testing.T) {
	// If the MTV is pushing the body downward, the body is below the platform —
	// never resolve regardless of velocity.
	assert.False(t, IsOneWayHit(FixedFromInt(4), FixedFromInt(2)))
	assert.False(t, IsOneWayHit(FixedFromInt(4), FixedFromInt(-2)))
}

func TestOneWayResolve_AppliesMtvAndGrounds(t *testing.T) {
	b := NewBody("p", Vec2{FixedFromInt(0), FixedFromInt(20)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity = Vec2{FixedZero, FixedFromInt(3)} // falling
	mtv := Vec2{FixedZero, FixedFromInt(-2)}      // platform pushes up 2
	ok := OneWayResolve(b, mtv)
	assert.True(t, ok)
	assert.Equal(t, FixedFromInt(18), b.Position.Y, "y resolved up by 2")
	assert.Equal(t, FixedZero, b.Velocity.Y, "downward vel zeroed")
	assert.True(t, b.Grounded)
}

func TestOneWayResolve_PassesWhenJumpingUp(t *testing.T) {
	b := NewBody("p", Vec2{FixedFromInt(0), FixedFromInt(20)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity = Vec2{FixedZero, FixedFromInt(-3)} // jumping
	mtv := Vec2{FixedZero, FixedFromInt(-2)}
	ok := OneWayResolve(b, mtv)
	assert.False(t, ok)
	assert.Equal(t, FixedFromInt(20), b.Position.Y, "position unchanged")
}
