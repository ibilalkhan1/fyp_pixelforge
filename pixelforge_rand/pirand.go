// Package pixelforge_rand provides a seedable, deterministic random
// source the engine and the studio share.
//
// The package is a thin wrapper around math/rand/v2 with one extra
// guarantee on top of the stdlib: a single, process-global *rand.Rand
// that can be reseeded mid-process. Seeding the source produces
// a deterministic stream, which is exactly what the regression
// replayer in pixelforge_studio/capture needs to drive recorded
// inputs against a known-good seed.
//
// Games that never call Seed inherit today's behaviour — the package
// auto-seeds from time.Now().UnixNano() during init.
package pixelforge_rand

import (
	"math/rand/v2"
	"sync"
	"time"
)

var (
	mu      sync.RWMutex
	current *rand.Rand
	seed    uint64
)

func init() {
	Seed(uint64(time.Now().UnixNano()))
}

// Seed reseeds the package-global random source.
//
// The mixing factor (0x9E3779B97F4A7C15, the 64-bit golden ratio) is
// added to seed in the second PCG slot so callers can pass a single
// "logical" seed yet still drive both PCG state words to distinct
// values, which is what math/rand/v2.NewPCG expects.
func Seed(s uint64) {
	mu.Lock()
	defer mu.Unlock()
	seed = s
	current = rand.New(rand.NewPCG(s, s^0x9E3779B97F4A7C15))
}

// CurrentSeed returns the seed last passed to Seed.
//
// The capture recorder logs this on EventInit so a captured session
// naturally carries its seed; regression replay reads it back via
// the seed.txt sidecar.
func CurrentSeed() uint64 {
	mu.RLock()
	defer mu.RUnlock()
	return seed
}

// Source returns the active *rand.Rand for callers that need to
// interop with stdlib functions accepting a *rand.Rand.
func Source() *rand.Rand {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Float64 returns a pseudo-random number in [0.0, 1.0).
func Float64() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return current.Float64()
}

// IntN returns a non-negative pseudo-random integer in [0, n).
// IntN panics if n <= 0 (matches math/rand/v2.IntN behaviour).
func IntN(n int) int {
	mu.RLock()
	defer mu.RUnlock()
	return current.IntN(n)
}

// Int returns a non-negative pseudo-random int.
func Int() int {
	mu.RLock()
	defer mu.RUnlock()
	return current.Int()
}

// Uint64 returns a pseudo-random uint64.
func Uint64() uint64 {
	mu.RLock()
	defer mu.RUnlock()
	return current.Uint64()
}

// Uint32 returns a pseudo-random uint32.
func Uint32() uint32 {
	mu.RLock()
	defer mu.RUnlock()
	return current.Uint32()
}

// Shuffle pseudo-randomizes the order of n elements via swap(i, j).
func Shuffle(n int, swap func(i, j int)) {
	mu.RLock()
	defer mu.RUnlock()
	current.Shuffle(n, swap)
}

// NormFloat64 returns a normally distributed float64 with mean 0 and
// standard deviation 1.
func NormFloat64() float64 {
	mu.RLock()
	defer mu.RUnlock()
	return current.NormFloat64()
}

// Perm returns a pseudo-random permutation of the integers [0, n).
func Perm(n int) []int {
	mu.RLock()
	defer mu.RUnlock()
	return current.Perm(n)
}
