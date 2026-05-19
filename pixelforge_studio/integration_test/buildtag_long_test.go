//go:build long

package integration_test

// isLongBuildTag reports whether the suite is running under
// `-tags=long`. Long-tag builds answer true so default-tag
// scaffold tests can skip when they'd otherwise hit the real
// codegen path.
func isLongBuildTag() bool { return true }
