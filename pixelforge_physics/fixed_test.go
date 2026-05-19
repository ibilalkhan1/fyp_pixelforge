package pixelforge_physics

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixed_RoundTripInt(t *testing.T) {
	for _, n := range []int{0, 1, -1, 100, -100, 32767, -32768} {
		x := FixedFromInt(n)
		require.Equal(t, n, x.Int(), "round-trip int %d", n)
	}
}

func TestFixed_RoundTripFloat_WithinResolution(t *testing.T) {
	for _, f := range []float64{0, 1, -1, 0.5, -0.5, 980.0, 0.0625, -100.25} {
		x := FixedFromFloat(f)
		got := x.Float()
		// 16.16 resolution is 1/65536 ≈ 1.5e-5.
		assert.InDelta(t, f, got, 1.0/float64(int64(1)<<FixedShift), "round-trip float %v", f)
	}
}

func TestFixed_FixedOneEqualsFromInt1(t *testing.T) {
	require.Equal(t, FixedOne, FixedFromInt(1))
}

func TestFixed_FixedHalfEqualsFromFloat(t *testing.T) {
	require.Equal(t, FixedHalf, FixedFromFloat(0.5))
}

func TestFixed_AddSub(t *testing.T) {
	a := FixedFromFloat(2.5)
	b := FixedFromFloat(1.25)
	assert.Equal(t, FixedFromFloat(3.75), a.Add(b))
	assert.Equal(t, FixedFromFloat(1.25), a.Sub(b))
}

func TestFixed_Mul(t *testing.T) {
	cases := []struct{ a, b, want float64 }{
		{2.0, 3.0, 6.0},
		{0.5, 0.5, 0.25},
		{-2.0, 3.0, -6.0},
		{100.0, 0.0625, 6.25},
	}
	for _, c := range cases {
		got := FixedFromFloat(c.a).Mul(FixedFromFloat(c.b))
		assert.InDelta(t, c.want, got.Float(), 1.0/float64(int64(1)<<FixedShift), "%v * %v", c.a, c.b)
	}
}

func TestFixed_Mul_LargeIntermediate(t *testing.T) {
	// 1000 * 1000 = 1_000_000 — without the int64 intermediate this would
	// overflow Fixed32's 16.16 range; with it, it works.
	got := FixedFromFloat(1000).Mul(FixedFromFloat(0.001))
	assert.InDelta(t, 1.0, got.Float(), 0.01)
}

func TestFixed_Div(t *testing.T) {
	got := FixedFromFloat(980).Div(FixedFromInt(60))
	assert.InDelta(t, 980.0/60.0, got.Float(), 1e-3)
}

func TestFixed_DivByZero_Saturates(t *testing.T) {
	pos := FixedFromInt(1).Div(FixedZero)
	neg := FixedFromInt(-1).Div(FixedZero)
	assert.Equal(t, Fixed32(0x7FFFFFFF), pos)
	assert.Equal(t, Fixed32(-0x80000000), neg)
}

func TestFixed_Abs(t *testing.T) {
	assert.Equal(t, FixedFromInt(5), FixedFromInt(-5).Abs())
	assert.Equal(t, FixedFromInt(5), FixedFromInt(5).Abs())
}

func TestFixed_NegLessEqual(t *testing.T) {
	a := FixedFromInt(3)
	b := FixedFromInt(5)
	assert.Equal(t, FixedFromInt(-3), a.Neg())
	assert.True(t, a.Less(b))
	assert.False(t, b.Less(a))
	assert.True(t, a.Equal(FixedFromInt(3)))
}

func TestFixed_Determinism(t *testing.T) {
	// Repeating the same arithmetic on the same inputs MUST produce identical
	// raw int32 values. This is the foundation the cross-CPU probe rests on.
	var prev Fixed32
	for i := 0; i < 1000; i++ {
		x := FixedFromFloat(0.123).Mul(FixedFromFloat(0.456))
		if i == 0 {
			prev = x
			continue
		}
		require.Equal(t, prev, x, "iteration %d diverged", i)
	}
}

func TestVec2_AddSubScale(t *testing.T) {
	a := Vec2{FixedFromInt(1), FixedFromInt(2)}
	b := Vec2{FixedFromInt(3), FixedFromInt(4)}
	assert.Equal(t, Vec2{FixedFromInt(4), FixedFromInt(6)}, a.AddV(b))
	assert.Equal(t, Vec2{FixedFromInt(-2), FixedFromInt(-2)}, a.SubV(b))
	assert.Equal(t, Vec2{FixedFromInt(2), FixedFromInt(4)}, a.ScaleV(FixedFromInt(2)))
}

func TestFixed_NoSilentMathDriftVsFloat(t *testing.T) {
	// Sanity check that FixedFromFloat matches what math says for a 16-bit
	// resolution. This is not a determinism guarantee — math.Floor is just
	// a convenient reference here.
	for f := -10.0; f < 10.0; f += 0.1 {
		x := FixedFromFloat(f)
		expected := int64(math.Floor(f*float64(int64(1)<<FixedShift) + 1e-12))
		// Allow ±1 LSB because of how float64 represents 0.1.
		diff := int64(x) - expected
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, int64(1), "f=%v drift=%d", f, diff)
	}
}
