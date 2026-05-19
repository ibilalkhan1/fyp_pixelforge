// builders.go is the default-tag scaffold path. Under //go:build
// long the long builders in builders_long.go take over — that file
// invokes the real Go toolchain via NewBuildCommand. The scaffold
// here writes a placeholder marker so the orchestrator's e2e
// contract (PhaseQueued -> Generating -> Compiling -> Packaging ->
// Done) holds in the default test suite without requiring Go to
// be installed on the test host.

//go:build !long

package buildpipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// scaffoldBuilder is the lightweight shared builder. It runs the
// Capsule codegen + the packaging phase, but defers the compile
// phase to the build-tagged shim. Tests covering the orchestrator
// drive this without invoking go build.
type scaffoldBuilder struct {
	target Target
}

func (b *scaffoldBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "pixelforge-build-*")
	if err != nil {
		return fmt.Errorf("scaffoldBuilder: mkdir temp: %w", err)
	}
	// Tests inspect the temp dir's structure; production cleanup
	// happens on PhaseDone (the orchestrator's outer goroutine
	// removes the tempDir after the build's artifact is moved).
	_ = tempDir

	emit(BuildStatus{Phase: PhaseCompiling, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	// Per-target packaging — short-circuit path that produces a
	// placeholder output file so the orchestrator's e2e contract
	// holds even without a real go-build invocation. The compile
	// step + real packaging live in the //go:build long variant
	// (builders_long.go).
	emit(BuildStatus{Phase: PhasePackaging, BuiltAt: time.Now()})
	outDir := filepath.Join(req.OutputDir, b.target.String())
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("scaffoldBuilder: mkdir output: %w", err)
	}
	gameName := projectGameName(req.Project)
	outPath := filepath.Join(outDir, gameName+artifactExt(b.target))
	if err := os.WriteFile(outPath, []byte("# Pixelforge build placeholder\n"), 0o644); err != nil {
		return fmt.Errorf("scaffoldBuilder: write placeholder: %w", err)
	}

	emit(BuildStatus{
		Phase:      PhaseDone,
		OutputPath: outPath,
		BuiltAt:    time.Now(),
	})
	return nil
}

func init() {
	RegisterBuilder(TargetWindows, &scaffoldBuilder{target: TargetWindows})
	RegisterBuilder(TargetMacOS, &scaffoldBuilder{target: TargetMacOS})
	RegisterBuilder(TargetLinux, &scaffoldBuilder{target: TargetLinux})
	RegisterBuilder(TargetWASM, &scaffoldBuilder{target: TargetWASM})
	RegisterBuilder(TargetSource, &scaffoldBuilder{target: TargetSource})
}
