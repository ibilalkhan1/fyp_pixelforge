//go:build darwin

package pixelforge_cart

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// adhocSign runs `codesign --force --sign - <path>` to attach an
// ad-hoc signature to the freshly appended binary. The ad-hoc
// identity (the lone "-" sentinel) requires no Apple Developer
// account, no entitlements, and no notarization — it's the
// minimum Apple Silicon will accept to actually launch a binary
// that the loader has modified.
//
// Failure modes:
//   - codesign not in PATH → ErrCodesignUnavailable (caller can
//     still ship unsigned on darwin/amd64).
//   - codesign exits non-zero → ErrCodesignFailed wrapping the
//     captured stderr so build-host operators see the exact
//     complaint.
//
// The `--force` flag tells codesign to overwrite any existing
// signature, which is the relevant case: the player binary was
// signed by the Go toolchain at link time and our append-cart
// step invalidated that signature.
func adhocSign(path string) error {
	binary, err := exec.LookPath("codesign")
	if err != nil {
		// Wrap so callers see the LookPath context but still match
		// the sentinel via errors.Is.
		return fmt.Errorf("%w: %v", ErrCodesignUnavailable, err)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(binary, "--force", "--sign", "-", path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("%w: %s (stderr: %s)", ErrCodesignFailed, exitErr.Error(), bytes.TrimSpace(stderr.Bytes()))
		}
		return fmt.Errorf("%w: %v (stderr: %s)", ErrCodesignFailed, err, bytes.TrimSpace(stderr.Bytes()))
	}
	return nil
}
