package pixelforge_physics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrate_GravityAccumulatesOverOneSecond(t *testing.T) {
	// Mario-style config: 980 px/s² gravity. After 60 ticks at 1/60 dt,
	// velocity.Y should be approximately 980 (within accumulated rounding).
	w := NewWorld(MarioConfig())
	b := NewBody("hero", Vec2{}, FixedFromInt(16), FixedFromInt(16))
	w.AddBody(b)
	dt := FixedFromFloat(1.0 / 60.0)
	for i := 0; i < 60; i++ {
		Integrate(b, w, dt)
	}
	gotV := b.Velocity.Y.Float()
	assert.InDelta(t, 980.0, gotV, 5.0, "vy after 1s should ≈ 980")
}

func TestIntegrate_VelocityUpdatesPosition(t *testing.T) {
	w := NewWorld(PhysicsConfig{})
	b := NewBody("p", Vec2{FixedFromInt(0), FixedFromInt(0)},
		FixedFromInt(8), FixedFromInt(8))
	b.Velocity = Vec2{FixedFromInt(2), FixedFromInt(0)} // 2 units / tick
	w.AddBody(b)

	for i := 0; i < 5; i++ {
		Integrate(b, w, FixedFromFloat(1.0/60.0))
	}
	// 5 ticks of +2 = +10
	assert.Equal(t, FixedFromInt(10), b.Position.X)
}

func TestIntegrate_StaticBodySkipped(t *testing.T) {
	w := NewWorld(MarioConfig())
	b := NewBody("wall", Vec2{}, FixedFromInt(16), FixedFromInt(16))
	b.Static = true
	w.AddBody(b)
	for i := 0; i < 100; i++ {
		Integrate(b, w, FixedFromFloat(1.0/60.0))
	}
	assert.Equal(t, FixedZero, b.Velocity.Y)
	assert.Equal(t, FixedZero, b.Position.Y)
}

func TestIntegrate_DragDampensVelocity(t *testing.T) {
	cfg := PhysicsConfig{
		Drag: FixedFromFloat(0.5), // 50% per tick — aggressive but illustrative
	}
	w := NewWorld(cfg)
	b := NewBody("p", Vec2{}, FixedFromInt(1), FixedFromInt(1))
	b.Velocity = Vec2{FixedFromInt(100), FixedZero}
	w.AddBody(b)
	Integrate(b, w, FixedFromFloat(1.0/60.0))
	// After one tick at 50% drag, velocity should be ~50.
	assert.InDelta(t, 50.0, b.Velocity.X.Float(), 1.0)
}

func TestApplyJump_SetsUpwardVelocity(t *testing.T) {
	b := NewBody("p", Vec2{}, FixedFromInt(8), FixedFromInt(8))
	b.Grounded = true
	ApplyJump(b, FixedFromFloat(200))
	assert.Equal(t, FixedFromFloat(-200), b.Velocity.Y)
	assert.False(t, b.Grounded)
}

func TestApplyImpulse_Accumulates(t *testing.T) {
	b := NewBody("p", Vec2{}, FixedFromInt(8), FixedFromInt(8))
	ApplyImpulse(b, FixedFromInt(3), FixedFromInt(4))
	ApplyImpulse(b, FixedFromInt(1), FixedFromInt(-2))
	assert.Equal(t, FixedFromInt(4), b.Velocity.X)
	assert.Equal(t, FixedFromInt(2), b.Velocity.Y)
}

func TestIntegrate_DeterministicAcrossRuns(t *testing.T) {
	// Two parallel Worlds with identical initial state must produce
	// identical body positions over N ticks.
	mk := func() (*World, *Body) {
		w := NewWorld(MarioConfig())
		b := NewBody("hero", Vec2{FixedFromInt(50), FixedFromInt(0)},
			FixedFromInt(16), FixedFromInt(16))
		w.AddBody(b)
		return w, b
	}
	w1, b1 := mk()
	w2, b2 := mk()
	dt := FixedFromFloat(1.0 / 60.0)
	for i := 0; i < 1000; i++ {
		Integrate(b1, w1, dt)
		Integrate(b2, w2, dt)
	}
	require.Equal(t, b1.Position, b2.Position, "same inputs must produce same positions")
	require.Equal(t, b1.Velocity, b2.Velocity)
}
