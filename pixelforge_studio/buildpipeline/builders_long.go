// builders_long.go is the //go:build long real-build path. Built
// with `-tags=long` the studio (and the long-tag CI job) replaces
// the placeholder scaffolds in builders.go with builders that
// actually produce runnable artifacts.
//
// Plan-009 U3 rewired both the host and WASM builders: instead of
// emitting a per-cart main.go + invoking `go build`, they call
// codegen.EncodeCart to produce the cart payload bytes, locate the
// matching pre-built pixelforge-player binary via the discovery
// chain in builders_long_helpers.go, and either append the cart
// onto a copy of the player (host) or inline both the WASM module
// and the cart's base64 into the HTML shell (WASM target). The
// effect: no Go toolchain is required at user-build time when the
// studio installer shipped with embedded player binaries.
//
// Production callers — the studio's Build button — apply the tag
// automatically so users don't need to know about it. The default
// test suite stays headless (no Go toolchain dependency) because
// builders.go is `//go:build !long`.

//go:build long

package buildpipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cart"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/codegen"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/modulepath"
)

// codegenOptionsFor builds codegen.Options carrying the engine
// override fields the long builders care about. Maps the build-
// pipeline-side ModuleStrategy enum to modulepath.Strategy. The
// host + WASM builders use this only for the sourceLongBuilder
// (the "Source" target still emits a per-cart Go tree); the
// universal-player path no longer touches codegen.Options.
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

// hostLongBuilder runs the universal-player + cart-append pipeline
// for the host's native target: codegen.EncodeCart → resolve player
// binary via the discovery chain (explicit override / cache hit /
// embedded / developer go-build) → (Windows only) BuildWindowsSyso
// onto a player-binary copy → (macOS only) materialise the .app
// skeleton → pixelforge_cart.Append. Output lands at
// req.OutputDir/host/<gameName>[.exe|.app].
type hostLongBuilder struct{}

func (b *hostLongBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	target := HostTarget()

	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	cartBytes, err := codegen.EncodeCart(req.Project)
	if err != nil {
		return fmt.Errorf("hostLongBuilder: encode cart: %w", err)
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhaseCompiling, BuiltAt: time.Now()})

	// "Compiling" now covers the player-binary discovery: cache
	// hits + embed extracts are fast; the developer `go build`
	// fallback is slow. Either way the user sees the phase fire,
	// preserving the orchestrator's UX contract.
	resolved, err := resolvePlayerBinary(ctx, req, target)
	if err != nil {
		return fmt.Errorf("hostLongBuilder: resolve player binary: %w", err)
	}
	defer resolved.Cleanup()

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhasePackaging, BuiltAt: time.Now()})

	outDir := filepath.Join(req.OutputDir, "host")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("hostLongBuilder: mkdir output: %w", err)
	}
	gameName := projectGameName(req.Project)
	outPath := filepath.Join(outDir, gameName+artifactExt(target))

	// Windows: bake the icon .syso into a staged copy of the player
	// binary BEFORE appending the cart. The Go linker already ran
	// when the player binary was built upstream; the .syso path
	// from plan-008 needs the resource to be visible at link time,
	// which means we have to embed it via a different mechanism
	// for an already-linked binary. For U3 we re-link the player
	// via go build when the target is Windows — slower than the
	// pure-copy path but produces a branded .exe. Future polish
	// could swap to a post-link resource injector.
	playerPath := resolved.Path
	if target == TargetWindows {
		relinked, cleanup, relinkErr := relinkWindowsPlayerWithIcon(ctx, req)
		if relinkErr != nil {
			return fmt.Errorf("hostLongBuilder: windows relink with icon: %w", relinkErr)
		}
		defer cleanup()
		playerPath = relinked
	}

	// macOS .app bundle: append into the .app directory and let
	// pixelforge_cart.Append resolve the inner Mach-O path
	// (Foo.app/Contents/MacOS/Foo). The .icns + Info.plist drop in
	// AFTER the append so codesign sees a complete bundle.
	appendTarget := outPath
	if target == TargetMacOS {
		if err := os.MkdirAll(filepath.Join(outDir, gameName+".app", "Contents", "MacOS"), 0o755); err != nil {
			return fmt.Errorf("hostLongBuilder: mkdir .app: %w", err)
		}
	}

	if err := pixelforge_cart.Append(playerPath, cartBytes, appendTarget); err != nil {
		return fmt.Errorf("hostLongBuilder: append cart: %w", err)
	}

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
		version := ""
		if req.Project != nil {
			version = req.Project.Version
		}
		plistPath := filepath.Join(outDir, gameName+".app", "Contents", "Info.plist")
		plist := macInfoPlist(gameName, version)
		if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
			return fmt.Errorf("hostLongBuilder: write Info.plist: %w", err)
		}
	}

	emit(BuildStatus{
		Phase:      PhaseDone,
		OutputPath: outPath,
		BuiltAt:    time.Now(),
	})
	return nil
}

