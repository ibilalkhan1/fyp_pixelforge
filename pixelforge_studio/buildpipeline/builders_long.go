// builders_long.go is the //go:build long real-build path. Built
// with `-tags=long` the studio (and the long-tag CI job) replaces
// the placeholder scaffolds in builders.go with builders that
// actually shell out to `go build` and produce runnable artifacts.
//
// Production callers — the studio's Build button — apply the tag
// automatically so users don't need to know about it. The default
// test suite stays headless (no Go toolchain dependency) because
// builders.go is `//go:build !long`.

//go:build long

package buildpipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// codegenOptionsFor builds codegen.Options carrying the engine
// override fields the long builders care about. Maps the build-
// pipeline-side ModuleStrategy enum to modulepath.Strategy.
func codegenOptionsFor(req BuildRequest, projectSourcePath string) codegen.Options {
	opts := codegen.Options{
		Force:             true,
		RunGoModTidy:      true,
		ProjectSourcePath: projectSourcePath,
		EnginePath:        req.EnginePath,
		EngineVersion:     req.EngineVersion,
		Credits:           req.Credits,
	}
	switch req.EngineStrategy {
	case ModuleStrategyDevReplace:
		opts.Strategy = modulepath.StrategyDevReplace
	case ModuleStrategyPublishedVersion:
		opts.Strategy = modulepath.StrategyPublishedVersion
	}
	return opts
}

// hostLongBuilder runs the full pipeline for the host's native
// target: codegen.Generate → (Windows only) BuildWindowsSyso →
// `go build -tags=long -o <out>` from the generated outDir. On
// success, the resulting binary is copied/renamed into
// req.OutputDir/host/<gameName><ext> so the orchestrator's
// reported OutputPath is stable across platforms.
type hostLongBuilder struct{}

func (b *hostLongBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	target := HostTarget()

	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "pixelforge-host-build-*")
	if err != nil {
		return fmt.Errorf("hostLongBuilder: mkdir temp: %w", err)
	}
	// tempDir is preserved on failure for post-mortem inspection
	// (the failing source is more useful than a clean tree). On
	// success the orchestrator's outer goroutine doesn't currently
	// sweep it; a future polish unit can wire that in if disk
	// usage becomes an issue.

	if _, err := codegen.Generate(req.Project, tempDir, codegenOptionsFor(req, req.ProjectPath)); err != nil {
		return fmt.Errorf("hostLongBuilder: generate: %w", err)
	}

	// Windows host: drop the .syso so the Go linker picks up the
	// brand icon when `go build` runs.
	if target == TargetWindows {
		version := ""
		if req.Project != nil {
			version = req.Project.Version
		}
		gameName := projectGameName(req.Project)
		if err := BuildWindowsSyso(tempDir, gameName, version, "amd64"); err != nil {
			return fmt.Errorf("hostLongBuilder: windows syso: %w", err)
		}
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhaseCompiling, BuiltAt: time.Now()})

	outDir := filepath.Join(req.OutputDir, "host")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("hostLongBuilder: mkdir output: %w", err)
	}
	gameName := projectGameName(req.Project)
	outPath := filepath.Join(outDir, gameName+artifactExt(target))

	// macOS .app packaging keeps the binary at <name>.app/Contents/MacOS/<name>.
	binaryPath := outPath
	if target == TargetMacOS {
		binaryPath = filepath.Join(outDir, gameName+".app", "Contents", "MacOS", gameName)
		if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
			return fmt.Errorf("hostLongBuilder: mkdir .app: %w", err)
		}
	}

	cmd, err := NewBuildCommand(ctx, target, "build", "-tags=long", "-o", binaryPath, ".")
	if err != nil {
		return fmt.Errorf("hostLongBuilder: build command: %w", err)
	}
	cmd.Dir = tempDir
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrBuildCancelled
		}
		return fmt.Errorf("hostLongBuilder: go build failed: %w\noutput:\n%s", runErr, string(combined))
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhasePackaging, BuiltAt: time.Now()})

	// macOS finalisation: drop the .icns into Contents/Resources/
	// and a minimal Info.plist next to it so Finder shows the
	// branded icon.
	if target == TargetMacOS {
		resourcesDir := filepath.Join(outDir, gameName+".app", "Contents", "Resources")
		if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
			return fmt.Errorf("hostLongBuilder: mkdir Resources: %w", err)
		}
		icnsBytes, err := BuildLogoICNS()
		if err != nil {
			return fmt.Errorf("hostLongBuilder: build icns: %w", err)
		}
		if err := os.WriteFile(filepath.Join(resourcesDir, "AppIcon.icns"), icnsBytes, 0o644); err != nil {
			return fmt.Errorf("hostLongBuilder: write AppIcon.icns: %w", err)
		}
		plistPath := filepath.Join(outDir, gameName+".app", "Contents", "Info.plist")
		plist := macInfoPlist(gameName, req.Project.Version)
		if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
			return fmt.Errorf("hostLongBuilder: write Info.plist: %w", err)
		}
		// outPath for macOS is the .app bundle directory.
	}

	emit(BuildStatus{
		Phase:      PhaseDone,
		OutputPath: outPath,
		BuiltAt:    time.Now(),
	})
	return nil
}

