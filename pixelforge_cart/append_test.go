//go:build !js

package pixelforge_cart

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFakePlayer writes some deterministic "binary" bytes to a
// tmp file and returns the path. The bytes don't actually need to
// be a real Mach-O / ELF / PE — Append only copies them verbatim
// and the round-trip test only checks payload extraction, never
// execution. The chmod here mirrors what the real player binary
// would land with on disk (0o755 from go build).
func writeFakePlayer(t *testing.T, dir string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, "fake-player")
	require.NoError(t, os.WriteFile(path, content, 0o755))
	return path
}

func TestAppend_RoundTrip(t *testing.T) {
	if runtime.GOOS == "darwin" {
		// Append on darwin invokes codesign as a final step. If the
		// host lacks codesign we'd misclassify success as failure,
		// and even when it's present the signing step adds time +
		// can fail on hosts with restrictive entitlements. The
		// round-trip semantics are platform-independent so we drive
		// them from the non-darwin paths.
		t.Skip("darwin handled separately via codesign-aware tests")
	}
	dir := t.TempDir()
	playerBytes := bytes.Repeat([]byte{0xAA, 0xBB}, 64)
	playerPath := writeFakePlayer(t, dir, playerBytes)
	cart := []byte("hello cart payload")
	out := filepath.Join(dir, "shipped")

	require.NoError(t, Append(playerPath, cart, out))

	// File now contains: player + cart + footer.
	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, int64(len(playerBytes)+len(cart)+FooterSize), info.Size())

	// Exec bit preserved on Unix.
	if runtime.GOOS != "windows" {
		assert.NotZero(t, info.Mode()&0o100, "exec bit must be set on Unix")
	}

	// Drive ReadSelf via the open-and-read helper rather than
	// re-execing the appended file (the player bytes aren't a real
	// executable on this host).
	f, err := os.Open(out)
	require.NoError(t, err)
	defer f.Close()
	got, err := readFromFile(f)
	require.NoError(t, err)
	assert.Equal(t, cart, got, "ReadSelf must recover the appended payload")
}

func TestAppend_LargePayloadBoundary(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("darwin handled separately via codesign-aware tests")
	}
	dir := t.TempDir()
	playerPath := writeFakePlayer(t, dir, []byte{0x01, 0x02, 0x03})

	// 1 MiB payload — well under MaxPayloadSize, exercises a full
	// pipeline with non-trivial bytes.
	cart := make([]byte, 1<<20)
	for i := range cart {
		cart[i] = byte(i * 7)
	}
	out := filepath.Join(dir, "shipped-big")
	require.NoError(t, Append(playerPath, cart, out))

	f, err := os.Open(out)
	require.NoError(t, err)
	defer f.Close()
	got, err := readFromFile(f)
	require.NoError(t, err)
	assert.Equal(t, cart, got)
}

func TestAppend_PayloadTooLargeRejected(t *testing.T) {
	// We can't allocate a >256 MB slice in a unit test without
	// burning a lot of RAM, so we exercise the boundary check by
	// hand-crafting a slice whose declared length exceeds the cap
	// via a wrapper. Append's check is on len(cart) directly, so we
	// just construct the smallest possible slice that bypasses it
	// — but len() honestly reports its length, so the only way to
	// test the limit is with a real allocation just above the cap.
	// Skip the actual allocation if the host can't afford it; the
	// reader-side oversize test in selfread_test.go also covers
	// the limit semantics without needing the memory.
	if testing.Short() {
		t.Skip("skipping large allocation test in -short")
	}
	dir := t.TempDir()
	playerPath := writeFakePlayer(t, dir, []byte{0x00})

	// Just-over-cap payload.
	cart := make([]byte, MaxPayloadSize+1)
	out := filepath.Join(dir, "shipped-oversize")
	err := Append(playerPath, cart, out)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPayloadTooLarge)

	// Ensure no output file leaked through.
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "Append must not leave output behind on size rejection")
}

func TestAppend_AppBundleResolvesInnerPath(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("darwin path runs codesign on the .app; tested separately")
	}
	dir := t.TempDir()
	playerPath := writeFakePlayer(t, dir, []byte{0xDE, 0xAD, 0xBE, 0xEF})
	cart := []byte("cart")
	outerPath := filepath.Join(dir, "Foo.app")

	require.NoError(t, Append(playerPath, cart, outerPath))

	// The .app outer is a directory; the inner exec is the real
	// target. Verify the resolved inner path exists at the
	// canonical macOS bundle location.
	innerPath := filepath.Join(outerPath, "Contents", "MacOS", "Foo")
	info, err := os.Stat(innerPath)
	require.NoError(t, err, "inner executable must exist at .app/Contents/MacOS/Foo")
	assert.False(t, info.IsDir(), "inner path must be the executable file, not a dir")

	// Round-trip: ReadSelf-equivalent extraction must work on the
	// inner file.
	f, err := os.Open(innerPath)
	require.NoError(t, err)
	defer f.Close()
	got, err := readFromFile(f)
	require.NoError(t, err)
	assert.Equal(t, cart, got)
}

func TestAppend_TmpFileCleanedOnSizeRejection(t *testing.T) {
	dir := t.TempDir()
	playerPath := writeFakePlayer(t, dir, []byte{0x42})
	out := filepath.Join(dir, "shipped")

	// The over-size short-circuit fires before any tmp file is
	// created. Make sure no .tmp leaked into the output dir.
	cart := make([]byte, MaxPayloadSize+1)
	_ = Append(playerPath, cart, out)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotEqual(t, "shipped.tmp", e.Name(), "over-size rejection must not leave a .tmp behind")
	}
}
