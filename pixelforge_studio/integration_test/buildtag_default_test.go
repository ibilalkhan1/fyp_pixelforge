//go:build !long

package integration_test

// isLongBuildTag reports whether the suite is running under
// `-tags=long`. Default-tag builds answer false; long-tag builds
// answer true (see buildtag_long_test.go).
func isLongBuildTag() bool { return false }