// relinkWindowsPlayerWithIcon is the U3 stopgap for the Windows
// icon problem: the resolved player binary is already linked, so
// dropping a .syso next to it doesn't change the .exe's embedded
// resources. We invoke `go build -tags=long` against
// cmd/pixelforge-player WITH a fresh syso in the source tree so
// the Go linker picks it up. Returns the relinked path + a
// cleanup func the caller defers.
//
// This costs the no-Go-user-on-Windows path a Go toolchain
// dependency at build time. A follow-up unit can swap to a
// post-link resource injector (e.g. github.com/akavel/rsrc).
func relinkWindowsPlayerWithIcon(ctx context.Context, req BuildRequest) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "pixelforge-windows-relink-*")
	if err != nil {
		return "", noopCleanup, fmt.Errorf("relinkWindowsPlayerWithIcon: mkdir temp: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	gameName := projectGameName(req.Project)
	version := ""
	if req.Project != nil {
		version = req.Project.Version
	}
	if err := BuildWindowsSyso(tmpDir, gameName, version, "amd64"); err != nil {
		cleanup()
		return "", noopCleanup, fmt.Errorf("relinkWindowsPlayerWithIcon: build syso: %w", err)
	}

	outPath := filepath.Join(tmpDir, "pixelforge-player.exe")
	cmd, err := NewBuildCommand(ctx, TargetWindows, "build", "-tags=long", "-o", outPath,
		"github.com/ibilalkhan1/fyp_pixelforge/cmd/pixelforge-player")
	if err != nil {
		cleanup()
		return "", noopCleanup, fmt.Errorf("relinkWindowsPlayerWithIcon: build command: %w", err)
	}
	combined, runErr := cmd.CombinedOutput()
	if runErr != nil {
		cleanup()
		return "", noopCleanup, fmt.Errorf("relinkWindowsPlayerWithIcon: go build failed: %w\noutput:\n%s", runErr, string(combined))
	}
	return outPath, cleanup, nil
}

// wasmLongBuilder runs the universal-player + inline-cart pipeline
// for the WASM target: codegen.EncodeCart → resolve the GOOS=js
// player WASM module via the discovery chain → BundleWASM inlines
// both the WASM and the cart base64 (plus the click-to-start
// splash that wires window.__pixelforgeCart before go.run) into a
// single self-contained .html file.
type wasmLongBuilder struct{}

func (b *wasmLongBuilder) Build(ctx context.Context, req BuildRequest, emit func(BuildStatus)) error {
	emit(BuildStatus{Phase: PhaseGenerating, BuiltAt: time.Now()})
	if err := contextCheck(ctx); err != nil {
		return err
	}

	cartBytes, err := codegen.EncodeCart(req.Project)
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: encode cart: %w", err)
	}

	if err := contextCheck(ctx); err != nil {
		return err
	}
	emit(BuildStatus{Phase: PhaseCompiling, BuiltAt: time.Now()})

	resolved, err := resolvePlayerBinary(ctx, req, TargetWASM)
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: resolve wasm player: %w", err)
	}
	defer resolved.Cleanup()

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

	// Plan-009 U23: BundleWASMWithSize also writes the
	// gzip-compressed sibling, optionally pre-optimises via
	// wasm-opt, and returns raw + gzip sizes so the toast can
	// name the cost. wasm-opt is silently skipped when absent;
	// gzip is always written.
	report, err := BundleWASMWithSize(
		resolved.Path, wasmExecPath, cartBytes, gameName, favicon,
		outPath, req.Credits,
		BundleWASMOptions{WriteGzipSibling: true, OptimizeWasm: true},
	)
	if err != nil {
		return fmt.Errorf("wasmLongBuilder: bundle: %w", err)
	}

	// Hard-cap gate: 30MB raw HTML fails the build unless the
	// caller flipped ForceLargeWASM. The error carries the
	// report so the failure toast names the exact byte count.
	if report.ExceededError() && !req.ForceLargeWASM {
		return &WASMTooLargeError{Report: report}
	}

	emit(BuildStatus{
		Phase:      PhaseDone,
		OutputPath: outPath,
		BuiltAt:    time.Now(),
		SizeReport: &report,
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
// Preserved for users who want the editable Go source from the
// "Source" target; the host + WASM builders no longer touch this
// codepath.
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
