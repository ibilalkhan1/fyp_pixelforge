package buildpipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Target names the platform a build is producing. Drives env-var
// composition (GOOS/GOARCH) plus per-target packaging logic.
type Target int

const (
	TargetWindows Target = iota
	TargetMacOS
	TargetLinux
	TargetWASM
	TargetSource
)

// String returns the lowercase identifier the UI + output paths
// share. Filesystem-safe.
func (t Target) String() string {
	switch t {
	case TargetWindows:
		return "windows"
	case TargetMacOS:
		return "macos"
	case TargetLinux:
		return "linux"
	case TargetWASM:
		return "wasm"
	case TargetSource:
		return "source"
	default:
		return "unknown"
	}
}

// DisplayName is the user-facing label the Build workspace renders.
func (t Target) DisplayName() string {
	switch t {
	case TargetWindows:
		return "Windows .exe"
	case TargetMacOS:
		return "macOS .app"
	case TargetLinux:
		return "Linux binary"
	case TargetWASM:
		return "WASM single-HTML"
	case TargetSource:
		return "Source"
	default:
		return "Unknown target"
	}
}

// GOOS returns the GOOS env-var value for go build invocations.
// Source target returns "" (no compile).
func (t Target) GOOS() string {
	switch t {
	case TargetWindows:
		return "windows"
	case TargetMacOS:
		return "darwin"
	case TargetLinux:
		return "linux"
	case TargetWASM:
		return "js"
	}
	return ""
}

// GOARCH returns the GOARCH env-var value for go build. macOS is
// arm64-only in v1 (Intel users use Rosetta per origin scope).
func (t Target) GOARCH() string {
	switch t {
	case TargetWindows, TargetLinux:
		return "amd64"
	case TargetMacOS:
		return "arm64"
	case TargetWASM:
		return "wasm"
	}
	return ""
}

// AllNonSourceTargets returns every target except Source. The
// Build workspace dispatches these to the toolchain helper; Source
// short-circuits the toolchain entirely.
var AllNonSourceTargets = []Target{
	TargetWindows, TargetMacOS, TargetLinux, TargetWASM,
}

// AllTargets includes Source.
var AllTargets = []Target{
	TargetWindows, TargetMacOS, TargetLinux, TargetWASM, TargetSource,
}

var (
	executableDir     string
	executableDirOnce sync.Once
)

// SetExecutableDir overrides the auto-detected directory. Used by
// tests to point ResolveGoBinary at a temp dir's go-sdk shim.
func SetExecutableDir(dir string) {
	executableDirOnce.Do(func() {})
	executableDir = dir
}

// executableDirOrAuto returns the override if set, otherwise
// derives from os.Executable.
func executableDirOrAuto() string {
	if executableDir != "" {
		return executableDir
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// ResolveGoBinary returns the absolute path to the Go toolchain to
// invoke. Prefers a vendored SDK under <executableDir>/go-sdk/bin
// (shipped by the studio installer); falls back to exec.LookPath
// for source-build deployments where the developer has Go on PATH.
func ResolveGoBinary() (string, error) {
	binName := "go"
	if runtime.GOOS == "windows" {
		binName = "go.exe"
	}
	vendored := filepath.Join(executableDirOrAuto(), "go-sdk", "bin", binName)
	if info, err := os.Stat(vendored); err == nil && !info.IsDir() {
		return vendored, nil
	}
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}
	return "", errors.New("buildpipeline: Go toolchain not found (no vendored SDK + no `go` on PATH)")
}

// ResolveGoRoot returns the toolchain's GOROOT. Vendored SDKs sit
// at <executableDir>/go-sdk; PATH installs are queried via
// `go env GOROOT`.
func ResolveGoRoot() (string, error) {
	vendored := filepath.Join(executableDirOrAuto(), "go-sdk")
	if info, err := os.Stat(vendored); err == nil && info.IsDir() {
		return vendored, nil
	}
	bin, err := ResolveGoBinary()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(bin, "env", "GOROOT").Output()
	if err != nil {
		return "", fmt.Errorf("buildpipeline: `go env GOROOT` failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ResolveWasmExecJS returns the path to wasm_exec.js inside the
// active GOROOT. Go 1.24+ puts it under lib/wasm; earlier versions
// under misc/wasm. The function tries both, returns the first hit.
func ResolveWasmExecJS() (string, error) {
	root, err := ResolveGoRoot()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(root, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(root, "misc", "wasm", "wasm_exec.js"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("buildpipeline: wasm_exec.js not found under %s", root)
}

// NewBuildCommand wraps exec.CommandContext with the env vars +
// argv shape every per-target builder uses. Filters out user-set
// GOTOOLCHAIN / GOPATH overrides so the build behaves predictably
// regardless of the developer's local setup.
//
// The Source target shouldn't call NewBuildCommand — it generates
// files without invoking the toolchain.
func NewBuildCommand(ctx context.Context, target Target, args ...string) (*exec.Cmd, error) {
	bin, err := ResolveGoBinary()
	if err != nil {
		return nil, err
	}
	root, err := ResolveGoRoot()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = filteredEnv()
	cmd.Env = appendEnv(cmd.Env, "GOROOT", root)
	if target.GOOS() != "" {
		cmd.Env = appendEnv(cmd.Env, "GOOS", target.GOOS())
	}
	if target.GOARCH() != "" {
		cmd.Env = appendEnv(cmd.Env, "GOARCH", target.GOARCH())
	}
	// CGO selection differs per target: Ebitengine's desktop path
	// needs CGO_ENABLED=1 to link the native graphics + audio
	// backends; the WASM path requires CGO_ENABLED=0 because the
	// js/wasm toolchain doesn't accept cgo. Source-only and unknown
	// targets default to 0 — they don't compile native code at all.
	cmd.Env = appendEnv(cmd.Env, "CGO_ENABLED", cgoEnabledFor(target))
	return cmd, nil
}

// cgoEnabledFor returns "1" for Host-native targets (which need
// Ebitengine's cgo bindings) and "0" everywhere else.
func cgoEnabledFor(t Target) string {
	switch t {
	case TargetWindows, TargetMacOS, TargetLinux:
		return "1"
	}
	return "0"
}

// filteredEnv returns os.Environ() minus GOTOOLCHAIN and GOPATH so
// the build's toolchain selection is fully deterministic from the
// resolved binary + GOROOT.
func filteredEnv() []string {
	original := os.Environ()
	out := make([]string, 0, len(original))
	for _, kv := range original {
		if strings.HasPrefix(kv, "GOTOOLCHAIN=") {
			continue
		}
		if strings.HasPrefix(kv, "GOPATH=") {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// appendEnv replaces any existing entry for key with key=value.
func appendEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
