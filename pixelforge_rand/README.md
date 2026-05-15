# `pixelforge_rand`

A seedable, deterministic random source for the engine and the studio.

The package is a thin wrapper around `math/rand/v2` with one extra
guarantee: a single, process-global `*rand.Rand` that can be reseeded
mid-process. Seeding the source produces a deterministic stream, which
is exactly what the regression replayer in `pixelforge_studio/capture`
needs to drive recorded inputs against a known-good seed.

## When to seed

- **Game code**: usually not. Games inherit a time-seeded source on
  startup, so behaviour matches today's `math/rand` default.
- **Regression replay**: the replayer calls `pirand.Seed(loggedSeed)`
  before driving recorded inputs.
- **Tests**: `pirand.Seed(N)` at the top of a test gives reproducible
  randomness.

## API

```go
pirand.Seed(uint64)        // reset the global source
pirand.CurrentSeed()       // last seed passed to Seed
pirand.Float64()           // [0.0, 1.0)
pirand.IntN(n)             // [0, n); panics if n <= 0
pirand.Int()               // non-negative int
pirand.Uint64()
pirand.Uint32()
pirand.Shuffle(n, swap)
pirand.NormFloat64()
pirand.Perm(n)
pirand.Source()            // *rand.Rand for stdlib interop
```

## v1 vs v2 trade-off

`math/rand/v2` ships a cleaner `Source` interface than v1 and supports
the PCG generator out of the box. Go 1.24 (this repo's minimum) has
v2; there is no reason to keep a foothold in v1.

## Interaction with regression replay

The capture recorder logs `pirand.CurrentSeed()` on `EventInit` and
writes it to `seed.txt` when a frame is promoted to a regression test.
On replay, `pf-studio test` reads `seed.txt` and calls
`pirand.Seed(seed)` before driving the recorded input log. Same
seed + same inputs ⇒ same frames.
