package pixelforge_cart

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AppCartFilename is the cart envelope stored in a macOS app's Resources.
const AppCartFilename = "game.pfcart"

func writeAppBundle(playerPath string, cart []byte, bundlePath string) error {
	name := strings.TrimSuffix(filepath.Base(bundlePath), ".app")
	executable := filepath.Join(bundlePath, "Contents", "MacOS", name)
	resources := filepath.Join(bundlePath, "Contents", "Resources")
	for _, dir := range []string{filepath.Dir(executable), resources} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create app directory: %w", err)
		}
	}
	player, err := os.ReadFile(playerPath)
	if err != nil {
		return fmt.Errorf("read app player: %w", err)
	}
	if err := writeBundleFile(executable, player, 0o755); err != nil {
		return err
	}
	payload := make([]byte, 0, len(cart)+FooterSize)
	payload = append(payload, cart...)
	payload = append(payload, EncodeFooter(cart)...)
	return writeBundleFile(filepath.Join(resources, AppCartFilename), payload, 0o644)
}

func writeBundleFile(path string, data []byte, mode fs.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write app file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename app file: %w", err)
	}
	return nil
}

// SignAppBundle ad-hoc signs a completed macOS bundle. Call only after its
// executable, cart, icon and Info.plist are in place. Other hosts are a no-op.
func SignAppBundle(path string) error { return adhocSign(path) }

func appCartPath(executable string) (string, bool) {
	macos := filepath.Dir(executable)
	contents := filepath.Dir(macos)
	bundle := filepath.Dir(contents)
	if filepath.Base(macos) != "MacOS" || filepath.Base(contents) != "Contents" || !strings.HasSuffix(bundle, ".app") {
		return "", false
	}
	return filepath.Join(contents, "Resources", AppCartFilename), true
}
