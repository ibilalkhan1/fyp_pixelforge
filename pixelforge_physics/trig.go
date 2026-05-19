package pixelforge_physics

import "math"

// sinLUT is a 65536-entry sine lookup, computed at init() from math.Sin.
//
// Index i corresponds to the angle 2π * i / 65536 radians. The stored value
// is the Fixed32 raw representation (16.16) of sin(angle) — i.e. an int32
// scaled by 1<<16.
//
// We compute this at init() rather than embedding a binary blob because the
// Go runtime's math.Sin is invoked at process start (not in the hot loop) so
// FMA emission / libm divergence cannot affect the values consumers see.
// The table is then read deterministically by index for the rest of the
// process lifetime.
//
// Memory cost: 65536 * 4 bytes = 256 KB read-only data.
var sinLUT [65536]int32

func init() {
	const N = 65536
	for i := 0; i < N; i++ {
		angle := 2 * math.Pi * float64(i) / float64(N)
		// Round to nearest before storing — gives ~1 LSB max error vs the
		// platonic ideal, which is good enough for game physics. (Rounding
		// is fine here because it happens once at init on a stable input
		// table, not in the hot loop.)
		sinLUT[i] = int32(math.Floor(math.Sin(angle)*float64(int64(1)<<FixedShift) + 0.5))
	}
}

// indexFromAngle maps a Fixed32 angle (in radians) to a LUT index in
// [0, 65536). The LUT spans 2π radians, so index = (angle / 2π) * 65536.
//
// We compute angle * (65536 / 2π) entirely in integer math:
//
//	scale := 65536 / (2π) ≈ 10430.378...
//	indexRaw := (angle * scale) / FixedOne
//	index    := indexRaw mod 65536
//
// The chosen scale below is precomputed at the comment level — we cannot
// store 10430.378 cleanly in Fixed32 (it overflows mul intermediates at
// large angles), so we just multiply the Fixed32 raw value by a constant
// 10430 (close enough — 1 LSB per full revolution of drift, which the LUT
// resolution already swallows) and shift.
func indexFromAngle(angle Fixed32) uint16 {
	// scale = round(65536^2 / (2π * 65536)) = round(65536 / 2π) ≈ 10430
	// We multiply the raw int32 by 10430 (int64 to be safe) then shift down
	// by FixedShift to undo the fixed-point scaling.
	const scale int64 = 10430 // ≈ 65536 / (2π) to integer precision
	idx := (int64(angle) * scale) >> FixedShift
	return uint16(uint32(idx) & 0xFFFF) // mod 65536
}

// SinFix returns sin(angle) where angle is in Fixed32 radians. Output is in
// Fixed32 ([-1, 1] mapped to [-65536, 65536]).
//
// The function is bit-identical across CPUs because it does table lookup
// only after init — no float ops in the hot path.
func SinFix(angle Fixed32) Fixed32 {
	return Fixed32(sinLUT[indexFromAngle(angle)])
}

// CosFix returns cos(angle) = sin(angle + π/2). We shift the LUT index by
// a quarter of the table size (16384) to avoid recomputing.
func CosFix(angle Fixed32) Fixed32 {
	idx := indexFromAngle(angle)
	return Fixed32(sinLUT[uint16(uint32(idx)+16384)&0xFFFF])
}

// SinDeg256 returns sin(angle) where angle is degrees scaled by 256
// (i.e. 90° = 23040). This is a friendlier API for callers who think in
// degrees — Asteroids' apply_thrust passes degrees directly.
//
// 360° * 256 = 92160, which is the period; we mod by it to fold.
func SinDeg256(degree256 uint16) Fixed32 {
	// 92160 entries cover a full revolution; the LUT has 65536 entries that
	// cover a full revolution. So index = degree256 * 65536 / 92160 =
	// degree256 * 8192 / 11520.
	//
	// Simpler: convert via the Fixed32 path. 1° = π/180 radians.
	// rad = degree256 / 256 * (π/180) = degree256 * π / (256 * 180)
	//
	// We compute the LUT index directly: 1 unit of degree256 corresponds to
	// 65536 / (360 * 256) = 65536 / 92160 of a LUT slot. We use a precomputed
	// integer scale: round(65536 * 256 / 92160) = round(182.044) = 182.
	//
	// idx = (degree256 * 182) >> 8 — but that drifts by ~0.025% per step.
	// Better: use 64-bit multiply by 65536/92160 expressed as 65536*N/92160.
	//
	// scale = 65536 / 92160 ≈ 0.71111, so index = degree256 * 65536 / 92160.
	idx := (uint32(degree256) * 65536) / 92160
	return Fixed32(sinLUT[idx&0xFFFF])
}

// CosDeg256 returns cos(angle) where angle is degrees * 256.
func CosDeg256(degree256 uint16) Fixed32 {
	idx := (uint32(degree256)*65536)/92160 + 16384
	return Fixed32(sinLUT[idx&0xFFFF])
}

// AtanFix returns atan2(y, x) in Fixed32 radians using a deterministic
// integer approximation. The result is in (-π, π], matching math.Atan2.
//
// Implementation: we reduce to the first octant (where |y| <= |x| and both
// are non-negative), use a linear approximation atan(r) ≈ (π/4)·r for
// r ∈ [0, 1], then reflect/rotate into the correct octant. Max error is
// ~0.07 rad (~4°), enough for game logic like aiming arrows or computing
// surface normals — we never use atan2 for tight collision math.
//
// We deliberately avoid math.Atan2 to keep the function bit-identical across
// CPUs.
func AtanFix(y, x Fixed32) Fixed32 {
	const piFixed = Fixed32(205887)  // round(π * 65536)
	const piOver2 = Fixed32(102944)  // round(π/2 * 65536)
	const piOver4 = Fixed32(51472)   // round(π/4 * 65536)

	if x == 0 && y == 0 {
		return 0
	}
	if x == 0 {
		if y > 0 {
			return piOver2
		}
		return -piOver2
	}

	absY := y.Abs()
	absX := x.Abs()

	// First-octant angle in [0, π/4].
	var base Fixed32
	if absX >= absY {
		// r = |y|/|x| ∈ [0, 1]; base = (π/4) * r
		r := absY.Div(absX)
		base = piOver4.Mul(r)
	} else {
		// r = |x|/|y| ∈ [0, 1); first-octant complement: base = π/2 - (π/4) * r
		r := absX.Div(absY)
		base = piOver2.Sub(piOver4.Mul(r))
	}

	// Now place into the correct quadrant.
	switch {
	case x > 0 && y >= 0:
		return base
	case x < 0 && y >= 0:
		return piFixed.Sub(base)
	case x < 0 && y < 0:
		return piFixed.Sub(base).Neg()
	default: // x > 0, y < 0
		return base.Neg()
	}
}
