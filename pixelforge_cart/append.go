package pixelforge_cart

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Append ships a player and cart. For a .app output, it copies the executable
// unchanged and stores the cart envelope in Contents/Resources/game.pfcart.
// Call SignAppBundle after writing the bundle's metadata and other resources.
// Appending bytes to a Mach-O executable would violate macOS signing rules.
// Other outputs retain the executable + cart + v2-footer layout.
// Writes use a temporary file and rename. Oversize carts are rejected first.
func Append(playerBinaryPath string, cart []byte, outputPath string) error {
	if uint64(len(cart)) > MaxPayloadSize {
		return fmt.Errorf("%w: cart is %d bytes, limit %d", ErrPayloadTooLarge, len(cart), MaxPayloadSize)
	}

	if strings.HasSuffix(outputPath, ".app") {
		return writeAppBundle(playerBinaryPath, cart, outputPath)
	}
	resolvedOutput := outputPath

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
