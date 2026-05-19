package pixelforge_physics

// Fixed32 is a signed 16.16 fixed-point number. The raw value 1<<16
// represents 1.0. Mul/Div use int64 intermediates so they cannot overflow
// across the representable range.
//
// All physics math in this package goes through Fixed32 to guarantee
// bit-identical behaviour across CPUs — math.Float* calls do not give that
// guarantee.
type Fixed32 int32

// FixedShift is the number of fractional bits.
const FixedShift = 16

// FixedOne is the Fixed32 value representing 1.0.
const FixedOne Fixed32 = 1 << FixedShift

// FixedHalf is the Fixed32 value representing 0.5.
const FixedHalf Fixed32 = 1 << (FixedShift - 1)

// FixedZero is the Fixed32 value representing 0.
const FixedZero Fixed32 = 0

// FixedFromInt converts an integer to Fixed32.
func FixedFromInt(i int) Fixed32 {
	return Fixed32(i) << FixedShift
}

// FixedFromFloat converts a float64 to Fixed32. Truncation, not rounding —
// matching the float-to-int conversion semantics of Go's spec, which is
// well-defined across CPUs.
func FixedFromFloat(f float64) Fixed32 {
	return Fixed32(f * float64(int64(1)<<FixedShift))
}

// Float converts a Fixed32 to a float64. Loss-less because float64 has 52
// fraction bits and Fixed32 has at most 32 significant bits.
func (x Fixed32) Float() float64 {
	return float64(x) / float64(int64(1)<<FixedShift)
}

// Int returns the integer part of x (truncating toward zero).
func (x Fixed32) Int() int {
	return int(x >> FixedShift)
}

// Add returns a + b.
func (a Fixed32) Add(b Fixed32) Fixed32 {
	return a + b
}

// Sub returns a - b.
func (a Fixed32) Sub(b Fixed32) Fixed32 {
	return a - b
}

// Mul returns a * b. Uses an int64 intermediate to avoid overflow on the
// shift-down.
func (a Fixed32) Mul(b Fixed32) Fixed32 {
	return Fixed32((int64(a) * int64(b)) >> FixedShift)
}

// Div returns a / b. Uses an int64 intermediate to preserve precision after
// the up-shift.
func (a Fixed32) Div(b Fixed32) Fixed32 {
	if b == 0 {
		// Conservative behaviour — return saturated max/min instead of
		// panicking. Physics code should never divide by zero, but the
		// substrate must not bring the runtime down if it does.
		if a >= 0 {
			return Fixed32(0x7FFFFFFF)
		}
		return Fixed32(-0x80000000)
	}
	return Fixed32((int64(a) << FixedShift) / int64(b))
}

// Neg returns -x.
func (x Fixed32) Neg() Fixed32 {
	return -x
}

// Abs returns |x|.
func (x Fixed32) Abs() Fixed32 {
	if x < 0 {
		return -x
	}
	return x
}

// Less reports whether a < b.
func (a Fixed32) Less(b Fixed32) bool { return a < b }

// Equal reports whether a == b.
func (a Fixed32) Equal(b Fixed32) bool { return a == b }

// Vec2 is a pair of Fixed32 values used for positions and velocities.
type Vec2 struct {
	X, Y Fixed32
}

// AddV returns a + b component-wise.
func (a Vec2) AddV(b Vec2) Vec2 { return Vec2{a.X.Add(b.X), a.Y.Add(b.Y)} }

// SubV returns a - b component-wise.
func (a Vec2) SubV(b Vec2) Vec2 { return Vec2{a.X.Sub(b.X), a.Y.Sub(b.Y)} }

// ScaleV returns a scaled by s.
func (a Vec2) ScaleV(s Fixed32) Vec2 { return Vec2{a.X.Mul(s), a.Y.Mul(s)} }
