package pixelforge_cart

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Append copies the player binary at playerBinaryPath, appends the
// cart payload plus a v2 footer, and writes the result atomically
// to outputPath. On Unix the result is chmod 0o755 so it's
// launchable. On darwin the function additionally runs
// `codesign --force --sign - <outputPath>` as the final step.
//
// Atomicity: the data lands at <outputPath>.tmp first, then we
// os.Rename onto outputPath. A crash mid-write leaves a stale .tmp
// the caller can clean up; the real target is never half-written.
//
// .app bundles: if outputPath ends in ".app", the function writes
// into "<outputPath>/Contents/MacOS/<base-without-.app>" — the
// macOS bundle's actual executable lives there, not at the .app
// directory itself. The parent dirs are created lazily.
//
// Payload bound: payloads larger than MaxPayloadSize are rejected
// with ErrPayloadTooLarge (the same cap the reader enforces) so
// the writer can't produce a binary the reader would refuse.
//
// Codesign failure modes (darwin only):
//   - codesign missing from PATH → returns ErrCodesignUnavailable.
//     The binary is still on disk and runnable on darwin/amd64;
//     darwin/arm64 callers must decide whether to ship unsigned.
//   - codesign exits non-zero → returns ErrCodesignFailed wrapping
//     the captured stderr.
//
// In both codesign failure modes the appended binary remains in
// place — the caller can inspect or re-sign manually.
func Append(playerBinaryPath string, cart []byte, outputPath string) error {
	if uint64(len(cart)) > MaxPayloadSize {
		return fmt.Errorf("%w: cart is %d bytes, limit %d", ErrPayloadTooLarge, len(cart), MaxPayloadSize)
	}

	// .app bundle redirect: outer ".../Foo.app" → inner
	// ".../Foo.app/Contents/MacOS/Foo". The Mach-O executable
	// inside the bundle is what we modify; the bundle wrapper is
	// just a directory.
	resolvedOutput := outputPath
	if strings.HasSuffix(outputPath, ".app") {
		base := strings.TrimSuffix(filepath.Base(outputPath), ".app")
		resolvedOutput = filepath.Join(outputPath, "Contents", "MacOS", base)
		if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0o755); err != nil {
			return fmt.Errorf("pixelforge_cart: mkdir bundle MacOS dir: %w", err)
		}
	}

	playerBytes, err := os.ReadFile(playerBinaryPath)
	if err != nil {
		return fmt.Errorf("pixelforge_cart: read player binary %s: %w", playerBinaryPath, err)
	}

	footer := EncodeFooter(cart)

	// Atomic write via tmp + rename, mirroring the asset-library
	// downloader's pattern. The .tmp suffix avoids colliding with
	// any concurrent reader of the final path.
	tmpPath := resolvedOutput + ".tmp"
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("pixelforge_cart: create tmp %s: %w", tmpPath, err)
	}
	writeErr := func() error {
		if _, err := tmp.Write(playerBytes); err != nil {
			return fmt.Errorf("write player bytes: %w", err)
		}
		if _, err := tmp.Write(cart); err != nil {
			return fmt.Errorf("write cart bytes: %w", err)
		}
		if _, err := tmp.Write(footer); err != nil {
			return fmt.Errorf("write footer: %w", err)
		}
		return nil
	}()
	if closeErr := tmp.Close(); closeErr != nil && writeErr == nil {
		writeErr = fmt.Errorf("close tmp: %w", closeErr)
	}
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("pixelforge_cart: %w", writeErr)
	}

	// Preserve the executable bit on Unix. Windows ignores chmod
	// — runnability there is driven by the .exe extension which the
	// build host appends to outputPath upstream.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0o755); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("pixelforge_cart: chmod 0755 %s: %w", tmpPath, err)
		}
	}

	if err := os.Rename(tmpPath, resolvedOutput); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("pixelforge_cart: rename %s -> %s: %w", tmpPath, resolvedOutput, err)
	}

	// Final step on darwin: ad-hoc sign so Apple Silicon will
	// actually launch the modified binary. Returned as-is so the
	// caller can distinguish the missing-codesign case (still
	// shippable in some configurations) from a hard signing
	// failure.
	if runtime.GOOS == "darwin" {
		// On a .app bundle the signing target is the outer
		// directory (codesign descends into Contents/MacOS itself);
		// otherwise we sign the bare executable.
		signTarget := resolvedOutput
		if strings.HasSuffix(outputPath, ".app") {
			signTarget = outputPath
		}
		if err := adhocSign(signTarget); err != nil {
			return err
		}
	}
	return nil
}
