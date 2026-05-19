package buildpipeline

// Internal test (package buildpipeline, not _test) so we can poke
// the wasmOptLookPath / wasmOptRunCommand seams without exporting
// them. Tests that only care about the public API live in
// wasm_optimizer_external_test.go via package buildpipeline_test.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withStubbedWasmOpt swaps both seams for the supplied stubs and
// restores the originals on cleanup. Tests use this to simulate
// "present + works", "absent", and "present but fails" without
// touching the real PATH.
func withStubbedWasmOpt(t *testing.T, lookPath func(string) (string, error), run func(string, ...string) error) {
	t.Helper()
	origLook := wasmOptLookPath
	origRun := wasmOptRunCommand
	wasmOptLookPath = lookPath
	wasmOptRunCommand = run
	t.Cleanup(func() {
		wasmOptLookPath = origLook
		wasmOptRunCommand = origRun
	})
}

func TestIsWasmOptAvailable_PresentReturnsTrue(t *testing.T) {
	withStubbedWasmOpt(t,
		func(name string) (string, error) {
			assert.Equal(t, "wasm-opt", name)
			return "/usr/local/bin/wasm-opt", nil
		},
		nil,
	)
	assert.True(t, IsWasmOptAvailable())
}

func TestIsWasmOptAvailable_AbsentReturnsFalse(t *testing.T) {
	withStubbedWasmOpt(t,
		func(string) (string, error) { return "", errors.New("not found") },
		nil,
	)
	assert.False(t, IsWasmOptAvailable())
}

// TestOptimizeWasm_InvokesCorrectArgs covers the exact command
// shape: `wasm-opt -Oz --enable-bulk-memory <in> -o <out>`. The
// stub captures the args; the test asserts on them.
func TestOptimizeWasm_InvokesCorrectArgs(t *testing.T) {
	var gotName string
	var gotArgs []string
	withStubbedWasmOpt(t,
		func(string) (string, error) { return "/fake/wasm-opt", nil },
		func(name string, args ...string) error {
			gotName = name
			gotArgs = args
			return nil
		},
	)
	require.NoError(t, OptimizeWasm("/tmp/in.wasm", "/tmp/out.wasm"))
	assert.Equal(t, "wasm-opt", gotName)
	assert.Equal(t, []string{"-Oz", "--enable-bulk-memory", "/tmp/in.wasm", "-o", "/tmp/out.wasm"}, gotArgs)
}

func TestOptimizeWasm_EmptyInPathReturnsError(t *testing.T) {
	err := OptimizeWasm("", "/tmp/out.wasm")
	require.Error(t, err)
}

func TestOptimizeWasm_EmptyOutPathReturnsError(t *testing.T) {
	err := OptimizeWasm("/tmp/in.wasm", "")
	require.Error(t, err)
}

// TestOptimizeWasm_AbsentReturnsError covers the case where a
// caller invokes OptimizeWasm without first checking
// IsWasmOptAvailable — they get a clear "not on PATH" error.
func TestOptimizeWasm_AbsentReturnsError(t *testing.T) {
	withStubbedWasmOpt(t,
		func(string) (string, error) { return "", errors.New("not found") },
		nil,
	)
	err := OptimizeWasm("/tmp/in.wasm", "/tmp/out.wasm")
	require.Error(t, err)
}

// TestOptimizeWasm_RunFailureWrapped covers the case where wasm-opt
// is present but the invocation itself fails (e.g., invalid wasm
// module). Caller gets the wrapped error.
func TestOptimizeWasm_RunFailureWrapped(t *testing.T) {
	withStubbedWasmOpt(t,
		func(string) (string, error) { return "/fake/wasm-opt", nil },
		func(string, ...string) error { return errors.New("bad module") },
	)
	err := OptimizeWasm("/tmp/in.wasm", "/tmp/out.wasm")
	require.Error(t, err)
}
