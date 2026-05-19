//go:build !js

package pixelforge_cart

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSelfFile assembles a fake "shipped binary" on disk:
// prefix bytes + payload + footer. Used to drive readFromFile
// without invoking the full Append flow (which would also chmod
// and codesign on darwin). Tests pass the file handle directly so
// we exercise the same code path ReadSelf takes.
func writeSelfFile(t *testing.T, prefix, payload []byte, footer []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "self")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(prefix)
	require.NoError(t, err)
	_, err = f.Write(payload)
	require.NoError(t, err)
	_, err = f.Write(footer)
	require.NoError(t, err)
	return path
}

func TestReadFromFile_HappyPath(t *testing.T) {
	payload := []byte("hello world cart")
	prefix := []byte("PLAYER-BINARY-BYTES-")
	path := writeSelfFile(t, prefix, payload, EncodeFooter(payload))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	got, err := readFromFile(f)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestReadFromFile_EmptyPayload(t *testing.T) {
	// 0-byte payload is a legitimate edge case (an "empty cart"
	// shipped player should still decode cleanly).
	payload := []byte{}
	path := writeSelfFile(t, []byte("PLAYER"), payload, EncodeFooter(payload))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	got, err := readFromFile(f)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestReadFromFile_NoFooter(t *testing.T) {
	// A bare "player" with no footer at all — fresh build that
	// nobody has shipped through Append yet. ReadSelf must distinguish
	// this from a corrupt cart so the player main can exit cleanly.
	dir := t.TempDir()
	path := filepath.Join(dir, "bare")
	require.NoError(t, os.WriteFile(path, []byte("just-a-bare-player-binary-with-no-footer-bytes"), 0o644))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCart)
}

func TestReadFromFile_FileSmallerThanFooter(t *testing.T) {
	// File shorter than FooterSize bytes — definitely not shipped.
	// Should surface as ErrNoCart (the file is too small to even
	// contain a footer; it can't possibly be a v2 cart).
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCart)
}

func TestReadFromFile_CRCMismatch(t *testing.T) {
	// Footer claims a CRC that doesn't match the payload — the
	// reader must catch this and return ErrCorruptCart.
	payload := []byte("the original payload bytes")
	footer := EncodeFooter(payload)
	// Flip the first byte of the CRC field so it no longer matches.
	footer[0] ^= 0xFF
	path := writeSelfFile(t, []byte("PLAYER"), payload, footer)

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCorruptCart)
}

func TestReadFromFile_TruncatedPayload(t *testing.T) {
	// Hand-craft a file whose footer claims length=1000 but the file
	// itself is only big enough for ~10 bytes of payload. The reader
	// must reject this without reading past the start of the file.
	dir := t.TempDir()
	path := filepath.Join(dir, "trunc")

	footer := make([]byte, FooterSize)
	binary.LittleEndian.PutUint32(footer[0:4], 0xDEADBEEF)
	binary.LittleEndian.PutUint64(footer[4:12], 1000) // huge declared length
	copy(footer[12:20], MagicV2)

	// File is just a small prefix + the footer — no room for 1000
	// payload bytes.
	prefix := []byte("xx")
	body := append(prefix, footer...)
	require.NoError(t, os.WriteFile(path, body, 0o644))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCorruptCart)
}

func TestReadFromFile_OversizeRejectedBeforeAllocation(t *testing.T) {
	// Synthesize a small file whose footer LIES — claiming a 300 MiB
	// payload. Critically, the file itself is tiny, so any allocation
	// attempt would either over-allocate (bad) or fail the subsequent
	// ReadAt with a separate error. The cap must short-circuit the
	// allocation entirely and return ErrPayloadTooLarge.
	dir := t.TempDir()
	path := filepath.Join(dir, "lies")

	footer := make([]byte, FooterSize)
	binary.LittleEndian.PutUint32(footer[0:4], 0xDEADBEEF)
	binary.LittleEndian.PutUint64(footer[4:12], 300<<20) // > 256 MiB cap
	copy(footer[12:20], MagicV2)
	require.NoError(t, os.WriteFile(path, footer, 0o644))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPayloadTooLarge, "300MB declared length must short-circuit on the cap, not OOM")
}

func TestReadFromFile_VersionMismatch(t *testing.T) {
	// Footer with PFORGEv1 magic — caller must see a precise
	// "version not supported" error rather than a CRC failure.
	payload := []byte("hi")
	footer := make([]byte, FooterSize)
	binary.LittleEndian.PutUint32(footer[0:4], crc32.ChecksumIEEE(payload))
	binary.LittleEndian.PutUint64(footer[4:12], uint64(len(payload)))
	copy(footer[12:20], "PFORGEv1")
	path := writeSelfFile(t, []byte("PLAYER"), payload, footer)

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = readFromFile(f)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVersionMismatch)
	assert.Contains(t, err.Error(), "PFORGEv1")
}

func TestResolveSelfPath_LinuxUsesProcSelfExe(t *testing.T) {
	// On Linux the implementation prefers /proc/self/exe — verify
	// this without exec'ing a subprocess. Other OSes get the
	// EvalSymlinks branch and aren't asserted-on here.
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only branch")
	}
	path, err := resolveSelfPath()
	require.NoError(t, err)
	assert.Equal(t, "/proc/self/exe", path)
}

func TestResolveSelfPath_NonLinuxResolvesSymlinks(t *testing.T) {
	// On non-Linux platforms resolveSelfPath goes through
	// os.Executable + filepath.EvalSymlinks. The returned path
	// must point at an existing file (the test binary itself).
	if runtime.GOOS == "linux" {
		t.Skip("non-Linux-only branch")
	}
	path, err := resolveSelfPath()
	require.NoError(t, err)
	_, err = os.Stat(path)
	assert.NoError(t, err, "resolved self path must exist")
}
