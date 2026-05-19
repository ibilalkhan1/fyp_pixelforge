package buildpipeline_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func TestTarget_StringAndDisplayName(t *testing.T) {
	cases := []struct {
		t            buildpipeline.Target
		name, displ  string
	}{
		{buildpipeline.TargetWindows, "windows", "Windows .exe"},
		{buildpipeline.TargetMacOS, "macos", "macOS .app"},
		{buildpipeline.TargetLinux, "linux", "Linux binary"},
		{buildpipeline.TargetWASM, "wasm", "WASM single-HTML"},
		{buildpipeline.TargetSource, "source", "Source"},
	}
	for _, c := range cases {
		assert.Equal(t, c.name, c.t.String())
		assert.Equal(t, c.displ, c.t.DisplayName())
	}
}

func TestTarget_GOOSAndGOARCH(t *testing.T) {
	cases := []struct {
		t     buildpipeline.Target
		os, a string
	}{
		{buildpipeline.TargetWindows, "windows", "amd64"},
		{buildpipeline.TargetMacOS, "darwin", "arm64"},
		{buildpipeline.TargetLinux, "linux", "amd64"},
		{buildpipeline.TargetWASM, "js", "wasm"},
		{buildpipeline.TargetSource, "", ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.os, c.t.GOOS(), c.t.DisplayName())
		assert.Equal(t, c.a, c.t.GOARCH())
	}
}

func TestAllNonSourceTargetsExcludesSource(t *testing.T) {
	for _, tg := range buildpipeline.AllNonSourceTargets {
		assert.NotEqual(t, buildpipeline.TargetSource, tg)
	}
	assert.Len(t, buildpipeline.AllNonSourceTargets, 4)
}

func TestAllTargetsIncludesSource(t *testing.T) {
	found := false
	for _, tg := range buildpipeline.AllTargets {
		if tg == buildpipeline.TargetSource {
			found = true
		}
	}
	assert.True(t, found)
	assert.Len(t, buildpipeline.AllTargets, 5)
}

func TestResolveGoBinary_PrefersVendoredSDK(t *testing.T) {
	// Create a temp dir with a go-sdk/bin/go shim.
	dir := t.TempDir()
	sdkBin := filepath.Join(dir, "go-sdk", "bin")
	require.NoError(t, os.MkdirAll(sdkBin, 0o755))
	binName := "go"
	if runtime.GOOS == "windows" {
		binName = "go.exe"
	}
	shimPath := filepath.Join(sdkBin, binName)
	require.NoError(t, os.WriteFile(shimPath, []byte("#!/bin/sh\necho mock\n"), 0o755))

	buildpipeline.SetExecutableDir(dir)
	t.Cleanup(func() { buildpipeline.SetExecutableDir("") })

	got, err := buildpipeline.ResolveGoBinary()
	require.NoError(t, err)
	assert.Equal(t, shimPath, got)
}

func TestResolveGoBinary_FallsBackToPATH(t *testing.T) {
	buildpipeline.SetExecutableDir(t.TempDir()) // empty — no vendored SDK
	t.Cleanup(func() { buildpipeline.SetExecutableDir("") })

	got, err := buildpipeline.ResolveGoBinary()
	require.NoError(t, err)
	assert.NotEmpty(t, got, "PATH fallback resolves to the test runner's Go binary")
}

func TestResolveGoRoot_ReturnsNonEmpty(t *testing.T) {
	root, err := buildpipeline.ResolveGoRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
}

func TestResolveGoRoot_PrefersVendoredSDK(t *testing.T) {
	dir := t.TempDir()
	sdkDir := filepath.Join(dir, "go-sdk")
	require.NoError(t, os.MkdirAll(sdkDir, 0o755))
	buildpipeline.SetExecutableDir(dir)
	t.Cleanup(func() { buildpipeline.SetExecutableDir("") })

	got, err := buildpipeline.ResolveGoRoot()
	require.NoError(t, err)
	assert.Equal(t, sdkDir, got)
}

func TestNewBuildCommand_SetsGOOSAndGOARCHFromTarget(t *testing.T) {
	cmd, err := buildpipeline.NewBuildCommand(context.Background(), buildpipeline.TargetWindows, "version")
	require.NoError(t, err)
	assert.Contains(t, cmd.Env, "GOOS=windows")
	assert.Contains(t, cmd.Env, "GOARCH=amd64")
}

// TestNewBuildCommand_HostNativeTargetsGetCGOEnabled covers the
// plan-008 U3 invariant: Ebitengine's desktop path links native
// graphics/audio backends through cgo, so Windows/macOS/Linux
// builds must set CGO_ENABLED=1.
func TestNewBuildCommand_HostNativeTargetsGetCGOEnabled(t *testing.T) {
	for _, target := range []buildpipeline.Target{
		buildpipeline.TargetWindows, buildpipeline.TargetMacOS, buildpipeline.TargetLinux,
	} {
		cmd, err := buildpipeline.NewBuildCommand(context.Background(), target, "version")
		require.NoError(t, err)
		assert.Contains(t, cmd.Env, "CGO_ENABLED=1",
			"target %s must enable cgo (Ebitengine native backend)", target)
	}
}

// TestNewBuildCommand_WASMDisablesCGO: js/wasm toolchain doesn't
// accept cgo, so the WASM target must set CGO_ENABLED=0.
func TestNewBuildCommand_WASMDisablesCGO(t *testing.T) {
	cmd, err := buildpipeline.NewBuildCommand(context.Background(), buildpipeline.TargetWASM, "version")
	require.NoError(t, err)
	assert.Contains(t, cmd.Env, "CGO_ENABLED=0")
}

func TestNewBuildCommand_DropsUserGOTOOLCHAIN(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "local")
	cmd, err := buildpipeline.NewBuildCommand(context.Background(), buildpipeline.TargetLinux, "version")
	require.NoError(t, err)
	for _, kv := range cmd.Env {
		assert.NotContains(t, kv, "GOTOOLCHAIN=")
	}
}

func TestNewBuildCommand_SetsGOROOT(t *testing.T) {
	cmd, err := buildpipeline.NewBuildCommand(context.Background(), buildpipeline.TargetLinux, "version")
	require.NoError(t, err)
	hasGOROOT := false
	for _, kv := range cmd.Env {
		if len(kv) > 7 && kv[:7] == "GOROOT=" {
			hasGOROOT = true
			break
		}
	}
	assert.True(t, hasGOROOT, "NewBuildCommand stamps GOROOT into the env")
}

func TestNewBuildCommand_SourceTargetSkipsGOOSGOARCH(t *testing.T) {
	cmd, err := buildpipeline.NewBuildCommand(context.Background(), buildpipeline.TargetSource, "version")
	require.NoError(t, err)
	for _, kv := range cmd.Env {
		assert.NotEqual(t, "GOOS=", kv[:min(len(kv), 5)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
