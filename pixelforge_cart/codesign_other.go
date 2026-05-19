//go:build !darwin

package pixelforge_cart

// adhocSign is a no-op on non-darwin hosts. Linux + Windows
// runtimes don't enforce a signature requirement on locally
// produced binaries, and the Authenticode signing path Windows
// builds use is handled by an external step BEFORE cart append
// (documented in doc.go).
func adhocSign(path string) error {
	_ = path
	return nil
}
