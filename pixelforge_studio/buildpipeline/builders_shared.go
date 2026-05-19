package buildpipeline

import (
	"context"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// builders_shared.go carries the helpers both the //go:build !long
// scaffold path (builders.go) and the //go:build long real-build
// path (builders_long.go) share. Splitting them out of the build-
// tagged files keeps the dependency surface obvious: the helpers
// here are pure data + cancellation plumbing, no toolchain or
// codegen calls.

// artifactExt returns the per-target output filename extension.
func artifactExt(t Target) string {
	switch t {
	case TargetWindows:
		return ".exe"
	case TargetMacOS:
		return ".app"
	case TargetWASM:
		return ".html"
	case TargetLinux:
		return ""
	case TargetSource:
		return ""
	}
	return ""
}

// projectGameName returns the sanitised game name the build
// pipeline uses for output filenames. Empty / nil projects fall
// back to "game".
func projectGameName(p *pixelforge_project.Project) string {
	if p == nil || p.Name == "" {
		return "game"
	}
	return sanitizeForFilename(p.Name)
}

// contextCheck early-returns ErrBuildCancelled when ctx is done.
// Each phase boundary calls this so cancellation surfaces
// promptly rather than after a full step completes.
func contextCheck(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return ErrBuildCancelled
	}
	return nil
}
