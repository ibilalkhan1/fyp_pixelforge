// Package pixelforge_physics is the deterministic physics substrate for
// Pixelforge. It wraps github.com/solarlune/resolv (v0.8.1, MIT) for broad-phase
// SAT collision detection and adds a thin layer of features the engine needs:
// tile-grid AABB collision, gravity integrator, one-way platforms, ladder
// state-machine, screen-wrap, and a LUT-based deterministic trig API.
//
// # Substrate choice — resolv
//
// resolv is pure-Go, WASM-safe, MIT-licensed, and ships generic SAT with a
// broad-phase grid. It is well-tested in the Ebitengine community and has no
// CGo dependencies. We accept its float64 internals as a calculated trade —
// see the determinism notes below — and pin to v0.8.1 to keep behaviour stable.
//
// # Thin-layer rationale
//
// resolv is a collision library, not a physics engine. It has no integrator,
// no gravity, no "grounded" concept, no one-way platforms, no screen-wrap,
// and no ladder semantics. This package fills those gaps without bringing in
// a heavyweight engine (Box2D-Go, Chipmunk-Go), which would expand the WASM
// payload and add CGo where we cannot afford it.
//
// # Recipes
//
// One-way platform: register the platform shape normally with the World's
// Space; when the platformer integrator resolves an MTV against it, run the
// resolved vector through IsOneWayHit — if the player is moving downward
// (vel.Y >= 0) and the MTV pushes upward (mtv.Y < 0), apply the resolution;
// otherwise let the body pass through. See oneway.go.
//
// Slope (right-triangle): represent the slope as a resolv.ConvexPolygon with
// the three corners. The standard MTV resolution will push the body along
// the slope normal, which produces the expected "walk up the slope" effect.
// The one-way variant of a slope (passes through from below) reuses the
// IsOneWayHit gate.
//
// # Determinism strategy (LOAD-BEARING)
//
// Pixelforge ships pixel-hash CI gates for reference games (U11, U14, U16,
// U18). The hash strategy depends on physics determinism. We therefore:
//
//   - Route ALL game-side math through Fixed32 (16.16 fixed point) — never
//     touch math.Sin/Cos/Atan2 directly. Use trig.go's LUT.
//   - Generate input frames identically per replay seed.
//   - Avoid goroutines, time.Now(), or unseeded math/rand inside this package.
//   - When we call into resolv (which uses float64 internally including
//     math.Sqrt for SAT), we treat its MTV output as a transient float and
//     immediately truncate it back to Fixed32 (NOT round — truncation is
//     deterministic across rounding modes; rounding is implementation-defined
//     on some platforms). This bounds cross-CPU divergence to at most one
//     Fixed32 LSB (1/65536 of a unit) per resolution step.
//
// # Cross-CPU determinism status — PROBE PENDING
//
// The same-CPU determinism contract is proven: TestDeterminism_SameCPU_ByteEqual
// asserts ten identical 1000-tick simulations produce byte-equal Fixed32
// positions on a single process. This is sufficient for the within-runner
// invariants U11+ rely on.
//
// The cross-CPU contract is NOT yet proven. resolv's float64 SAT plus a
// possible FMA emission on arm64 means byte-equal positions across
// linux/amd64 and darwin/arm64 are theoretically at risk. We commit the
// determinism probe framework (testdata/cross_cpu_baseline.dat) so U25's CI
// matrix can flip the switch:
//
//   - If TestDeterminism_CrossCPU_ProbeFramework passes on both arms, we
//     keep byte-equal as the contract.
//   - If it fails, U6's Execution-note fallback applies: (a) downgrade to
//     tolerance-bucket comparison (a per-Fixed32 LSB tolerance); or (b)
//     restrict pixel-hash CI to a single CPU leg.
//
// The decision lands in U25; this unit just gates it.
//
// # Fixed32 16.16 arithmetic
//
// type Fixed32 int32 is a signed 32-bit fixed-point number with 16 fractional
// bits. The raw value 1<<16 represents 1.0. The representable range is
// approximately ±32768.0 with a resolution of 1/65536 ≈ 0.0000153. This
// covers the full screen-coordinate space at sub-pixel precision; we never
// need values outside it. Mul/Div use int64 intermediates to avoid overflow.
//
// # LUT trig rationale
//
// math.Sin and friends are not guaranteed bit-identical across CPUs (the Go
// runtime calls into libm or platform intrinsics that may emit FMA). A
// 65536-entry sin LUT keyed by a Fixed32 angle gives us bit-identical
// trigonometry everywhere, at the cost of ~256 KB of read-only data baked
// into the binary at init().
package pixelforge_physics
