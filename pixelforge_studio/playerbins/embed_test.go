package playerbins

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerBinaryFor_FixturePresent(t *testing.T) {
	// bins/test-fake/pixelforge-player is committed alongside this
	// test file as a stable fixture so embed lookup is exercised
	// regardless of whether `make playerbins` has populated real
	// per-OS binaries.
	got, err := PlayerBinaryFor("test", "fake")
	require.NoError(t, err)
	assert.NotEmpty(t, got, "fixture must return non-empty bytes")
	assert.Contains(t, string(got), "PIXELFORGE-PLAYER-TEST-FIXTURE",
		"fixture content sanity check")
}

func TestPlayerBinaryFor_UnknownReturnsNotEmbedded(t *testing.T) {
	_, err := PlayerBinaryFor("plan9", "mips64")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotEmbedded),
		"unknown OS/arch should soft-miss via ErrNotEmbedded so the build pipeline falls through, got: %v", err)
}

func TestPlayerBinaryFor_EmptyArgsRejected(t *testing.T) {
	_, err := PlayerBinaryFor("", "amd64")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotEmbedded),
		"empty arg is a programmer error, not a soft miss")

	_, err = PlayerBinaryFor("linux", "")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotEmbedded),
		"empty arg is a programmer error, not a soft miss")
}

func TestPlayerBinaryFilename_WindowsSuffix(t *testing.T) {
	assert.Equal(t, "pixelforge-player.exe", playerBinaryFilename("windows"))
}

func TestPlayerBinaryFilename_WasmSuffix(t *testing.T) {
	assert.Equal(t, "pixelforge-player.wasm", playerBinaryFilename("js"))
}

func TestPlayerBinaryFilename_DefaultBare(t *testing.T) {
	assert.Equal(t, "pixelforge-player", playerBinaryFilename("linux"))
	assert.Equal(t, "pixelforge-player", playerBinaryFilename("darwin"))
}