// wasmLongBuilder runs codegen.Generate, GOOS=js GOARCH=wasm
// go build, then BundleWASM to wrap the resulting .wasm into a
// single self-contained .html.
type wasmLongBuilder struct{}

func (b *wasmLongBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "pixelforge-wasm-build-*")
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: mkdir temp: %w", err)
	}
	if _, err := codegen.Generate(req.Project, tempDir, codegenOptionsFor(req, req.ProjectPath)); err != nil {
		return fmt.Errorf("wasmLongBuilder: generate: %w", err)
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhaseCompiling, BuiltAt: time.Now()})

	wasmTempPath := filepath.Join(tempDir, "game.wasm")
	cmd, err := NewBuildCommand(ctx, TargetWASM, "build", "-tags=long", "-o", wasmTempPath, ".")
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: build command: %w", err)
	}
	cmd.Dir = tempDir
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ErrBuildCancelled
		}
		return fmt.Errorf("wasmLongBuilder: go build failed: %w\noutput:\n%s", runErr, string(combined))
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhasePackaging, BuiltAt: time.Now()})

	wasmExecPath, err := ResolveWasmExecJS()
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: resolve wasm_exec.js: %w", err)
	}
	favicon, err := GenerateFavicon()
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: favicon: %w", err)
	}

	outDir := filepath.Join(req.OutputDir, "wasm")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("wasmLongBuilder: mkdir output: %w", err)
	}
	gameName := projectGameName(req.Project)
	outPath := filepath.Join(outDir, gameName+".html")
	if err := BundleWASM(wasmTempPath, wasmExecPath, gameName, favicon, outPath, req.Credits); err != nil {
		return fmt.Errorf("wasmLongBuilder: bundle: %w", err)
	}

	emit(BuildStatus{
		Phase:      PhaseDone,
		OutputPath: outPath,
		BuiltAt:    time.Now(),
	})
	return nil
}

// macInfoPlist returns a minimal Info.plist that names the
// branded icon. Real production polish (LSUIElement, bundle ID,
// CFBundleVersion semantics) is deferred to a follow-up packaging
// pass.
func macInfoPlist(gameName, version string) string {
	if version == "" {
		version = "1.0"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>` + gameName + `</string>
    <key>CFBundleExecutable</key>
    <string>` + gameName + `</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleShortVersionString</key>
    <string>` + version + `</string>
    <key>CFBundleVersion</key>
    <string>` + version + `</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
</dict>
</plist>
`
}

func init() {
	// Under //go:build long the scaffold builders in builders.go
	// are excluded; this init() owns the entire builder registry.
	// We register the long builders for the host's native target
	// and WASM (the only two targets the two-button Build UI
	// surfaces) plus the scaffold for Source so codegen-only test
	// flows that exercise the orchestrator still pass.
	RegisterBuilder(HostTarget(), &hostLongBuilder{})
	RegisterBuilder(TargetWASM, &wasmLongBuilder{})
	RegisterBuilder(TargetSource, &sourceLongBuilder{})

	// Foreign-OS native targets get a builder that always rejects —
	// the orchestrator's preflight (CanBuildOnHost) already rejects
	// them before dispatch, but a registered builder makes the
	// failure obvious if a test or downstream caller bypasses the
	// preflight via direct LookupBuilder.
	for _, t := range []Target{TargetWindows, TargetMacOS, TargetLinux} {
		if t == HostTarget() {
			continue
		}
		RegisterBuilder(t, &rejectingBuilder{target: t})
	}
}

// sourceLongBuilder is the codegen-only path. No `go build`
// invocation — the output is the generated source tree itself.
type sourceLongBuilder struct{}

func (b *sourceLongBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	outDir := filepath.Join(req.OutputDir, "source")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("sourceLongBuilder: mkdir: %w", err)
	}
	opts := codegenOptionsFor(req, req.ProjectPath)
	opts.RunGoModTidy = false // source target just generates files
	if _, err := codegen.Generate(req.Project, outDir, opts); err != nil {
		return fmt.Errorf("sourceLongBuilder: generate: %w", err)
	}
	emit(BuildStatus{Phase: PhaseDone, OutputPath: outDir, BuiltAt: time.Now()})
	return nil
}

// rejectingBuilder is the long-tag stand-in for cross-OS native
// targets. Builds always emit PhaseFailed with a clear message;
// callers reaching this path bypassed the orchestrator's
// preflight.
type rejectingBuilder struct {
	target Target
}

func (b *rejectingBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	emit(BuildStatus{
		Phase:   PhaseFailed,
		Err:     &CrossCompileNotSupportedError{Target: b.target, HostOS: runtime.GOOS},
		BuiltAt: time.Now(),
	})
	return &CrossCompileNotSupportedError{Target: b.target, HostOS: runtime.GOOS}
}
