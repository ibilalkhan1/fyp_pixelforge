package pixelforge_rand

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeed_DeterministicSequence(t *testing.T) {
	Seed(42)
	first := make([]float64, 10)
	for i := range first {
		first[i] = Float64()
	}

	Seed(42)
	second := make([]float64, 10)
	for i := range second {
		second[i] = Float64()
	}

	assert.Equal(t, first, second, "same seed should produce same sequence")
}

func TestSeed_CurrentSeedRoundTrip(t *testing.T) {
	Seed(12345)
	assert.Equal(t, uint64(12345), CurrentSeed())
}

func TestIntN_InRange(t *testing.T) {
	Seed(7)
	for i := 0; i < 100; i++ {
		v := IntN(100)
		assert.GreaterOrEqual(t, v, 0)
		assert.Less(t, v, 100)
	}
}

func TestIntN_ZeroPanics(t *testing.T) {
	Seed(1)
	assert.Panics(t, func() { IntN(0) })
}

func TestSeed_ResetsSequence(t *testing.T) {
	Seed(99)
	a := Float64()
	_ = Float64()
	_ = Float64()
	Seed(99)
	b := Float64()
	assert.Equal(t, a, b, "reseeding restarts the sequence")
}

func TestSource_NotNil(t *testing.T) {
	Seed(1)
	require.NotNil(t, Source())
}

func TestShuffle_DeterministicGivenSeed(t *testing.T) {
	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	Seed(2024)
	Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })

	Seed(2024)
	Shuffle(len(b), func(i, j int) { b[i], b[j] = b[j], b[i] })

	assert.Equal(t, a, b, "shuffle with same seed produces same permutation")
}

func TestPerm_DeterministicGivenSeed(t *testing.T) {
	Seed(33)
	a := Perm(20)
	Seed(33)
	b := Perm(20)
	assert.Equal(t, a, b)
}

func TestUint64_DeterministicGivenSeed(t *testing.T) {
	Seed(8)
	a := Uint64()
	Seed(8)
	b := Uint64()
	assert.Equal(t, a, b)
}

func TestUint32_DeterministicGivenSeed(t *testing.T) {
	Seed(9)
	a := Uint32()
	Seed(9)
	b := Uint32()
	assert.Equal(t, a, b)
}
