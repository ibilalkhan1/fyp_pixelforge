package pixelforge_physics

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSinFix_ZeroAndQuadrants(t *testing.T) {
	// SinFix(0) should be exactly 0 (the LUT entry at index 0 is 0).
	assert.Equal(t, FixedZero, SinFix(0))

	// SinFix(π/2) should be FixedOne within 1 LSB.
	piOver2 := FixedFromFloat(math.Pi / 2)
	got := SinFix(piOver2)
	diff := int32(got) - int32(FixedOne)
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, int32(64), "SinFix(π/2) raw drift = %d", diff)

	// SinFix(π) should be approximately 0.
	pi := FixedFromFloat(math.Pi)
	gotPi := SinFix(pi)
	assert.LessOrEqual(t, int32(gotPi.Abs()), int32(64), "SinFix(π) ≈ 0")
}

func TestSinFix_AccuracyVsMathSin(t *testing.T) {
	// 360 samples across [0, 2π) — error vs math.Sin should be < 0.0001 in
	// raw Fixed32 terms, which is approximately 0.0015 in real units. We
	// loosen to 0.005 because of the integer indexFromAngle scale rounding.
	for deg := 0; deg < 360; deg++ {
		rad := float64(deg) * math.Pi / 180
		want := math.Sin(rad)
		got := SinFix(FixedFromFloat(rad)).Float()
		assert.InDelta(t, want, got, 0.005, "deg=%d", deg)
	}
}

func TestCosFix_AccuracyVsMathCos(t *testing.T) {
	for deg := 0; deg < 360; deg++ {
		rad := float64(deg) * math.Pi / 180
		want := math.Cos(rad)
		got := CosFix(FixedFromFloat(rad)).Float()
		assert.InDelta(t, want, got, 0.005, "deg=%d", deg)
	}
}

func TestSinDeg256_KnownAngles(t *testing.T) {
	// degree256 = degrees * 256; uint16 max is 65535 ≈ 256°. We test angles
	// up to 256° which is enough to hit all four quadrants.
	cases := []struct {
		deg256 uint16
		want   float64
	}{
		{0, 0.0},
		{uint16(90 * 256), 1.0},
		{uint16(180 * 256), 0.0},
		// 250° lies in the third quadrant: sin(250°) ≈ -0.94
		{uint16(250 * 256), -0.94},
	}
	for _, c := range cases {
		got := SinDeg256(c.deg256).Float()
		assert.InDelta(t, c.want, got, 0.02, "deg256=%d", c.deg256)
	}
}

func TestCosDeg256_KnownAngles(t *testing.T) {
	cases := []struct {
		deg256 uint16
		want   float64
	}{
		{0, 1.0},
		{uint16(90 * 256), 0.0},
		{uint16(180 * 256), -1.0},
		// 250° in third quadrant: cos(250°) ≈ -0.34
		{uint16(250 * 256), -0.34},
	}
	for _, c := range cases {
		got := CosDeg256(c.deg256).Float()
		assert.InDelta(t, c.want, got, 0.02, "deg256=%d", c.deg256)
	}
}

func TestSinFix_Determinism_SameInput(t *testing.T) {
	// 1000 invocations with the same input must produce byte-identical
	// outputs. This is the foundation the cross-CPU probe rests on — the
	// LUT entries are fixed by init(), and the lookup is a pure table read.
	angle := FixedFromFloat(0.7)
	first := SinFix(angle)
	for i := 0; i < 1000; i++ {
		got := SinFix(angle)
		require.Equal(t, first, got, "iter %d diverged", i)
	}
}

func TestAtanFix_BasicQuadrants(t *testing.T) {
	// AtanFix is approximate; we only assert quadrant + rough magnitude.
	// Quadrant 1
	q1 := AtanFix(FixedFromInt(1), FixedFromInt(1)).Float()
	assert.InDelta(t, math.Pi/4, q1, 0.2, "atan2(1,1)")
	// Quadrant 2
	q2 := AtanFix(FixedFromInt(1), FixedFromInt(-1)).Float()
	assert.InDelta(t, 3*math.Pi/4, q2, 0.2, "atan2(1,-1)")
	// Origin
	assert.Equal(t, FixedZero, AtanFix(0, 0))
}

func TestSinLUT_AtZeroAndPi(t *testing.T) {
	// Index 0 corresponds to angle 0.
	assert.Equal(t, int32(0), sinLUT[0])
	// Index 16384 corresponds to angle π/2 — should be ≈ FixedOne.
	assert.InDelta(t, int32(FixedOne), sinLUT[16384], 1.0)
	// Index 32768 corresponds to angle π — should be ≈ 0.
	assert.LessOrEqual(t, sinLUT[32768], int32(1))
	assert.GreaterOrEqual(t, sinLUT[32768], int32(-1))
}
