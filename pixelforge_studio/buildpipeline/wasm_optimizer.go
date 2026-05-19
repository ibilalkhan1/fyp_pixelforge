package buildpipeline

import (
	"errors"
	"fmt"
	"os/exec"
)

// Plan-009 U23 — optional wasm-opt invocation.
//
// `wasm-opt` is part of the Binaryen toolchain. When present on
// the build host it can shrink the GOOS=js/GOARCH=wasm module 20-
// 30% with -Oz (size-first optimisation) while preserving
// semantics. We do NOT require it; the bundler emits a working
// .html either way. The studio UX is "silent skip when absent" —
// users who care about size install Binaryen themselves; users
// who don't never see a warning.

// wasmOptLookPath is the seam tests stub. Production code uses
// exec.LookPath; tests can swap it for a stub that returns a
// known path (or a known error).
var wasmOptLookPath = exec.LookPath

// wasmOptRunCommand is the seam tests stub. Production code wraps
// exec.Command + Run; tests can record invocations without
// actually executing wasm-opt.
var wasmOptRunCommand = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("buildpipeline: wasm-opt failed: %w (output: %s)", err, string(out))
	}
	return nil
}

// IsWasmOptAvailable reports whether the `wasm-opt` binary is
// resolvable on PATH. Cheap (PATH walk + stat) so callers can
// invoke per-build without caching.
//
// Returns false on any LookPath error — that includes both "not
// installed" and "installed but unreadable", which the caller
// shouldn't distinguish: in either case we skip optimization.
func IsWasmOptAvailable() bool {
	_, err := wasmOptLookPath("wasm-opt")
	return err == nil
}

// OptimizeWasm runs `wasm-opt -Oz --enable-bulk-memory <inPath>
// -o <outPath>`. Returns nil on success, a wrapped error on
// failure. Caller decides what to do on error — typically log it
// and fall through to the unoptimized input (the bundler still
// produces a working artifact).
//
// -Oz: size-first optimisation. -O3 is faster code but bigger
// binary; -Oz is what we want for the WASM-over-the-wire path.
// --enable-bulk-memory: Go 1.21+'s WASM output uses bulk memory
// operations; without the flag wasm-opt rejects the module with
// a parse error.
//
// Both inPath and outPath must be non-empty. Empty paths return
// an error rather than letting wasm-opt produce a confusing
// "could not open" diagnostic.
func OptimizeWasm(inPath, outPath string) error {
	if inPath == "" {
		return errors.New("buildpipeline.OptimizeWasm: inPath is empty")
	}
	if outPath == "" {
		return errors.New("buildpipeline.OptimizeWasm: outPath is empty")
	}
	if _, err := wasmOptLookPath("wasm-opt"); err != nil {
		return fmt.Errorf("buildpipeline.OptimizeWasm: wasm-opt not found on PATH: %w", err)
	}
	return wasmOptRunCommand("wasm-opt", "-Oz", "--enable-bulk-memory", inPath, "-o", outPath)
}
