//go:build !js

package pixelforge_cart

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
)

// ReadSelf reads the cart payload appended to the running
// executable. The function opens the executable once and holds a
// single *os.File for every read — the descriptor is inode-bound
// after open, so a path-level swap mid-read cannot redirect the
// reader to a different file.
//
// Resolution order:
//
//   - On Linux, /proc/self/exe is opened directly. The kernel
//     ensures /proc/self/exe is a magic symlink to the actual
//     running process image, immune to filesystem-side races.
//   - Elsewhere, os.Executable() + filepath.EvalSymlinks resolves
//     the executable path; we then os.Open the resolved real path.
//
// Errors:
//   - ErrNoCart            magic absent (unshipped player)
//   - ErrVersionMismatch   magic present but wrong version
//   - ErrPayloadTooLarge   declared length > MaxPayloadSize
//     (returned BEFORE any allocation of the payload buffer)
//   - ErrCorruptCart       short file, truncated payload, CRC fail
func ReadSelf() ([]byte, error) {
	path, err := resolveSelfPath()
	if err != nil {
		return nil, fmt.Errorf("pixelforge_cart: resolve self: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("pixelforge_cart: open self %s: %w", path, err)
	}
	defer f.Close()
	return readFromFile(f)
}

// resolveSelfPath returns a stable filesystem path for the running
// executable. Split out so tests can verify the Linux branch picks
// /proc/self/exe without invoking the full ReadSelf flow.
func resolveSelfPath() (string, error) {
	if runtime.GOOS == "linux" {
		// /proc/self/exe is a magic symlink the kernel guarantees
		// resolves to the running image — no userland EvalSymlinks
		// race to worry about.
		return "/proc/self/exe", nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		// EvalSymlinks failure is non-fatal — fall back to the raw
		// path. On macOS the executable is sometimes already the
		// real file with no symlink in front of it.
		return exe, nil
	}
	return resolved, nil
}

// readFromFile is the file-handle-bound read path. Exposed inside
// the package so selfread_test.go can drive it against a
// synthesized test binary without spawning a subprocess.
func readFromFile(f *os.File) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("pixelforge_cart: stat self: %w", err)
	}
	size := info.Size()
	if size < int64(FooterSize) {
		// File too small to even contain a footer — definitely not
		// a shipped player.
		return nil, ErrNoCart
	}

	footer := make([]byte, FooterSize)
	if _, err := f.ReadAt(footer, size-int64(FooterSize)); err != nil {
		return nil, fmt.Errorf("pixelforge_cart: read footer: %w", err)
	}

	length, crc, err := DecodeFooter(footer)
	if err != nil {
		// ErrNoCart / ErrVersionMismatch / ErrCorruptCart all
		// propagate as-is; DecodeFooter has already classified
		// the failure mode for us.
		return nil, err
	}

	// Enforce the 256 MB cap BEFORE allocating — the whole point
	// of the limit is to refuse hostile footers without first
	// honoring their malicious length claim. ErrPayloadTooLarge
	// is wrapped with the observed length so build-host operators
	// can diagnose corruption.
	if length > MaxPayloadSize {
		return nil, fmt.Errorf("%w: declared length %d, limit %d", ErrPayloadTooLarge, length, MaxPayloadSize)
	}

	// Truncated file: declared length doesn't fit between BOF and
	// the footer. Surface as corrupt rather than try to read past
	// the start of the file.
	payloadStart := size - int64(FooterSize) - int64(length)
	if payloadStart < 0 {
		return nil, fmt.Errorf("%w: declared length %d exceeds available bytes", ErrCorruptCart, length)
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := f.ReadAt(payload, payloadStart); err != nil {
			return nil, fmt.Errorf("%w: read payload: %v", ErrCorruptCart, err)
		}
	}

	if crc32.ChecksumIEEE(payload) != crc {
		return nil, fmt.Errorf("%w: CRC32 mismatch", ErrCorruptCart)
	}
	return payload, nil
}
